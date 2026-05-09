package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/meindokuse/cloud-drive/user-service-new/config"
	userservice "github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/closer"
)

type App struct {
	cfg        *config.Config
	db         *mongo.Database
	storages   *storage.Registry
	services   *userservice.Registry
	httpSrv    *http.Server
	messageBus *messagebus.Registry
}

func New(_ context.Context) *App {
	return &App{cfg: config.Instance()}
}

func (a *App) Run(ctx context.Context) {
	if err := a.init(ctx); err != nil {
		slog.ErrorContext(ctx, "init failed", "error", err)
		os.Exit(1)
	}

	if a.messageBus.Consumer != nil {
		go a.messageBus.Consumer.Start(ctx)
	}

	go func() {
		slog.InfoContext(ctx, "user-service starting", "http_addr", a.cfg.Server.HTTPAddr)
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "http server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(ctx, a.cfg.Graceful.Timeout)
	defer cancel()
	_ = a.httpSrv.Shutdown(shutdownCtx)

	c := closer.New(a.cfg.Graceful.Timeout)
	if a.messageBus.Consumer != nil {
		c.Add(a.messageBus.Consumer.Stop)
	}
	c.Add(func(ctx context.Context) error { return a.db.Client().Disconnect(ctx) })
	c.CloseAll()
}
