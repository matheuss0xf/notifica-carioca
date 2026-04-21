package ports

import (
	"context"
	"errors"
)

// ErrCacheMiss indicates a cache miss — the caller should fall through to the database.
var ErrCacheMiss = errors.New("cache miss")

// UnreadCache caches the unread notification count per citizen.
type UnreadCache interface {
	// Get returns the cached unread count. Returns ErrCacheMiss if not cached.
	Get(ctx context.Context, cpfHash string) (int64, error)

	// Set stores the unread count in cache.
	Set(ctx context.Context, cpfHash string, count int64) error

	// Invalidate removes the cached count (e.g., after a new notification or read).
	Invalidate(ctx context.Context, cpfHash string) error
}
