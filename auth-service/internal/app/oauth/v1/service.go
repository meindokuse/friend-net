package v1

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/gateway/oauth"
	oauthservice "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth"
)

const (
	oauthStateCookieGoogleLogin = "oauth_state_google_login"
	oauthStateCookieGoogleLink  = "oauth_state_google_link"
)

// Implementation handles HTTP requests for OAuth
type Implementation struct {
	services       *oauthservice.Registry
	googleProvider *oauth.GoogleClient
	config         config.ControllerConfig
}

// NewOAuthService creates a new OAuth HTTP service
func NewOAuthService(
	services *oauthservice.Registry,
	googleProvider *oauth.GoogleClient,
	cfg config.ControllerConfig,
) *Implementation {
	if cfg.RefreshCookieName == "" {
		cfg.RefreshCookieName = "refresh_token"
	}

	return &Implementation{
		services:       services,
		googleProvider: googleProvider,
		config:         cfg,
	}
}

func (i *Implementation) setRefreshCookie(ctx *gin.Context, refreshToken string, maxAge int) {
	ctx.SetCookie(
		i.config.RefreshCookieName,
		refreshToken,
		maxAge,
		"/",
		i.config.CookieDomain,
		i.config.CookieSecure,
		true,
	)
}

func (i *Implementation) generateRandomState() string {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(stateBytes)
}

func (i *Implementation) setOAuthStateCookie(ctx *gin.Context, name, state string) {
	const maxAgeSeconds = 10 * 60
	ctx.SetCookie(
		name,
		state,
		maxAgeSeconds,
		"/",
		i.config.CookieDomain,
		i.config.CookieSecure,
		true,
	)
}

func (i *Implementation) clearOAuthStateCookie(ctx *gin.Context, name string) {
	ctx.SetCookie(
		name,
		"",
		-1,
		"/",
		i.config.CookieDomain,
		i.config.CookieSecure,
		true,
	)
}

func (i *Implementation) validateOAuthState(ctx *gin.Context, cookieName, callbackState string) bool {
	storedState, err := ctx.Cookie(cookieName)
	if err != nil {
		return false
	}
	return storedState != "" && strings.TrimSpace(callbackState) == storedState
}
