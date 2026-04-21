package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

type stubNotificationLister struct {
	listFn func(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error)
}

func (s *stubNotificationLister) ListNotifications(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error) {
	if s.listFn != nil {
		return s.listFn(ctx, cpfHash, cursor, limit)
	}
	return &domain.NotificationPage{}, nil
}

type stubUnreadCounter struct {
	countFn func(ctx context.Context, cpfHash string) (int64, error)
}

func (s *stubUnreadCounter) GetUnreadCount(ctx context.Context, cpfHash string) (int64, error) {
	if s.countFn != nil {
		return s.countFn(ctx, cpfHash)
	}
	return 0, nil
}

type stubNotificationMarkerUseCase struct {
	markFn func(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error)
}

func (s *stubNotificationMarkerUseCase) MarkAsRead(ctx context.Context, id uuid.UUID, cpfHash string) (bool, error) {
	if s.markFn != nil {
		return s.markFn(ctx, id, cpfHash)
	}
	return false, nil
}

func TestNotificationHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	expectedID := uuid.New()
	expectedCursor := "cursor-1"

	tests := []struct {
		name       string
		cpfHash    string
		rawQuery   string
		lister     *stubNotificationLister
		wantStatus int
		assertBody func(t *testing.T, body []byte)
	}{
		{
			name:       "unauthorized without cpf hash",
			wantStatus: http.StatusUnauthorized,
			assertBody: func(t *testing.T, body []byte) {
				t.Helper()
				var response httpx.ErrorResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if response.Code != "unauthorized" {
					t.Fatalf("expected unauthorized code, got %q", response.Code)
				}
			},
		},
		{
			name:    "success passes cursor and capped limit",
			cpfHash: "hashed-cpf",
			rawQuery: "cursor=abc&limit=999",
			lister: &stubNotificationLister{
				listFn: func(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error) {
					if cpfHash != "hashed-cpf" {
						t.Fatalf("expected cpfHash hashed-cpf, got %q", cpfHash)
					}
					if cursor == nil || *cursor != "abc" {
						t.Fatalf("expected cursor abc, got %#v", cursor)
					}
					if limit != 50 {
						t.Fatalf("expected capped limit 50, got %d", limit)
					}
					return &domain.NotificationPage{
						Data: []domain.Notification{{
							ID:        expectedID,
							ChamadoID: "CH-1",
							Tipo:      "status_change",
							StatusNovo:"done",
							Titulo:    "Titulo",
							CreatedAt: now,
						}},
						NextCursor: &expectedCursor,
						HasMore:    true,
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			assertBody: func(t *testing.T, body []byte) {
				t.Helper()
				var response domain.NotificationPage
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if len(response.Data) != 1 || response.Data[0].ID != expectedID {
					t.Fatalf("unexpected page data: %#v", response)
				}
				if response.NextCursor == nil || *response.NextCursor != expectedCursor {
					t.Fatalf("unexpected next cursor: %#v", response.NextCursor)
				}
				if !response.HasMore {
					t.Fatalf("expected has_more true")
				}
			},
		},
		{
			name:    "invalid limit falls back to default",
			cpfHash: "hashed-cpf",
			rawQuery: "limit=abc",
			lister: &stubNotificationLister{
				listFn: func(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error) {
					if cursor != nil {
						t.Fatalf("expected nil cursor, got %#v", cursor)
					}
					if limit != 20 {
						t.Fatalf("expected default limit 20, got %d", limit)
					}
					return &domain.NotificationPage{}, nil
				},
			},
			wantStatus: http.StatusOK,
			assertBody: func(t *testing.T, body []byte) {},
		},
		{
			name:    "internal error from lister",
			cpfHash: "hashed-cpf",
			lister: &stubNotificationLister{
				listFn: func(ctx context.Context, cpfHash string, cursor *string, limit int) (*domain.NotificationPage, error) {
					return nil, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
			assertBody: func(t *testing.T, body []byte) {
				t.Helper()
				var response httpx.ErrorResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if response.Code != "internal_error" {
					t.Fatalf("expected internal_error code, got %q", response.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewNotificationHandler(tt.lister, &stubUnreadCounter{}, &stubNotificationMarkerUseCase{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?"+tt.rawQuery, nil)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			if tt.cpfHash != "" {
				c.Set("cpf_hash", tt.cpfHash)
			}

			handler.List(c)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			tt.assertBody(t, rec.Body.Bytes())
		})
	}
}

func TestNotificationHandlerMarkAsRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()

	tests := []struct {
		name       string
		cpfHash    string
		paramID    string
		marker     *stubNotificationMarkerUseCase
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unauthorized without cpf hash",
			paramID:    id.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "invalid notification id",
			cpfHash:    "hashed-cpf",
			paramID:    "bad-id",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_notification_id",
		},
		{
			name:    "not found or already read",
			cpfHash: "hashed-cpf",
			paramID: id.String(),
			marker: &stubNotificationMarkerUseCase{
				markFn: func(ctx context.Context, gotID uuid.UUID, cpfHash string) (bool, error) {
					return false, nil
				},
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "notification_not_found",
		},
		{
			name:    "internal error",
			cpfHash: "hashed-cpf",
			paramID: id.String(),
			marker: &stubNotificationMarkerUseCase{
				markFn: func(ctx context.Context, gotID uuid.UUID, cpfHash string) (bool, error) {
					return false, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
		{
			name:    "marks as read successfully",
			cpfHash: "hashed-cpf",
			paramID: id.String(),
			marker: &stubNotificationMarkerUseCase{
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
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewNotificationHandler(&stubNotificationLister{}, &stubUnreadCounter{}, tt.marker)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+tt.paramID+"/read", nil)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.paramID}}
			if tt.cpfHash != "" {
				c.Set("cpf_hash", tt.cpfHash)
			}

			handler.MarkAsRead(c)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			if tt.wantCode != "" {
				var response httpx.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if response.Code != tt.wantCode {
					t.Fatalf("expected code %q, got %q", tt.wantCode, response.Code)
				}
			} else if !strings.Contains(rec.Body.String(), "notification marked as read") {
				t.Fatalf("unexpected success body: %s", rec.Body.String())
			}
		})
	}
}

func TestNotificationHandlerUnreadCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		cpfHash    string
		counter    *stubUnreadCounter
		wantStatus int
		wantBody   string
	}{
		{
			name:       "unauthorized without cpf hash",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"code":"unauthorized"`,
		},
		{
			name:    "internal error",
			cpfHash: "hashed-cpf",
			counter: &stubUnreadCounter{
				countFn: func(ctx context.Context, cpfHash string) (int64, error) {
					return 0, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `"code":"internal_error"`,
		},
		{
			name:    "success",
			cpfHash: "hashed-cpf",
			counter: &stubUnreadCounter{
				countFn: func(ctx context.Context, cpfHash string) (int64, error) {
					if cpfHash != "hashed-cpf" {
						t.Fatalf("expected cpf hash hashed-cpf, got %q", cpfHash)
					}
					return 7, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   `"count":7`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewNotificationHandler(&stubNotificationLister{}, tt.counter, &stubNotificationMarkerUseCase{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			if tt.cpfHash != "" {
				c.Set("cpf_hash", tt.cpfHash)
			}

			handler.UnreadCount(c)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("expected body to contain %q, got %s", tt.wantBody, rec.Body.String())
			}
		})
	}
}

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want int
	}{
		{name: "valid number", in: "10", max: 50, want: 10},
		{name: "caps at max", in: "999", max: 50, want: 50},
		{name: "invalid alpha", in: "1a", max: 50, want: 0},
		{name: "empty string", in: "", max: 50, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePositiveInt(tt.in, tt.max); got != tt.want {
				t.Fatalf("parsePositiveInt(%q, %d) = %d, want %d", tt.in, tt.max, got, tt.want)
			}
		})
	}
}
