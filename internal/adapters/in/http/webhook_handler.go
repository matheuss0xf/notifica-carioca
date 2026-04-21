package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// WebhookHandler handles incoming webhook events from the city system.
type WebhookHandler struct {
	svc ports.WebhookUseCase
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(svc ports.WebhookUseCase) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// HandleStatusChange processes a status change webhook.
// POST /api/v1/webhooks/status-change
func (h *WebhookHandler) HandleStatusChange(c *gin.Context) {
	var event domain.WebhookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		httpx.JSONError(c, http.StatusBadRequest, "invalid_request", "invalid request payload")
		return
	}

	event = normalizeWebhookEvent(event)
	if field, message := validateWebhookEvent(event); field != "" {
		httpx.JSONFieldError(c, http.StatusBadRequest, "invalid_field", message, field)
		return
	}

	notification, err := h.svc.ProcessWebhook(c.Request.Context(), event)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCPF) {
			httpx.JSONFieldError(c, http.StatusBadRequest, "invalid_cpf", "invalid cpf", "cpf")
			return
		}
		httpx.JSONError(c, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	if notification == nil {
		c.JSON(http.StatusOK, gin.H{
			"message":    "webhook already processed for chamado",
			"chamado_id": event.ChamadoID,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "notification created",
		"notification_id": notification.ID,
	})
}

func normalizeWebhookEvent(event domain.WebhookEvent) domain.WebhookEvent {
	event.ChamadoID = strings.TrimSpace(event.ChamadoID)
	event.Tipo = strings.TrimSpace(event.Tipo)
	event.CPF = strings.TrimSpace(event.CPF)
	event.StatusAnterior = strings.TrimSpace(event.StatusAnterior)
	event.StatusNovo = strings.TrimSpace(event.StatusNovo)
	event.Titulo = strings.TrimSpace(event.Titulo)
	event.Descricao = strings.TrimSpace(event.Descricao)
	return event
}

func validateWebhookEvent(event domain.WebhookEvent) (string, string) {
	switch {
	case event.ChamadoID == "":
		return "chamado_id", "chamado_id is required"
	case event.Tipo == "":
		return "tipo", "tipo is required"
	case event.Tipo != "status_change":
		return "tipo", "tipo must be status_change"
	case event.CPF == "":
		return "cpf", "cpf is required"
	case event.StatusNovo == "":
		return "status_novo", "status_novo is required"
	case event.Titulo == "":
		return "titulo", "titulo is required"
	case event.Timestamp.IsZero():
		return "timestamp", "timestamp is required"
	default:
		return "", ""
	}
}
