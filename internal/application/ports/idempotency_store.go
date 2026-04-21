package ports

import "context"

// IdempotencyStore tracks whether an event was already processed (fast-path dedup).
type IdempotencyStore interface {
	// Exists returns true if the key is already registered.
	Exists(ctx context.Context, key string) bool

	// Set marks a key as processed.
	Set(ctx context.Context, key string) error
}
