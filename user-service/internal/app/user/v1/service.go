package v1

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	userservice "github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user"
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
	r.Use(middleware.Recoverer)
	r.Route("/users", func(r chi.Router) {
		r.Get("/search", i.SearchUsers)
		r.Get("/{id}", i.GetUserByID)
		r.Get("/username/{username}", i.GetUserByUsername)
		r.Post("/batch", i.GetUsersByIDs)
		r.Post("/", i.CreateUser)
		r.Group(func(r chi.Router) {
			r.Use(i.AuthMiddleware)
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

type contextKey string

const userIDKey contextKey = "user_id"

func (i *Implementation) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("X-User-ID")
		if header == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		id, err := uuid.Parse(header)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, id)))
	})
}

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(userIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}
