package middleware

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerPassesThroughRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restore := setTestLogger(t)
	defer restore()

	router := gin.New()
	router.Use(NewRequestLogger().Handle())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestRequestLoggerHandlesFallbackRouteAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restore := setTestLogger(t)
	defer restore()

	router := gin.New()
	router.Use(NewRequestLogger().Handle())
	router.GET("/boom", func(c *gin.Context) {
		_ = c.Error(errors.New("handler failed"))
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unmatched route, got %d", rec.Code)
	}
}

func TestRequestLoggerSkipsHealthAndReadyLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buffer := &bytes.Buffer{}
	restore := setBufferedLogger(t, buffer)
	defer restore()

	router := gin.New()
	router.Use(NewRequestLogger().Handle())
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/ready", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/notifications", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for _, path := range []string{"/health", "/ready", "/notifications"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	logOutput := buffer.String()
	if strings.Contains(logOutput, "/health") {
		t.Fatalf("expected /health to be skipped from logs, got %q", logOutput)
	}
	if strings.Contains(logOutput, "/ready") {
		t.Fatalf("expected /ready to be skipped from logs, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "/notifications") {
		t.Fatalf("expected non-probe route to be logged, got %q", logOutput)
	}
}

func setTestLogger(t *testing.T) func() {
	t.Helper()
	return setBufferedLogger(t, &bytes.Buffer{})
}

func setBufferedLogger(t *testing.T, buffer *bytes.Buffer) func() {
	t.Helper()

	previous := slog.Default()
	logger := slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	return func() {
		slog.SetDefault(previous)
	}
}
