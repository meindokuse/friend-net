package v1

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	analytic "github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/pkg/logger"
)

type Implementation struct {
	services *analytic.Registry
}

func New(services *analytic.Registry) *Implementation {
	return &Implementation{services: services}
}

func (i *Implementation) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

	r.Route("/analytics", func(r chi.Router) {
		r.Get("/stats", i.GetStats)
		r.Get("/events", i.ListEvents)
		r.Post("/events", i.CreateEvent)
		r.Delete("/events/{id}", i.DeleteEvent)
	})

	return r
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := middleware.GetReqID(r.Context())
		ctx := logger.InitRequestContext(r.Context(), traceID, r.URL.Path)

		rlc := &reqLogCtx{}
		ctx = r.Context()

		slog.InfoContext(ctx, "request received", "method", r.Method, "route", r.URL.Path)

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r.WithContext(ctx))

		status := ww.Status()
		attrs := []any{
			"status_code", status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if rlc.errMsg != "" {
			attrs = append(attrs, "error", rlc.errMsg)
		}
		switch {
		case status >= 500:
			slog.ErrorContext(ctx, "response sent", attrs...)
		case status >= 400:
			slog.WarnContext(ctx, "response sent", attrs...)
		default:
			slog.InfoContext(ctx, "response sent", attrs...)
		}
	})
}
