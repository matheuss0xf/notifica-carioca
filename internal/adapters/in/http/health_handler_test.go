package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubReadinessChecker struct {
	checkFn func(ctx context.Context) (bool, int, error)
}

func (s *stubReadinessChecker) Check(ctx context.Context) (bool, int, error) {
	if s.checkFn != nil {
		return s.checkFn(ctx)
	}
	return true, 0, nil
}

func TestHealthHandlerReadinessSanitizesDependencyErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(&stubReadinessChecker{
		checkFn: func(ctx context.Context) (bool, int, error) {
			return false, 3, errors.New("dial tcp redis.internal:6379: connect: connection refused")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.Readiness(c)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if _, ok := body["error"]; ok {
		t.Fatal("expected readiness response to omit raw error details")
	}
	if got := body["status"]; got != "not_ready" {
		t.Fatalf("expected status not_ready, got %v", got)
	}
}

func TestHealthHandlerLiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(&stubReadinessChecker{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.Liveness(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "{\"status\":\"ok\"}" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHealthHandlerReadinessReturnsNotReadyWith200WhenCheckerReportsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(&stubReadinessChecker{
		checkFn: func(ctx context.Context) (bool, int, error) {
			return false, 2, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	handler.Readiness(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "{\"status\":\"not_ready\",\"ws_connections\":2}" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
