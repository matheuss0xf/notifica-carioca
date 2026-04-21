package redis

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestIdempotencyStoreSetAndExists(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	store := NewIdempotencyStore(client, time.Minute)

	if store.Exists(context.Background(), "event:1") {
		t.Fatalf("expected missing key to return false")
	}
	if err := store.Set(context.Background(), "event:1"); err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}
	if !store.Exists(context.Background(), "event:1") {
		t.Fatalf("expected existing key to return true")
	}
}
