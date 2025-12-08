package dynamo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

// Client wraps the DynamoDB client.
type Client struct {
	DB *dynamodb.Client
}

// Config holds DynamoDB configuration.
type Config struct {
	Region   string
	Endpoint string // Optional: for local development (e.g., "http://localhost:8000")
	// For local development only - in production use IAM roles
	AccessKeyID     string
	SecretAccessKey string
}

// NewClient creates a new DynamoDB client.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	var opts []func(*config.LoadOptions) error

	opts = append(opts, config.WithRegion(cfg.Region))

	// For local development with DynamoDB Local
	if cfg.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == dynamodb.ServiceID {
					return aws.Endpoint{
						URL:           cfg.Endpoint,
						SigningRegion: cfg.Region,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			})
		opts = append(opts, config.WithEndpointResolverWithOptions(customResolver))

		// Always use static credentials for local to avoid SDK trying to reach AWS metadata
		accessKey := cfg.AccessKeyID
		secretKey := cfg.SecretAccessKey
		if accessKey == "" {
			accessKey = "local"
		}
		if secretKey == "" {
			secretKey = "local"
		}
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsCfg)

	// Verify connectivity with retry
	if err := pingWithRetry(ctx, client); err != nil {
		return nil, err
	}

	return &Client{DB: client}, nil
}

// pingWithRetry attempts to ping DynamoDB with exponential backoff.
func pingWithRetry(ctx context.Context, client *dynamodb.Client) error {
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := client.ListTables(pingCtx, &dynamodb.ListTablesInput{Limit: aws.Int32(1)})
		cancel()

		if err == nil {
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("dynamodb ping failed after %d attempts: %w", maxRetries, err)
		}

		slog.Warn("dynamodb ping failed, retrying",
			"attempt", attempt,
			"backoff", backoff,
			"err", err)
		time.Sleep(backoff)
		backoff = min(backoff*2, maxBackoff)
	}

	return nil
}

// Ping checks DynamoDB connectivity by listing tables.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.DB.ListTables(ctx, &dynamodb.ListTablesInput{Limit: aws.Int32(1)})
	return err
}
