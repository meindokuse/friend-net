package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


func (i *Implementation) Validate(ctx *gin.Context) {
    // 1. Берем токен ТОЛЬКО из заголовка (как это делает Traefik)
    authHeader := ctx.GetHeader("Authorization")
    token := extractBearerToken(authHeader)

    if token == "" {
        ctx.AbortWithStatus(http.StatusUnauthorized) // 401 для Traefik
        return
    }

    // 2. Вызываем бизнес-логику проверки
    result, err := i.services.Introspect.Introspect(ctx.Request.Context(), token)
    if err != nil || !result.Active {
        ctx.AbortWithStatus(http.StatusUnauthorized)
        return
    }

    // 3. ПРОФИТ: Прокидываем данные о пользователе дальше через заголовки
    // Traefik возьмет эти X-заголовки и отправит их в user-service
    ctx.Header("X-Account-ID", result.AccountID)
    ctx.Header("X-Session-ID", result.SessionID)

    ctx.Status(http.StatusOK) // Тело пустое, только 200 OK
}