package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
	"go.mongodb.org/mongo-driver/bson"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OfferRepoMongo struct {
	coll      *mongodrv.Collection
	opTimeout time.Duration
}

func NewOfferRepo(db *mongodrv.Database, opTimeout time.Duration) *OfferRepoMongo {
	return &OfferRepoMongo{
		coll:      db.Collection(ColOffers),
		opTimeout: opTimeout,
	}
}

func (repo *OfferRepoMongo) Create(ctx context.Context, offer core.Offer) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toOfferDoc(offer)
	_, err := repo.coll.InsertOne(ctx, doc)
	if err != nil {
		var we mongodrv.WriteException
		if errors.As(err, &we) {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return core.ErrOfferExists
				}
			}
		}
		return fmt.Errorf("offers.insert: %w", err)
	}
	return nil
}

func (repo *OfferRepoMongo) Get(ctx context.Context, id string) (core.Offer, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc OfferDoc
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Offer{}, core.ErrOfferNotFound
		}
		return core.Offer{}, fmt.Errorf("offers.findOne: %w", err)
	}
	return fromOfferDoc(doc), nil
}

func (repo *OfferRepoMongo) GetByApplicationID(ctx context.Context, appID string) (core.Offer, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc OfferDoc
	err := repo.coll.FindOne(ctx, bson.M{"application_id": appID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Offer{}, core.ErrOfferNotFound
		}
		return core.Offer{}, fmt.Errorf("offers.findByApp: %w", err)
	}
	return fromOfferDoc(doc), nil
}

func (repo *OfferRepoMongo) Update(ctx context.Context, offer core.Offer) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toOfferDoc(offer)
	result, err := repo.coll.ReplaceOne(ctx, bson.M{"_id": offer.ID}, doc)
	if err != nil {
		return fmt.Errorf("offers.replace: %w", err)
	}
	if result.MatchedCount == 0 {
		return core.ErrOfferNotFound
	}
	return nil
}

func (repo *OfferRepoMongo) FindAccepted(ctx context.Context, limit int) ([]core.Offer, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	filter := bson.M{"status": string(core.OfferStatusAccepted)}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "accepted_at", Value: 1}})

	cursor, err := repo.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("offers.findAccepted: %w", err)
	}
	defer cursor.Close(ctx)

	var offers []core.Offer
	for cursor.Next(ctx) {
		var doc OfferDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("offers.decode: %w", err)
		}
		offers = append(offers, fromOfferDoc(doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("offers.cursor: %w", err)
	}

	return offers, nil
}

func (repo *OfferRepoMongo) ExpireOffers(ctx context.Context, before time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	filter := bson.M{
		"status":     string(core.OfferStatusPending),
		"expires_at": bson.M{"$lt": before},
	}
	update := bson.M{
		"$set": bson.M{"status": string(core.OfferStatusExpired)},
	}

	result, err := repo.coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("offers.expireMany: %w", err)
	}
	return result.ModifiedCount, nil
}
