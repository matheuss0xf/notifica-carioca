package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	handler "github.com/matheuss0xf/notifica-carioca/internal/adapters/in/http"
	"github.com/matheuss0xf/notifica-carioca/internal/adapters/in/middleware"
	"github.com/matheuss0xf/notifica-carioca/internal/adapters/out/postgres"
	redisadapter "github.com/matheuss0xf/notifica-carioca/internal/adapters/out/redis"
	runtimeadapter "github.com/matheuss0xf/notifica-carioca/internal/adapters/out/runtime"
	ws "github.com/matheuss0xf/notifica-carioca/internal/adapters/out/websocket"
	"github.com/matheuss0xf/notifica-carioca/internal/application"
	"github.com/matheuss0xf/notifica-carioca/internal/infra/config"
	"github.com/matheuss0xf/notifica-carioca/internal/infra/crypto"
	"github.com/matheuss0xf/notifica-carioca/internal/infra/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	migrator := postgres.NewMigrator("file://migrations")
	if err := migrator.Up(cfg.DatabaseURL); err != nil {
		slog.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied successfully")

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to postgres")

	redisOpts, err := goredis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to parse redis url", "error", err)
		os.Exit(1)
	}
	redisClient := goredis.NewClient(redisOpts)
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			slog.Warn("failed to close redis client", "error", closeErr)
		}
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("failed to ping redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to redis")

	repo := postgres.NewNotificationRepository(pool)
	cache := redisadapter.NewUnreadCache(redisClient, cfg.UnreadCacheTTL)
	idemp := redisadapter.NewIdempotencyStore(redisClient, cfg.IdempotencyTTL)
	publisher := redisadapter.NewPublisher(redisClient)
	subscriber := redisadapter.NewSubscriber(redisClient)
	hub := ws.NewHub()
	cpfHasher := crypto.NewCPFHasher(cfg.CPFHashKey)
	readiness := runtimeadapter.NewReadinessChecker(pool, redisClient, hub.ConnectedCount)

	webhookProcessor := application.NewWebhookProcessor(repo, cache, idemp, publisher, cpfHasher)
	notificationDispatcher := application.NewNotificationDispatcher(subscriber, hub)
	notificationReader := application.NewNotificationReader(repo, cache)
	notificationMarker := application.NewNotificationMarker(repo, cache)

	healthHandler := handler.NewHealthHandler(readiness)
	webhookHandler := handler.NewWebhookHandler(webhookProcessor)
	notifHandler := handler.NewNotificationHandler(notificationReader, notificationReader, notificationMarker)
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, cpfHasher)
	requestLogger := middleware.NewRequestLogger()
	signatureMiddleware := middleware.NewSignatureMiddleware(cfg.WebhookSecret)
	wsHandler := handler.NewWebSocketHandler(hub, authMiddleware, cfg.WSAllowedOrigins)

	go func() {
		if subErr := notificationDispatcher.Run(ctx); subErr != nil && ctx.Err() == nil {
			slog.Error("redis subscriber error", "error", subErr)
		}
	}()

	router := server.NewRouter(
		requestLogger.Handle(),
		healthHandler.Liveness,
		healthHandler.Readiness,
		signatureMiddleware.Handle(),
		authMiddleware.Handle(),
		webhookHandler.HandleStatusChange,
		notifHandler.List,
		notifHandler.UnreadCount,
		notifHandler.MarkAsRead,
		wsHandler.Handle,
	)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("shutdown signal received", "signal", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()

		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Error("server shutdown error", "error", shutdownErr)
		}
		cancel()
	}()

	slog.Info("server starting", "port", cfg.ServerPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
