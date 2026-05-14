package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Validate handles GET /auth/validate — called by Traefik forwardAuth.
// Returns 200 + X-Account-ID/X-Session-ID headers on success, 401 on failure.
func (i *Implementation) Validate(ctx *gin.Context) {
	token := extractBearerToken(ctx.GetHeader("Authorization"))
	if token == "" {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	result, err := i.services.Introspect.Introspect(ctx.Request.Context(), token)
	if err != nil || !result.Active {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	ctx.Header("X-Account-ID", result.AccountID)
	ctx.Header("X-Session-ID", result.SessionID)
	ctx.Status(http.StatusOK)
}

// Introspect handles POST /auth/introspect — public token inspection endpoint.
func (i *Implementation) Introspect(ctx *gin.Context) {
	token := extractBearerToken(ctx.GetHeader("Authorization"))

	if token == "" {
		var req struct {
			AccessToken string `json:"access_token"`
		}
		if err := ctx.ShouldBindJSON(&req); err == nil {
			token = strings.TrimSpace(req.AccessToken)
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
