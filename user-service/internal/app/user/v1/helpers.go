package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/apperr"
)

// reqLogCtx is a per-request mutable cell. requestLogger allocates it and
// passes it downstream via context; error helpers write the message so the
// middleware can include it in the "response sent" log at the correct level.
type reqLogCtx struct{ errMsg string }
type reqLogCtxKey struct{}

func storeErrMsg(ctx context.Context, msg string) {
	if rlc, ok := ctx.Value(reqLogCtxKey{}).(*reqLogCtx); ok {
		rlc.errMsg = msg
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError records the error message for the response log and writes the HTTP error response.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	storeErrMsg(r.Context(), msg)
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

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apperr.ErrInvalidInput),
		errors.Is(err, entity.ErrDisplayNameRequired),
		errors.Is(err, entity.ErrDisplayNameTooLong),
		errors.Is(err, entity.ErrBioTooLong),
		errors.Is(err, entity.ErrInvalidPrivacyLevel),
		errors.Is(err, entity.ErrEmailOrPhoneRequired):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrUserNotFound):
		writeError(w, r, http.StatusNotFound, "user not found")
	case errors.Is(err, entity.ErrUsernameAlreadyTaken),
		errors.Is(err, entity.ErrEmailAlreadyTaken),
		errors.Is(err, entity.ErrPhoneAlreadyTaken),
		errors.Is(err, entity.ErrVersionConflict):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrAlreadyDeleted):
		writeError(w, r, http.StatusGone, "user already deleted")
	default:
		slog.ErrorContext(r.Context(), "unhandled usecase error", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal server error")
	}
}

type userResponse struct {
	ID            uuid.UUID      `json:"id"`
	Username      string         `json:"username"`
	Email         *string        `json:"email,omitempty"`
	Phone         *string        `json:"phone,omitempty"`
	DisplayName   string         `json:"display_name"`
	Bio           *string        `json:"bio,omitempty"`
	AvatarURL     *string        `json:"avatar_url,omitempty"`
	EmailVerified bool           `json:"email_verified"`
	PhoneVerified bool           `json:"phone_verified"`
	IsActive      bool           `json:"is_active"`
	Privacy       privacyResponse `json:"privacy"`
	Language      string         `json:"language"`
	Timezone      string         `json:"timezone"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	LastSeenAt    *time.Time     `json:"last_seen_at,omitempty"`
	Version       int            `json:"version"`
}

type privacyResponse struct {
	WhoCanMessage     string `json:"who_can_message"`
	WhoCanSeeLastSeen string `json:"who_can_see_last_seen"`
	WhoCanSeeProfile  string `json:"who_can_see_profile"`
}

type publicUserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Bio         *string    `json:"bio,omitempty"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

func toUserResponse(u *entity.User) *userResponse {
	var email, phone *string
	if u.Email() != nil {
		s := u.Email().String()
		email = &s
	}
	if u.Phone() != nil {
		s := u.Phone().String()
		phone = &s
	}
	return &userResponse{
		ID:            u.ID(),
		Username:      u.Username().String(),
		Email:         email,
		Phone:         phone,
		DisplayName:   u.Profile().DisplayName,
		Bio:           u.Profile().Bio,
		AvatarURL:     u.Profile().AvatarURL,
		EmailVerified: u.Verification().EmailVerified,
		PhoneVerified: u.Verification().PhoneVerified,
		IsActive:      u.IsActive(),
		Privacy: privacyResponse{
			WhoCanMessage:     string(u.Settings().Privacy.WhoCanMessage),
			WhoCanSeeLastSeen: string(u.Settings().Privacy.WhoCanSeeLastSeen),
			WhoCanSeeProfile:  string(u.Settings().Privacy.WhoCanSeeProfile),
		},
		Language:   u.Settings().Language,
		Timezone:   u.Settings().Timezone,
		CreatedAt:  u.CreatedAt(),
		UpdatedAt:  u.UpdatedAt(),
		LastSeenAt: u.LastSeenAt(),
		Version:    u.Version(),
	}
}

func toPublicUserResponse(u *entity.User) *publicUserResponse {
	return &publicUserResponse{
		ID:          u.ID(),
		Username:    u.Username().String(),
		DisplayName: u.Profile().DisplayName,
		Bio:         u.Profile().Bio,
		AvatarURL:   u.Profile().AvatarURL,
		LastSeenAt:  u.LastSeenAt(),
	}
}
