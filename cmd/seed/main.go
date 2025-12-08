package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/internal/platform/config"
	"github.com/MrKriegler/go-insurance/internal/platform/logging"
	"github.com/MrKriegler/go-insurance/internal/store/dynamo"
	"github.com/MrKriegler/go-insurance/internal/store/mongo"
)

func main() {
	cfg := config.MustLoad()
	log := logging.New(cfg.Env)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var productRepo core.ProductRepo

	if cfg.DBType == "dynamodb" {
		// Connect to DynamoDB
		log.Info("connecting to DynamoDB", "region", cfg.AWSRegion, "endpoint", cfg.DynamoDBEndpoint)
		client, err := dynamo.NewClient(ctx, dynamo.Config{
			Region:          cfg.AWSRegion,
			Endpoint:        cfg.DynamoDBEndpoint,
			AccessKeyID:     cfg.AWSAccessKeyID,
			SecretAccessKey: cfg.AWSSecretAccessKey,
		})
		if err != nil {
			log.Error("failed to connect to DynamoDB", "err", err)
			os.Exit(1)
		}

		// Ensure tables exist
		if err := dynamo.EnsureTables(ctx, client.DB, log); err != nil {
			log.Error("failed to ensure tables", "err", err)
			os.Exit(1)
		}

		productRepo = dynamo.NewProductRepo(client.DB)
	} else {
		// Connect to MongoDB
		log.Info("connecting to MongoDB", "uri", cfg.MongoURI)
		client, err := mongo.NewClient(cfg)
		if err != nil {
			log.Error("failed to connect to MongoDB", "err", err)
			os.Exit(1)
		}
		defer client.Close(ctx)

		productRepo = mongo.NewProductRepo(client.DB, 5*time.Second)
	}

	log.Info("seeding products")
	seedProducts(ctx, productRepo)
	log.Info("done seeding")
}

func seedProducts(ctx context.Context, repo core.ProductRepo) {
	products := []core.Product{
		{
			Slug:        "term-life-10",
			Name:        "10-Year Term Life",
			TermYears:   10,
			MinCoverage: 50000,
			MaxCoverage: 500000,
			BaseRate:    0.25, // per $1,000 coverage per month
		},
		{
			Slug:        "term-life-20",
			Name:        "20-Year Term Life",
			TermYears:   20,
			MinCoverage: 50000,
			MaxCoverage: 1000000,
			BaseRate:    0.35,
		},
		{
			Slug:        "term-life-30",
			Name:        "30-Year Term Life",
			TermYears:   30,
			MinCoverage: 100000,
			MaxCoverage: 2000000,
			BaseRate:    0.45,
		},
		{
			Slug:        "whole-life",
			Name:        "Whole Life",
			TermYears:   99,
			MinCoverage: 25000,
			MaxCoverage: 500000,
			BaseRate:    1.50,
		},
		{
			Slug:        "senior-life",
			Name:        "Senior Term Life (Ages 50-80)",
			TermYears:   15,
			MinCoverage: 10000,
			MaxCoverage: 100000,
			BaseRate:    2.00,
		},
	}

	for _, p := range products {
		if err := repo.UpsertBySlug(ctx, p); err != nil {
			fmt.Printf("failed to seed %s: %v\n", p.Slug, err)
		} else {
			fmt.Printf("seeded: %s\n", p.Name)
		}
	}
}
