package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
)

// NotificationMarker handles notification update use cases.
type NotificationMarker struct {
	repo  ports.NotificationRepository
	cache ports.UnreadCache
}

// NewNotificationMarker creates the command-side notification use case.
func NewNotificationMarker(repo ports.NotificationRepository, cache ports.UnreadCache) *NotificationMarker {
	return &NotificationMarker{
		repo:  repo,
		cache: cache,
	}
}

// MarkAsRead marks a notification as read, ensuring ownership via cpfHash.
func (m *NotificationMarker) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	updated, err := m.repo.MarkAsRead(ctx, id, cpfHash)
	if err != nil {
		return false, fmt.Errorf("marking as read: %w", err)
	}

	if updated {
		if invErr := m.cache.Invalidate(ctx, cpfHash); invErr != nil {
			slog.Warn("failed to invalidate unread cache", "error", invErr)
		}
	}

	return updated, nil
}
