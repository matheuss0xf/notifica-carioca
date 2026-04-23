package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/httpx"
	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

// AuthMiddleware validates JWT tokens and sets cpf_hash in the context.
type AuthMiddleware struct {
	jwtSecret string
	hasher    cpfHasher
}

type cpfHasher interface {
	Hash(cpf string) string
}

// NewAuthMiddleware creates a new JWT authentication middleware.
func NewAuthMiddleware(jwtSecret string, hasher cpfHasher) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
		hasher:    hasher,
	}
}

// Handle returns a Gin middleware that validates JWT and extracts CPF.
func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		cpfHash, err := m.AuthenticateRequest(c.Request)
		if err != nil {
			httpx.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
			return
		}

		c.Set("cpf_hash", cpfHash)
		c.Next()
	}
}

// AuthenticateRequest validates the request token and returns the hashed CPF identity.
func (m *AuthMiddleware) AuthenticateRequest(r *http.Request) (string, error) {
	tokenString := extractBearerToken(r)
	if tokenString == "" {
		return "", errors.New("missing authorization token")
	}

	return m.AuthenticateToken(tokenString)
}

// AuthenticateToken validates a raw token string and returns the hashed CPF identity.
func (m *AuthMiddleware) AuthenticateToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("missing authorization token")
	}

	cpf, err := m.ParseJWT(tokenString)
	if err != nil {
		return "", err
	}

	return m.hasher.Hash(cpf), nil
}

// ParseJWT validates the token and extracts the CPF from preferred_username.
// Exported so the WebSocket handler can reuse it directly.
func (m *AuthMiddleware) ParseJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.jwtSecret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	rawCPF, ok := claims["preferred_username"].(string)
	if !ok || rawCPF == "" {
		return "", errors.New("preferred_username not found in token")
	}

	cpf, err := domain.ValidateCPF(rawCPF)
	if err != nil {
		return "", err
	}

	return cpf, nil
}

// extractBearerToken extracts the token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

// ExtractToken extracts JWT from the Authorization header, falling back to the
// token query parameter for browser WebSocket clients.
func ExtractToken(r *http.Request) string {
	if token := extractBearerToken(r); token != "" {
		return token
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}
