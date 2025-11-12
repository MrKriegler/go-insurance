package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
	"go.mongodb.org/mongo-driver/bson"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
)

type QuoteRepoMongo struct {
	coll      *mongodrv.Collection
	opTimeout time.Duration
}

func NewQuoteRepo(db *mongodrv.Database, opTimeout time.Duration) *QuoteRepoMongo {
	return &QuoteRepoMongo{
		coll:      db.Collection(ColQuotes),
		opTimeout: opTimeout,
	}
}

func (repo *QuoteRepoMongo) Create(ctx context.Context, q core.Quote) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	quote := toQuoteDoc(q)
	_, err := repo.coll.InsertOne(ctx, quote)
	if err != nil {
		// map dup key -> core.ErrConflict
		var we mongodrv.WriteException
		if errors.As(err, &we) {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return core.ErrConflict
				}
			}
		}
		return fmt.Errorf("quotes.insert: %w", err)
	}
	return nil
}

func (repo *QuoteRepoMongo) Get(ctx context.Context, id string) (core.Quote, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var quote QuoteDoc
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&quote)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Quote{}, core.ErrNotFound
		}
		return core.Quote{}, fmt.Errorf("quotes.findOne: %w", err)
	}
	return fromQuoteDoc(quote), nil
}
