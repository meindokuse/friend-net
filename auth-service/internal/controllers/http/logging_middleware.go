package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

func RequestContextLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		traceID := ctx.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}

		reqPath := ctx.FullPath()
		if reqPath == "" {
			reqPath = ctx.Request.URL.Path
		}

		reqCtx := context.WithValue(ctx.Request.Context(), sharedlogger.LogFieldsKey, &sharedlogger.LogEntry{
			TraceID: traceID,
			ReqPath: reqPath,
			Extra: map[string]interface{}{
				"method": ctx.Request.Method,
			},
		})

		ctx.Request = ctx.Request.WithContext(reqCtx)
		ctx.Header("X-Trace-ID", traceID)

		startedAt := time.Now()
		slog.InfoContext(reqCtx, "http request started")

		ctx.Next()

		status := ctx.Writer.Status()
		duration := time.Since(startedAt)

		switch {
		case status >= 500:
			slog.ErrorContext(ctx.Request.Context(), "http request finished", slog.Int("status", status), slog.Duration("duration", duration))
		case status >= 400:
			slog.WarnContext(ctx.Request.Context(), "http request finished", slog.Int("status", status), slog.Duration("duration", duration))
		default:
			slog.InfoContext(ctx.Request.Context(), "http request finished", slog.Int("status", status), slog.Duration("duration", duration))
		}
	}
}
