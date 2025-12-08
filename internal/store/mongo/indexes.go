package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := ensureQuotesIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure quotes indexes: %w", err)
	}
	if err := ensureProductsIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure products indexes: %w", err)
	}
	if err := ensureApplicationsIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure applications indexes: %w", err)
	}
	if err := ensureUnderwritingCasesIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure underwriting_cases indexes: %w", err)
	}
	if err := ensureOffersIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure offers indexes: %w", err)
	}
	if err := ensurePoliciesIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure policies indexes: %w", err)
	}
	return nil
}

func ensureQuotesIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(ColQuotes)
	models := []mongo.IndexModel{
		newIndex("product_slug", 1, "quotes_product_slug", false),
		newIndex("created_at", 1, "quotes_created_at", false),
		newTTLIndex("expires_at", "quotes_expiry_ttl", 0), // optional: enable later if you want TTL
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func ensureProductsIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(ColProducts)
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetName("products_slug_unique").SetUnique(true),
		},
		{Keys: bson.D{{Key: "term_years", Value: 1}},
			Options: options.Index().SetName("products_term_years_asc"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func ensureApplicationsIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(ColApplications)
	models := []mongo.IndexModel{
		newIndex("quote_id", 1, "apps_quote_id", false),
		newIndex("status", 1, "apps_status", false),
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func ensureUnderwritingCasesIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(ColUnderwriting)
	models := []mongo.IndexModel{
		newIndex("application_id", 1, "uwc_application_id_unique", true),
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func ensureOffersIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(ColOffers)
	models := []mongo.IndexModel{
		newIndex("application_id", 1, "offers_application_id_unique", true),
		newIndex("status", 1, "offers_status", false),
		newTTLIndex("expires_at", "offers_expiry_ttl", 0),
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func ensurePoliciesIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(ColPolicies)
	models := []mongo.IndexModel{
		newIndex("number", 1, "policies_number_unique", true),
		newIndex("application_id", 1, "policies_application_id", false),
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func newIndex(field string, asc int32, name string, unique bool) mongo.IndexModel {
	opts := options.Index().SetName(name)
	if unique {
		opts = opts.SetUnique(true)
	}
	return mongo.IndexModel{
		Keys:    bson.D{{Key: field, Value: asc}},
		Options: opts,
	}
}

func newTTLIndex(field, name string, expireAfterSeconds int32) mongo.IndexModel {
	return mongo.IndexModel{
		Keys:    bson.D{{Key: field, Value: 1}},
		Options: options.Index().SetName(name).SetExpireAfterSeconds(expireAfterSeconds),
	}
}
