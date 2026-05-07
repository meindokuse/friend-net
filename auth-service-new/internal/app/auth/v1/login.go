package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	authlogin "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/login"
	authrefresh "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/refresh"
	authregister "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/register"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// Login handles POST /auth/login
func (i *Implementation) Login(ctx *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	dto := authlogin.LoginDTO{
		Email:       request.Email,
		Password:    request.Password,
		Fingerprint: i.requestFingerprint(ctx),
		IP:          ctx.ClientIP(),
		UserAgent:   ctx.Request.UserAgent(),
	}

	result, err := i.services.Login.Login(ctx.Request.Context(), dto)
	if err != nil {
		i.respondWithError(ctx, err)
		return
	}

	i.respondWithTokenPairFromLogin(ctx, http.StatusOK, result)
}

// Register handles POST /auth/register
func (i *Implementation) Register(ctx *gin.Context) {
	var request struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		DisplayName string `json:"display_name"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Register creates account but doesn't return tokens
	// User needs to login separately
	_, err := i.services.Register.Register(ctx.Request.Context(), authregister.RegisterDTO{
		Email:       request.Email,
		Password:    request.Password,
		DisplayName: request.DisplayName,
	})
	if err != nil {
		i.respondWithError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "account created"})
}

// Refresh handles POST /auth/refresh
func (i *Implementation) Refresh(ctx *gin.Context) {
	refreshToken := i.readRefreshToken(ctx)
	if refreshToken == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "refresh token is required"})
		return
	}

	dto := authrefresh.RefreshDTO{
		RefreshToken: refreshToken,
		Fingerprint:  i.requestFingerprint(ctx),
	}

	result, err := i.services.Refresh.Refresh(ctx.Request.Context(), dto)
	if err != nil {
		i.respondWithError(ctx, err)
		return
	}

	i.respondWithTokenPairFromRefresh(ctx, http.StatusOK, result)
}

func (i *Implementation) requestFingerprint(ctx *gin.Context) string {
	fingerprint := strings.TrimSpace(ctx.GetHeader("X-Device-Fingerprint"))
	if fingerprint != "" {
		return fingerprint
	}
	return ctx.Request.UserAgent()
}

func (i *Implementation) readRefreshToken(ctx *gin.Context) string {
	// Try header first
	refreshToken := strings.TrimSpace(ctx.GetHeader("X-Refresh-Token"))
	if refreshToken != "" {
		return refreshToken
	}

	// Try cookie
	if cookieToken, err := ctx.Cookie(i.config.RefreshCookieName); err == nil && cookieToken != "" {
		return cookieToken
	}

	// Try body
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := ctx.ShouldBindJSON(&request); err == nil {
		return strings.TrimSpace(request.RefreshToken)
	}

	return ""
}

func (i *Implementation) respondWithError(ctx *gin.Context, err error) {
	statusCode := http.StatusInternalServerError

	var terr *terror.Error
	if terror.IsNotFound(err) {
		statusCode = http.StatusNotFound
	} else if terror.IsUnauthorized(err) {
		statusCode = http.StatusUnauthorized
	} else if terror.IsConflict(err) {
		statusCode = http.StatusConflict
	} else if terror.IsBadRequest(err) {
		statusCode = http.StatusBadRequest
	}

	ctx.JSON(statusCode, gin.H{"error": err.Error()})
}
