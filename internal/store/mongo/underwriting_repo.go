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

type UnderwritingRepoMongo struct {
	coll      *mongodrv.Collection
	opTimeout time.Duration
}

func NewUnderwritingRepo(db *mongodrv.Database, opTimeout time.Duration) *UnderwritingRepoMongo {
	return &UnderwritingRepoMongo{
		coll:      db.Collection(ColUnderwriting),
		opTimeout: opTimeout,
	}
}

func (repo *UnderwritingRepoMongo) Create(ctx context.Context, uw core.UnderwritingCase) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toUnderwritingCaseDoc(uw)
	_, err := repo.coll.InsertOne(ctx, doc)
	if err != nil {
		var we mongodrv.WriteException
		if errors.As(err, &we) {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return core.ErrUWCaseExists
				}
			}
		}
		return fmt.Errorf("underwriting.insert: %w", err)
	}
	return nil
}

func (repo *UnderwritingRepoMongo) Get(ctx context.Context, id string) (core.UnderwritingCase, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc UnderwritingCaseDoc
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.UnderwritingCase{}, core.ErrUWCaseNotFound
		}
		return core.UnderwritingCase{}, fmt.Errorf("underwriting.findOne: %w", err)
	}
	return fromUnderwritingCaseDoc(doc), nil
}

func (repo *UnderwritingRepoMongo) GetByApplicationID(ctx context.Context, appID string) (core.UnderwritingCase, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc UnderwritingCaseDoc
	err := repo.coll.FindOne(ctx, bson.M{"application_id": appID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.UnderwritingCase{}, core.ErrUWCaseNotFound
		}
		return core.UnderwritingCase{}, fmt.Errorf("underwriting.findByApp: %w", err)
	}
	return fromUnderwritingCaseDoc(doc), nil
}

func (repo *UnderwritingRepoMongo) Update(ctx context.Context, uw core.UnderwritingCase) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toUnderwritingCaseDoc(uw)
	result, err := repo.coll.ReplaceOne(ctx, bson.M{"_id": uw.ID}, doc)
	if err != nil {
		return fmt.Errorf("underwriting.replace: %w", err)
	}
	if result.MatchedCount == 0 {
		return core.ErrUWCaseNotFound
	}
	return nil
}

func (repo *UnderwritingRepoMongo) FindPending(ctx context.Context, limit int) ([]core.UnderwritingCase, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	filter := bson.M{"decision": string(core.UWDecisionPending)}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := repo.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("underwriting.findPending: %w", err)
	}
	defer cursor.Close(ctx)

	var cases []core.UnderwritingCase
	for cursor.Next(ctx) {
		var doc UnderwritingCaseDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("underwriting.decode: %w", err)
		}
		cases = append(cases, fromUnderwritingCaseDoc(doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("underwriting.cursor: %w", err)
	}

	return cases, nil
}

func (repo *UnderwritingRepoMongo) FindReferred(ctx context.Context, limit int) ([]core.UnderwritingCase, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	filter := bson.M{"decision": string(core.UWDecisionReferred)}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := repo.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("underwriting.findReferred: %w", err)
	}
	defer cursor.Close(ctx)

	var cases []core.UnderwritingCase
	for cursor.Next(ctx) {
		var doc UnderwritingCaseDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("underwriting.decode: %w", err)
		}
		cases = append(cases, fromUnderwritingCaseDoc(doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("underwriting.cursor: %w", err)
	}

	return cases, nil
}
