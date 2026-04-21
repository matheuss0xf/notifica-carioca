package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRouterRegistersRoutes(t *testing.T) {
	router := NewRouter(
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
		func(c *gin.Context) { c.Status(http.StatusAccepted) },
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Status(http.StatusCreated) },
		func(c *gin.Context) { c.Status(http.StatusOK) },
		func(c *gin.Context) { c.Status(http.StatusOK) },
		func(c *gin.Context) { c.Status(http.StatusOK) },
		func(c *gin.Context) { c.Status(http.StatusSwitchingProtocols) },
	)

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{name: "health", method: http.MethodGet, target: "/health", wantStatus: http.StatusNoContent},
		{name: "ready", method: http.MethodGet, target: "/ready", wantStatus: http.StatusAccepted},
		{name: "webhook", method: http.MethodPost, target: "/api/v1/webhooks/status-change", wantStatus: http.StatusCreated},
		{name: "list notifications", method: http.MethodGet, target: "/api/v1/notifications", wantStatus: http.StatusOK},
		{name: "unread count", method: http.MethodGet, target: "/api/v1/notifications/unread-count", wantStatus: http.StatusOK},
		{name: "mark as read", method: http.MethodPatch, target: "/api/v1/notifications/123/read", wantStatus: http.StatusOK},
		{name: "websocket route", method: http.MethodGet, target: "/ws", wantStatus: http.StatusSwitchingProtocols},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
