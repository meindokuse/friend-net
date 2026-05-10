package internal

import (
	"context"
	"log/slog"
	"net/http"

	userservice "github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/change_email"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/change_phone"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/create_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/delete_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/get_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/list_users"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/search_users"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_last_seen"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_profile"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_settings"
	userv1 "github.com/meindokuse/cloud-drive/user-service-new/internal/app/user/v1"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/storage"
	mongoconn "github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/connector/mongo"
)

func (a *App) init(ctx context.Context) error {
	if err := a.initMongo(ctx); err != nil {
		return err
	}
	if err := a.initStorages(); err != nil {
		return err
	}
	a.initServices()
	a.initMessageBus()
	a.initHTTPServer()
	return nil
}

func (a *App) initMongo(ctx context.Context) error {
	db, err := mongoconn.NewDatabase(ctx, a.cfg.Mongo)
	if err != nil {
		return err
	}
	a.db = db
	slog.InfoContext(ctx, "mongodb connected")
	return nil
}

func (a *App) initStorages() error {
	reg, err := storage.NewRegistry(a.db)
	if err != nil {
		return err
	}
	a.storages = reg
	return nil
}

func (a *App) initServices() {
	repo := a.storages.User
	a.services = &userservice.Registry{
		CreateUser:     create_user.NewService(repo),
		GetUser:        get_user.NewService(repo),
		UpdateProfile:  update_profile.NewService(repo),
		UpdateSettings: update_settings.NewService(repo),
		ChangeEmail:    change_email.NewService(repo),
		ChangePhone:    change_phone.NewService(repo),
		DeleteUser:     delete_user.NewService(repo),
		UpdateLastSeen: update_last_seen.NewService(repo),
		SearchUsers:    search_users.NewService(repo),
		ListUsers:      list_users.NewService(repo),
	}
}

func (a *App) initMessageBus() {
	a.messageBus = messagebus.NewRegistry(a.cfg.Kafka, a.services.CreateUser, a.storages.Idempotency, slog.Default())
}

func (a *App) initHTTPServer() {
	h := userv1.New(a.services)
	a.httpSrv = &http.Server{
		Addr:    a.cfg.Server.HTTPAddr,
		Handler: h.Router(),
	}
}
