package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	// MinJWTSecretLength defines the minimum recommended key length (256 bits / 32 bytes) for HMAC-SHA256 in production.
	MinJWTSecretLength = 32
	// DevFallbackJWTSecret is only permitted in non-production environments to streamline local dev.
	DevFallbackJWTSecret = "dev-only-insecure-jwt-secret-key-32bytes-min!!"
	// LegacyInsecureJWTSecret is the deprecated known default that must never be permitted.
	LegacyInsecureJWTSecret = "super-secret-jwt-key-change-in-prod"
	// LegacyAdminPassword is the deprecated known default admin password that must never be permitted.
	LegacyAdminPassword = "Admin@4Yaar2026!"
)

// Config holds all application configuration values parsed from environment variables.
type Config struct {
	AppPort          string
	AppEnv           string
	AppBaseURL       string
	AllowedOrigins   []string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	AdminBootstrapEmail    string
	AdminBootstrapPassword string

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

// IsProduction reports whether the application is running in a production environment.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production") || strings.EqualFold(c.AppEnv, "prod")
}

// LoadConfig reads configuration from the environment and optional .env file.
func LoadConfig() (*Config, error) {
	// Attempt to load .env file if it exists, ignoring errors if not present
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found or error reading .env, reading from system environment")
	}

	appEnv := getEnv("APP_ENV", "development")
	isProd := strings.EqualFold(appEnv, "production") || strings.EqualFold(appEnv, "prod")

	jwtSecret := os.Getenv("JWT_SECRET")
	if isProd {
		if jwtSecret == "" {
			return nil, errors.New("JWT_SECRET must be explicitly set in production")
		}
		if len(jwtSecret) < MinJWTSecretLength {
			return nil, fmt.Errorf("JWT_SECRET must be at least %d characters in production for security", MinJWTSecretLength)
		}
		if jwtSecret == DevFallbackJWTSecret || jwtSecret == LegacyInsecureJWTSecret {
			return nil, errors.New("JWT_SECRET in production cannot use known default/development credentials")
		}
	} else {
		if jwtSecret == "" {
			slog.Warn("JWT_SECRET not provided; using development fallback secret. Never use this in production.")
			jwtSecret = DevFallbackJWTSecret
		}
	}

	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}

	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}

	appBaseURL := getEnv("APP_BASE_URL", "http://localhost:8080")

	// Parse configured allowed origins for WebSocket connections and CORS
	var allowedOrigins []string
	if rawOrigins := os.Getenv("ALLOWED_ORIGINS"); rawOrigins != "" {
		for _, o := range strings.Split(rawOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	} else {
		if isProd {
			if appBaseURL != "" {
				allowedOrigins = []string{appBaseURL}
			}
		} else {
			allowedOrigins = []string{
				"http://localhost:3000",
				"http://localhost:5173",
				"http://localhost:8080",
				appBaseURL,
			}
		}
	}

	adminEmail := strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_EMAIL"))
	adminPassword := os.Getenv("ADMIN_BOOTSTRAP_PASSWORD")
	if adminEmail != "" || adminPassword != "" {
		if adminEmail == "" || adminPassword == "" {
			return nil, errors.New("both ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD must be provided if either is set")
		}
		if len(adminPassword) < 8 {
			return nil, errors.New("ADMIN_BOOTSTRAP_PASSWORD must be at least 8 characters")
		}
		if adminPassword == LegacyAdminPassword {
			return nil, errors.New("ADMIN_BOOTSTRAP_PASSWORD cannot use legacy default credentials")
		}
	}

	cfg := &Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		AppEnv:           appEnv,
		AppBaseURL:       appBaseURL,
		AllowedOrigins:   allowedOrigins,
		JWTSecret:        jwtSecret,
		JWTAccessExpiry:  accessExpiry,
		JWTRefreshExpiry: refreshExpiry,

		AdminBootstrapEmail:    adminEmail,
		AdminBootstrapPassword: adminPassword,

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
