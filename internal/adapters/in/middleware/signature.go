package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/infra/crypto"
)

// SignatureMiddleware validates the X-Signature-256 HMAC header on webhook requests.
type SignatureMiddleware struct {
	secret string
}

// NewSignatureMiddleware creates a new webhook signature validation middleware.
func NewSignatureMiddleware(secret string) *SignatureMiddleware {
	return &SignatureMiddleware{secret: secret}
}

// Handle returns a Gin middleware that validates HMAC-SHA256 signatures.
// It reads the full body, validates the signature, then resets the body for downstream handlers.
func (m *SignatureMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Signature-256")
		if signature == "" {
			httpx.AbortError(c, http.StatusUnauthorized, "missing_signature", "missing signature")
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			httpx.AbortError(c, http.StatusBadRequest, "invalid_request_body", "failed to read body")
			return
		}

		if !crypto.ValidateSignature(body, signature, m.secret) {
			httpx.AbortError(c, http.StatusUnauthorized, "invalid_signature", "invalid signature")
			return
		}

		// Reset body for downstream handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Next()
	}
}
