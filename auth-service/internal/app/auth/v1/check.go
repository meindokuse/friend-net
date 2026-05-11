package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (i *Implementation) Validate(ctx *gin.Context) {
	accountID, ok := i.requireAuth(ctx)
	if !ok {
		return
	}

	sessionID, _ := ctx.Get("session_id")

	// Traefik возьмёт эти заголовки и добавит их в запрос к upstream сервису
	ctx.Header("X-Account-Id", accountID)
	ctx.Header("X-Session-Id", sessionID.(string))
	ctx.Status(http.StatusOK)
}
