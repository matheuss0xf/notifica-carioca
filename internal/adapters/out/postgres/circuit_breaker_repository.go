package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
	"github.com/matheuss0xf/notifica-carioca/internal/infra/resilience"
)

// CircuitBreakerNotificationRepository wraps the repository with a breaker.
type CircuitBreakerNotificationRepository struct {
	next    *NotificationRepository
	breaker *resilience.CircuitBreaker
}

// NewCircuitBreakerNotificationRepository wraps a repository with breaker protection.
func NewCircuitBreakerNotificationRepository(next *NotificationRepository, breaker *resilience.CircuitBreaker) *CircuitBreakerNotificationRepository {
	return &CircuitBreakerNotificationRepository{next: next, breaker: breaker}
}

func (r *CircuitBreakerNotificationRepository) Create(ctx context.Context, n *domain.Notification) (bool, error) {
	var created bool
	err := r.breaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		created, execErr = r.next.Create(ctx, n)
		return execErr
	})
	if err != nil {
		return false, fmt.Errorf("postgres breaker create: %w", err)
	}
	return created, nil
}

func (r *CircuitBreakerNotificationRepository) ListByOwner(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
	var notifications []domain.Notification
	err := r.breaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		notifications, execErr = r.next.ListByOwner(ctx, cpfHash, cursor, limit)
		return execErr
	})
	if err != nil {
		return nil, fmt.Errorf("postgres breaker list: %w", err)
	}
	return notifications, nil
}

func (r *CircuitBreakerNotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	var updated bool
	err := r.breaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		updated, execErr = r.next.MarkAsRead(ctx, id, cpfHash)
		return execErr
	})
	if err != nil {
		return false, fmt.Errorf("postgres breaker mark read: %w", err)
	}
	return updated, nil
}

func (r *CircuitBreakerNotificationRepository) CountUnread(ctx context.Context, cpfHash string) (int64, error) {
	var count int64
	err := r.breaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		count, execErr = r.next.CountUnread(ctx, cpfHash)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("postgres breaker count unread: %w", err)
	}
	return count, nil
}
