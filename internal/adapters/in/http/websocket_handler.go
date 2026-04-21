package handler

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/middleware"
	ws "github.com/matheuss0xf/notifica-carioca/internal/adapters/out/websocket"
)

// WebSocketHandler handles WebSocket upgrade and registration.
type WebSocketHandler struct {
	hub            *ws.Hub
	auth           tokenAuthenticator
	allowedOrigins []string
}

type tokenAuthenticator interface {
	AuthenticateToken(tokenString string) (string, error)
}

// NewWebSocketHandler creates a new WebSocket handler.
func NewWebSocketHandler(hub *ws.Hub, auth tokenAuthenticator, allowedOrigins []string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:            hub,
		auth:           auth,
		allowedOrigins: append([]string(nil), allowedOrigins...),
	}
}

func (h *WebSocketHandler) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	return slices.Contains(h.allowedOrigins, origin)
}

// Handle upgrades an HTTP connection to WebSocket.
// GET /ws (Authorization header or ?token= query param)
func (h *WebSocketHandler) Handle(c *gin.Context) {
	// Extract JWT from header or query param fallback
	tokenString := middleware.ExtractToken(c.Request)
	if tokenString == "" {
		httpx.JSONError(c, http.StatusUnauthorized, "missing_token", "missing authorization token")
		return
	}

	cpfHash, err := h.auth.AuthenticateToken(tokenString)
	if err != nil {
		httpx.JSONError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
		return
	}

	// Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.originAllowed,
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "error", err)
		return
	}

	client := ws.NewClient(h.hub, conn, cpfHash)
	h.hub.Register(client)

	// Start read/write pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}
