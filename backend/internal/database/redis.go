package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient initializes and connects to the Redis service.
func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL (%s): %w", cfg.RedisURL, err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	slog.Info("Redis connection established successfully", "url", cfg.RedisURL)
	return client, nil
}
