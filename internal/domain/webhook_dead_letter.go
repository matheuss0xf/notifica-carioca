package domain

import "time"

// WebhookDeadLetter stores the minimum data needed to inspect or replay a webhook
// that failed after validation but before notification persistence completed.
type WebhookDeadLetter struct {
	FailedAt       time.Time           `json:"failed_at"`
	Stage          string              `json:"stage"`
	Reason         string              `json:"reason"`
	CPFHash        string              `json:"cpf_hash"`
	IdempotencyKey string              `json:"idempotency_key"`
	Event          WebhookDeadLetterEvent `json:"event"`
}

// WebhookDeadLetterEvent keeps the validated webhook payload without storing the raw CPF.
type WebhookDeadLetterEvent struct {
	ChamadoID      string    `json:"chamado_id"`
	Tipo           string    `json:"tipo"`
	StatusAnterior string    `json:"status_anterior,omitempty"`
	StatusNovo     string    `json:"status_novo"`
	Titulo         string    `json:"titulo"`
	Descricao      string    `json:"descricao,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}
