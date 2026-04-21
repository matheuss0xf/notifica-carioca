package application

import (
	"context"
	"errors"
	"testing"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type stubNotificationSubscriber struct {
	consumeFn func(ctx context.Context, handler func(context.Context, string, domain.Notification) error) error
}

func (s *stubNotificationSubscriber) Consume(
	ctx context.Context,
	handler func(context.Context, string, domain.Notification) error,
) error {
	if s.consumeFn != nil {
		return s.consumeFn(ctx, handler)
	}
	return nil
}

type recordingBroadcaster struct {
	cpfHash      string
	notification domain.Notification
	calls        int
	err          error
}

func (b *recordingBroadcaster) BroadcastNotification(
	ctx context.Context,
	cpfHash string,
	notification domain.Notification,
) error {
	_ = ctx
	b.calls++
	b.cpfHash = cpfHash
	b.notification = notification
	return b.err
}

func TestNotificationDispatcherRunBroadcastsConsumedEvents(t *testing.T) {
	expected := domain.Notification{ChamadoID: "CH-123"}
	broadcaster := &recordingBroadcaster{}
	dispatcher := NewNotificationDispatcher(
		&stubNotificationSubscriber{
			consumeFn: func(ctx context.Context, handler func(context.Context, string, domain.Notification) error) error {
				return handler(ctx, "cpf-hash", expected)
			},
		},
		broadcaster,
	)

	if err := dispatcher.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if broadcaster.calls != 1 {
		t.Fatalf("expected 1 broadcast, got %d", broadcaster.calls)
	}
	if broadcaster.cpfHash != "cpf-hash" {
		t.Fatalf("expected cpf-hash broadcast target, got %q", broadcaster.cpfHash)
	}
	if broadcaster.notification.ChamadoID != expected.ChamadoID {
		t.Fatalf("expected notification %q, got %q", expected.ChamadoID, broadcaster.notification.ChamadoID)
	}
}

func TestNotificationDispatcherRunWrapsBroadcastErrors(t *testing.T) {
	dispatcher := NewNotificationDispatcher(
		&stubNotificationSubscriber{
			consumeFn: func(ctx context.Context, handler func(context.Context, string, domain.Notification) error) error {
				return handler(ctx, "cpf-hash", domain.Notification{})
			},
		},
		&recordingBroadcaster{err: errors.New("ws down")},
	)

	err := dispatcher.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "broadcasting notification: ws down" {
		t.Fatalf("unexpected error: %v", err)
	}
}
