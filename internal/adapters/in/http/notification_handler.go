package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/application/ports"
)

// NotificationHandler handles REST API requests for citizen notifications.
type NotificationHandler struct {
	lister  ports.NotificationLister
	counter ports.UnreadCounter
	marker  ports.NotificationMarker
}

// NewNotificationHandler creates a new notification REST handler.
func NewNotificationHandler(
	lister ports.NotificationLister,
	counter ports.UnreadCounter,
	marker ports.NotificationMarker,
) *NotificationHandler {
	return &NotificationHandler{
		lister:  lister,
		counter: counter,
		marker:  marker,
	}
}

// List returns paginated notifications for the authenticated citizen.
// GET /api/v1/notifications?cursor=<uuid>&limit=20
func (h *NotificationHandler) List(c *gin.Context) {
	cpfHash := c.GetString("cpf_hash")
	if cpfHash == "" {
		httpx.JSONError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var cursor *string
	if cursorParam := c.Query("cursor"); cursorParam != "" {
		cursor = &cursorParam
	}

	limit := 20
	if l := c.DefaultQuery("limit", "20"); l != "" {
		if parsed := parsePositiveInt(l, 50); parsed > 0 {
			limit = parsed
		}
	}

	page, err := h.lister.ListNotifications(c.Request.Context(), cpfHash, cursor, limit)
	if err != nil {
		httpx.JSONError(c, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	c.JSON(http.StatusOK, page)
}

// MarkAsRead marks a notification as read.
// PATCH /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	cpfHash := c.GetString("cpf_hash")
	if cpfHash == "" {
		httpx.JSONError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.JSONFieldError(c, http.StatusBadRequest, "invalid_notification_id", "invalid notification id", "id")
		return
	}

	updated, err := h.marker.MarkAsRead(c.Request.Context(), notificationID, cpfHash)
	if err != nil {
		httpx.JSONError(c, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	if !updated {
		httpx.JSONError(c, http.StatusNotFound, "notification_not_found", "notification not found or already read")
		return
	}

	c.JSON(http.StatusOK, httpx.MessageResponse{Message: "notification marked as read"})
}

// UnreadCount returns the total unread notifications for the authenticated citizen.
// GET /api/v1/notifications/unread-count
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	cpfHash := c.GetString("cpf_hash")
	if cpfHash == "" {
		httpx.JSONError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	count, err := h.counter.GetUnreadCount(c.Request.Context(), cpfHash)
	if err != nil {
		httpx.JSONError(c, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// parsePositiveInt safely parses a string to int, capping at max.
func parsePositiveInt(s string, max int) int {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
		if n > max {
			return max
		}
	}
	return n
}
