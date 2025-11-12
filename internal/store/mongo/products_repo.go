package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/internal/platform/ids"
	"go.mongodb.org/mongo-driver/bson"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepoMongo struct {
	coll      *mongodrv.Collection
	opTimeout time.Duration
}

func NewProductRepo(db *mongodrv.Database, opTimeout time.Duration) *ProductRepoMongo {
	return &ProductRepoMongo{
		coll:      db.Collection(ColProducts),
		opTimeout: opTimeout,
	}
}

// Lists all products. returns an empty slice if none found.
func (r *ProductRepoMongo) List(ctx context.Context) ([]core.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	cur, err := r.coll.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("products.find: %w", err)
	}
	defer cur.Close(ctx)

	var products []core.Product
	for cur.Next(ctx) {
		var product ProductDoc
		if err := cur.Decode(&product); err != nil {
			return nil, fmt.Errorf("products.decode: %w", err)
		}
		products = append(products, fromProductDoc(product))
	}
	// Check for errors during iteration
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("products.cursor: %w", err)
	}
	return products, nil
}

// Gets a product by Slug. Returns core.ErrNotFound if not found.
func (r *ProductRepoMongo) GetBySlug(ctx context.Context, slug string) (core.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()
	var doc ProductDoc
	err := r.coll.FindOne(ctx, bson.M{"slug": slug}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Product{}, core.ErrNotFound
		}
		return core.Product{}, fmt.Errorf("products.getBySlug: %w", err)
	}
	return fromProductDoc(doc), nil
}

// Gets a product by ID. Returns core.ErrNotFound if not found.
func (r *ProductRepoMongo) GetByID(ctx context.Context, id string) (core.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	var product ProductDoc
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Product{}, core.ErrNotFound
		}
		return core.Product{}, fmt.Errorf("products.findOne: %w", err)
	}
	return fromProductDoc(product), nil
}

// Upserts a product by Slug. If no existing product with the given Slug exists, a new one is created.
func (r *ProductRepoMongo) UpsertBySlug(ctx context.Context, p core.Product) error {
	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	// Ensure we never overwrite the _id unless inserting
	set := bson.M{
		"slug":         p.Slug,
		"name":         p.Name,
		"term_years":   p.TermYears,
		"min_coverage": p.MinCoverage,
		"max_coverage": p.MaxCoverage,
		"base_rate":    p.BaseRate,
	}
	setOnInsert := bson.M{"_id": p.ID}
	if p.ID == "" {
		setOnInsert["_id"] = ids.New()
	}

	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"slug": p.Slug}, // match by slug
		bson.M{"$set": set, "$setOnInsert": setOnInsert},
		options.Update().SetUpsert(true),
	)
	return err
}
