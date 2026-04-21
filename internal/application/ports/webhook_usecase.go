package ports

import (
	"context"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// WebhookUseCase defines the inbound application boundary for webhook processing.
type WebhookUseCase interface {
	ProcessWebhook(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error)
}
