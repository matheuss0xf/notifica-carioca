package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

func TestWebhookDeadLetterQueueEnqueue(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	queue := NewWebhookDeadLetterQueue(client, "webhook:dlq:test")

	deadLetter := domain.WebhookDeadLetter{
		FailedAt:       time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC),
		Stage:          "persistence",
		Reason:         "db down",
		CPFHash:        "hashed:52998224725",
		IdempotencyKey: "idemp:key",
		Event: domain.WebhookDeadLetterEvent{
			ChamadoID:  "CH-1",
			Tipo:       "status_change",
			StatusNovo: "em_execucao",
			Titulo:     "Titulo",
			Descricao:  "Descricao",
			Timestamp:  time.Date(2026, 4, 22, 1, 59, 0, 0, time.UTC),
		},
	}

	if err := queue.Enqueue(context.Background(), deadLetter); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	items, err := client.LRange(context.Background(), "webhook:dlq:test", 0, -1).Result()
	if err != nil {
		t.Fatalf("reading redis list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one dead-letter entry, got %d", len(items))
	}

	var got domain.WebhookDeadLetter
	if err := json.Unmarshal([]byte(items[0]), &got); err != nil {
		t.Fatalf("unmarshal dead-letter payload: %v", err)
	}

	if got.CPFHash != deadLetter.CPFHash || got.Event.ChamadoID != deadLetter.Event.ChamadoID {
		t.Fatalf("unexpected dead-letter payload: %#v", got)
	}
}
