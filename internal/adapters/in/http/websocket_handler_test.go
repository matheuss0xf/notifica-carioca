package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	ws "github.com/matheuss0xf/notifica-carioca/internal/adapters/out/websocket"
)

type stubTokenAuthenticator struct {
	authenticateFn func(tokenString string) (string, error)
}

func (s *stubTokenAuthenticator) AuthenticateToken(tokenString string) (string, error) {
	if s.authenticateFn != nil {
		return s.authenticateFn(tokenString)
	}
	return "hashed", nil
}

func TestWebSocketOriginAllowed(t *testing.T) {
	handler := NewWebSocketHandler(ws.NewHub(), &stubTokenAuthenticator{}, []string{
		"https://app.example.com",
		"https://citizen.example.com",
	})

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "non browser request without origin", origin: "", want: true},
		{name: "allowed origin", origin: "https://app.example.com", want: true},
		{name: "disallowed origin", origin: "https://evil.example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			if got := handler.originAllowed(req); got != tt.want {
				t.Fatalf("originAllowed(%q) = %v; want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestWebSocketHandleRejectsMissingOrInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		header     string
		auth       *stubTokenAuthenticator
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing token",
			auth:       &stubTokenAuthenticator{},
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"code":"missing_token"`,
		},
		{
			name:   "invalid token",
			header: "Bearer bad-token",
			auth: &stubTokenAuthenticator{
				authenticateFn: func(tokenString string) (string, error) {
					return "", errors.New("bad token")
				},
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"code":"invalid_token"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewWebSocketHandler(ws.NewHub(), tt.auth, nil)
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req

			handler.Handle(c)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if body := rec.Body.String(); body == "" || !strings.Contains(body, tt.wantBody) {
				t.Fatalf("expected body to contain %q, got %s", tt.wantBody, body)
			}
		})
	}
}

func TestWebSocketHandleUpgradesAndRegistersClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub()
	handler := NewWebSocketHandler(hub, &stubTokenAuthenticator{
		authenticateFn: func(tokenString string) (string, error) {
			if tokenString != "good-token" {
				t.Fatalf("expected token good-token, got %q", tokenString)
			}
			return "hashed-cpf", nil
		},
	}, []string{"http://example.com"})

	router := gin.New()
	router.GET("/ws", handler.Handle)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	header := http.Header{}
	header.Set("Authorization", "Bearer good-token")
	header.Set("Origin", "http://example.com")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial websocket: %v (resp=%v)", err, resp)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close websocket: %v", closeErr)
		}
	})

	if got := hub.ConnectedCount(); got != 1 {
		t.Fatalf("expected 1 connected client, got %d", got)
	}
}
