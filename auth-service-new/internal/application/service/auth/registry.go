package auth

import (
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/get_sessions"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/introspect"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/login"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/logout"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/refresh"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/register"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/revoke_session"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/gateway/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
)

// Registry contains all auth use cases
type Registry struct {
	Login         *login.Service
	Register      *register.Service
	Refresh       *refresh.Service
	Logout        *logout.Service
	Introspect    *introspect.Service
	GetSessions   *get_sessions.Service
	RevokeSession *revoke_session.Service
}

// NewRegistry creates a new auth service registry
func NewRegistry(
	storage *storage.Registry,
	oauthGateway *oauth.Registry,
	jwtManager *jwt.Manager,
	hasher *pass.Hasher,
	refreshTTL int64,
) *Registry {
	return &Registry{
		Login: login.NewService(
			storage.Account,
			storage.Session,
			hasher,
			jwtManager,
		),
		Register: register.NewService(
			storage.Account,
			storage.Outbox,
			hasher,
		),
		Refresh: refresh.NewService(
			storage.Session,
			jwtManager,
		),
		Logout: logout.NewService(
			storage.Session,
			jwtManager,
		),
		Introspect: introspect.NewService(
			storage.Session,
			jwtManager,
		),
		GetSessions: get_sessions.NewService(
			storage.Session,
		),
		RevokeSession: revoke_session.NewService(
			storage.Session,
		),
	}
}
