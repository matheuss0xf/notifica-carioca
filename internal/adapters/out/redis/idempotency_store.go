package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// IdempotencyStore implements ports.IdempotencyStore using Redis SETNX.
type IdempotencyStore struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewIdempotencyStore creates a new Redis-backed idempotency store.
func NewIdempotencyStore(client *goredis.Client, ttl time.Duration) *IdempotencyStore {
	return &IdempotencyStore{client: client, ttl: ttl}
}

// Exists returns true if the key is already registered (event was already processed).
func (s *IdempotencyStore) Exists(ctx context.Context, key string) bool {
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		// Redis failure — return false so processing continues to DB check (resilient)
		slog.Warn("redis idempotency check failed", "error", err)
		return false
	}
	return exists > 0
}

// Set marks a key as processed with a TTL.
func (s *IdempotencyStore) Set(ctx context.Context, key string) error {
	if err := s.client.Set(ctx, key, "1", s.ttl).Err(); err != nil {
		return fmt.Errorf("setting idempotency key: %w", err)
	}
	return nil
}
