package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewNotificationRepository(t *testing.T) {
	pool := &pgxpool.Pool{}
	repo := NewNotificationRepository(pool)
	if repo == nil {
		t.Fatal("expected repository instance")
	}
	if repo.pool != pool {
		t.Fatal("expected repository to keep provided pool")
	}
}
