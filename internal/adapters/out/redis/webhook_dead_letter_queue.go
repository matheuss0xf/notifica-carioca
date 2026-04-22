package redis

import (
	"context"
	"encoding/json"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

const defaultWebhookDLQKey = "webhook:dlq"
const defaultWebhookDLQMaxLen = 1000

// WebhookDeadLetterQueue stores failed webhook persistence attempts in Redis.
type WebhookDeadLetterQueue struct {
	client *goredis.Client
	key    string
	maxLen int64
}

// NewWebhookDeadLetterQueue creates a Redis-backed dead letter queue.
func NewWebhookDeadLetterQueue(client *goredis.Client, key string, maxLen int64) *WebhookDeadLetterQueue {
	if key == "" {
		key = defaultWebhookDLQKey
	}
	if maxLen <= 0 {
		maxLen = defaultWebhookDLQMaxLen
	}

	return &WebhookDeadLetterQueue{
		client: client,
		key:    key,
		maxLen: maxLen,
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
	if err := q.client.LTrim(ctx, q.key, 0, q.maxLen-1).Err(); err != nil {
		return fmt.Errorf("trimming webhook dead letter queue: %w", err)
	}

	return nil
}
