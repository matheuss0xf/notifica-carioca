package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// WebhookProcessor orchestrates incoming webhook events into persisted notifications.
type WebhookProcessor struct {
	repo      ports.NotificationRepository
	cache     ports.UnreadCache
	idemp     ports.IdempotencyStore
	dlq       ports.WebhookDeadLetterQueue
	publisher ports.EventPublisher
	hasher    ports.CPFHasher
}

// NewWebhookProcessor creates the webhook processing use case.
func NewWebhookProcessor(
	repo ports.NotificationRepository,
	cache ports.UnreadCache,
	idemp ports.IdempotencyStore,
	dlq ports.WebhookDeadLetterQueue,
	publisher ports.EventPublisher,
	hasher ports.CPFHasher,
) *WebhookProcessor {
	return &WebhookProcessor{
		repo:      repo,
		cache:     cache,
		idemp:     idemp,
		dlq:       dlq,
		publisher: publisher,
		hasher:    hasher,
	}
}

// ProcessWebhook handles an incoming webhook event.
// Returns the created notification or nil if it was a duplicate.
func (p *WebhookProcessor) ProcessWebhook(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error) {
	cpf, err := domain.ValidateCPF(event.CPF)
	if err != nil {
		return nil, err
	}

	cpfHash := p.hasher.Hash(cpf)
	idempKey := buildIdempotencyKey(event)

	if p.idemp.Exists(ctx, idempKey) {
		slog.Debug("webhook duplicate detected via dedup store", "chamado_id", event.ChamadoID)
		return nil, nil
	}

	var statusAnterior *string
	if event.StatusAnterior != "" {
		statusAnterior = &event.StatusAnterior
	}
	var descricao *string
	if event.Descricao != "" {
		descricao = &event.Descricao
	}

	notification := &domain.Notification{
		ID:             uuid.New(),
		ChamadoID:      event.ChamadoID,
		CPFHash:        cpfHash,
		Tipo:           event.Tipo,
		StatusAnterior: statusAnterior,
		StatusNovo:     event.StatusNovo,
		Titulo:         event.Titulo,
		Descricao:      descricao,
		EventTimestamp: event.Timestamp,
	}

	created, err := p.repo.Create(ctx, notification)
	if err != nil {
		deadLetter := domain.WebhookDeadLetter{
			FailedAt:       time.Now().UTC(),
			Stage:          "persistence",
			Reason:         err.Error(),
			CPFHash:        cpfHash,
			IdempotencyKey: idempKey,
			Event: domain.WebhookDeadLetterEvent{
				ChamadoID:      event.ChamadoID,
				Tipo:           event.Tipo,
				StatusAnterior: event.StatusAnterior,
				StatusNovo:     event.StatusNovo,
				Titulo:         event.Titulo,
				Descricao:      event.Descricao,
				Timestamp:      event.Timestamp,
			},
		}
		if dlqErr := p.dlq.Enqueue(ctx, deadLetter); dlqErr != nil {
			slog.Error("failed to enqueue webhook dead letter", "error", dlqErr, "chamado_id", event.ChamadoID)
		}
		return nil, fmt.Errorf("creating notification: %w", err)
	}
	if !created {
		slog.Debug("webhook duplicate detected via db constraint", "chamado_id", event.ChamadoID)
		return nil, nil
	}

	if setErr := p.idemp.Set(ctx, idempKey); setErr != nil {
		slog.Warn("failed to set idempotency key", "error", setErr)
	}

	if invErr := p.cache.Invalidate(ctx, cpfHash); invErr != nil {
		slog.Warn("failed to invalidate unread cache", "error", invErr)
	}

	if pubErr := p.publisher.Publish(ctx, cpfHash, notification); pubErr != nil {
		slog.Error("failed to publish notification", "error", pubErr, "notification_id", notification.ID)
	}

	slog.Info("notification created",
		"notification_id", notification.ID,
		"chamado_id", event.ChamadoID,
		"status", event.StatusNovo,
	)

	return notification, nil
}
