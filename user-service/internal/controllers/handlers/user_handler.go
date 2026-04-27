package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
	"github.com/meindokuse/cloud-drive/user-service/internal/dto"
	usecase "github.com/meindokuse/cloud-drive/user-service/internal/usecase/user"
)

// UserHandler содержит все HTTP-хендлеры для работы с пользователями.
type UserHandler struct {
	svc *usecase.Service
}

func NewUserHandler(svc *usecase.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

// ─── POST /users ──────────────────────────────────────────────────────────────

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.svc.CreateUser(r.Context(), usecase.CreateUserInput{
		ID:          req.ID,
		Username:    req.Username,
		Email:       req.Email,
		Phone:       req.Phone,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.FromUserOutput(out))
}

// ─── GET /users/me ────────────────────────────────────────────────────────────

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromUserOutput(out))
}

// ─── GET /users/{id} ─────────────────────────────────────────────────────────

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	out, err := h.svc.GetPublicUserByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromPublicUserOutput(out))
}

// ─── GET /users/username/{username} ──────────────────────────────────────────

func (h *UserHandler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.GetPublicUserByUsername(r.Context(), chi.URLParam(r, "username"))
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromPublicUserOutput(out))
}

// ─── POST /users/batch ────────────────────────────────────────────────────────

func (h *UserHandler) GetUsersByIDs(w http.ResponseWriter, r *http.Request) {
	var req dto.GetUsersByIDsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.svc.GetUsersByIDs(r.Context(), req.IDs)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.UsersByIDsResponse{
		Items: dto.FromPublicUserOutputs(out),
	})
}

// ─── PATCH /users/me/profile ─────────────────────────────────────────────────

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.svc.UpdateProfile(r.Context(), usecase.UpdateProfileInput{
		UserID:      userID,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		Version:     req.Version,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromUserOutput(out))
}

// ─── PATCH /users/me/settings ────────────────────────────────────────────────

func (h *UserHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.UpdateSettingsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.svc.UpdateSettings(r.Context(), usecase.UpdateSettingsInput{
		UserID:            userID,
		WhoCanMessage:     req.WhoCanMessage,
		WhoCanSeeLastSeen: req.WhoCanSeeLastSeen,
		WhoCanSeeProfile:  req.WhoCanSeeProfile,
		Language:          req.Language,
		Timezone:          req.Timezone,
		Version:           req.Version,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromUserOutput(out))
}

// ─── PATCH /users/me/email ────────────────────────────────────────────────────

func (h *UserHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangeEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.svc.ChangeEmail(r.Context(), usecase.ChangeEmailInput{
		UserID:  userID,
		Email:   req.Email,
		Version: req.Version,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromUserOutput(out))
}

// ─── PATCH /users/me/phone ────────────────────────────────────────────────────

func (h *UserHandler) ChangePhone(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangePhoneRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.svc.ChangePhone(r.Context(), usecase.ChangePhoneInput{
		UserID:  userID,
		Phone:   req.Phone,
		Version: req.Version,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.FromUserOutput(out))
}

// ─── DELETE /users/me ─────────────────────────────────────────────────────────

func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.DeleteUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.DeleteUser(r.Context(), usecase.DeleteUserInput{
		UserID:  userID,
		Version: req.Version,
	}); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── GET /users?cursor=...&limit=... ─────────────────────────────────────────

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := parseIntQuery(q.Get("limit"), 20)

	params := domainuser.ListParams{Limit: limit}

	if raw := q.Get("cursor"); raw != "" {
		payload, err := dto.DecodeCursor(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		params.Cursor = &domainuser.UsernameCursor{
			Username: payload.Username,
			ID:       payload.ID,
		}
	}

	_, paged, err := h.svc.List(r.Context(), params)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	resp := dto.ListUsersResponse{
		Items:   dto.FromPublicUserOutputs(pagedToPublicOutputs(paged.Items)),
		HasMore: paged.HasMore,
	}
	if paged.HasMore {
		encoded, err := dto.EncodeCursor(dto.CursorUsernamePayload{
			Username: paged.NextCursor.Username,
			ID:       paged.NextCursor.ID,
		})
		if err == nil {
			resp.NextCursor = &encoded
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ─── POST /users/me/last-seen ─────────────────────────────────────────────────

func (h *UserHandler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.svc.UpdateLastSeen(r.Context(), userID); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── GET /users/search?q=...&limit=...&offset=... ────────────────────────────

func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	out, err := h.svc.SearchUsers(r.Context(), usecase.SearchUsersInput{
		Query:  q.Get("q"),
		Limit:  parseIntQuery(q.Get("limit"), 20),
		Offset: parseIntQuery(q.Get("offset"), 0),
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.UsersByIDsResponse{
		Items: dto.FromPublicUserOutputs(out),
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type contextKey string

const ctxUserID contextKey = "user_id"

// userIDFromCtx извлекает UserID из контекста запроса.
// Проставляется auth middleware перед вызовом хендлера.
func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(ctxUserID)
	if v == nil {
		return uuid.UUID{}, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseIntQuery(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}

// pagedToPublicOutputs конвертирует []*domainuser.User → []*usecase.PublicUserOutput.
func pagedToPublicOutputs(users []*domainuser.User) []*usecase.PublicUserOutput {
	out := make([]*usecase.PublicUserOutput, 0, len(users))
	for _, u := range users {
		out = append(out, &usecase.PublicUserOutput{
			ID:          u.ID(),
			Username:    u.Username().String(),
			DisplayName: u.Profile().DisplayName,
			Bio:         u.Profile().Bio,
			AvatarURL:   u.Profile().AvatarURL,
			LastSeenAt:  u.LastSeenAt(),
		})
	}
	return out
}

// writeUsecaseError маппит доменные/usecase ошибки в HTTP статусы.
func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, domainuser.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")

	case errors.Is(err, domainuser.ErrUsernameAlreadyTaken),
		errors.Is(err, domainuser.ErrEmailAlreadyTaken),
		errors.Is(err, domainuser.ErrPhoneAlreadyTaken):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, domainuser.ErrVersionConflict):
		writeError(w, http.StatusConflict, "version conflict, please retry")

	case errors.Is(err, domainuser.ErrAlreadyDeleted):
		writeError(w, http.StatusGone, "user already deleted")

	case errors.Is(err, domainuser.ErrDisplayNameRequired),
		errors.Is(err, domainuser.ErrDisplayNameTooLong),
		errors.Is(err, domainuser.ErrBioTooLong),
		errors.Is(err, domainuser.ErrInvalidPrivacyLevel),
		errors.Is(err, domainuser.ErrEmailOrPhoneRequired):
		writeError(w, http.StatusBadRequest, err.Error())

	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
