package internal

import (
	"context"
	"log/slog"
	"net/http"

	analytic "github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/delete_event"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/get_stats"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/ingest_event"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/list_events"
	analyticv1 "github.com/meindokuse/cloud-drive/analytic-service/internal/app/analytic/v1"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/storage"
	chconn "github.com/meindokuse/cloud-drive/analytic-service/internal/pkg/connector/clickhouse"
)

func (a *App) init(ctx context.Context) error {
	if err := a.initClickHouse(ctx); err != nil {
		return err
	}
	if err := a.initStorages(ctx); err != nil {
		return err
	}
	a.initServices()
	a.initMessageBus()
	a.initHTTPServer()
	return nil
}

func (a *App) initClickHouse(ctx context.Context) error {
	conn, err := chconn.NewConn(ctx, a.cfg.ClickHouse)
	if err != nil {
		return err
	}
	a.chConn = conn
	slog.InfoContext(ctx, "clickhouse connected")
	return nil
}

func (a *App) initStorages(ctx context.Context) error {
	reg, err := storage.NewRegistry(ctx, a.chConn, a.cfg.Batcher)
	if err != nil {
		return err
	}
	a.storages = reg
	return nil
}

func (a *App) initServices() {
	store := a.storages.Event
	a.services = &analytic.Registry{
		IngestEvent: ingest_event.NewService(store),
		GetStats:    get_stats.NewService(store),
		ListEvents:  list_events.NewService(store),
		DeleteEvent: delete_event.NewService(store),
	}
}

func (a *App) initMessageBus() {
	a.messageBus = messagebus.NewRegistry(a.cfg.Kafka, a.services.IngestEvent)
}

func (a *App) initHTTPServer() {
	h := analyticv1.New(a.services)
	a.httpSrv = &http.Server{
		Addr:    a.cfg.Server.HTTPAddr,
		Handler: h.Router(),
	}
}
