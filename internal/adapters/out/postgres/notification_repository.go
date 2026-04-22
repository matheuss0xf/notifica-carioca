package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// NotificationRepository implements ports.NotificationRepository using PostgreSQL.
type NotificationRepository struct {
	pool notificationStore
}

type notificationStore interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// NewNotificationRepository creates a new repository backed by a pgx connection pool.
func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

// Create inserts a notification using ON CONFLICT for idempotency.
// Returns (true, nil) if created, (false, nil) if duplicate.
func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) (bool, error) {
	query := `
		INSERT INTO notifications (id, chamado_id, cpf_hash, tipo, status_anterior, status_novo, titulo, descricao, event_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (chamado_id, status_novo, event_timestamp) DO NOTHING
		RETURNING id`

	var returnedID uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		n.ID, n.ChamadoID, n.CPFHash, n.Tipo,
		n.StatusAnterior, n.StatusNovo, n.Titulo,
		n.Descricao, n.EventTimestamp,
	).Scan(&returnedID)

	if err == pgx.ErrNoRows {
		return false, nil // Duplicate — idempotent
	}
	if err != nil {
		return false, fmt.Errorf("inserting notification: %w", err)
	}
	return true, nil
}

// ListByOwner returns paginated notifications for a CPF hash using cursor-based pagination.
// The cursor is the UUID of the last item from the previous page.
func (r *NotificationRepository) ListByOwner(ctx context.Context, cpfHash string, cursor *string, limit int) ([]domain.Notification, error) {
	var rows pgx.Rows
	var err error

	if cursor != nil {
		cursorID, parseErr := uuid.Parse(*cursor)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid cursor: %w", parseErr)
		}

		query := `
			SELECT id, chamado_id, tipo, status_anterior, status_novo,
			       titulo, descricao, read_at, event_timestamp, created_at
			FROM notifications
			WHERE cpf_hash = $1
			  AND (created_at, id) < (
			      SELECT created_at, id
			      FROM notifications
			      WHERE id = $2 AND cpf_hash = $1
			  )
			ORDER BY created_at DESC, id DESC
			LIMIT $3`

		rows, err = r.pool.Query(ctx, query, cpfHash, cursorID, limit+1)
	} else {
		query := `
			SELECT id, chamado_id, tipo, status_anterior, status_novo,
			       titulo, descricao, read_at, event_timestamp, created_at
			FROM notifications
			WHERE cpf_hash = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`

		rows, err = r.pool.Query(ctx, query, cpfHash, limit+1)
	}

	if err != nil {
		return nil, fmt.Errorf("querying notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]domain.Notification, 0, limit)
	for rows.Next() {
		var n domain.Notification
		if scanErr := rows.Scan(
			&n.ID, &n.ChamadoID, &n.Tipo, &n.StatusAnterior,
			&n.StatusNovo, &n.Titulo, &n.Descricao,
			&n.ReadAt, &n.EventTimestamp, &n.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scanning notification: %w", scanErr)
		}
		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// MarkAsRead sets read_at for a notification owned by cpfHash.
// Returns true if the notification was updated, false if not found or already read.
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	query := `
		UPDATE notifications
		SET read_at = NOW()
		WHERE id = $1 AND cpf_hash = $2 AND read_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id, cpfHash)
	if err != nil {
		return false, fmt.Errorf("marking notification as read: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// CountUnread returns the number of unread notifications for a CPF hash.
func (r *NotificationRepository) CountUnread(ctx context.Context, cpfHash string) (int64, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE cpf_hash = $1 AND read_at IS NULL`

	var count int64
	if err := r.pool.QueryRow(ctx, query, cpfHash).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}
