package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type readinessChecker interface {
	Check(ctx context.Context) (bool, int, error)
}

// HealthHandler exposes liveness and readiness endpoints.
type HealthHandler struct {
	readiness readinessChecker
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(readiness readinessChecker) *HealthHandler {
	return &HealthHandler{readiness: readiness}
}

// Liveness reports whether the process is up.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness reports whether runtime dependencies are healthy.
func (h *HealthHandler) Readiness(c *gin.Context) {
	ready, wsConnections, err := h.readiness.Check(c.Request.Context())
	if err != nil {
		slog.Warn("readiness check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":         "not_ready",
			"ws_connections": wsConnections,
		})
		return
	}

	status := "ready"
	if !ready {
		status = "not_ready"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         status,
		"ws_connections": wsConnections,
	})
}
