package config

import (
	"reflect"
	"testing"
)

func TestNormalizeOrigins(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "trims whitespace and drops empty values",
			input: []string{" https://app.example.com ", "", "   ", "https://citizen.example.com"},
			want:  []string{"https://app.example.com", "https://citizen.example.com"},
		},
		{
			name:  "empty input",
			input: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeOrigins(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeOrigins(%v) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadParsesAndNormalizesEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://notifica:notifica@localhost:5432/notifica_carioca?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://default:secret@localhost:6379/0")
	t.Setenv("WEBHOOK_SECRET", "webhook-secret")
	t.Setenv("CPF_HASH_KEY", "cpf-hash-key")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("WS_ALLOWED_ORIGINS", " https://app.example.com , ,https://citizen.example.com ")
	t.Setenv("RATE_LIMIT_WINDOW", "2m")
	t.Setenv("WEBHOOK_RATE_LIMIT", "10")
	t.Setenv("NOTIFICATIONS_RATE_LIMIT", "20")
	t.Setenv("WEBSOCKET_RATE_LIMIT", "5")
	t.Setenv("ENABLE_HSTS", "true")
	t.Setenv("HSTS_MAX_AGE_SECONDS", "123")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.DatabaseURL == "" || cfg.RedisURL == "" {
		t.Fatalf("expected required URLs to be parsed: %#v", cfg)
	}
	wantOrigins := []string{"https://app.example.com", "https://citizen.example.com"}
	if !reflect.DeepEqual(cfg.WSAllowedOrigins, wantOrigins) {
		t.Fatalf("expected normalized origins %v, got %v", wantOrigins, cfg.WSAllowedOrigins)
	}
	if cfg.WebhookRateLimit != 10 || cfg.NotificationsRateLimit != 20 || cfg.WebSocketRateLimit != 5 {
		t.Fatalf("expected rate limits to be parsed, got %#v", cfg)
	}
	if !cfg.EnableHSTS || cfg.HSTSMaxAgeSeconds != 123 {
		t.Fatalf("expected HSTS config to be parsed, got %#v", cfg)
	}
}
