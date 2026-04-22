package redis

import (
	"context"
	"encoding/json"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

const defaultWebhookDLQKey = "webhook:dlq"

// WebhookDeadLetterQueue stores failed webhook persistence attempts in Redis.
type WebhookDeadLetterQueue struct {
	client *goredis.Client
	key    string
}

// NewWebhookDeadLetterQueue creates a Redis-backed dead letter queue.
func NewWebhookDeadLetterQueue(client *goredis.Client, key string) *WebhookDeadLetterQueue {
	if key == "" {
		key = defaultWebhookDLQKey
	}

	return &WebhookDeadLetterQueue{
		client: client,
		key:    key,
	}
}

// Enqueue pushes a dead-letter payload into Redis for later inspection or replay.
func (q *WebhookDeadLetterQueue) Enqueue(ctx context.Context, deadLetter domain.WebhookDeadLetter) error {
	payload, err := json.Marshal(deadLetter)
	if err != nil {
		return fmt.Errorf("marshaling webhook dead letter: %w", err)
	}

	if err := q.client.LPush(ctx, q.key, payload).Err(); err != nil {
		return fmt.Errorf("pushing webhook dead letter: %w", err)
	}

	return nil
}
