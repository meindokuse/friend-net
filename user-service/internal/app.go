package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/meindokuse/cloud-drive/user-service-new/config"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/processor"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/closer"
	userv1 "github.com/meindokuse/cloud-drive/user-service/internal/app/user/v1"
	userservice "github.com/meindokuse/cloud-drive/user-service/internal/application/service/user"
	userstorage "github.com/meindokuse/cloud-drive/user-service/internal/infrastructure/storage/user"
	mongoconn "github.com/meindokuse/cloud-drive/user-service/internal/pkg/connector/mongo"
	"go.mongodb.org/mongo-driver/mongo"
)

type App struct {
	cfg      *config.Config
	db       *mongo.Database
	services *userservice.Service
	httpSrv  *http.Server
	consumer *messagebus.Consumer
}

func New(ctx context.Context) *App {
	_ = ctx
	return &App{cfg: config.Instance()}
}

func (a *App) Run(ctx context.Context) {
	if err := a.init(ctx); err != nil {
		slog.ErrorContext(ctx, "init failed", "error", err)
		os.Exit(1)
	}
	if a.consumer != nil {
		go a.consumer.Start(ctx)
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
	if a.consumer != nil {
		c.Add(a.consumer.Stop)
	}
	c.Add(func(ctx context.Context) error { return a.db.Client().Disconnect(ctx) })
	c.CloseAll()
}

func (a *App) init(ctx context.Context) error {
	db, err := mongoconn.NewDatabase(ctx, a.cfg.Mongo)
	if err != nil {
		return err
	}
	a.db = db
	storage, err := userstorage.NewStorage(db)
	if err != nil {
		return err
	}
	a.services = userservice.NewService(storage)
	handler := userv1.New(a.services)
	router := handler.Router()
	a.httpSrv = &http.Server{Addr: a.cfg.Server.HTTPAddr, Handler: router}
	if a.cfg.Kafka.Enabled {
		accountProcessor := processor.NewAccountCreatedProcessor(a.services)
		a.consumer = messagebus.NewConsumer(a.cfg.Kafka.Brokers, a.cfg.Kafka.Topic, a.cfg.Kafka.GroupID, accountProcessor, slog.Default())
	}
	return nil
}
