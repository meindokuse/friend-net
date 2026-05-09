package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	oauthlink "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/link"
)

// LinkGoogle handles GET /auth/link/google
func (i *Implementation) LinkGoogle(ctx *gin.Context) {
	state := i.generateRandomState()
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
