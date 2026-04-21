package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// NotificationLister defines the inbound application boundary for notification listing.
type NotificationLister interface {
	ListNotifications(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error)
}

// NotificationMarker defines the inbound application boundary for notification state changes.
type NotificationMarker interface {
	MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error)
}

// UnreadCounter defines the inbound application boundary for unread count queries.
type UnreadCounter interface {
	GetUnreadCount(ctx context.Context, cpfHash string) (int64, error)
}
