package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type markerRepoStub struct {
	markFn func(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error)
}

func (s *markerRepoStub) Create(ctx context.Context, n *domain.Notification) (bool, error) {
	return true, nil
}

func (s *markerRepoStub) ListByOwner(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
	return nil, nil
}

func (s *markerRepoStub) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	if s.markFn != nil {
		return s.markFn(ctx, id, cpfHash)
	}
	return false, nil
}

func (s *markerRepoStub) CountUnread(ctx context.Context, cpfHash string) (int64, error) {
	return 0, nil
}

type markerCacheStub struct {
	invalidateFn func(ctx context.Context, cpfHash string) error
}

func (s *markerCacheStub) Get(ctx context.Context, cpfHash string) (int64, error) { return 0, nil }
func (s *markerCacheStub) Set(ctx context.Context, cpfHash string, count int64) error {
	return nil
}
func (s *markerCacheStub) Invalidate(ctx context.Context, cpfHash string) error {
	if s.invalidateFn != nil {
		return s.invalidateFn(ctx, cpfHash)
	}
	return nil
}

func TestNotificationMarkerMarkAsRead(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name      string
		repo      *markerRepoStub
		cache     *markerCacheStub
		want      bool
		wantErr   bool
	}{
		{
			name: "updated invalidates cache",
			repo: &markerRepoStub{
				markFn: func(ctx context.Context, gotID uuid.UUID, cpfHash string) (bool, error) {
					if gotID != id {
						t.Fatalf("expected id %s, got %s", id, gotID)
					}
					if cpfHash != "hashed-cpf" {
						t.Fatalf("expected cpf hash hashed-cpf, got %q", cpfHash)
					}
					return true, nil
				},
			},
			cache: &markerCacheStub{
				invalidateFn: func(ctx context.Context, cpfHash string) error {
					if cpfHash != "hashed-cpf" {
						t.Fatalf("expected cpf hash hashed-cpf, got %q", cpfHash)
					}
					return nil
				},
			},
			want: true,
		},
		{
			name: "not updated does not fail",
			repo: &markerRepoStub{
				markFn: func(ctx context.Context, gotID uuid.UUID, cpfHash string) (bool, error) {
					return false, nil
				},
			},
			cache: &markerCacheStub{},
			want:  false,
		},
		{
			name: "cache invalidation failure is swallowed",
			repo: &markerRepoStub{
				markFn: func(ctx context.Context, gotID uuid.UUID, cpfHash string) (bool, error) {
					return true, nil
				},
			},
			cache: &markerCacheStub{
				invalidateFn: func(ctx context.Context, cpfHash string) error {
					return errors.New("cache down")
				},
			},
			want: true,
		},
		{
			name: "repo error is returned",
			repo: &markerRepoStub{
				markFn: func(ctx context.Context, gotID uuid.UUID, cpfHash string) (bool, error) {
					return false, errors.New("db down")
				},
			},
			cache:   &markerCacheStub{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marker := NewNotificationMarker(tt.repo, tt.cache)
			got, err := marker.MarkAsRead(context.Background(), id, "hashed-cpf")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected updated=%v, got %v", tt.want, got)
			}
		})
	}
}
