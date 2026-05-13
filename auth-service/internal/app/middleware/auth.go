package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AccountIDKey = "account_id"
	SessionIDKey = "session_id"
)

// RequireAccountID reads the X-Account-ID header forwarded by Traefik forwardAuth
// and stores it in the Gin context. Returns 401 if the header is absent.
func RequireAccountID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		accountID := strings.TrimSpace(ctx.GetHeader("X-Account-ID"))
		if accountID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx.Set(AccountIDKey, accountID)

		if sessionID := strings.TrimSpace(ctx.GetHeader("X-Session-ID")); sessionID != "" {
			ctx.Set(SessionIDKey, sessionID)
		}

		ctx.Next()
	}
}
