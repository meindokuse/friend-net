package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	authservice "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth"
	oauthservice "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/gateway/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/closer"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
)

// outboxFlusher is the minimal interface the App needs for shutdown coordination.
type outboxFlusher interface {
	Done() <-chan struct{}
}

// App is the main application structure
type App struct {
	cfg *config.Config

	pool   *pgxpool.Pool
	rdb    *redis.Client
	jwt    *jwt.Manager
	hasher *pass.Hasher

	storages      *storage.Registry
	authServices  *authservice.Registry
	oauthServices *oauthservice.Registry

	oauthGateway *oauth.Registry
	messageBus   *messagebus.Registry

	httpServer  *http.Server
	flusher     outboxFlusher
	flushCancel context.CancelFunc
}

// New creates a new App instance
func New(_ context.Context) *App {
	return &App{
		cfg: config.Instance(),
	}
}

// Run starts the application
func (a *App) Run(ctx context.Context) {
	if err := a.init(ctx); err != nil {
		slog.ErrorContext(ctx, "initialization failed", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.InfoContext(ctx, "auth-service starting", "http_addr", a.cfg.Server.HTTPAddr)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "http server error", "error", err)
			os.Exit(1)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	slog.InfoContext(ctx, "shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(ctx, a.cfg.Graceful.Timeout)
	defer cancel()

	// Stop accepting HTTP traffic first.
	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "http shutdown error", "error", err)
	}

	// Cancel the flusher and wait for its drain flush to complete.
	// This must happen before the pool and producer are closed.
	if a.flushCancel != nil {
		a.flushCancel()
		select {
		case <-a.flusher.Done():
		case <-shutdownCtx.Done():
			slog.WarnContext(ctx, "flusher drain timed out")
		}
	}

	c := closer.New(a.cfg.Graceful.Timeout)
	c.Add(func(_ context.Context) error {
		a.pool.Close()
		return nil
	})
	c.Add(func(ctx context.Context) error {
		return a.rdb.Close()
	})
	if a.messageBus.Producer != nil {
		c.Add(a.messageBus.Producer.Close)
	}
	c.CloseAll()

	slog.InfoContext(ctx, "auth-service stopped")
}
