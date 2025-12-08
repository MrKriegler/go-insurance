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

type ApplicationRepoMongo struct {
	coll      *mongodrv.Collection
	opTimeout time.Duration
}

func NewApplicationRepo(db *mongodrv.Database, opTimeout time.Duration) *ApplicationRepoMongo {
	return &ApplicationRepoMongo{
		coll:      db.Collection(ColApplications),
		opTimeout: opTimeout,
	}
}

func (repo *ApplicationRepoMongo) Create(ctx context.Context, app core.Application) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toApplicationDoc(app)
	_, err := repo.coll.InsertOne(ctx, doc)
	if err != nil {
		var we mongodrv.WriteException
		if errors.As(err, &we) {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return core.ErrConflict
				}
			}
		}
		return fmt.Errorf("applications.insert: %w", err)
	}
	return nil
}

func (repo *ApplicationRepoMongo) Get(ctx context.Context, id string) (core.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	var doc ApplicationDoc
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongodrv.ErrNoDocuments) {
			return core.Application{}, core.ErrApplicationNotFound
		}
		return core.Application{}, fmt.Errorf("applications.findOne: %w", err)
	}
	return fromApplicationDoc(doc), nil
}

func (repo *ApplicationRepoMongo) Update(ctx context.Context, app core.Application) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	doc := toApplicationDoc(app)
	result, err := repo.coll.ReplaceOne(ctx, bson.M{"_id": app.ID}, doc)
	if err != nil {
		return fmt.Errorf("applications.replace: %w", err)
	}
	if result.MatchedCount == 0 {
		return core.ErrApplicationNotFound
	}
	return nil
}

func (repo *ApplicationRepoMongo) UpdateStatus(ctx context.Context, id string, status core.ApplicationStatus, updatedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":     string(status),
			"updated_at": updatedAt,
		},
	}

	result, err := repo.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("applications.updateStatus: %w", err)
	}
	if result.MatchedCount == 0 {
		return core.ErrApplicationNotFound
	}
	return nil
}

func (repo *ApplicationRepoMongo) FindByStatus(ctx context.Context, status core.ApplicationStatus, limit int) ([]core.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, repo.opTimeout)
	defer cancel()

	filter := bson.M{"status": string(status)}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := repo.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("applications.find: %w", err)
	}
	defer cursor.Close(ctx)

	var apps []core.Application
	for cursor.Next(ctx) {
		var doc ApplicationDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("applications.decode: %w", err)
		}
		apps = append(apps, fromApplicationDoc(doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("applications.cursor: %w", err)
	}

	return apps, nil
}
