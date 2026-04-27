package dto

import (
	"time"

	"github.com/google/uuid"
)

// ─── Requests ────────────────────────────────────────────────────────────────

// CreateUserRequest — входящий запрос на создание пользователя.
// ID опционален: если не передан, usecase генерирует сам (обычно передаётся от auth-service).
type CreateUserRequest struct {
	ID          *uuid.UUID `json:"id,omitempty"`
	Username    string     `json:"username"`
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	DisplayName string     `json:"display_name"`
}

// UpdateProfileRequest — обновление публичного профиля.
type UpdateProfileRequest struct {
	DisplayName string  `json:"display_name"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Version     int     `json:"version"`
}

// UpdateSettingsRequest — обновление настроек приватности и локали.
type UpdateSettingsRequest struct {
	WhoCanMessage     string `json:"who_can_message"`
	WhoCanSeeLastSeen string `json:"who_can_see_last_seen"`
	WhoCanSeeProfile  string `json:"who_can_see_profile"`
	Language          string `json:"language"`
	Timezone          string `json:"timezone"`
	Version           int    `json:"version"`
}

// ChangeEmailRequest — смена email.
type ChangeEmailRequest struct {
	Email   string `json:"email"`
	Version int    `json:"version"`
}

// ChangePhoneRequest — смена телефона.
type ChangePhoneRequest struct {
	Phone   string `json:"phone"`
	Version int    `json:"version"`
}

// DeleteUserRequest — удаление аккаунта (soft delete).
type DeleteUserRequest struct {
	Version int `json:"version"`
}

// ListUsersRequest — параметры запроса списка пользователей с keyset-пагинацией.
type ListUsersRequest struct {
	Cursor string `json:"cursor,omitempty"` // base64-encoded CursorUsernamePayload
	Limit  int    `json:"limit,omitempty"`
}

// GetUsersByIDsRequest — batch-запрос публичных профилей (inter-service).
type GetUsersByIDsRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

// ─── Responses ───────────────────────────────────────────────────────────────

// UserResponse — полный профиль пользователя. Отдаётся владельцу аккаунта (/users/me).
type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       *string   `json:"email,omitempty"`
	Phone       *string   `json:"phone,omitempty"`
	DisplayName string    `json:"display_name"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`

	EmailVerified bool `json:"email_verified"`
	PhoneVerified bool `json:"phone_verified"`

	Privacy  PrivacyResponse `json:"privacy"`
	Language string          `json:"language"`
	Timezone string          `json:"timezone"`

	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	Version    int        `json:"version"`
}

// PublicUserResponse — публичный профиль. Отдаётся при просмотре чужого аккаунта.
// Не содержит email, phone, настройки приватности.
type PublicUserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Bio         *string    `json:"bio,omitempty"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

// PrivacyResponse — настройки приватности в ответе.
type PrivacyResponse struct {
	WhoCanMessage     string `json:"who_can_message"`
	WhoCanSeeLastSeen string `json:"who_can_see_last_seen"`
	WhoCanSeeProfile  string `json:"who_can_see_profile"`
}

// ListUsersResponse — страница пользователей с курсором для следующей страницы.
type ListUsersResponse struct {
	Items      []*PublicUserResponse `json:"items"`
	NextCursor *string               `json:"next_cursor,omitempty"` // nil если страниц больше нет
	HasMore    bool                  `json:"has_more"`
}

// UsersByIDsResponse — ответ на batch-запрос.
type UsersByIDsResponse struct {
	Items []*PublicUserResponse `json:"items"`
}
