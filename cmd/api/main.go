// cmd/api/main.go
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
	healthhttp "github.com/MrKriegler/go-insurance/internal/http/health"

	"github.com/MrKriegler/go-insurance/internal/http/handlers"
	"github.com/MrKriegler/go-insurance/internal/platform/config"
	"github.com/MrKriegler/go-insurance/internal/platform/logging"
	"github.com/MrKriegler/go-insurance/internal/store/mongo"
)

func main() {
	// --- Config & Logger ---
	cfg := config.MustLoad()
	log := logging.New(cfg.Env)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Info("starting server", "addr", addr, "env", cfg.Env)

	// Root ctx with SIGINT/SIGTERM
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Mongo ---
	log.Info("connecting mongo", "uri", cfg.MongoURI, "db", cfg.MongoDB)
	mc, err := mongo.NewClient(cfg)
	if err != nil {
		log.Error("mongo connect failed", "err", err)
		os.Exit(1)
	}
	defer mc.Close(context.Background())

	// Ensure indexes
	{
		ctx, cancel := context.WithTimeout(rootCtx, 15*time.Second)
		defer cancel()
		if err := mongo.EnsureIndexes(ctx, mc.DB); err != nil {
			log.Error("ensure indexes failed", "err", err)
			os.Exit(1)
		}
	}

	// --- Build deps for handlers ---
	productRepo := mongo.NewProductRepo(mc.DB, time.Duration(cfg.MongoOpTimeoutMs)*time.Millisecond)

	productsH := handlers.NewProductHandler(productRepo, log)
	quotesH := handlers.NewQuoteHandler()     // keep simple for now
	appsH := handlers.NewApplicationHandler() // keep simple for now
	uwH := handlers.NewUWHandler()
	offersH := handlers.NewOfferHandler()
	policiesH := handlers.NewPolicyHandler()

	// --- Outer router: health + /api/v1 mount ---
	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Logger, chimw.Recoverer)
	r.Use(chimw.Timeout(time.Duration(cfg.HTTPRequestTimeoutSec) * time.Second))

	r.Mount("/", healthhttp.New(
		log,
		mc, // your mongo client implements Ping(ctx) error
		time.Duration(cfg.MongoOpTimeoutMs)*time.Millisecond,
	))

	// Build API subrouter (adds JSON content-type inside)
	api := transporthttp.NewRouter(transporthttp.Deps{
		Mounts: []handlers.Mountable{
			productsH, quotesH, appsH, uwH, offersH, policiesH,
		},
	})

	r.Mount("/api/v1", api)

	// --- HTTP Server ---
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
		log.Info("listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	// --- Shutdown / Exit ---
	select {
	case <-rootCtx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
		log.Info("shutdown complete")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}
}
