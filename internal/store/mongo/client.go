package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/MrKriegler/go-insurance/internal/platform/config"
)

const (
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

type MongoClient struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func NewClient(cfg *config.Config) (*MongoClient, error) {
	clientOpts := options.Client().ApplyURI(cfg.MongoURI)

	var client *mongo.Client
	var err error

	// Retry connection with exponential backoff
	backoff := initialBackoff
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(cfg.MongoConnectTimeoutSec)*time.Second)

		client, err = mongo.Connect(ctx, clientOpts)
		if err != nil {
			cancel()
			if attempt == maxRetries {
				return nil, fmt.Errorf("connect to mongo after %d attempts: %w", maxRetries, err)
			}
			slog.Warn("mongo connect failed, retrying",
				"attempt", attempt,
				"backoff", backoff,
				"err", err)
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Verify connection
		if err = client.Ping(ctx, nil); err != nil {
			cancel()
			_ = client.Disconnect(context.Background())
			if attempt == maxRetries {
				return nil, fmt.Errorf("ping mongo after %d attempts: %w", maxRetries, err)
			}
			slog.Warn("mongo ping failed, retrying",
				"attempt", attempt,
				"backoff", backoff,
				"err", err)
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		cancel()
		break
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
