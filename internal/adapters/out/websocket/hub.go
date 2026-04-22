package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendBufSize    = 256
)

// Client represents a single WebSocket connection for a citizen.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	CpfHash string
	send    chan []byte
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, cpfHash string) *Client {
	return &Client{
		hub:     hub,
		conn:    conn,
		CpfHash: cpfHash,
		send:    make(chan []byte, sendBufSize),
	}
}

// WritePump sends messages from the send channel to the WebSocket connection.
// Runs in its own goroutine per client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			slog.Debug("ws close error", "error", err)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Error("ws set write deadline error", "error", err)
				return
			}
			if !ok {
				// Hub closed the channel
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					slog.Debug("ws close frame error", "error", err)
				}
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Error("ws write error", "error", err)
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Error("ws set write deadline error", "error", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump reads messages from the WebSocket (handles pong, detects disconnect).
// Runs in its own goroutine per client.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		if err := c.conn.Close(); err != nil {
			slog.Debug("ws close error", "error", err)
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Error("ws set read deadline error", "error", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			slog.Error("ws set read deadline error", "error", err)
			return err
		}
		return nil
	})

	// We don't expect messages from the client, just keep reading to detect disconnect
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Hub manages all active WebSocket connections grouped by cpf_hash.
type Hub struct {
	mu          sync.RWMutex
	connections map[string]map[*Client]struct{} // cpf_hash -> set of clients
}

func redactCPFHash(cpfHash string) string {
	if len(cpfHash) <= 8 {
		return cpfHash
	}
	return cpfHash[:8] + "..."
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]map[*Client]struct{}),
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.connections[client.CpfHash]; !ok {
		h.connections[client.CpfHash] = make(map[*Client]struct{})
	}
	h.connections[client.CpfHash][client] = struct{}{}

	slog.Info("ws client registered", "cpf_hash", redactCPFHash(client.CpfHash))
}

// Unregister removes a client from the hub and closes its send channel.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.connections[client.CpfHash]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)

			if len(clients) == 0 {
				delete(h.connections, client.CpfHash)
			}
		}
	}

	slog.Info("ws client unregistered", "cpf_hash", redactCPFHash(client.CpfHash))
}

func (h *Hub) clientsForCPF(cpfHash string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.connections[cpfHash]
	if !ok {
		return nil
	}

	snapshot := make([]*Client, 0, len(clients))
	for client := range clients {
		snapshot = append(snapshot, client)
	}

	return snapshot
}

// BroadcastNotification sends a notification to all connected clients for a given cpf_hash.
func (h *Hub) BroadcastNotification(ctx context.Context, cpfHash string, notification domain.Notification) error {
	_ = ctx

	data, err := json.Marshal(notification)
	if err != nil {
		slog.Error("ws broadcast marshal error", "error", err)
		return err
	}

	clients := h.clientsForCPF(cpfHash)
	if len(clients) == 0 {
		return nil
	}

	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			// Client buffer full — disconnect
			slog.Warn("ws client buffer full, disconnecting", "cpf_hash", redactCPFHash(cpfHash))
			go h.Unregister(client)
		}
	}

	return nil
}

// Broadcast preserves the existing hub API for callers that do not use the application port.
func (h *Hub) Broadcast(cpfHash string, notification domain.Notification) {
	if err := h.BroadcastNotification(context.Background(), cpfHash, notification); err != nil {
		slog.Error("ws broadcast error", "error", err)
	}
}

// ConnectedCount returns the total number of active connections.
func (h *Hub) ConnectedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.connections {
		count += len(clients)
	}
	return count
}
