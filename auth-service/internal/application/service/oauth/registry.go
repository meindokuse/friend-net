package oauth

import (
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/providers"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/get_linked"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/link"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/login"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/unlink"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
)

// Registry contains all OAuth use cases
type Registry struct {
	Login     *login.Service
	Link      *link.Service
	Unlink    *unlink.Service
	GetLinked *get_linked.Service
}

// NewRegistry creates a new OAuth service registry
func NewRegistry(
	storage *storage.Registry,
	providers map[entity.OAuthProvider]providers.OAuthProviderGateway,
	jwtManager *jwt.Manager,
	refreshTTL int64,
) *Registry {
	return &Registry{
		Login: login.NewService(
			storage.Account,
			storage.OAuth,
			storage.Session,
			storage.Outbox,
			providers,
			jwtManager,
			refreshTTL,
		),
		Link: link.NewService(
			storage.OAuth,
			providers,
		),
		Unlink: unlink.NewService(
			storage.OAuth,
		),
		GetLinked: get_linked.NewService(
			storage.OAuth,
		),
	}
}
