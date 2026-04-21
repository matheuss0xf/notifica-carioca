package runtime

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// ReadinessChecker probes runtime dependencies used by the application.
type ReadinessChecker struct {
	pingDB        func(context.Context) error
	pingRedis     func(context.Context) error
	wsConnections func() int
}

// NewReadinessChecker creates a readiness adapter for infrastructure dependencies.
func NewReadinessChecker(db *pgxpool.Pool, redis *goredis.Client, wsConnections func() int) *ReadinessChecker {
	return &ReadinessChecker{
		pingDB:        db.Ping,
		pingRedis:     func(ctx context.Context) error { return redis.Ping(ctx).Err() },
		wsConnections: wsConnections,
	}
}

// Check verifies whether the runtime dependencies are available.
func (c *ReadinessChecker) Check(ctx context.Context) (bool, int, error) {
	if err := c.pingDB(ctx); err != nil {
		return false, c.wsConnections(), fmt.Errorf("postgres unavailable: %w", err)
	}

	if err := c.pingRedis(ctx); err != nil {
		return false, c.wsConnections(), fmt.Errorf("redis unavailable: %w", err)
	}

	return true, c.wsConnections(), nil
}
