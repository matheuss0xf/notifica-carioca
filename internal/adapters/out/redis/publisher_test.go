package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

func TestPublisherAndSubscriber(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	publisher := NewPublisher(client)
	subscriber := NewSubscriber(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	received := make(chan struct {
		cpfHash string
		n       domain.Notification
	}, 1)

	go func() {
		done <- subscriber.Consume(ctx, func(ctx context.Context, cpfHash string, n domain.Notification) error {
			received <- struct {
				cpfHash string
				n       domain.Notification
			}{cpfHash: cpfHash, n: n}
			cancel()
			return nil
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		subscribers, err := client.Publish(context.Background(), channelName, "probe").Result()
		if err != nil {
			t.Fatalf("publish probe: %v", err)
		}
		if subscribers > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for subscriber registration")
		}
		time.Sleep(10 * time.Millisecond)
	}

	notification := &domain.Notification{
		ID:        uuid.New(),
		ChamadoID: "CH-1",
		Titulo:    "Titulo",
	}

	if err := publisher.Publish(context.Background(), "hashed-cpf", notification); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	select {
	case got := <-received:
		if got.cpfHash != "hashed-cpf" || got.n.ID != notification.ID {
			t.Fatalf("unexpected received message: %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pubsub message")
	}

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected consume to stop on context cancellation, got %v", err)
	}
}
