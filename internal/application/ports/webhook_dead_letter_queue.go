package ports

import (
	"context"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// WebhookDeadLetterQueue stores validated webhook events that failed during persistence.
type WebhookDeadLetterQueue interface {
	Enqueue(ctx context.Context, deadLetter domain.WebhookDeadLetter) error
}
