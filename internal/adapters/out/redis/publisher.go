package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

const channelName = "notifications"

// publishedMessage is the payload sent over Redis Pub/Sub.
type publishedMessage struct {
	CPFHash      string              `json:"cpf_hash"`
	Notification domain.Notification `json:"notification"`
}

// Publisher implements ports.EventPublisher using Redis Pub/Sub.
type Publisher struct {
	client *goredis.Client
}

// NewPublisher creates a new Redis Pub/Sub publisher.
func NewPublisher(client *goredis.Client) *Publisher {
	return &Publisher{client: client}
}

// Publish sends a notification event to the shared channel.
func (p *Publisher) Publish(ctx context.Context, cpfHash string, n *domain.Notification) error {
	msg := publishedMessage{
		CPFHash:      cpfHash,
		Notification: *n,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling pubsub message: %w", err)
	}

	if err := p.client.Publish(ctx, channelName, data).Err(); err != nil {
		return fmt.Errorf("publishing to redis: %w", err)
	}

	return nil
}

// Subscriber listens to the shared notification channel.
type Subscriber struct {
	client *goredis.Client
}

// NewSubscriber creates a new Redis Pub/Sub subscriber.
func NewSubscriber(client *goredis.Client) *Subscriber {
	return &Subscriber{client: client}
}

// Consume starts listening on the notifications channel.
// The handler is called for each received message. Blocks until ctx is cancelled.
func (s *Subscriber) Consume(ctx context.Context, handler func(context.Context, string, domain.Notification) error) error {
	sub := s.client.Subscribe(ctx, channelName)
	defer func() {
		if closeErr := sub.Close(); closeErr != nil {
			slog.Warn("closing redis subscription", "error", closeErr)
		}
	}()

	// Wait for subscription confirmation
	if _, err := sub.Receive(ctx); err != nil {
		return fmt.Errorf("subscribing to channel %q: %w", channelName, err)
	}

	slog.Info("subscribed to redis pub/sub", "channel", channelName)

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}

			var m publishedMessage
			if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
				slog.Error("unmarshaling pubsub message", "error", err)
				continue
			}

			if err := handler(ctx, m.CPFHash, m.Notification); err != nil {
				slog.Error("handling pubsub message", "error", err)
			}
		}
	}
}
