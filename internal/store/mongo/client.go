package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/MrKriegler/go-insurance/internal/platform/config"
)

type MongoClient struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func NewClient(cfg *config.Config) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.MongoConnectTimeoutSec)*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	db := client.Database(cfg.MongoDB)
	return &MongoClient{Client: client, DB: db}, nil
}

// Ping verifies connectivity (used by /readyz).
func (c *MongoClient) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx, nil)
}

// Close gracefully disconnects from MongoDB.
func (m *MongoClient) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
