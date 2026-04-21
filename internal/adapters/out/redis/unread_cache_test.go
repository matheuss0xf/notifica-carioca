package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
)

func TestUnreadCacheLifecycle(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	cache := NewUnreadCache(client, time.Minute)

	if _, err := cache.Get(context.Background(), "cpf-1"); !errors.Is(err, ports.ErrCacheMiss) {
		t.Fatalf("expected cache miss, got %v", err)
	}

	if err := cache.Set(context.Background(), "cpf-1", 7); err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}

	got, err := cache.Get(context.Background(), "cpf-1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got != 7 {
		t.Fatalf("expected cached unread count 7, got %d", got)
	}

	if err := cache.Invalidate(context.Background(), "cpf-1"); err != nil {
		t.Fatalf("unexpected invalidate error: %v", err)
	}
	if _, err := cache.Get(context.Background(), "cpf-1"); !errors.Is(err, ports.ErrCacheMiss) {
		t.Fatalf("expected cache miss after invalidate, got %v", err)
	}
	if cacheKey("cpf-1") != "unread:cpf-1" {
		t.Fatalf("unexpected cache key: %s", cacheKey("cpf-1"))
	}
}
