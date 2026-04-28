package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
)

// ──────────────────────────────────────────────────────────────
// Value objects внутри агрегата User
// ──────────────────────────────────────────────────────────────

type Profile struct {
	DisplayName string
	Bio         *string
	AvatarURL   *string
}

type PrivacyLevel string

const (
	PrivacyEveryone PrivacyLevel = "everyone"
	PrivacyFriends  PrivacyLevel = "friends"
	PrivacyNobody   PrivacyLevel = "nobody"
)

func (p PrivacyLevel) IsValid() bool {
	switch p {
	case PrivacyEveryone, PrivacyFriends, PrivacyNobody:
		return true
	}
	return false
}

type PrivacySettings struct {
	WhoCanMessage     PrivacyLevel
	WhoCanSeeLastSeen PrivacyLevel
	WhoCanSeeProfile  PrivacyLevel
}

type Settings struct {
	Privacy  PrivacySettings
	Language string
	Timezone string
}

type Verification struct {
	EmailVerified bool
	PhoneVerified bool
}

// ──────────────────────────────────────────────────────────────
// User — корень агрегата
// ──────────────────────────────────────────────────────────────

// User — доменная сущность. Инварианты защищены: нельзя создать невалидный User.
// Поля приватные; изменения только через бизнес-методы.
type User struct {
	id           uuid.UUID
	username     vo.Username
	email        *vo.Email
	phone        *vo.Phone
	profile      Profile
	settings     Settings
	verification Verification
	isActive     bool
	createdAt    time.Time
	updatedAt    time.Time
	lastSeenAt   *time.Time
	deletedAt    *time.Time
	version      int // для optimistic locking
}

// NewUser — фабрика создания нового пользователя. Применяет инварианты.
func NewUser(
	id uuid.UUID,
	username vo.Username,
	email *vo.Email,
	phone *vo.Phone,
	displayName string,
) (*User, error) {
	if email == nil && phone == nil {
		return nil, ErrEmailOrPhoneRequired
	}
	if displayName == "" {
		return nil, ErrDisplayNameRequired
	}
	if len(displayName) > 64 {
		return nil, ErrDisplayNameTooLong
	}

	now := time.Now().UTC()
	return &User{
		id:       id,
		username: username,
		email:    email,
		phone:    phone,
		profile: Profile{
			DisplayName: displayName,
		},
		settings:  defaultSettings(),
		isActive:  true,
		createdAt: now,
		updatedAt: now,
		version:   1,
	}, nil
}

// Reconstruct — восстановление из БД. Не валидирует (данные в БД считаем валидными).
// Используется только в маппере репозитория.
func Reconstruct(
	id uuid.UUID,
	username vo.Username,
	email *vo.Email,
	phone *vo.Phone,
	profile Profile,
	settings Settings,
	verification Verification,
	isActive bool,
	createdAt, updatedAt time.Time,
	lastSeenAt, deletedAt *time.Time,
	version int,
) *User {
	return &User{
		id:           id,
		username:     username,
		email:        email,
		phone:        phone,
		profile:      profile,
		settings:     settings,
		verification: verification,
		isActive:     isActive,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		lastSeenAt:   lastSeenAt,
		deletedAt:    deletedAt,
		version:      version,
	}
}

// ─── Getters ──────────────────────────────────────────────────

func (u *User) ID() uuid.UUID              { return u.id }
func (u *User) Username() vo.Username      { return u.username }
func (u *User) Email() *vo.Email           { return u.email }
func (u *User) Phone() *vo.Phone           { return u.phone }
func (u *User) Profile() Profile           { return u.profile }
func (u *User) Settings() Settings         { return u.settings }
func (u *User) Verification() Verification { return u.verification }
func (u *User) IsActive() bool             { return u.isActive }
func (u *User) CreatedAt() time.Time       { return u.createdAt }
func (u *User) UpdatedAt() time.Time       { return u.updatedAt }
func (u *User) LastSeenAt() *time.Time     { return u.lastSeenAt }
func (u *User) DeletedAt() *time.Time      { return u.deletedAt }
func (u *User) Version() int               { return u.version }
func (u *User) IsDeleted() bool            { return u.deletedAt != nil }

// ─── Business methods ────────────────────────────────────────

// UpdateProfile — обновление профиля с валидацией.
func (u *User) UpdateProfile(displayName string, bio, avatarURL *string) error {
	if displayName == "" {
		return ErrDisplayNameRequired
	}
	if len(displayName) > 64 {
		return ErrDisplayNameTooLong
	}
	if bio != nil && len(*bio) > 500 {
		return ErrBioTooLong
	}

	u.profile.DisplayName = displayName
	u.profile.Bio = bio
	u.profile.AvatarURL = avatarURL
	u.touch()
	return nil
}

// UpdateSettings — замена settings целиком с валидацией privacy levels.
func (u *User) UpdateSettings(s Settings) error {
	if !s.Privacy.WhoCanMessage.IsValid() ||
		!s.Privacy.WhoCanSeeLastSeen.IsValid() ||
		!s.Privacy.WhoCanSeeProfile.IsValid() {
		return ErrInvalidPrivacyLevel
	}
	u.settings = s
	u.touch()
	return nil
}

// VerifyEmail — пометить email подтверждённым.
func (u *User) VerifyEmail() {
	u.verification.EmailVerified = true
	u.touch()
}

// VerifyPhone — пометить телефон подтверждённым.
func (u *User) VerifyPhone() {
	u.verification.PhoneVerified = true
	u.touch()
}

// UpdateLastSeen — обновление "последнего визита".
// НЕ бампит version: last_seen обновляется очень часто,
// не считаем это существенным изменением для optimistic lock.
func (u *User) UpdateLastSeen() {
	now := time.Now().UTC()
	u.lastSeenAt = &now
}

// SoftDelete — пометка удалённым. Hard delete запрещён.
func (u *User) SoftDelete() error {
	if u.IsDeleted() {
		return ErrAlreadyDeleted
	}
	now := time.Now().UTC()
	u.deletedAt = &now
	u.isActive = false
	u.touch()
	return nil
}

// ChangeEmail — смена email. После смены email считается неподтверждённым.
func (u *User) ChangeEmail(email vo.Email) {
	u.email = &email
	u.verification.EmailVerified = false
	u.touch()
}

// ChangePhone — смена телефона. После смены телефон считается неподтверждённым.
func (u *User) ChangePhone(phone vo.Phone) {
	u.phone = &phone
	u.verification.PhoneVerified = false
	u.touch()
}

// touch — внутренний helper: меняем updated_at + bump version.
func (u *User) touch() {
	u.updatedAt = time.Now().UTC()
	u.version++
}

// ─── Defaults ────────────────────────────────────────────────

func defaultSettings() Settings {
	return Settings{
		Privacy: PrivacySettings{
			WhoCanMessage:     PrivacyEveryone,
			WhoCanSeeLastSeen: PrivacyEveryone,
			WhoCanSeeProfile:  PrivacyEveryone,
		},
		Language: "en",
		Timezone: "UTC",
	}
}
