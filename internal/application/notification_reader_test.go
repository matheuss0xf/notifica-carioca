package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type readerRepoStub struct {
	listFn  func(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error)
	countFn func(ctx context.Context, cpfHash string) (int64, error)
}

func (s *readerRepoStub) Create(ctx context.Context, n *domain.Notification) (bool, error) {
	return true, nil
}

func (s *readerRepoStub) ListByOwner(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
	if s.listFn != nil {
		return s.listFn(ctx, cpfHash, cursor, limit)
	}
	return nil, nil
}

func (s *readerRepoStub) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	return false, nil
}

func (s *readerRepoStub) CountUnread(ctx context.Context, cpfHash string) (int64, error) {
	if s.countFn != nil {
		return s.countFn(ctx, cpfHash)
	}
	return 0, nil
}

type readerCacheStub struct {
	getFn        func(ctx context.Context, cpfHash string) (int64, error)
	setFn        func(ctx context.Context, cpfHash string, count int64) error
	invalidateFn func(ctx context.Context, cpfHash string) error
}

func (s *readerCacheStub) Get(ctx context.Context, cpfHash string) (int64, error) {
	if s.getFn != nil {
		return s.getFn(ctx, cpfHash)
	}
	return 0, ports.ErrCacheMiss
}

func (s *readerCacheStub) Set(ctx context.Context, cpfHash string, count int64) error {
	if s.setFn != nil {
		return s.setFn(ctx, cpfHash, count)
	}
	return nil
}

func (s *readerCacheStub) Invalidate(ctx context.Context, cpfHash string) error {
	if s.invalidateFn != nil {
		return s.invalidateFn(ctx, cpfHash)
	}
	return nil
}

func TestNotificationReaderListNotifications(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	thirdID := uuid.New()

	tests := []struct {
		name      string
		limit     int
		cursor    *string
		repo      *readerRepoStub
		assertion func(t *testing.T, page *domain.NotificationPage, err error)
	}{
		{
			name:  "invalid limit falls back to default and computes next cursor",
			limit: 0,
			cursor: func() *string {
				s := "cursor-1"
				return &s
			}(),
			repo: &readerRepoStub{
				listFn: func(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
					if limit != 20 {
						t.Fatalf("expected default limit 20, got %d", limit)
					}
					if cursor == nil || *cursor != "cursor-1" {
						t.Fatalf("unexpected cursor: %#v", cursor)
					}
					return []domain.Notification{
						{ID: firstID},
						{ID: secondID},
						{ID: thirdID},
					}, nil
				},
			},
			assertion: func(t *testing.T, page *domain.NotificationPage, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if page == nil || len(page.Data) != 3 || page.HasMore {
					t.Fatalf("unexpected page for default limit: %#v", page)
				}
			},
		},
		{
			name:  "trims extra item and returns next cursor",
			limit: 2,
			repo: &readerRepoStub{
				listFn: func(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
					if limit != 2 {
						t.Fatalf("expected limit 2, got %d", limit)
					}
					return []domain.Notification{
						{ID: firstID},
						{ID: secondID},
						{ID: thirdID},
					}, nil
				},
			},
			assertion: func(t *testing.T, page *domain.NotificationPage, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !page.HasMore {
					t.Fatalf("expected HasMore true")
				}
				if len(page.Data) != 2 {
					t.Fatalf("expected 2 notifications, got %d", len(page.Data))
				}
				if page.NextCursor == nil || *page.NextCursor != secondID.String() {
					t.Fatalf("unexpected next cursor: %#v", page.NextCursor)
				}
			},
		},
		{
			name:  "wraps repository error",
			limit: 10,
			repo: &readerRepoStub{
				listFn: func(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
					return nil, errors.New("db down")
				},
			},
			assertion: func(t *testing.T, page *domain.NotificationPage, err error) {
				if err == nil || page != nil {
					t.Fatalf("expected error, got page=%#v err=%v", page, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewNotificationReader(tt.repo, &readerCacheStub{})
			page, err := reader.ListNotifications(context.Background(), "hashed-cpf", tt.cursor, tt.limit)
			tt.assertion(t, page, err)
		})
	}
}

func TestNotificationReaderGetUnreadCount(t *testing.T) {
	tests := []struct {
		name      string
		repo      *readerRepoStub
		cache     *readerCacheStub
		wantCount int64
		wantErr   bool
	}{
		{
			name: "uses cached count",
			cache: &readerCacheStub{
				getFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 9, nil
				},
			},
			repo:      &readerRepoStub{},
			wantCount: 9,
		},
		{
			name: "cache miss falls back to repo and stores cache",
			cache: &readerCacheStub{
				getFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 0, ports.ErrCacheMiss
				},
				setFn: func(ctx context.Context, cpfHash string, count int64) error {
					if cpfHash != "hashed-cpf" || count != 4 {
						t.Fatalf("unexpected cache set cpf=%q count=%d", cpfHash, count)
					}
					return nil
				},
			},
			repo: &readerRepoStub{
				countFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 4, nil
				},
			},
			wantCount: 4,
		},
		{
			name: "cache set failure does not fail request",
			cache: &readerCacheStub{
				getFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 0, ports.ErrCacheMiss
				},
				setFn: func(ctx context.Context, cpfHash string, count int64) error {
					return errors.New("cache down")
				},
			},
			repo: &readerRepoStub{
				countFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 2, nil
				},
			},
			wantCount: 2,
		},
		{
			name: "repository error is returned",
			cache: &readerCacheStub{
				getFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 0, ports.ErrCacheMiss
				},
			},
			repo: &readerRepoStub{
				countFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 0, errors.New("db down")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewNotificationReader(tt.repo, tt.cache)
			got, err := reader.GetUnreadCount(context.Background(), "hashed-cpf")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantCount {
				t.Fatalf("expected count %d, got %d", tt.wantCount, got)
			}
		})
	}
}
