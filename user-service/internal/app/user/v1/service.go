package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
)

type Implementation struct{ service *user.Service }

func New(service *user.Service) *Implementation { return &Implementation{service: service} }

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

type key string

const userIDKey key = "user_id"

func (i *Implementation) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("X-User-ID"))
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

func (i *Implementation) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          *uuid.UUID `json:"id,omitempty"`
		Username    string     `json:"username"`
		Email       *string    `json:"email,omitempty"`
		Phone       *string    `json:"phone,omitempty"`
		DisplayName string     `json:"display_name"`
	}
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	out, err := i.service.CreateUser(r.Context(), user.CreateUserInput{ID: req.ID, Username: req.Username, Email: req.Email, Phone: req.Phone, DisplayName: req.DisplayName})
	if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusCreated, out)
}
func (i *Implementation) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	out, err := i.service.GetUserByID(r.Context(), userID); if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id")); if err != nil { writeError(w, http.StatusBadRequest, "invalid user id"); return }
	out, err := i.service.GetPublicUserByID(r.Context(), id); if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	out, err := i.service.GetPublicUserByUsername(r.Context(), chi.URLParam(r, "username")); if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) GetUsersByIDs(w http.ResponseWriter, r *http.Request) {
	var req struct{ IDs []uuid.UUID `json:"ids"` }
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	out, err := i.service.GetUsersByIDs(r.Context(), req.IDs); if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}
func (i *Implementation) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	var req struct{ DisplayName string `json:"display_name"`; Bio *string `json:"bio,omitempty"`; AvatarURL *string `json:"avatar_url,omitempty"`; Version int `json:"version"` }
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	out, err := i.service.UpdateProfile(r.Context(), user.UpdateProfileInput{UserID: userID, DisplayName: req.DisplayName, Bio: req.Bio, AvatarURL: req.AvatarURL, Version: req.Version})
	if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	var req struct {
		WhoCanMessage string `json:"who_can_message"`; WhoCanSeeLastSeen string `json:"who_can_see_last_seen"`; WhoCanSeeProfile string `json:"who_can_see_profile"`
		Language string `json:"language"`; Timezone string `json:"timezone"`; Version int `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	out, err := i.service.UpdateSettings(r.Context(), userID, user.UpdateSettingsInput{WhoCanMessage: req.WhoCanMessage, WhoCanSeeLastSeen: req.WhoCanSeeLastSeen, WhoCanSeeProfile: req.WhoCanSeeProfile, Language: req.Language, Timezone: req.Timezone, V: req.Version})
	if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	var req struct { Email string `json:"email"`; Version int `json:"version"` }
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	out, err := i.service.ChangeEmail(r.Context(), user.ChangeEmailInput{UserID: userID, Email: req.Email, Version: req.Version})
	if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) ChangePhone(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	var req struct { Phone string `json:"phone"`; Version int `json:"version"` }
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	out, err := i.service.ChangePhone(r.Context(), user.ChangePhoneInput{UserID: userID, Phone: req.Phone, Version: req.Version})
	if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, out)
}
func (i *Implementation) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	var req struct{ Version int `json:"version"` }
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	if err := i.service.DeleteUser(r.Context(), user.DeleteUserInput{UserID: userID, Version: req.Version}); err != nil { writeUsecaseError(w, err); return }
	w.WriteHeader(http.StatusNoContent)
}
func (i *Implementation) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r); if !ok { writeError(w, http.StatusUnauthorized, "unauthorized"); return }
	if err := i.service.UpdateLastSeen(r.Context(), userID); err != nil { writeUsecaseError(w, err); return }
	w.WriteHeader(http.StatusNoContent)
}
func (i *Implementation) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := i.service.SearchUsers(r.Context(), user.SearchUsersInput{Query: q.Get("q"), Limit: parseIntQuery(q.Get("limit"), 20), Offset: parseIntQuery(q.Get("offset"), 0)})
	if err != nil { writeUsecaseError(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}
func (i *Implementation) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query(); limit := parseIntQuery(q.Get("limit"), 20); params := entity.ListParams{Limit: limit}
	if raw := q.Get("cursor"); raw != "" {
		b, err := base64.URLEncoding.DecodeString(raw); if err != nil { writeError(w, http.StatusBadRequest, "invalid cursor"); return }
		var payload struct{ Username string `json:"username"`; ID uuid.UUID `json:"id"` }
		if err := json.Unmarshal(b, &payload); err != nil { writeError(w, http.StatusBadRequest, "invalid cursor"); return }
		params.Cursor = &entity.UsernameCursor{Username: payload.Username, ID: payload.ID}
	}
	_, paged, err := i.service.List(r.Context(), params); if err != nil { writeUsecaseError(w, err); return }
	resp := map[string]any{"items": paged.Items, "has_more": paged.HasMore}
	if paged.HasMore {
		b, _ := json.Marshal(map[string]any{"username": paged.NextCursor.Username, "id": paged.NextCursor.ID})
		c := base64.URLEncoding.EncodeToString(b); resp["next_cursor"] = c
	}
	writeJSON(w, http.StatusOK, resp)
}

func decodeJSON(r *http.Request, dst any) error { defer r.Body.Close(); return json.NewDecoder(r.Body).Decode(dst) }
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json"); w.WriteHeader(status); _ = json.NewEncoder(w).Encode(body)
}
func writeError(w http.ResponseWriter, status int, msg string) { writeJSON(w, status, map[string]string{"error": msg}) }
func parseIntQuery(s string, def int) int { if s == "" { return def }; var n int; if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 { return def }; return n }

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidInput), errors.Is(err, entity.ErrDisplayNameRequired), errors.Is(err, entity.ErrDisplayNameTooLong), errors.Is(err, entity.ErrBioTooLong), errors.Is(err, entity.ErrInvalidPrivacyLevel), errors.Is(err, entity.ErrEmailOrPhoneRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, entity.ErrUsernameAlreadyTaken), errors.Is(err, entity.ErrEmailAlreadyTaken), errors.Is(err, entity.ErrPhoneAlreadyTaken), errors.Is(err, entity.ErrVersionConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrAlreadyDeleted):
		writeError(w, http.StatusGone, "user already deleted")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
