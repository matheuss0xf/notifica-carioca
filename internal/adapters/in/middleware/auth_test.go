package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type recordingHasher struct {
	input string
}

func (h *recordingHasher) Hash(cpf string) string {
	h.input = cpf
	return "hashed:" + cpf
}

func signedToken(t *testing.T, secret string, preferredUsername string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"preferred_username": preferredUsername,
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	return signed
}

func TestAuthenticateTokenNormalizesValidCPF(t *testing.T) {
	hasher := &recordingHasher{}
	m := NewAuthMiddleware("secret", hasher)

	token := signedToken(t, "secret", "529.982.247-25")
	hash, err := m.AuthenticateToken(token)
	if err != nil {
		t.Fatalf("AuthenticateToken returned error: %v", err)
	}

	if hasher.input != "52998224725" {
		t.Fatalf("expected normalized cpf, got %q", hasher.input)
	}
	if hash != "hashed:52998224725" {
		t.Fatalf("unexpected hash %q", hash)
	}
}

func TestAuthenticateTokenRejectsInvalidCPF(t *testing.T) {
	m := NewAuthMiddleware("secret", &recordingHasher{})

	token := signedToken(t, "secret", "12345678901")
	if _, err := m.AuthenticateToken(token); err == nil {
		t.Fatal("expected error for invalid cpf, got nil")
	}
}

func TestExtractTokenPrefersHeaderAndFallsBackToQueryString(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")

	if got := ExtractToken(req); got != "header-token" {
		t.Fatalf("expected header token, got %q", got)
	}

	req = httptest.NewRequest("GET", "/ws?token=query-token", nil)
	if got := ExtractToken(req); got != "query-token" {
		t.Fatalf("expected query token without Authorization header, got %q", got)
	}
}

func TestAuthenticateRequestRequiresBearerHeader(t *testing.T) {
	m := NewAuthMiddleware("secret", &recordingHasher{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)

	if _, err := m.AuthenticateRequest(req); err == nil {
		t.Fatal("expected error for missing Authorization header")
	}
}

func TestAuthMiddlewareHandleSetsCPFHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher := &recordingHasher{}
	m := NewAuthMiddleware("secret", hasher)
	token := signedToken(t, "secret", "529.982.247-25")

	router := gin.New()
	router.Use(m.Handle())
	router.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("cpf_hash"))
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "hashed:52998224725" {
		t.Fatalf("unexpected cpf hash body: %s", rec.Body.String())
	}
}

func TestAuthMiddlewareHandleRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewAuthMiddleware("secret", &recordingHasher{})

	router := gin.New()
	router.Use(m.Handle())
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
