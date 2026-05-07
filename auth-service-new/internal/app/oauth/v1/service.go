package v1

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	oauthlink "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/link"
	oauthlogin "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/login"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/gateway/oauth"
)

const (
	oauthStateCookieGoogleLogin = "oauth_state_google_login"
	oauthStateCookieGoogleLink  = "oauth_state_google_link"
)

// Implementation handles HTTP requests for OAuth
type Implementation struct {
	services       *oauth.Registry
	googleProvider *oauth.GoogleClient
	config         config.ControllerConfig
}

// NewOAuthService creates a new OAuth HTTP service
func NewOAuthService(
	services *oauth.Registry,
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

// GoogleAuth handles GET /auth/google - redirects to Google
func (i *Implementation) GoogleAuth(ctx *gin.Context) {
	state := generateRandomState()
	if state == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate oauth state"})
		return
	}
	i.setOAuthStateCookie(ctx, oauthStateCookieGoogleLogin, state)
	authURL := i.googleProvider.AuthURL(state)
	ctx.Redirect(http.StatusFound, authURL)
}

// GoogleCallback handles GET /auth/google/callback
func (i *Implementation) GoogleCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")
	if code == "" || state == "" || !i.validateOAuthState(ctx, oauthStateCookieGoogleLogin, state) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state or code"})
		return
	}

	dto := oauthlogin.LoginDTO{
		Provider:    entity.OAuthGoogle,
		Code:        code,
		State:       state,
		Fingerprint: ctx.Request.UserAgent(),
		IP:          ctx.ClientIP(),
		UserAgent:   ctx.Request.UserAgent(),
	}

	result, err := i.services.Login.Login(ctx.Request.Context(), dto)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	refreshMaxAge := int(time.Until(time.Now().Add(30 * 24 * time.Hour)).Seconds())
	i.setRefreshCookie(ctx, result.RefreshToken, refreshMaxAge)
	i.clearOAuthStateCookie(ctx, oauthStateCookieGoogleLogin)

	ctx.JSON(http.StatusOK, gin.H{
		"access_token":       result.AccessToken,
		"refresh_token":      result.RefreshToken,
		"token_type":         result.TokenType,
		"expires_in":         result.ExpiresIn,
		"expires_at":         result.ExpiresAt,
		"refresh_expires_at": result.RefreshExpiresAt,
		"account_id":         result.AccountID,
		"is_new_user":        result.IsNewUser,
	})
}

// LinkGoogle handles GET /auth/link/google
func (i *Implementation) LinkGoogle(ctx *gin.Context) {
	state := generateRandomState()
	if state == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate oauth state"})
		return
	}
	i.setOAuthStateCookie(ctx, oauthStateCookieGoogleLink, state)
	authURL := i.googleProvider.AuthURL(state)
	ctx.Redirect(http.StatusFound, authURL)
}

// LinkGoogleCallback handles GET /auth/link/google/callback
func (i *Implementation) LinkGoogleCallback(ctx *gin.Context) {
	accountID, ok := ctx.Get("account_id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	code := ctx.Query("code")
	state := ctx.Query("state")
	if code == "" || state == "" || !i.validateOAuthState(ctx, oauthStateCookieGoogleLink, state) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state or code"})
		return
	}

	dto := oauthlink.LinkDTO{
		Provider:  entity.OAuthGoogle,
		Code:      code,
		State:     state,
		AccountID: accountID.(string),
	}

	if err := i.services.Link.Link(ctx.Request.Context(), dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	i.clearOAuthStateCookie(ctx, oauthStateCookieGoogleLink)
	ctx.JSON(http.StatusOK, gin.H{"message": "account linked successfully"})
}

// GetLinkedAccounts handles GET /auth/linked
func (i *Implementation) GetLinkedAccounts(ctx *gin.Context) {
	accountID, ok := ctx.Get("account_id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accounts, err := i.services.GetLinked.GetLinked(ctx.Request.Context(), accountID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"linked_accounts": accounts})
}

// Unlink handles DELETE /auth/linked/:provider
func (i *Implementation) Unlink(ctx *gin.Context) {
	accountID, ok := ctx.Get("account_id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	provider := entity.OAuthProvider(ctx.Param("provider"))
	if err := i.services.Unlink.Unlink(ctx.Request.Context(), accountID.(string), provider); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "provider unlinked"})
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

func generateRandomState() string {
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
