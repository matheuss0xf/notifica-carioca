package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
)

type rateLimitBucket struct {
	count   int
	resetAt time.Time
}

// RateLimiter enforces a fixed-window request limit per client IP.
type RateLimiter struct {
	name   string
	limit  int
	window time.Duration

	mu      sync.Mutex
	buckets map[string]rateLimitBucket
}

// NewRateLimiter creates a new in-memory fixed-window rate limiter.
func NewRateLimiter(name string, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		name:    name,
		limit:   limit,
		window:  window,
		buckets: make(map[string]rateLimitBucket),
	}
}

// Handle returns a Gin middleware that enforces the configured rate limit.
func (m *RateLimiter) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.limit <= 0 || m.window <= 0 {
			c.Next()
			return
		}

		allowed, remaining, retryAfter := m.allow(c.ClientIP())
		c.Header("X-RateLimit-Limit", strconv.Itoa(m.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			httpx.AbortError(c, http.StatusTooManyRequests, "rate_limited", fmt.Sprintf("%s rate limit exceeded", m.name))
			return
		}

		c.Next()
	}
}

func (m *RateLimiter) allow(key string) (allowed bool, remaining int, retryAfter time.Duration) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	if key == "" {
		key = "unknown"
	}

	bucket, ok := m.buckets[key]
	if !ok || now.After(bucket.resetAt) {
		bucket = rateLimitBucket{count: 0, resetAt: now.Add(m.window)}
	}

	if bucket.count >= m.limit {
		return false, 0, time.Until(bucket.resetAt)
	}

	bucket.count++
	m.buckets[key] = bucket
	return true, m.limit - bucket.count, 0
}
