package application

import (
	"context"
	"fmt"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// NotificationDispatcher bridges notification events from a subscriber into a broadcaster.
type NotificationDispatcher struct {
	subscriber  ports.NotificationEventSubscriber
	broadcaster ports.NotificationBroadcaster
}

// NewNotificationDispatcher creates the event dispatching use case.
func NewNotificationDispatcher(
	subscriber ports.NotificationEventSubscriber,
	broadcaster ports.NotificationBroadcaster,
) *NotificationDispatcher {
	return &NotificationDispatcher{
		subscriber:  subscriber,
		broadcaster: broadcaster,
	}
}

// Run consumes notification events and forwards them to connected clients.
func (d *NotificationDispatcher) Run(ctx context.Context) error {
	return d.subscriber.Consume(ctx, func(ctx context.Context, cpfHash string, notification domain.Notification) error {
		if err := d.broadcaster.BroadcastNotification(ctx, cpfHash, notification); err != nil {
			return fmt.Errorf("broadcasting notification: %w", err)
		}
		return nil
	})
}
