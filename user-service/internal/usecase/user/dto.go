package user

import (
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// INPUT DTO
// ─────────────────────────────────────────────

type CreateUserInput struct {
	// ID пришёл извне (auth-service генерит его при регистрации и передаёт сюда).
	// Если nil — usecase сгенерирует сам.
	ID          *uuid.UUID
	Username    string
	Email       *string
	Phone       *string
	DisplayName string
}

type UpdateProfileInput struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	Version     int // optimistic lock
}

type UpdateSettingsInput struct {
	UserID            uuid.UUID
	WhoCanMessage     string
	WhoCanSeeLastSeen string
	WhoCanSeeProfile  string
	Language          string
	Timezone          string
	Version           int
}

type ChangeEmailInput struct {
	UserID  uuid.UUID
	Email   string
	Version int
}

type ChangePhoneInput struct {
	UserID  uuid.UUID
	Phone   string
	Version int
}

type SearchUsersInput struct {
	Query  string
	Limit  int
	Offset int
}

type DeleteUserInput struct {
	UserID  uuid.UUID
	Version int
}

// ─────────────────────────────────────────────
// OUTPUT DTO
// ─────────────────────────────────────────────

// UserOutput — полный профиль (для "своего" пользователя).
type UserOutput struct {
	ID          uuid.UUID
	Username    string
	Email       *string
	Phone       *string
	DisplayName string
	Bio         *string
	AvatarURL   *string

	EmailVerified bool
	PhoneVerified bool

	Privacy  PrivacyOutput
	Language string
	Timezone string

	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastSeenAt *time.Time
	Version    int
}

type PrivacyOutput struct {
	WhoCanMessage     string
	WhoCanSeeLastSeen string
	WhoCanSeeProfile  string
}

// PublicUserOutput — урезанный профиль для чужих пользователей.
// Не содержит email/phone/settings.
type PublicUserOutput struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	Bio         *string
	AvatarURL   *string
	LastSeenAt  *time.Time
}
