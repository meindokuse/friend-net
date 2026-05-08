package v1

import (
	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	authservice "github.com/meindokuse/cloud-drive/auth-service/internal/application/service/auth"
)

// Implementation handles HTTP requests for auth
type Implementation struct {
	services *authservice.Registry
	config   config.ControllerConfig
}

// NewAuthService creates a new auth HTTP service
func NewAuthService(services *authservice.Registry, cfg config.ControllerConfig) *Implementation {
	if cfg.RefreshCookieName == "" {
		cfg.RefreshCookieName = "refresh_token"
	}

	return &Implementation{
		services: services,
		config:   cfg,
	}
}
