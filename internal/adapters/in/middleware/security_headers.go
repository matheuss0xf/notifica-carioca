package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets a baseline of defensive HTTP response headers.
type SecurityHeaders struct {
	enableHSTS        bool
	hstsMaxAgeSeconds int
}

// NewSecurityHeaders creates a new security headers middleware.
func NewSecurityHeaders(enableHSTS bool, hstsMaxAgeSeconds int) *SecurityHeaders {
	return &SecurityHeaders{
		enableHSTS:        enableHSTS,
		hstsMaxAgeSeconds: hstsMaxAgeSeconds,
	}
}

// Handle returns a Gin middleware that appends security headers to all responses.
func (m *SecurityHeaders) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if m.enableHSTS && m.hstsMaxAgeSeconds > 0 {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", m.hstsMaxAgeSeconds))
		}
		c.Next()
	}
}
