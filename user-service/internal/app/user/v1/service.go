package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	userservice "github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/logger"
)

type Implementation struct {
	services *userservice.Registry
}

func New(services *userservice.Registry) *Implementation {
	return &Implementation{services: services}
}

func (i *Implementation) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Route("/users", func(r chi.Router) {
		r.Get("/search", i.SearchUsers)
		r.Get("/{id}", i.GetUserByID)
		r.Get("/username/{username}", i.GetUserByUsername)
		r.Post("/batch", i.GetUsersByIDs)
		r.Post("/", i.CreateUser)
		r.Group(func(r chi.Router) {
			r.Use(ExtractUserIDMiddleware)
			r.Get("/me", i.GetMe)
			r.Delete("/me", i.DeleteMe)
			r.Patch("/me/profile", i.UpdateProfile)
			r.Patch("/me/settings", i.UpdateSettings)
			r.Patch("/me/email", i.ChangeEmail)
			r.Patch("/me/phone", i.ChangePhone)
			r.Post("/me/last-seen", i.UpdateLastSeen)
			r.Get("/me/list", i.ListUsers)
		})
	})
	return r
}

// requestLogger is the outermost HTTP middleware. It:
//  1. Seeds the context with trace_id, path, and user_id (from X-User-ID header if present).
//  2. Injects a *reqLogCtx cell so downstream error helpers can record the error message.
//  3. Logs "request received" at INFO before the handler runs.
//  4. Logs "response sent" after the handler at INFO (2xx/3xx), WARN (4xx), ERROR (5xx),
//     including the error message written by the handler.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := middleware.GetReqID(r.Context())
		ctx := logger.InitRequestContext(r.Context(), traceID, r.URL.Path)

		if uid := r.Header.Get("X-User-ID"); uid != "" {
			ctx = logger.WithUserIDEntry(ctx, uid)
		}

		rlc := &reqLogCtx{}
		ctx = context.WithValue(ctx, reqLogCtxKey{}, rlc)

		slog.InfoContext(ctx, "request received",
			"method", r.Method,
			"route", r.URL.Path,
		)

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

type contextKey string

const userIDKey contextKey = "user_id"

// func (i *Implementation) AuthMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		header := r.Header.Get("X-User-ID")
// 		if header == "" {
// 			writeError(w, r, http.StatusUnauthorized, "unauthorized")
// 			return
// 		}
// 		id, err := uuid.Parse(header)
// 		if err != nil {
// 			writeError(w, r, http.StatusUnauthorized, "unauthorized")
// 			return
// 		}
// 		ctx := context.WithValue(r.Context(), userIDKey, id)
// 		ctx = logger.WithUserIDEntry(ctx, id.String())
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(userIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

func ExtractUserIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Достаем то, что прислал Traefik
        headerID := r.Header.Get("X-Account-Id")
        if headerID == "" {
			slog.WarnContext(r.Context(),"miss user-data")
            next.ServeHTTP(w, r)
            return
        }

        // Парсим строку в uuid.UUID
        id, err := uuid.Parse(headerID)
        if err != nil {
			slog.WarnContext(r.Context(),"error user-data parse")
            next.ServeHTTP(w, r)
            return
        }

        // Кладем в контекст
        ctx := context.WithValue(r.Context(), userIDKey, id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}