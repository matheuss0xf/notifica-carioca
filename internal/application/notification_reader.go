package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// NotificationReader handles notification query use cases.
type NotificationReader struct {
	repo  ports.NotificationRepository
	cache ports.UnreadCache
}

// NewNotificationReader creates the query-side notification use case.
func NewNotificationReader(repo ports.NotificationRepository, cache ports.UnreadCache) *NotificationReader {
	return &NotificationReader{
		repo:  repo,
		cache: cache,
	}
}

// ListNotifications returns a paginated list of notifications for a citizen.
func (r *NotificationReader) ListNotifications(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	notifications, err := r.repo.ListByOwner(ctx, cpfHash, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}

	page := &domain.NotificationPage{
		HasMore: len(notifications) > limit,
	}
	if page.HasMore {
		notifications = notifications[:limit]
	}

	page.Data = notifications

	if len(notifications) > 0 && page.HasMore {
		lastID := notifications[len(notifications)-1].ID.String()
		page.NextCursor = &lastID
	}

	return page, nil
}

// GetUnreadCount returns the unread count, using cache when available.
func (r *NotificationReader) GetUnreadCount(ctx context.Context, cpfHash string) (int64, error) {
	cached, err := r.cache.Get(ctx, cpfHash)
	if err == nil {
		return cached, nil
	}

	count, err := r.repo.CountUnread(ctx, cpfHash)
	if err != nil {
		return 0, fmt.Errorf("getting unread count: %w", err)
	}

	if setErr := r.cache.Set(ctx, cpfHash, count); setErr != nil {
		slog.Error("failed to cache unread count", "error", setErr)
	}

	return count, nil
}
