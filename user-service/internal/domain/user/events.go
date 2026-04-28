package user

import (
	"time"

	"github.com/google/uuid"
)

// Event — интерфейс доменного события.
type Event interface {
	EventName() string
	OccurredAt() time.Time
	AggregateID() uuid.UUID
}

// ─── UserCreated ─────────────────────────────────────────────

type UserCreated struct {
	UserID      uuid.UUID
	Username    string
	Email       *string
	Phone       *string
	DisplayName string
	At          time.Time
}

func NewUserCreated(u *User) UserCreated {
	var email, phone *string
	if u.Email() != nil {
		s := u.Email().String()
		email = &s
	}
	if u.Phone() != nil {
		s := u.Phone().String()
		phone = &s
	}
	return UserCreated{
		UserID:      u.ID(),
		Username:    u.Username().String(),
		Email:       email,
		Phone:       phone,
		DisplayName: u.Profile().DisplayName,
		At:          u.CreatedAt(),
	}
}

func (e UserCreated) EventName() string      { return "user.created" }
func (e UserCreated) OccurredAt() time.Time  { return e.At }
func (e UserCreated) AggregateID() uuid.UUID { return e.UserID }

// ─── UserProfileUpdated ──────────────────────────────────────

type UserProfileUpdated struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	At          time.Time
}

func NewUserProfileUpdated(u *User) UserProfileUpdated {
	return UserProfileUpdated{
		UserID:      u.ID(),
		DisplayName: u.Profile().DisplayName,
		Bio:         u.Profile().Bio,
		AvatarURL:   u.Profile().AvatarURL,
		At:          u.UpdatedAt(),
	}
}

func (e UserProfileUpdated) EventName() string      { return "user.profile.updated" }
func (e UserProfileUpdated) OccurredAt() time.Time  { return e.At }
func (e UserProfileUpdated) AggregateID() uuid.UUID { return e.UserID }

// ─── UserDeleted ─────────────────────────────────────────────

type UserDeleted struct {
	UserID uuid.UUID
	At     time.Time
}

func NewUserDeleted(u *User) UserDeleted {
	return UserDeleted{
		UserID: u.ID(),
		At:     *u.DeletedAt(),
	}
}

func (e UserDeleted) EventName() string      { return "user.deleted" }
func (e UserDeleted) OccurredAt() time.Time  { return e.At }
func (e UserDeleted) AggregateID() uuid.UUID { return e.UserID }
