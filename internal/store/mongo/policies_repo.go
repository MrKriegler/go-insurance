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

type PolicyRepoMongo struct {
	coll      *mongodrv.Collection
	counters  *mongodrv.Collection
	opTimeout time.Duration
}

func NewPolicyRepo(db *mongodrv.Database, opTimeout time.Duration) *PolicyRepoMongo {
	return &PolicyRepoMongo{
		coll:      db.Collection(ColPolicies),
		counters:  db.Collection("counters"),
		opTimeout: opTimeout,
	}
}

func (repo *PolicyRepoMongo) Create(ctx context.Context, policy core.Policy) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toPolicyDoc(policy)
	_, err := repo.coll.InsertOne(ctx, doc)
	if err != nil {
		var we mongodrv.WriteException
		if errors.As(err, &we) {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return core.ErrPolicyExists
				}
			}
		}
		return fmt.Errorf("policies.insert: %w", err)
	}
	return nil
}

func (repo *PolicyRepoMongo) Get(ctx context.Context, id string) (core.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc PolicyDoc
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Policy{}, core.ErrPolicyNotFound
		}
		return core.Policy{}, fmt.Errorf("policies.findOne: %w", err)
	}
	return fromPolicyDoc(doc), nil
}

func (repo *PolicyRepoMongo) GetByNumber(ctx context.Context, number string) (core.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc PolicyDoc
	err := repo.coll.FindOne(ctx, bson.M{"number": number}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Policy{}, core.ErrPolicyNotFound
		}
		return core.Policy{}, fmt.Errorf("policies.findByNumber: %w", err)
	}
	return fromPolicyDoc(doc), nil
}

func (repo *PolicyRepoMongo) GetByOfferID(ctx context.Context, offerID string) (core.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc PolicyDoc
	err := repo.coll.FindOne(ctx, bson.M{"offer_id": offerID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Policy{}, core.ErrPolicyNotFound
		}
		return core.Policy{}, fmt.Errorf("policies.findByOffer: %w", err)
	}
	return fromPolicyDoc(doc), nil
}

func (repo *PolicyRepoMongo) GetByApplicationID(ctx context.Context, appID string) (core.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc PolicyDoc
	err := repo.coll.FindOne(ctx, bson.M{"application_id": appID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Policy{}, core.ErrPolicyNotFound
		}
		return core.Policy{}, fmt.Errorf("policies.findByApp: %w", err)
	}
	return fromPolicyDoc(doc), nil
}

func (repo *PolicyRepoMongo) List(ctx context.Context, filter core.PolicyFilter, limit, offset int) ([]core.Policy, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	mongoFilter := bson.M{}
	if filter.ApplicationID != "" {
		mongoFilter["application_id"] = filter.ApplicationID
	}
	if filter.Status != "" {
		mongoFilter["status"] = string(filter.Status)
	}

	// Get total count
	total, err := repo.coll.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, 0, fmt.Errorf("policies.count: %w", err)
	}

	// Get paginated results
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "issued_at", Value: -1}})

	cursor, err := repo.coll.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("policies.find: %w", err)
	}
	defer cursor.Close(ctx)

	var policies []core.Policy
	for cursor.Next(ctx) {
		var doc PolicyDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, fmt.Errorf("policies.decode: %w", err)
		}
		policies = append(policies, fromPolicyDoc(doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("policies.cursor: %w", err)
	}

	return policies, total, nil
}

func (repo *PolicyRepoMongo) NextPolicyNumber(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	year := time.Now().Year()
	counterID := fmt.Sprintf("policy_%d", year)

	// Atomic increment using FindOneAndUpdate with upsert
	filter := bson.M{"_id": counterID}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result struct {
		Seq int64 `bson:"seq"`
	}

	err := repo.counters.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("policies.nextNumber: %w", err)
	}

	// Format: POL-YYYY-NNNNNN
	return fmt.Sprintf("POL-%d-%06d", year, result.Seq), nil
}
