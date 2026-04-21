package ports

import (
	"context"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// NotificationEventSubscriber consumes notification events from an external stream.
type NotificationEventSubscriber interface {
	Consume(ctx context.Context, handler func(context.Context, string, domain.Notification) error) error
}

// NotificationBroadcaster pushes notifications to connected clients.
type NotificationBroadcaster interface {
	BroadcastNotification(ctx context.Context, cpfHash string, notification domain.Notification) error
}
