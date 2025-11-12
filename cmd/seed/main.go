package main

import (
	"context"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/internal/platform/config"
	"github.com/MrKriegler/go-insurance/internal/platform/logging"
	"github.com/MrKriegler/go-insurance/internal/store/mongo"
)

func main() {
	cfg := config.MustLoad()
	log := logging.New(cfg.Env)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Connect to Mongo
	client, err := mongo.NewClient(cfg)
	if err != nil {
		log.Error("failed to connect to MongoDB", "err", err)
		return
	}
	defer client.Close(ctx)

	db := client.DB
	productRepo := mongo.NewProductRepo(db, 5*time.Second)

	log.Info("seeding products")

	seedProducts(ctx, productRepo)

	log.Info("done seeding")
}

func seedProducts(ctx context.Context, repo *mongo.ProductRepoMongo) {
	products := []core.Product{
		{
			Slug:        "term-life-10y-standard",
			Name:        "Standard Term Life 10-Year",
			TermYears:   10,
			MinCoverage: 50000,
			MaxCoverage: 500000,
			BaseRate:    1.20, // per 1,000 coverage per month
		},
		{
			Slug:        "term-life-20y-preferred",
			Name:        "Preferred Term Life 20-Year",
			TermYears:   20,
			MinCoverage: 100000,
			MaxCoverage: 1000000,
			BaseRate:    0.95,
		},
		{
			Slug:        "whole-life-level-premium",
			Name:        "Whole Life – Level Premium",
			TermYears:   99,
			MinCoverage: 25000,
			MaxCoverage: 2000000,
			BaseRate:    1.75,
		},
		{
			Slug:        "senior-life-65plus",
			Name:        "Senior Life 65+ Guaranteed Acceptance",
			TermYears:   15,
			MinCoverage: 10000,
			MaxCoverage: 100000,
			BaseRate:    2.40,
		},
		{
			Slug:        "family-protection-plan",
			Name:        "Family Protection Plan (Joint Life)",
			TermYears:   25,
			MinCoverage: 100000,
			MaxCoverage: 750000,
			BaseRate:    1.60,
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
