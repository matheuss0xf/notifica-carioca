package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

func TestClientsForCPFReturnsSnapshot(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "cpf-1")
	hub.Register(client)

	snapshot := hub.clientsForCPF("cpf-1")
	hub.Unregister(client)

	if len(snapshot) != 1 {
		t.Fatalf("expected snapshot with one client, got %d", len(snapshot))
	}
	if snapshot[0] != client {
		t.Fatal("expected snapshot to preserve the original client pointer")
	}
}

func TestBroadcastSendsToRegisteredClients(t *testing.T) {
	hub := NewHub()
	clientA := NewClient(hub, nil, "cpf-1")
	clientB := NewClient(hub, nil, "cpf-1")
	hub.Register(clientA)
	hub.Register(clientB)

	notification := domain.Notification{
		ID:        uuid.New(),
		ChamadoID: "CH-1",
		Titulo:    "title",
	}

	hub.Broadcast("cpf-1", notification)

	for _, client := range []*Client{clientA, clientB} {
		select {
		case msg := <-client.send:
			var got domain.Notification
			if err := json.Unmarshal(msg, &got); err != nil {
				t.Fatalf("failed to unmarshal broadcast payload: %v", err)
			}
			if got.ID != notification.ID {
				t.Fatalf("expected notification %s, got %s", notification.ID, got.ID)
			}
		default:
			t.Fatal("expected client to receive broadcast")
		}
	}
}

func TestConnectedCountTracksRegisteredClients(t *testing.T) {
	hub := NewHub()
	clientA := NewClient(hub, nil, "cpf-1")
	clientB := NewClient(hub, nil, "cpf-2")

	hub.Register(clientA)
	hub.Register(clientB)

	if got := hub.ConnectedCount(); got != 2 {
		t.Fatalf("expected 2 connections, got %d", got)
	}

	hub.Unregister(clientA)

	if got := hub.ConnectedCount(); got != 1 {
		t.Fatalf("expected 1 connection after unregister, got %d", got)
	}
}

func TestRedactCPFHash(t *testing.T) {
	if got := redactCPFHash("1234567"); got != "1234567" {
		t.Fatalf("expected short hash unchanged, got %q", got)
	}
	if got := redactCPFHash("1234567890"); got != "12345678..." {
		t.Fatalf("expected redacted hash, got %q", got)
	}
}

func TestBroadcastNotificationWithNoClientsDoesNothing(t *testing.T) {
	hub := NewHub()
	if err := hub.BroadcastNotification(context.Background(), "missing", domain.Notification{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestBroadcastNotificationDisconnectsClientWithFullBuffer(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "cpf-1")
	hub.Register(client)

	for i := 0; i < sendBufSize; i++ {
		client.send <- []byte("filled")
	}

	if err := hub.BroadcastNotification(context.Background(), "cpf-1", domain.Notification{ID: uuid.New()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for hub.ConnectedCount() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("expected client to be unregistered after buffer overflow")
		}
	}
}
