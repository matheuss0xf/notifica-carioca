package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
)

// UnreadCache implements ports.UnreadCache using Redis.
type UnreadCache struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewUnreadCache creates a new Redis-backed unread count cache.
func NewUnreadCache(client *goredis.Client, ttl time.Duration) *UnreadCache {
	return &UnreadCache{client: client, ttl: ttl}
}

// Get returns the cached unread count. Returns ports.ErrCacheMiss if not cached.
func (c *UnreadCache) Get(ctx context.Context, cpfHash string) (int64, error) {
	val, err := c.client.Get(ctx, cacheKey(cpfHash)).Int64()
	if err == goredis.Nil {
		return 0, ports.ErrCacheMiss
	}
	if err != nil {
		slog.Warn("redis cache get failed", "error", err)
		return 0, ports.ErrCacheMiss
	}
	return val, nil
}

// Set stores the unread count in cache.
func (c *UnreadCache) Set(ctx context.Context, cpfHash string, count int64) error {
	if err := c.client.Set(ctx, cacheKey(cpfHash), count, c.ttl).Err(); err != nil {
		return fmt.Errorf("setting unread cache: %w", err)
	}
	return nil
}

// Invalidate removes the cached count.
func (c *UnreadCache) Invalidate(ctx context.Context, cpfHash string) error {
	if err := c.client.Del(ctx, cacheKey(cpfHash)).Err(); err != nil {
		return fmt.Errorf("invalidating unread cache: %w", err)
	}
	return nil
}

func cacheKey(cpfHash string) string {
	return fmt.Sprintf("unread:%s", cpfHash)
}
