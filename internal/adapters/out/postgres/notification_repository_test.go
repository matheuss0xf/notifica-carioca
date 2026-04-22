package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return nil
}

type fakeRows struct {
	items []domain.Notification
	index int
	err   error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT 0") }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.index >= len(r.items) {
		return false
	}
	r.index++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	item := r.items[r.index-1]
	*(dest[0].(*uuid.UUID)) = item.ID
	*(dest[1].(*string)) = item.ChamadoID
	*(dest[2].(*string)) = item.Tipo
	*(dest[3].(**string)) = item.StatusAnterior
	*(dest[4].(*string)) = item.StatusNovo
	*(dest[5].(*string)) = item.Titulo
	*(dest[6].(**string)) = item.Descricao
	*(dest[7].(**time.Time)) = item.ReadAt
	*(dest[8].(*time.Time)) = item.EventTimestamp
	*(dest[9].(*time.Time)) = item.CreatedAt
	return nil
}
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

type fakeStore struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *fakeStore) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return s.queryRowFn(ctx, sql, args...)
}

func (s *fakeStore) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return s.queryFn(ctx, sql, args...)
}

func (s *fakeStore) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return s.execFn(ctx, sql, args...)
}

func TestNotificationRepositoryCreate(t *testing.T) {
	id := uuid.New()
	notification := &domain.Notification{ID: id}

	tests := []struct {
		name    string
		store   *fakeStore
		want    bool
		wantErr bool
	}{
		{
			name: "created",
			store: &fakeStore{
				queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*uuid.UUID)) = id
						return nil
					}}
				},
			},
			want: true,
		},
		{
			name: "duplicate returns false",
			store: &fakeStore{
				queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return fakeRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
				},
			},
		},
		{
			name: "scan error",
			store: &fakeStore{
				queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return fakeRow{scanFn: func(dest ...any) error { return errors.New("db down") }}
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &NotificationRepository{pool: tt.store}
			got, err := repo.Create(context.Background(), notification)
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
				t.Fatalf("expected created=%v, got %v", tt.want, got)
			}
		})
	}
}

func TestNotificationRepositoryListByOwner(t *testing.T) {
	now := time.Now().UTC()
	statusAnterior := "old"
	descricao := "desc"
	id := uuid.New()
	cursor := uuid.New().String()

	tests := []struct {
		name    string
		cursor  *string
		store   *fakeStore
		wantLen int
		wantErr bool
	}{
		{
			name:    "invalid cursor",
			cursor:  func() *string { s := "bad-cursor"; return &s }(),
			store:   &fakeStore{},
			wantErr: true,
		},
		{
			name:   "lists without cursor",
			cursor: nil,
			store: &fakeStore{
				queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
					if args[1].(int) != 3 {
						t.Fatalf("expected limit+1=3, got %v", args[1])
					}
					return &fakeRows{items: []domain.Notification{{
						ID:             id,
						ChamadoID:      "CH-1",
						Tipo:           "status_change",
						StatusAnterior: &statusAnterior,
						StatusNovo:     "done",
						Titulo:         "Titulo",
						Descricao:      &descricao,
						EventTimestamp: now,
						CreatedAt:      now,
					}}}, nil
				},
			},
			wantLen: 1,
		},
		{
			name:   "lists with cursor",
			cursor: &cursor,
			store: &fakeStore{
				queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
					if _, ok := args[1].(uuid.UUID); !ok {
						t.Fatalf("expected parsed uuid cursor, got %T", args[1])
					}
					if args[2].(int) != 3 {
						t.Fatalf("expected limit+1=3, got %v", args[2])
					}
					return &fakeRows{}, nil
				},
			},
		},
		{
			name:   "query error",
			cursor: nil,
			store: &fakeStore{
				queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
					return nil, errors.New("db down")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &NotificationRepository{pool: tt.store}
			got, err := repo.ListByOwner(context.Background(), "hashed", tt.cursor, 2)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d notifications, got %d", tt.wantLen, len(got))
			}
		})
	}
}

func TestNotificationRepositoryMarkAsReadAndCountUnread(t *testing.T) {
	id := uuid.New()

	repo := &NotificationRepository{pool: &fakeStore{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int64)) = 5
				return nil
			}}
		},
	}}

	updated, err := repo.MarkAsRead(context.Background(), id, "hashed")
	if err != nil {
		t.Fatalf("unexpected error marking as read: %v", err)
	}
	if !updated {
		t.Fatalf("expected updated=true")
	}

	count, err := repo.CountUnread(context.Background(), "hashed")
	if err != nil {
		t.Fatalf("unexpected error counting unread: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected unread count 5, got %d", count)
	}
}

func TestNotificationRepositoryErrors(t *testing.T) {
	id := uuid.New()
	repo := &NotificationRepository{pool: &fakeStore{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db down")
		},
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return fakeRow{scanFn: func(dest ...any) error { return errors.New("db down") }}
		},
	}}

	if _, err := repo.MarkAsRead(context.Background(), id, "hashed"); err == nil {
		t.Fatalf("expected mark as read error")
	}
	if _, err := repo.CountUnread(context.Background(), "hashed"); err == nil {
		t.Fatalf("expected count unread error")
	}
}
