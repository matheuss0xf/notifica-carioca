package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	cryptoutil "github.com/matheuss0xf/notifica-carioca/internal/infra/crypto"
)

func TestSignatureMiddlewareHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	body := `{"ok":true}`

	tests := []struct {
		name       string
		signature  string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing signature",
			body:       body,
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"code":"missing_signature"`,
		},
		{
			name:       "invalid signature",
			signature:  "sha256=deadbeef",
			body:       body,
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"code":"invalid_signature"`,
		},
		{
			name:       "valid signature forwards request body",
			signature:  cryptoutil.ComputeSignature([]byte(body), secret),
			body:       body,
			wantStatus: http.StatusOK,
			wantBody:   body,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(NewSignatureMiddleware(secret).Handle())
			router.POST("/webhook", func(c *gin.Context) {
				raw, _ := c.GetRawData()
				c.String(http.StatusOK, string(raw))
			})

			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tt.body))
			if tt.signature != "" {
				req.Header.Set("X-Signature-256", tt.signature)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("expected body to contain %q, got %s", tt.wantBody, rec.Body.String())
			}
		})
	}
}
