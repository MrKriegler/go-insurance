package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	transporthttp "github.com/MrKriegler/go-insurance/internal/http"
	"github.com/MrKriegler/go-insurance/internal/http/handlers"
	"github.com/MrKriegler/go-insurance/internal/platform/config"
	"github.com/MrKriegler/go-insurance/internal/platform/logging"
	"github.com/MrKriegler/go-insurance/internal/store/mongo"
)

func main() {
	// ---- Config & Logger ----
	cfg := config.MustLoad()
	log := logging.New(cfg.Env)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Info("Starting server on %s in %s mode", addr, cfg.Env)

	// Root context with cancel on SIGINT/SIGTERM for graceful shutdown
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- Mongo ----
	log.Info("Connecting to MongoDB at %s / DB: %s", cfg.MongoURI, cfg.MongoDB)
	mc, err := mongo.NewClient(cfg) // must return a struct with fields {Client *mongo.Client; DB *mongo.Database}
	if err != nil {
		log.Error("Failed to connect to MongoDB: %v", err)
		os.Exit(1)
	}
	defer mc.Close(context.Background())
	log.Info("Connected to MongoDB")

	// Ensure indexes (idempotent)
	{
		ctx, cancel := context.WithTimeout(rootCtx, 15*time.Second)
		defer cancel()
		if err := mongo.EnsureIndexes(ctx, mc.DB); err != nil {
			log.Error("ensure indexes: %v", err)
			os.Exit(1)
		}
	}

	// ---- Build empty handlers (no services yet) ----
	productsH := handlers.NewProductHandler()
	quotesH := handlers.NewQuoteHandler()
	appsH := handlers.NewApplicationHandler()
	uwH := handlers.NewUWHandler()
	offersH := handlers.NewOfferHandler()
	policiesH := handlers.NewPolicyHandler()

	// ---- Outer router (health, readiness, then mount API) ----
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(time.Duration(cfg.HTTPRequestTimeoutSec) * time.Second))

	// Liveness probe: process up
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Readiness probe: DB reachable within a tight SLA
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		ctx, cancel := context.WithTimeout(rootCtx, time.Duration(cfg.MongoOpTimeoutMs)*time.Millisecond)
		defer cancel()
		if err := mc.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	apiRouter := transporthttp.NewRouter(transporthttp.Deps{
		Mounts: []handlers.Mountable{
			productsH, quotesH, appsH, uwH, offersH, policiesH,
		},
	})
	r.Mount("/api/v1", apiRouter)

	// ---- HTTP Server ----
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       time.Duration(cfg.HTTPReadTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(cfg.HTTPWriteTimeoutSec) * time.Second,
		IdleTimeout:       time.Duration(cfg.HTTPIdleTimeoutSec) * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("Server started and listening on %s", addr)
		errCh <- srv.ListenAndServe()
	}()

	// ---- Wait for shutdown or error ----
	select {
	case <-rootCtx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
		log.Info("Shut down cleanly")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Error("Server failed: %v", err)
			os.Exit(1)
		}
	}
}
