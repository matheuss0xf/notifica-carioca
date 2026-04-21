package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type stubWebhookUseCase struct {
	processFn func(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error)
}

func (s *stubWebhookUseCase) ProcessWebhook(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error) {
	if s.processFn != nil {
		return s.processFn(ctx, event)
	}
	return nil, nil
}

func TestWebhookHandlerReturnsBadRequestForInvalidCPF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWebhookHandler(&stubWebhookUseCase{
		processFn: func(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error) {
			return nil, domain.ErrInvalidCPF
		},
	})

	body, err := json.Marshal(domain.WebhookEvent{
		ChamadoID:  "CH-1",
		Tipo:       "status_change",
		CPF:        "12345678901",
		StatusNovo: "done",
		Titulo:     "Titulo",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/status-change", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.HandleStatusChange(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var response httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if response.Code != "invalid_cpf" {
		t.Fatalf("expected invalid_cpf code, got %q", response.Code)
	}
	if response.Field != "cpf" {
		t.Fatalf("expected cpf field, got %q", response.Field)
	}
}

func TestWebhookHandlerReturnsCreatedForValidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	handler := NewWebhookHandler(&stubWebhookUseCase{
		processFn: func(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error) {
			return &domain.Notification{ID: id}, nil
		},
	})

	body, err := json.Marshal(domain.WebhookEvent{
		ChamadoID:  "CH-1",
		Tipo:       "status_change",
		CPF:        "52998224725",
		StatusNovo: "done",
		Titulo:     "Titulo",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/status-change", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.HandleStatusChange(c)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}

func TestWebhookHandlerReturnsProcessedMessageForDuplicateEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWebhookHandler(&stubWebhookUseCase{
		processFn: func(ctx context.Context, event domain.WebhookEvent) (*domain.Notification, error) {
			return nil, nil
		},
	})

	body, err := json.Marshal(domain.WebhookEvent{
		ChamadoID:  "CH-2024-001234",
		Tipo:       "status_change",
		CPF:        "52998224725",
		StatusNovo: "done",
		Titulo:     "Titulo",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/status-change", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.HandleStatusChange(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if response["message"] != "webhook already processed for chamado" {
		t.Fatalf("expected duplicate message, got %q", response["message"])
	}
	if response["chamado_id"] != "CH-2024-001234" {
		t.Fatalf("expected chamado_id in response, got %q", response["chamado_id"])
	}
}

func TestWebhookHandlerReturnsBadRequestForInvalidTipo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWebhookHandler(&stubWebhookUseCase{})

	body, err := json.Marshal(domain.WebhookEvent{
		ChamadoID:  "CH-1",
		Tipo:       "other_event",
		CPF:        "52998224725",
		StatusNovo: "done",
		Titulo:     "Titulo",
		Timestamp:  time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/status-change", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.HandleStatusChange(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var response httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if response.Code != "invalid_field" {
		t.Fatalf("expected invalid_field code, got %q", response.Code)
	}
	if response.Field != "tipo" {
		t.Fatalf("expected tipo field, got %q", response.Field)
	}
}

func TestValidateWebhookEventRequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		event     domain.WebhookEvent
		wantField string
	}{
		{name: "missing chamado_id", event: domain.WebhookEvent{}, wantField: "chamado_id"},
		{name: "missing tipo", event: domain.WebhookEvent{ChamadoID: "CH-1"}, wantField: "tipo"},
		{name: "missing cpf", event: domain.WebhookEvent{ChamadoID: "CH-1", Tipo: "status_change"}, wantField: "cpf"},
		{name: "missing status_novo", event: domain.WebhookEvent{ChamadoID: "CH-1", Tipo: "status_change", CPF: "52998224725"}, wantField: "status_novo"},
		{name: "missing titulo", event: domain.WebhookEvent{ChamadoID: "CH-1", Tipo: "status_change", CPF: "52998224725", StatusNovo: "done"}, wantField: "titulo"},
		{name: "missing timestamp", event: domain.WebhookEvent{ChamadoID: "CH-1", Tipo: "status_change", CPF: "52998224725", StatusNovo: "done", Titulo: "Titulo"}, wantField: "timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, _ := validateWebhookEvent(tt.event)
			if field != tt.wantField {
				t.Fatalf("expected field %q, got %q", tt.wantField, field)
			}
		})
	}
}
