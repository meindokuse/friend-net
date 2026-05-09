package v1

import "github.com/gin-gonic/gin"

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