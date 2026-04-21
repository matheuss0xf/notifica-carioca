package domain

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a citizen notification (domain entity).
type Notification struct {
	ID             uuid.UUID  `json:"id"`
	ChamadoID      string     `json:"chamado_id"`
	CPFHash        string     `json:"-"` // Never exposed in API responses
	Tipo           string     `json:"tipo"`
	StatusAnterior *string    `json:"status_anterior,omitempty"`
	StatusNovo     string     `json:"status_novo"`
	Titulo         string     `json:"titulo"`
	Descricao      *string    `json:"descricao,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	EventTimestamp time.Time  `json:"event_timestamp"`
	CreatedAt      time.Time  `json:"created_at"`
}

// NotificationPage represents a paginated response with cursor-based pagination.
type NotificationPage struct {
	Data       []Notification `json:"data"`
	NextCursor *string        `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}
