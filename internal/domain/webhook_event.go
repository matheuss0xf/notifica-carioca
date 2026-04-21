package domain

import "time"

// WebhookEvent represents an incoming status-change event from the city's system.
type WebhookEvent struct {
	ChamadoID      string    `json:"chamado_id" binding:"required"`
	Tipo           string    `json:"tipo" binding:"required"`
	CPF            string    `json:"cpf" binding:"required"`
	StatusAnterior string    `json:"status_anterior"`
	StatusNovo     string    `json:"status_novo" binding:"required"`
	Titulo         string    `json:"titulo" binding:"required"`
	Descricao      string    `json:"descricao"`
	Timestamp      time.Time `json:"timestamp" binding:"required"`
}
