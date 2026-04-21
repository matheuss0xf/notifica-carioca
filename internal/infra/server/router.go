package server

import (
	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures the Gin router with all routes.
func NewRouter(
	requestLogger gin.HandlerFunc,
	handleLiveness gin.HandlerFunc,
	handleReadiness gin.HandlerFunc,
	webhookAuth gin.HandlerFunc,
	apiAuth gin.HandlerFunc,
	handleWebhook gin.HandlerFunc,
	listNotifications gin.HandlerFunc,
	unreadCount gin.HandlerFunc,
	markAsRead gin.HandlerFunc,
	handleWebSocket gin.HandlerFunc,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger)

	// Health checks (no auth)
	r.GET("/health", handleLiveness)
	r.GET("/ready", handleReadiness)

	v1 := r.Group("/api/v1")

	webhooks := v1.Group("/webhooks")
	webhooks.Use(webhookAuth)
	{
		webhooks.POST("/status-change", handleWebhook)
	}

	notifications := v1.Group("/notifications")
	notifications.Use(apiAuth)
	{
		notifications.GET("", listNotifications)
		notifications.GET("/unread-count", unreadCount)
		notifications.PATCH("/:id/read", markAsRead)
	}

	r.GET("/ws", handleWebSocket)

	return r
}
