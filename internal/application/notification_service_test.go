package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type stubNotificationRepository struct {
	createFn func(ctx context.Context, n *domain.Notification) (bool, error)
}

func (s *stubNotificationRepository) Create(ctx context.Context, n *domain.Notification) (bool, error) {
	if s.createFn != nil {
		return s.createFn(ctx, n)
	}
	return true, nil
}

func (s *stubNotificationRepository) ListByOwner(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
	return nil, nil
}

func (s *stubNotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	return false, nil
}

func (s *stubNotificationRepository) CountUnread(ctx context.Context, cpfHash string) (int64, error) {
	return 0, nil
}

type recordingIdempotencyStore struct {
	exists map[string]bool
	sets   []string
	setErr error
}

func (s *recordingIdempotencyStore) Exists(ctx context.Context, key string) bool {
	return s.exists[key]
}

func (s *recordingIdempotencyStore) Set(ctx context.Context, key string) error {
	s.sets = append(s.sets, key)
	return s.setErr
}

type stubUnreadCache struct {
	invalidateErr error
	invalidated   []string
}

func (s *stubUnreadCache) Get(ctx context.Context, cpfHash string) (int64, error) {
	return 0, context.Canceled
}
func (s *stubUnreadCache) Set(ctx context.Context, cpfHash string, count int64) error {
	return nil
}
func (s *stubUnreadCache) Invalidate(ctx context.Context, cpfHash string) error {
	s.invalidated = append(s.invalidated, cpfHash)
	return s.invalidateErr
}

type stubPublisher struct {
	publishErr error
	published  []string
}

func (s *stubPublisher) Publish(ctx context.Context, cpfHash string, n *domain.Notification) error {
	s.published = append(s.published, cpfHash)
	return s.publishErr
}

type stubHasher struct{}

func (s *stubHasher) Hash(cpf string) string { return "hashed:" + cpf }

func TestBuildIdempotencyKeyPreservesSubsecondPrecision(t *testing.T) {
	base := domain.WebhookEvent{
		ChamadoID:  "CH-1",
		StatusNovo: "done",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
	}

	tests := []struct {
		name  string
		event domain.WebhookEvent
	}{
		{
			name:  "nanoseconds 123",
			event: base,
		},
		{
			name: "nanoseconds 456",
			event: domain.WebhookEvent{
				ChamadoID:  base.ChamadoID,
				StatusNovo: base.StatusNovo,
				Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 456, time.UTC),
			},
		},
	}

	keys := make([]string, 0, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys = append(keys, buildIdempotencyKey(tt.event))
		})
	}

	if keys[0] == keys[1] {
		t.Fatalf("expected different keys for sub-second timestamps, got %q", keys[0])
	}
}

func TestProcessWebhookUsesPreciseIdempotencyKey(t *testing.T) {
	idemp := &recordingIdempotencyStore{exists: map[string]bool{}}
	svc := NewWebhookProcessor(
		&stubNotificationRepository{},
		&stubUnreadCache{},
		idemp,
		&stubPublisher{},
		&stubHasher{},
	)

	events := []domain.WebhookEvent{
		{
			ChamadoID:  "CH-1",
			CPF:        "52998224725",
			Tipo:       "status_change",
			StatusNovo: "done",
			Titulo:     "First",
			Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
		},
		{
			ChamadoID:  "CH-1",
			CPF:        "529.982.247-25",
			Tipo:       "status_change",
			StatusNovo: "done",
			Titulo:     "Second",
			Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 456, time.UTC),
		},
	}

	for _, event := range events {
		if _, err := svc.ProcessWebhook(context.Background(), event); err != nil {
			t.Fatalf("ProcessWebhook returned error: %v", err)
		}
	}

	if len(idemp.sets) != 2 {
		t.Fatalf("expected 2 idempotency writes, got %d", len(idemp.sets))
	}
	if idemp.sets[0] == idemp.sets[1] {
		t.Fatalf("expected distinct idempotency keys, got %q", idemp.sets[0])
	}
}

func TestProcessWebhookRejectsInvalidCPF(t *testing.T) {
	svc := NewWebhookProcessor(
		&stubNotificationRepository{},
		&stubUnreadCache{},
		&recordingIdempotencyStore{exists: map[string]bool{}},
		&stubPublisher{},
		&stubHasher{},
	)

	_, err := svc.ProcessWebhook(context.Background(), domain.WebhookEvent{
		ChamadoID:  "CH-1",
		CPF:        "12345678901",
		Tipo:       "status_change",
		StatusNovo: "done",
		Titulo:     "Invalid",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
	})
	if !errors.Is(err, domain.ErrInvalidCPF) {
		t.Fatalf("expected ErrInvalidCPF, got %v", err)
	}
}

