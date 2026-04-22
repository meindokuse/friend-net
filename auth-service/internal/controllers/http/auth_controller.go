package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/user"
	"github.com/meindokuse/cloud-drive/auth-service/internal/dto"
	usecase "github.com/meindokuse/cloud-drive/auth-service/internal/usecase/auth"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

type AuthUseCase interface {
	LoginUser(ctx context.Context, input domain.LoginInput) (*domain.AuthResult, error)
	Register(ctx context.Context, registerData domain.Register) (string, error)
	Refresh(ctx context.Context, input domain.RefreshInput) (*domain.AuthResult, error)
	Logout(ctx context.Context, input domain.LogoutInput) error
	LogoutAll(ctx context.Context, userID string) error
	GetUserSessions(ctx context.Context, userID, currentSessionID string) ([]domain.SessionInfo, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	ValidateAccessToken(ctx context.Context, accessToken string) (*domain.AccessTokenInfo, error)
	RefreshTTL() time.Duration
}

type ControllerConfig struct {
	CookieDomain      string
	CookieSecure      bool
	RefreshCookieName string
}

type AuthController struct {
	useCase AuthUseCase
	config  ControllerConfig
}

func NewAuthController(useCase AuthUseCase, config ControllerConfig) *AuthController {
	if config.RefreshCookieName == "" {
		config.RefreshCookieName = "refresh_token"
	}

	return &AuthController{
		useCase: useCase,
		config:  config,
	}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var request dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		slog.WarnContext(ctx.Request.Context(), "register request validation failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	reqCtx := sharedlogger.WithField(ctx.Request.Context(), "email", request.Email)

	if _, err := c.useCase.Register(reqCtx, domain.Register{
		Email:    request.Email,
		Password: request.Password,
	}); err != nil {
		slog.WarnContext(reqCtx, "register request failed", slog.String("error", err.Error()))
		c.respondWithError(ctx, err)
		return
	}

	tokens, err := c.useCase.LoginUser(reqCtx, c.loginInputFromRequest(ctx, request.Email, request.Password))
	if err != nil {
		slog.WarnContext(reqCtx, "register autologin failed", slog.String("error", err.Error()))
		c.respondWithError(ctx, err)
		return
	}

	slog.InfoContext(reqCtx, "register request completed")
	c.respondWithTokenPair(ctx, http.StatusCreated, tokens)
}

func (c *AuthController) Login(ctx *gin.Context) {
	var request dto.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		slog.WarnContext(ctx.Request.Context(), "login request validation failed")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	reqCtx := sharedlogger.WithField(ctx.Request.Context(), "email", request.Email)

	tokens, err := c.useCase.LoginUser(reqCtx, c.loginInputFromRequest(ctx, request.Email, request.Password))
	if err != nil {
		slog.WarnContext(reqCtx, "login request failed", slog.String("error", err.Error()))
		c.respondWithError(ctx, err)
		return
	}

	slog.InfoContext(reqCtx, "login request completed")
	c.respondWithTokenPair(ctx, http.StatusOK, tokens)
}

func (c *AuthController) Refresh(ctx *gin.Context) {
	refreshToken := c.readRefreshToken(ctx)
	if refreshToken == "" {
		slog.WarnContext(ctx.Request.Context(), "refresh token is missing in request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "refresh token is required"})
		return
	}

	reqCtx := sharedlogger.WithField(ctx.Request.Context(), "refresh_flow", true)

	tokens, err := c.useCase.Refresh(reqCtx, domain.RefreshInput{
		RefreshToken: refreshToken,
		Fingerprint:  c.requestFingerprint(ctx),
	})
	if err != nil {
		slog.WarnContext(reqCtx, "refresh request failed", slog.String("error", err.Error()))
		c.respondWithError(ctx, err)
		return
	}

	slog.InfoContext(reqCtx, "refresh request completed")
	c.respondWithTokenPair(ctx, http.StatusOK, tokens)
}

func (c *AuthController) Logout(ctx *gin.Context) {
	accessToken := extractBearerToken(ctx.GetHeader("Authorization"))
	refreshToken := c.readRefreshToken(ctx)

	if err := c.useCase.Logout(ctx.Request.Context(), domain.LogoutInput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}); err != nil {
		c.respondWithError(ctx, err)
		return
	}

	c.clearRefreshCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (c *AuthController) LogoutAll(ctx *gin.Context) {
	authInfo, ok := c.requireAuth(ctx)
	if !ok {
		return
	}

	if err := c.useCase.LogoutAll(ctx.Request.Context(), authInfo.UserID); err != nil {
		c.respondWithError(ctx, err)
		return
	}

	c.clearRefreshCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

func (c *AuthController) Sessions(ctx *gin.Context) {
	authInfo, ok := c.requireAuth(ctx)
	if !ok {
		return
	}

	sessions, err := c.useCase.GetUserSessions(ctx.Request.Context(), authInfo.UserID, authInfo.SessionID)
	if err != nil {
		c.respondWithError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (c *AuthController) RevokeSession(ctx *gin.Context) {
	authInfo, ok := c.requireAuth(ctx)
	if !ok {
		return
	}

	sessionID := strings.TrimSpace(ctx.Param("session_id"))
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	if err := c.useCase.RevokeSession(ctx.Request.Context(), authInfo.UserID, sessionID); err != nil {
		c.respondWithError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

func (c *AuthController) Introspect(ctx *gin.Context) {
	token := extractBearerToken(ctx.GetHeader("Authorization"))

	if token == "" {
		var request dto.IntrospectRequest
		if err := ctx.ShouldBindJSON(&request); err == nil {
			token = strings.TrimSpace(request.AccessToken)
		}
	}

	if token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "access token is required"})
		return
	}

	info, err := c.useCase.ValidateAccessToken(ctx.Request.Context(), token)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"active":     true,
		"user_id":    info.UserID,
		"session_id": info.SessionID,
		"expires_at": info.ExpiresAt,
	})
}

func (c *AuthController) loginInputFromRequest(ctx *gin.Context, email, password string) domain.LoginInput {
	return domain.LoginInput{
		Email:       email,
		Password:    password,
		Fingerprint: c.requestFingerprint(ctx),
		IP:          ctx.ClientIP(),
		UserAgent:   ctx.Request.UserAgent(),
	}
}

func (c *AuthController) requestFingerprint(ctx *gin.Context) string {
	fingerprint := strings.TrimSpace(ctx.GetHeader("X-Device-Fingerprint"))
	if fingerprint != "" {
		return fingerprint
	}

	return ctx.Request.UserAgent()
}

func (c *AuthController) readRefreshToken(ctx *gin.Context) string {
	refreshToken := strings.TrimSpace(ctx.GetHeader("X-Refresh-Token"))
	if refreshToken != "" {
		return refreshToken
	}

	if cookieToken, err := ctx.Cookie(c.config.RefreshCookieName); err == nil && cookieToken != "" {
		return cookieToken
	}

	var request dto.RefreshRequest
	if err := ctx.ShouldBindJSON(&request); err == nil {
		return strings.TrimSpace(request.RefreshToken)
	}

	return ""
}

func (c *AuthController) requireAuth(ctx *gin.Context) (*domain.AccessTokenInfo, bool) {
	accessToken := extractBearerToken(ctx.GetHeader("Authorization"))
	if accessToken == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return nil, false
	}

	info, err := c.useCase.ValidateAccessToken(ctx.Request.Context(), accessToken)
	if err != nil {
		c.respondWithError(ctx, err)
		return nil, false
	}

	return info, true
}

func extractBearerToken(authHeader string) string {
	value := strings.TrimSpace(authHeader)
	if value == "" {
		return ""
	}

	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func (c *AuthController) respondWithTokenPair(ctx *gin.Context, statusCode int, tokens *domain.AuthResult) {
	if tokens == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "empty token response"})
		return
	}

	refreshMaxAge := int(time.Until(tokens.RefreshExpiresAt).Seconds())
	c.setRefreshCookie(ctx, tokens.RefreshToken, refreshMaxAge)

	ctx.JSON(statusCode, gin.H{
		"access_token":       tokens.AccessToken,
		"refresh_token":      tokens.RefreshToken,
		"token_type":         tokens.TokenType,
		"expires_in":         tokens.ExpiresIn,
		"expires_at":         tokens.ExpiresAt,
		"refresh_expires_at": tokens.RefreshExpiresAt,
		"user_id":            tokens.UserID,
	})
}

