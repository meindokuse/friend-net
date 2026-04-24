package http

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	"github.com/meindokuse/cloud-drive/auth-service/internal/usecase/oauth"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

type OAuthController struct {
	useCase        oauth.OAuthUseCase
	googleProvider oauth.OAuthProviderService
	config         ControllerConfig
}

func NewOAuthController(useCase oauth.OAuthUseCase, googleProvider oauth.OAuthProviderService, config ControllerConfig) *OAuthController {
	if config.RefreshCookieName == "" {
		config.RefreshCookieName = "refresh_token"
	}

	return &OAuthController{
		useCase:        useCase,
		googleProvider: googleProvider,
		config:         config,
	}
}

// /auth/google — перенаправление на Google
func (c *OAuthController) GoogleAuth(ctx *gin.Context) {
	state := generateRandomState() // CSRF token
	// Сохранить state в Redis (expires в 10 мин)
	// uc.validateState(ctx, state) — позже полезно

	authURL := c.googleProvider.AuthURL(state, "")
	slog.InfoContext(sharedlogger.WithField(ctx.Request.Context(), "provider", string(domain.OAuthGoogle)), "oauth google redirect prepared")

	ctx.Redirect(302, authURL)
}

// /auth/google/callback — callback от Google
func (c *OAuthController) GoogleCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")

	input := oauth.OAuthCallbackInput{
		Provider:    domain.OAuthGoogle,
		Code:        code,
		State:       state,
		RedirectURL: "/dashboard", // Куда вернуть после успеха
		RequestData: domain.RequestData{
			IPAddress:   ctx.ClientIP(),
			UserAgent:   ctx.Request.UserAgent(),
			Fingerprint: ctx.GetHeader("X-Device-Fingerprint"),
		},
	}

	reqCtx := sharedlogger.WithField(ctx.Request.Context(), "provider", string(domain.OAuthGoogle))

	output, err := c.useCase.Login(reqCtx, input)
	if err != nil {
		slog.WarnContext(reqCtx, "oauth google callback failed", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.SetCookie(
		c.config.RefreshCookieName,
		output.RefreshToken,
		int(time.Until(output.RefreshExpiresAt).Seconds()),
		"/",
		c.config.CookieDomain,
		c.config.CookieSecure,
		true,
	)

	// Вернуть JWT
	ctx.JSON(200, gin.H{
		"access_token":       output.AccessToken,
		"refresh_token":      output.RefreshToken,
		"token_type":         output.TokenType,
		"account_id":         output.AccountID,
		"is_new_user":        output.IsNewUser,
		"expires_at":         output.ExpiresAt,
		"refresh_expires_at": output.RefreshExpiresAt,
	})
	slog.InfoContext(reqCtx, "oauth google callback completed", slog.String("account_id", output.AccountID))
}

// /auth/link/google — привязка аккаунта (для уже залогиненного)
func (c *OAuthController) LinkGoogle(ctx *gin.Context) {
	// Получаем accountID из JWT (из middleware)
	_ = ctx.GetString("account_id")

	state := generateRandomState()
	// Сохранить state...

	authURL := c.googleProvider.AuthURL(state, "")
	slog.InfoContext(sharedlogger.WithField(ctx.Request.Context(), "provider", string(domain.OAuthGoogle)), "oauth google link redirect prepared")

	ctx.Redirect(302, authURL)
}

// /auth/link/google/callback — callback для Link
func (c *OAuthController) LinkGoogleCallback(ctx *gin.Context) {
	userID := ctx.GetString("account_id")
	code := ctx.Query("code")
	state := ctx.Query("state")

	input := oauth.OAuthLinkInput{
		Provider:         domain.OAuthGoogle,
		Code:             code,
		State:            state,
		CurrentAccountID: userID,
	}

	reqCtx := sharedlogger.WithFields(ctx.Request.Context(), map[string]interface{}{
		"provider":   string(domain.OAuthGoogle),
		"account_id": userID,
	})

	err := c.useCase.LinkAccount(reqCtx, input)
	if err != nil {
		slog.WarnContext(reqCtx, "oauth google link failed", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(reqCtx, "oauth google link completed")
	ctx.JSON(http.StatusOK, gin.H{"message": "account linked successfully"})
}

func generateRandomState() string {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(stateBytes)
}
