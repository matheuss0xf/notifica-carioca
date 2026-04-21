package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// NotificationRepository defines persistence operations for notifications.
type NotificationRepository interface {
	// Create inserts a notification. Returns true if created, false if duplicate.
	Create(ctx context.Context, n *domain.Notification) (created bool, err error)

	// ListByOwner returns paginated notifications for a CPF hash.
	// cursor is the UUID of the last item from the previous page.
	ListByOwner(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error)

	// MarkAsRead sets read_at for a notification owned by cpfHash.
	// Returns true if updated, false if not found or already read.
	MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error)

	// CountUnread returns the number of unread notifications for a CPF hash.
	CountUnread(ctx context.Context, cpfHash string) (int64, error)
}