func TestProcessWebhookReturnsNilForDuplicateFromStore(t *testing.T) {
	idemp := &recordingIdempotencyStore{
		exists: map[string]bool{
			buildIdempotencyKey(domain.WebhookEvent{
				ChamadoID:  "CH-1",
				StatusNovo: "done",
				Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
			}): true,
		},
	}

	svc := NewWebhookProcessor(
		&stubNotificationRepository{createFn: func(ctx context.Context, n *domain.Notification) (bool, error) {
			t.Fatal("repo.Create should not be called when idempotency store already knows the event")
			return false, nil
		}},
		&stubUnreadCache{},
		idemp,
		&stubPublisher{},
		&stubHasher{},
	)

	got, err := svc.ProcessWebhook(context.Background(), domain.WebhookEvent{
		ChamadoID:  "CH-1",
		CPF:        "52998224725",
		Tipo:       "status_change",
		StatusNovo: "done",
		Titulo:     "Duplicate",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil notification for duplicate, got %#v", got)
	}
}

func TestProcessWebhookReturnsNilForDuplicateFromDB(t *testing.T) {
	svc := NewWebhookProcessor(
		&stubNotificationRepository{createFn: func(ctx context.Context, n *domain.Notification) (bool, error) {
			return false, nil
		}},
		&stubUnreadCache{},
		&recordingIdempotencyStore{exists: map[string]bool{}},
		&stubPublisher{},
		&stubHasher{},
	)

	got, err := svc.ProcessWebhook(context.Background(), domain.WebhookEvent{
		ChamadoID:  "CH-1",
		CPF:        "52998224725",
		Tipo:       "status_change",
		StatusNovo: "done",
		Titulo:     "Duplicate",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil notification for db duplicate, got %#v", got)
	}
}

func TestProcessWebhookWrapsRepositoryError(t *testing.T) {
	svc := NewWebhookProcessor(
		&stubNotificationRepository{createFn: func(ctx context.Context, n *domain.Notification) (bool, error) {
			return false, errors.New("db down")
		}},
		&stubUnreadCache{},
		&recordingIdempotencyStore{exists: map[string]bool{}},
		&stubPublisher{},
		&stubHasher{},
	)

	if _, err := svc.ProcessWebhook(context.Background(), domain.WebhookEvent{
		ChamadoID:  "CH-1",
		CPF:        "52998224725",
		Tipo:       "status_change",
		StatusNovo: "done",
		Titulo:     "Error",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
	}); err == nil {
		t.Fatal("expected wrapped repository error")
	}
}

func TestProcessWebhookSwallowsPostPersistSideEffectErrors(t *testing.T) {
	cache := &stubUnreadCache{invalidateErr: errors.New("cache down")}
	publisher := &stubPublisher{publishErr: errors.New("pubsub down")}
	idemp := &recordingIdempotencyStore{exists: map[string]bool{}, setErr: errors.New("redis down")}

	svc := NewWebhookProcessor(
		&stubNotificationRepository{createFn: func(ctx context.Context, n *domain.Notification) (bool, error) {
			return true, nil
		}},
		cache,
		idemp,
		publisher,
		&stubHasher{},
	)

	got, err := svc.ProcessWebhook(context.Background(), domain.WebhookEvent{
		ChamadoID:      "CH-1",
		CPF:            "52998224725",
		Tipo:           "status_change",
		StatusAnterior: "em_analise",
		StatusNovo:     "done",
		Titulo:         "Titulo",
		Descricao:      "Descricao",
		Timestamp:      time.Date(2026, 4, 21, 12, 0, 0, 123, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected created notification")
	}
	if got.StatusAnterior == nil || *got.StatusAnterior != "em_analise" {
		t.Fatalf("expected status_anterior to be set, got %#v", got.StatusAnterior)
	}
	if got.Descricao == nil || *got.Descricao != "Descricao" {
		t.Fatalf("expected descricao to be set, got %#v", got.Descricao)
	}
	if len(idemp.sets) != 1 || len(cache.invalidated) != 1 || len(publisher.published) != 1 {
		t.Fatalf("expected side effects to be attempted once, got sets=%d invalidations=%d publishes=%d", len(idemp.sets), len(cache.invalidated), len(publisher.published))
	}
}
