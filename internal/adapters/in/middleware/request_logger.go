package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger emits a structured access log for every HTTP request.
type RequestLogger struct{}

// NewRequestLogger creates a request logging middleware.
func NewRequestLogger() *RequestLogger {
	return &RequestLogger{}
}

// Handle returns a Gin middleware that logs method, path, status, latency, and client metadata.
func (m *RequestLogger) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		if shouldSkipRequestLog(route, c.Request.URL.Path) {
			return
		}

		latency := time.Since(start)
		status := c.Writer.Status()

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		logger := slog.Default().With(
			"method", c.Request.Method,
			"route", route,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		)

		if userAgent := c.Request.UserAgent(); userAgent != "" {
			logger = logger.With("user_agent", userAgent)
		}

		if len(c.Errors) > 0 {
			logger = logger.With("errors", c.Errors.String())
		}

		logger.Log(c.Request.Context(), level, "http request completed")
	}
}

func shouldSkipRequestLog(route, path string) bool {
	return route == "/health" || route == "/ready" || path == "/health" || path == "/ready"
}
