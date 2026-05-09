package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/value_object"
)

var (
	ErrEmailOrPhoneRequired = errors.New("user: email or phone is required")
	ErrDisplayNameRequired  = errors.New("user: display name is required")
	ErrDisplayNameTooLong   = errors.New("user: display name too long (max 64)")
	ErrBioTooLong           = errors.New("user: bio too long (max 500)")
	ErrInvalidPrivacyLevel  = errors.New("user: invalid privacy level")
	ErrAlreadyDeleted       = errors.New("user: already deleted")
	ErrUserNotFound         = errors.New("user: not found")
	ErrUsernameAlreadyTaken = errors.New("user: username already taken")
	ErrEmailAlreadyTaken    = errors.New("user: email already taken")
	ErrPhoneAlreadyTaken    = errors.New("user: phone already taken")
	ErrVersionConflict      = errors.New("user: version conflict (optimistic lock)")
)

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
	return p == PrivacyEveryone || p == PrivacyFriends || p == PrivacyNobody
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
	version      int
}

func NewUser(id uuid.UUID, username vo.Username, email *vo.Email, phone *vo.Phone, displayName string) (*User, error) {
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
		settings: Settings{
			Privacy: PrivacySettings{
				WhoCanMessage:     PrivacyEveryone,
				WhoCanSeeLastSeen: PrivacyEveryone,
				WhoCanSeeProfile:  PrivacyEveryone,
			},
			Language: "en",
			Timezone: "UTC",
		},
		isActive:  true,
		createdAt: now,
		updatedAt: now,
		version:   1,
	}, nil
}

func Reconstruct(id uuid.UUID, username vo.Username, email *vo.Email, phone *vo.Phone, profile Profile, settings Settings, verification Verification, isActive bool, createdAt, updatedAt time.Time, lastSeenAt, deletedAt *time.Time, version int) *User {
	return &User{id: id, username: username, email: email, phone: phone, profile: profile, settings: settings, verification: verification, isActive: isActive, createdAt: createdAt, updatedAt: updatedAt, lastSeenAt: lastSeenAt, deletedAt: deletedAt, version: version}
}

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
	u.profile.DisplayName, u.profile.Bio, u.profile.AvatarURL = displayName, bio, avatarURL
	u.touch()
	return nil
}
func (u *User) UpdateSettings(s Settings) error {
	if !s.Privacy.WhoCanMessage.IsValid() || !s.Privacy.WhoCanSeeLastSeen.IsValid() || !s.Privacy.WhoCanSeeProfile.IsValid() {
		return ErrInvalidPrivacyLevel
	}
	u.settings = s
	u.touch()
	return nil
}
func (u *User) UpdateLastSeen() { now := time.Now().UTC(); u.lastSeenAt = &now }
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
func (u *User) ChangeEmail(email vo.Email) {
	u.email = &email
	u.verification.EmailVerified = false
	u.touch()
}
func (u *User) ChangePhone(phone vo.Phone) {
	u.phone = &phone
	u.verification.PhoneVerified = false
	u.touch()
}
func (u *User) touch() { u.updatedAt = time.Now().UTC(); u.version++ }

type UsernameCursor struct {
	Username string
	ID       uuid.UUID
}
type ListParams struct {
	Limit  int
	Cursor *UsernameCursor
}
type PagedUsers struct {
	Items      []*User
	NextCursor UsernameCursor
	HasMore    bool
}
