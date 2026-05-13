package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/meindokuse/cloud-drive/analytic-service/config"
	analytic "github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/pkg/closer"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/pkg/logger"
)

type App struct {
	cfg        *config.Config
	chConn     driver.Conn
	storages   *storage.Registry
	services   *analytic.Registry
	httpSrv    *http.Server
	messageBus *messagebus.Registry
}

func New(_ context.Context) *App {
	return &App{cfg: config.Instance()}
}

func (a *App) Run(ctx context.Context) {
	logger.Init(a.cfg.Logger.Level)

	if err := a.init(ctx); err != nil {
		slog.ErrorContext(ctx, "init failed", "error", err)
		os.Exit(1)
	}

	go a.storages.Event.Start(ctx)

	if a.messageBus.Consumer != nil {
		go a.messageBus.Consumer.Start(ctx)
	}

	go func() {
		slog.InfoContext(ctx, "analytic-service starting", "http_addr", a.cfg.Server.HTTPAddr)
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
	c.Add(a.storages.Event.Stop)
	c.Add(func(ctx context.Context) error { return a.chConn.Close() })
	c.CloseAll()
}
