package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	Env  string

	// Database selection: "dynamodb" or "mongo"
	DBType string

	// MongoDB settings (when DBType = "mongo")
	MongoURI string
	MongoDB  string

	// DynamoDB settings (when DBType = "dynamodb")
	AWSRegion          string
	DynamoDBEndpoint   string // Optional: for local development
	AWSAccessKeyID     string // Optional: for local development
	AWSSecretAccessKey string // Optional: for local development

	// Timeouts
	HTTPReadTimeoutSec     int
	HTTPWriteTimeoutSec    int
	HTTPIdleTimeoutSec     int
	HTTPRequestTimeoutSec  int
	MongoConnectTimeoutSec int
	MongoOpTimeoutMs       int

	// Worker settings
	WorkerIntervalSec int

	// Security settings (for demo)
	APIKey         string   // Simple API key for demo auth
	AllowedOrigins []string // CORS allowed origins
	RateLimitRPM   int      // Rate limit requests per minute
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")
	cfg := &Config{}

	cfg.Port = getEnv("PORT", "8080")
	cfg.Env = getEnv("ENV", "dev")
	cfg.DBType = getEnv("DB_TYPE", "dynamodb") // Default to DynamoDB

	// MongoDB settings (check both MONGODB_URI and MONGO_URI for compatibility)
	cfg.MongoURI = getEnv("MONGODB_URI", getEnv("MONGO_URI", ""))
	cfg.MongoDB = getEnv("MONGO_DB", "go_insurance")

	// DynamoDB settings
	cfg.AWSRegion = getEnv("AWS_REGION", "us-east-1")
	cfg.DynamoDBEndpoint = getEnv("DYNAMODB_ENDPOINT", "") // Empty means use AWS
	cfg.AWSAccessKeyID = getEnv("AWS_ACCESS_KEY_ID", "")
	cfg.AWSSecretAccessKey = getEnv("AWS_SECRET_ACCESS_KEY", "")

	cfg.HTTPReadTimeoutSec = getEnvAsInt("HTTP_READ_TIMEOUT_SEC", 10)
	cfg.HTTPWriteTimeoutSec = getEnvAsInt("HTTP_WRITE_TIMEOUT_SEC", 10)
	cfg.HTTPIdleTimeoutSec = getEnvAsInt("HTTP_IDLE_TIMEOUT_SEC", 120)
	cfg.HTTPRequestTimeoutSec = getEnvAsInt("HTTP_REQUEST_TIMEOUT_SEC", 30)
	cfg.MongoConnectTimeoutSec = getEnvAsInt("MONGO_CONNECT_TIMEOUT_SEC", 5)
	cfg.MongoOpTimeoutMs = getEnvAsInt("MONGO_OP_TIMEOUT_MS", 500)
	cfg.WorkerIntervalSec = getEnvAsInt("WORKER_INTERVAL_SEC", 5)

	// Security settings
	cfg.APIKey = getEnv("API_KEY", "")
	cfg.AllowedOrigins = getEnvAsSlice("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:8080"})
	cfg.RateLimitRPM = getEnvAsInt("RATE_LIMIT_RPM", 100) // 100 requests per minute

	// Validate required fields based on DB type
	if cfg.DBType == "mongo" && cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI is required when DB_TYPE=mongo")
	}

	// In production, API_KEY must be explicitly set
	if cfg.Env == "prod" && cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY is required in production environment")
	}

	// Default API key for development only
	if cfg.APIKey == "" {
		cfg.APIKey = "demo-api-key-12345"
	}

	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsSlice(key string, defaultVal []string) []string {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	// Split by comma and trim whitespace
	var result []string
	for _, s := range strings.Split(valStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return defaultVal
	}
	return result
}
