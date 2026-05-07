package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	authlogin "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/login"
	oauthlogin "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/login"
	authrefresh "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/refresh"
)

func (i *Implementation) respondWithTokenPairFromLogin(ctx *gin.Context, statusCode int, result *authlogin.Result) {
	i.respondWithTokenPair(ctx, statusCode, result.AccessToken, result.RefreshToken, result.TokenType, result.ExpiresIn, result.ExpiresAt, result.RefreshExpiresAt, result.AccountID, nil)
}

func (i *Implementation) respondWithTokenPairFromRefresh(ctx *gin.Context, statusCode int, result *authrefresh.Result) {
	i.respondWithTokenPair(ctx, statusCode, result.AccessToken, result.RefreshToken, result.TokenType, result.ExpiresIn, result.ExpiresAt, result.RefreshExpiresAt, result.AccountID, nil)
}

func (i *Implementation) respondWithTokenPairFromOAuth(ctx *gin.Context, statusCode int, result *oauthlogin.Result) {
	isNewUser := result.IsNewUser
	i.respondWithTokenPair(ctx, statusCode, result.AccessToken, result.RefreshToken, result.TokenType, result.ExpiresIn, result.ExpiresAt, result.RefreshExpiresAt, result.AccountID, &isNewUser)
}

func (i *Implementation) respondWithTokenPair(
	ctx *gin.Context,
	statusCode int,
	accessToken, refreshToken, tokenType string,
	expiresIn int64,
	expiresAt, refreshExpiresAt, accountID string,
	isNewUser *bool,
) {
	refreshMaxAge := int(time.Until(time.Now().Add(30 * 24 * time.Hour)).Seconds())
	i.setRefreshCookie(ctx, refreshToken, refreshMaxAge)

	response := gin.H{
		"access_token":       accessToken,
		"refresh_token":      refreshToken,
		"token_type":         tokenType,
		"expires_in":         expiresIn,
		"expires_at":         expiresAt,
		"refresh_expires_at": refreshExpiresAt,
		"account_id":         accountID,
	}
	if isNewUser != nil {
		response["is_new_user"] = *isNewUser
	}
	ctx.JSON(statusCode, response)
}

func (i *Implementation) setRefreshCookie(ctx *gin.Context, refreshToken string, maxAge int) {
	if maxAge < 0 {
		maxAge = 0
	}

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

func (i *Implementation) clearRefreshCookie(ctx *gin.Context) {
	ctx.SetCookie(
		i.config.RefreshCookieName,
		"",
		-1,
		"/",
		i.config.CookieDomain,
		i.config.CookieSecure,
		true,
	)
}