func (c *AuthController) setRefreshCookie(ctx *gin.Context, refreshToken string, maxAge int) {
	if maxAge < 0 {
		maxAge = 0
	}

	ctx.SetCookie(
		c.config.RefreshCookieName,
		refreshToken,
		maxAge,
		"/",
		c.config.CookieDomain,
		c.config.CookieSecure,
		true,
	)
}

func (c *AuthController) clearRefreshCookie(ctx *gin.Context) {
	ctx.SetCookie(
		c.config.RefreshCookieName,
		"",
		-1,
		"/",
		c.config.CookieDomain,
		c.config.CookieSecure,
		true,
	)
}

func (c *AuthController) respondWithError(ctx *gin.Context, err error) {
	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, usecase.ErrUserAlreadyExists):
		statusCode = http.StatusConflict
	case errors.Is(err, usecase.ErrInvalidCredentials), errors.Is(err, usecase.ErrInvalidToken), errors.Is(err, usecase.ErrSessionExpired), errors.Is(err, usecase.ErrSessionNotFound), errors.Is(err, usecase.ErrSessionRevoked), errors.Is(err, usecase.ErrTokenExpired), errors.Is(err, usecase.ErrFingerprintMismatch):
		statusCode = http.StatusUnauthorized
	case errors.Is(err, usecase.ErrInternal):
		statusCode = http.StatusInternalServerError
	default:
		statusCode = http.StatusBadRequest
	}

	ctx.JSON(statusCode, gin.H{"error": err.Error()})
}
