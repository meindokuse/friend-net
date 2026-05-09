package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	authlogout "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth/logout"
)

// Logout handles POST /auth/logout
func (i *Implementation) Logout(ctx *gin.Context) {
	accessToken := extractBearerToken(ctx.GetHeader("Authorization"))
	refreshToken := i.readRefreshToken(ctx)

	dto := authlogout.LogoutDTO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	if err := i.services.Logout.Logout(ctx.Request.Context(), dto); err != nil {
		i.respondWithError(ctx, err)
		return
	}

	i.clearRefreshCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// LogoutAll handles POST /auth/logout-all
func (i *Implementation) LogoutAll(ctx *gin.Context) {
	accountID, ok := i.requireAuth(ctx)
	if !ok {
		return
	}

	if err := i.services.Logout.LogoutAll(ctx.Request.Context(), accountID); err != nil {
		i.respondWithError(ctx, err)
		return
	}

	i.clearRefreshCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

// Sessions handles GET /auth/sessions
func (i *Implementation) Sessions(ctx *gin.Context) {
	accountID, ok := i.requireAuth(ctx)
	if !ok {
		return
	}

	sessionID, _ := ctx.Get("session_id")

	sessions, err := i.services.GetSessions.GetSessions(ctx.Request.Context(), accountID, sessionID.(string))
	if err != nil {
		i.respondWithError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// RevokeSession handles DELETE /auth/sessions/:session_id
func (i *Implementation) RevokeSession(ctx *gin.Context) {
	accountID, ok := i.requireAuth(ctx)
	if !ok {
		return
	}

	sessionID := strings.TrimSpace(ctx.Param("session_id"))
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	if err := i.services.RevokeSession.RevokeSession(ctx.Request.Context(), accountID, sessionID); err != nil {
		i.respondWithError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

// Introspect handles POST /auth/introspect
func (i *Implementation) Introspect(ctx *gin.Context) {
	token := extractBearerToken(ctx.GetHeader("Authorization"))

	if token == "" {
		var request struct {
			AccessToken string `json:"access_token"`
		}
		if err := ctx.ShouldBindJSON(&request); err == nil {
			token = strings.TrimSpace(request.AccessToken)
		}
	}

	if token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "access token is required"})
		return
	}

	result, err := i.services.Introspect.Introspect(ctx.Request.Context(), token)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"active":     result.Active,
		"account_id": result.AccountID,
		"session_id": result.SessionID,
		"expires_at": result.ExpiresAt,
	})
}

func (i *Implementation) requireAuth(ctx *gin.Context) (string, bool) {
	accessToken := extractBearerToken(ctx.GetHeader("Authorization"))
	if accessToken == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return "", false
	}

	result, err := i.services.Introspect.Introspect(ctx.Request.Context(), accessToken)
	if err != nil || !result.Active {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return "", false
	}

	ctx.Set("session_id", result.SessionID)
	return result.AccountID, true
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
