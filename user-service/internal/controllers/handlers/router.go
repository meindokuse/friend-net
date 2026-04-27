package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter собирает chi-роутер и регистрирует все маршруты.
func NewRouter(userHandler *UserHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Route("/users", func(r chi.Router) {
		// Публичные — без auth
		r.Get("/search", userHandler.SearchUsers)                    // GET /users/search?q=alice
		r.Get("/{id}", userHandler.GetUserByID)                      // GET /users/{id}
		r.Get("/username/{username}", userHandler.GetUserByUsername) // GET /users/username/alice
		r.Post("/batch", userHandler.GetUsersByIDs)                  // POST /users/batch
		r.Post("/", userHandler.CreateUser)                          // POST /users

		// Требуют auth (middleware проставляет user_id в ctx)
		r.Group(func(r chi.Router) {
			// r.Use(AuthMiddleware) — подключить когда будет готов
			r.Get("/me", userHandler.GetMe)                     // GET  /users/me
			r.Delete("/me", userHandler.DeleteMe)               // DELETE /users/me
			r.Patch("/me/profile", userHandler.UpdateProfile)   // PATCH /users/me/profile
			r.Patch("/me/settings", userHandler.UpdateSettings) // PATCH /users/me/settings
			r.Patch("/me/email", userHandler.ChangeEmail)       // PATCH /users/me/email
			r.Patch("/me/phone", userHandler.ChangePhone)       // PATCH /users/me/phone
			r.Post("/me/last-seen", userHandler.UpdateLastSeen) // POST  /users/me/last-seen
			r.Get("/me/list", userHandler.ListUsers)            // GET   /users/me/list?cursor=...&limit=...
		})
	})

	return r
}
