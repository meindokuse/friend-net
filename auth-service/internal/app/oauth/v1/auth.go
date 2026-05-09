package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	oauthlogin "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/login"
)

// GoogleAuth handles GET /auth/google - redirects to Google
func (i *Implementation) GoogleAuth(ctx *gin.Context) {
	state := i.generateRandomState()
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
