package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MrKriegler/go-insurance/internal/platform/config"
	"github.com/MrKriegler/go-insurance/internal/store/mongo"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	log.Println("Starting go-insurance API...")
	cfg := config.MustLoad()
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting server on %s in %s mode", addr, cfg.Env)

	// Connect to Mongo
	log.Printf("Connecting to MongoDB at %s / DB: %s", cfg.MongoURI, cfg.MongoDB)
	mongoClient, err := mongo.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(context.Background())
	log.Printf("Connected to MongoDB at %s / DB: %s", cfg.MongoURI, cfg.MongoDB)

	// ---- Setup router (Chi) ----
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Duration(cfg.HTTPRequestTimeoutSec) * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTPWriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTPIdleTimeoutSec) * time.Second,
	}

	log.Printf("Server started and listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
