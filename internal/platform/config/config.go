package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	Env  string

	MongoURI string
	MongoDB  string

	// Timeouts
	HTTPReadTimeoutSec     int
	HTTPWriteTimeoutSec    int
	HTTPIdleTimeoutSec     int
	HTTPRequestTimeoutSec  int
	MongoConnectTimeoutSec int
	MongoOpTimeoutMs       int
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")
	cfg := &Config{}

	cfg.Port = getEnv("PORT", "8080")
	cfg.Env = getEnv("ENV", "dev")
	cfg.MongoURI = getEnv("MONGO_URI", "")
	cfg.MongoDB = getEnv("MONGO_DB", "go_insurance")

	cfg.HTTPReadTimeoutSec = getEnvAsInt("HTTP_READ_TIMEOUT_SEC", 10)
	cfg.HTTPWriteTimeoutSec = getEnvAsInt("HTTP_WRITE_TIMEOUT_SEC", 10)
	cfg.HTTPIdleTimeoutSec = getEnvAsInt("HTTP_IDLE_TIMEOUT_SEC", 120)
	cfg.HTTPRequestTimeoutSec = getEnvAsInt("HTTP_REQUEST_TIMEOUT_SEC", 30)
	cfg.MongoConnectTimeoutSec = getEnvAsInt("MONGO_CONNECT_TIMEOUT_SEC", 5)
	cfg.MongoOpTimeoutMs = getEnvAsInt("MONGO_OP_TIMEOUT_MS", 500)

	if cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
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
