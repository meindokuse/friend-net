package logger

import (
	"context"
	"log/slog"
	"os"
)

const logFieldsKey = "log_fields"

type logEntry struct {
	TraceID string
	UserID  string
	ReqPath string
}

func Init(level string) {
	var l slog.Level
	switch level {
	case "debug", "DEBUG":
		l = slog.LevelDebug
	case "warn", "WARN":
		l = slog.LevelWarn
	case "error", "ERROR":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	slog.SetDefault(slog.New(h).With("service", "analytic-service"))
}

func InitRequestContext(ctx context.Context, traceID, reqPath string) context.Context {
	return context.WithValue(ctx, logFieldsKey, &logEntry{TraceID: traceID, ReqPath: reqPath})
}

func WithUserIDEntry(ctx context.Context, userID string) context.Context {
	if e, ok := ctx.Value(logFieldsKey).(*logEntry); ok && e != nil {
		updated := *e
		updated.UserID = userID
		return context.WithValue(ctx, logFieldsKey, &updated)
	}
	return context.WithValue(ctx, logFieldsKey, &logEntry{UserID: userID})
}
