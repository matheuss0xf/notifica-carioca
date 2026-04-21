package runtime

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func TestNewReadinessChecker(t *testing.T) {
	checker := NewReadinessChecker(&pgxpool.Pool{}, goredis.NewClient(&goredis.Options{Addr: "localhost:6379"}), func() int { return 1 })
	if checker == nil {
		t.Fatal("expected readiness checker instance")
	}
}
