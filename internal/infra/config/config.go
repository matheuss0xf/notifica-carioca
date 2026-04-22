package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration parsed from environment variables.
type Config struct {
	ServerPort        string        `env:"SERVER_PORT" envDefault:"8080"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`

	DatabaseURL string `env:"DATABASE_URL,required"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`

	WebhookSecret string `env:"WEBHOOK_SECRET,required"`
	CPFHashKey    string `env:"CPF_HASH_KEY,required"`
	JWTSecret     string `env:"JWT_SECRET,required"`

	WSAllowedOrigins []string `env:"WS_ALLOWED_ORIGINS" envSeparator:","`

	RateLimitWindow        time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	WebhookRateLimit       int           `env:"WEBHOOK_RATE_LIMIT" envDefault:"60"`
	NotificationsRateLimit int           `env:"NOTIFICATIONS_RATE_LIMIT" envDefault:"120"`
	WebSocketRateLimit     int           `env:"WEBSOCKET_RATE_LIMIT" envDefault:"30"`
	EnableHSTS             bool          `env:"ENABLE_HSTS" envDefault:"false"`
	HSTSMaxAgeSeconds      int           `env:"HSTS_MAX_AGE_SECONDS" envDefault:"31536000"`

	IdempotencyTTL time.Duration `env:"IDEMPOTENCY_TTL" envDefault:"24h"`
	UnreadCacheTTL time.Duration `env:"UNREAD_CACHE_TTL" envDefault:"1h"`
}

// Load parses environment variables into a Config struct.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	cfg.WSAllowedOrigins = normalizeOrigins(cfg.WSAllowedOrigins)
	return cfg, nil
}

func normalizeOrigins(origins []string) []string {
	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			normalized = append(normalized, origin)
		}
	}
	return normalized
}
