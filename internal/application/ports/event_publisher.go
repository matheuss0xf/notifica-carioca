package ports

import (
	"context"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// EventPublisher publishes notification events to connected subscribers.
type EventPublisher interface {
	Publish(ctx context.Context, cpfHash string, n *domain.Notification) error
}
