package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/logger"
)

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		traceID := uuid.NewString()

		ctx := logger.InitRequestContext(c.Request.Context(), traceID, c.FullPath())
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Trace-Id", traceID)

		slog.DebugContext(ctx, "request started",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"remote_addr", c.ClientIP(),
		)

		c.Next()

		status := c.Writer.Status()
		duration := time.Since(start)

		finalCtx := ctx
		if accountID, exists := c.Get(AccountIDKey); exists {
			if id, ok := accountID.(string); ok && id != "" {
				finalCtx = logger.WithUserIDEntry(finalCtx, id)
			}
		}

		lvl := slog.LevelInfo
		if status >= 500 {
			lvl = slog.LevelError
		} else if status >= 400 {
			lvl = slog.LevelWarn
		}

		slog.Log(finalCtx, lvl, "request completed",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)
	}
}
