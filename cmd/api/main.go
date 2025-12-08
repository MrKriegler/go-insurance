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
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/MrKriegler/go-insurance/internal/core"
	transporthttp "github.com/MrKriegler/go-insurance/internal/http"
	healthhttp "github.com/MrKriegler/go-insurance/internal/http/health"

	_ "github.com/MrKriegler/go-insurance/docs" // Swagger docs

	"github.com/MrKriegler/go-insurance/internal/http/handlers"
	"github.com/MrKriegler/go-insurance/internal/jobs"
	"github.com/MrKriegler/go-insurance/internal/middleware"
	"github.com/MrKriegler/go-insurance/internal/platform/config"
	"github.com/MrKriegler/go-insurance/internal/platform/logging"
	"github.com/MrKriegler/go-insurance/internal/store/dynamo"
	"github.com/MrKriegler/go-insurance/internal/store/mongo"
)

// Pinger interface for health checks
type Pinger interface {
	Ping(ctx context.Context) error
}

func main() {
	// --- Config & Logger ---
	cfg := config.MustLoad()
	log := logging.New(cfg.Env)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Info("starting server", "addr", addr, "env", cfg.Env, "db_type", cfg.DBType)

	// Root ctx with SIGINT/SIGTERM
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Initialize based on DB type ---
	var (
		productRepo core.ProductRepo
		quoteRepo   core.QuoteRepo
		appRepo     core.ApplicationRepo
		uwRepo      core.UnderwritingRepo
		offerRepo   core.OfferRepo
		policyRepo  core.PolicyRepo
		pinger      Pinger
	)

	if cfg.DBType == "dynamodb" {
		// --- DynamoDB ---
		log.Info("connecting dynamodb", "region", cfg.AWSRegion, "endpoint", cfg.DynamoDBEndpoint)
		dynamoClient, err := dynamo.NewClient(rootCtx, dynamo.Config{
			Region:          cfg.AWSRegion,
			Endpoint:        cfg.DynamoDBEndpoint,
			AccessKeyID:     cfg.AWSAccessKeyID,
			SecretAccessKey: cfg.AWSSecretAccessKey,
		})
		if err != nil {
			log.Error("dynamodb connect failed", "err", err)
			os.Exit(1)
		}

		// Ensure tables exist
		if err := dynamo.EnsureTables(rootCtx, dynamoClient.DB, log); err != nil {
			log.Error("ensure tables failed", "err", err)
			os.Exit(1)
		}

		// Create repos
		productRepo = dynamo.NewProductRepo(dynamoClient.DB)
		quoteRepo = dynamo.NewQuoteRepo(dynamoClient.DB)
		appRepo = dynamo.NewApplicationRepo(dynamoClient.DB)
		uwRepo = dynamo.NewUnderwritingRepo(dynamoClient.DB)
		offerRepo = dynamo.NewOfferRepo(dynamoClient.DB)
		policyRepo = dynamo.NewPolicyRepo(dynamoClient.DB)
		pinger = dynamoClient

	} else {
		// --- MongoDB ---
		log.Info("connecting mongo", "uri", cfg.MongoURI, "db", cfg.MongoDB)
		mongoClient, err := mongo.NewClient(cfg)
		if err != nil {
			log.Error("mongo connect failed", "err", err)
			os.Exit(1)
		}
		defer mongoClient.Close(context.Background())

		// Ensure indexes
		{
			ctx, cancel := context.WithTimeout(rootCtx, 15*time.Second)
			defer cancel()
			if err := mongo.EnsureIndexes(ctx, mongoClient.DB); err != nil {
				log.Error("ensure indexes failed", "err", err)
				os.Exit(1)
			}
		}

		// Create repos
		opTimeout := time.Duration(cfg.MongoOpTimeoutMs) * time.Millisecond
		productRepo = mongo.NewProductRepo(mongoClient.DB, opTimeout)
		quoteRepo = mongo.NewQuoteRepo(mongoClient.DB, opTimeout)
		appRepo = mongo.NewApplicationRepo(mongoClient.DB, opTimeout)
		uwRepo = mongo.NewUnderwritingRepo(mongoClient.DB, opTimeout)
		offerRepo = mongo.NewOfferRepo(mongoClient.DB, opTimeout)
		policyRepo = mongo.NewPolicyRepo(mongoClient.DB, opTimeout)
		pinger = mongoClient
	}

	// --- Services ---
	quoteService := core.NewQuoteService(productRepo, quoteRepo)
	appService := core.NewApplicationService(appRepo, quoteRepo)
	offerService := core.NewOfferService(offerRepo, appRepo)
	policyService := core.NewPolicyService(policyRepo, offerRepo, appRepo)
	uwService := core.NewUnderwritingService(uwRepo, appRepo, offerRepo)

	// --- Handlers ---
	productsH := handlers.NewProductHandler(productRepo, log)
	quotesH := handlers.NewQuoteHandler(quoteService, quoteRepo, log)
	appsH := handlers.NewApplicationHandler(appService, log)
	uwH := handlers.NewUWHandler(uwService, log)
	offersH := handlers.NewOfferHandler(offerService, log)
	policiesH := handlers.NewPolicyHandler(policyService, log)

	// --- Background Workers ---
	workerInterval := time.Duration(cfg.WorkerIntervalSec) * time.Second
	uwWorker := jobs.NewUnderwritingWorker(appRepo, uwService, workerInterval, log)
	issuanceWorker := jobs.NewIssuanceWorker(offerRepo, policyService, workerInterval, log)

	// Start workers
	go uwWorker.Start(rootCtx)
	go issuanceWorker.Start(rootCtx)
	log.Info("background workers started", "interval", workerInterval)

	// --- Outer router: health + /api/v1 mount ---
	r := chi.NewRouter()

	// Standard middleware
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Logger, chimw.Recoverer)
	r.Use(chimw.Timeout(time.Duration(cfg.HTTPRequestTimeoutSec) * time.Second))

	// Security middleware
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.LimitRequestBody(middleware.MaxBodySize))

	// Rate limiting (100 req/min default)
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPM, time.Minute)
	rateLimiter.StartWithContext(rootCtx) // Graceful shutdown support
	r.Use(rateLimiter.Middleware)

	// API key auth (skips health/swagger)
	r.Use(middleware.SimpleAPIKey(cfg.APIKey))

	r.Mount("/", healthhttp.New(
		log,
		pinger,
		time.Duration(cfg.MongoOpTimeoutMs)*time.Millisecond,
	))

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
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
