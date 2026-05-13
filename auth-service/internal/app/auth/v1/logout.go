package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/app/middleware"
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
	accountID := ctx.GetString(middleware.AccountIDKey)

	if err := i.services.Logout.LogoutAll(ctx.Request.Context(), accountID); err != nil {
		i.respondWithError(ctx, err)
		return
	}

	i.clearRefreshCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

// Sessions handles GET /auth/sessions
func (i *Implementation) Sessions(ctx *gin.Context) {
	accountID := ctx.GetString(middleware.AccountIDKey)
	sessionID := ctx.GetString(middleware.SessionIDKey)

	sessions, err := i.services.GetSessions.GetSessions(ctx.Request.Context(), accountID, sessionID)
	if err != nil {
		i.respondWithError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// RevokeSession handles DELETE /auth/sessions/:session_id
func (i *Implementation) RevokeSession(ctx *gin.Context) {
	accountID := ctx.GetString(middleware.AccountIDKey)

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

// extractBearerToken parses "Bearer <token>" from the Authorization header.
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
