package redis

import (
	"context"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
	"github.com/matheuss0xf/notifica-carioca/internal/infra/resilience"
)

// CircuitBreakerUnreadCache wraps the unread cache with a breaker.
type CircuitBreakerUnreadCache struct {
	next    *UnreadCache
	breaker *resilience.CircuitBreaker
}

func NewCircuitBreakerUnreadCache(next *UnreadCache, breaker *resilience.CircuitBreaker) *CircuitBreakerUnreadCache {
	return &CircuitBreakerUnreadCache{next: next, breaker: breaker}
}

func (c *CircuitBreakerUnreadCache) Get(ctx context.Context, cpfHash string) (int64, error) {
	var count int64
	err := c.breaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		count, execErr = c.next.Get(ctx, cpfHash)
		if execErr == ports.ErrCacheMiss {
			return nil
		}
		return execErr
	})
	if err != nil {
		return 0, ports.ErrCacheMiss
	}
	return count, nil
}

func (c *CircuitBreakerUnreadCache) Set(ctx context.Context, cpfHash string, count int64) error {
	return c.breaker.Execute(ctx, func(ctx context.Context) error {
		return c.next.Set(ctx, cpfHash, count)
	})
}

func (c *CircuitBreakerUnreadCache) Invalidate(ctx context.Context, cpfHash string) error {
	return c.breaker.Execute(ctx, func(ctx context.Context) error {
		return c.next.Invalidate(ctx, cpfHash)
	})
}

// CircuitBreakerIdempotencyStore wraps the idempotency store with a breaker.
type CircuitBreakerIdempotencyStore struct {
	next    *IdempotencyStore
	breaker *resilience.CircuitBreaker
}

func NewCircuitBreakerIdempotencyStore(next *IdempotencyStore, breaker *resilience.CircuitBreaker) *CircuitBreakerIdempotencyStore {
	return &CircuitBreakerIdempotencyStore{next: next, breaker: breaker}
}

func (s *CircuitBreakerIdempotencyStore) Exists(ctx context.Context, key string) bool {
	var exists bool
	if err := s.breaker.Execute(ctx, func(ctx context.Context) error {
		exists = s.next.Exists(ctx, key)
		return nil
	}); err != nil {
		return false
	}
	return exists
}

func (s *CircuitBreakerIdempotencyStore) Set(ctx context.Context, key string) error {
	return s.breaker.Execute(ctx, func(ctx context.Context) error {
		return s.next.Set(ctx, key)
	})
}

// CircuitBreakerPublisher wraps the event publisher with a breaker.
type CircuitBreakerPublisher struct {
	next    *Publisher
	breaker *resilience.CircuitBreaker
}

func NewCircuitBreakerPublisher(next *Publisher, breaker *resilience.CircuitBreaker) *CircuitBreakerPublisher {
	return &CircuitBreakerPublisher{next: next, breaker: breaker}
}

func (p *CircuitBreakerPublisher) Publish(ctx context.Context, cpfHash string, n *domain.Notification) error {
	return p.breaker.Execute(ctx, func(ctx context.Context) error {
		return p.next.Publish(ctx, cpfHash, n)
	})
}

// CircuitBreakerWebhookDeadLetterQueue wraps the DLQ writer with a breaker.
type CircuitBreakerWebhookDeadLetterQueue struct {
	next    *WebhookDeadLetterQueue
	breaker *resilience.CircuitBreaker
}

func NewCircuitBreakerWebhookDeadLetterQueue(next *WebhookDeadLetterQueue, breaker *resilience.CircuitBreaker) *CircuitBreakerWebhookDeadLetterQueue {
	return &CircuitBreakerWebhookDeadLetterQueue{next: next, breaker: breaker}
}

func (q *CircuitBreakerWebhookDeadLetterQueue) Enqueue(ctx context.Context, deadLetter domain.WebhookDeadLetter) error {
	return q.breaker.Execute(ctx, func(ctx context.Context) error {
		return q.next.Enqueue(ctx, deadLetter)
	})
}
