package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration values parsed from environment variables.
type Config struct {
	AppPort          string
	AppEnv           string
	AppBaseURL       string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisURL string

	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	S3Bucket           string
	SESFromEmail       string
}

// LoadConfig reads configuration from the environment and optional .env file.
func LoadConfig() (*Config, error) {
	// Attempt to load .env file if it exists, ignoring errors if not present
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found or error reading .env, reading from system environment")
	}

	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}

	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}

	cfg := &Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		AppEnv:           getEnv("APP_ENV", "development"),
		AppBaseURL:       getEnv("APP_BASE_URL", "http://localhost:8080"),
		JWTSecret:        getEnv("JWT_SECRET", "super-secret-jwt-key-change-in-prod"),
		JWTAccessExpiry:  accessExpiry,
		JWTRefreshExpiry: refreshExpiry,

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "4yaar"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		S3Bucket:           getEnv("S3_BUCKET", "4yaar-files"),
		SESFromEmail:       getEnv("SES_FROM_EMAIL", "noreply@yourdomain.com"),
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a fallback value.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
