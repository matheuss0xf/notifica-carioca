package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersHandleSetsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	headers := NewSecurityHeaders(true, 123)

	r := gin.New()
	r.Use(headers.Handle())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options nosniff, got %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatalf("expected Strict-Transport-Security to be set")
	}
}
