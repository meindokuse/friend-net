package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/value_object"
)

type User struct {
	ID          uuid.UUID  `bson:"_id"`
	Username    string     `bson:"username"`
	Email       *string    `bson:"email,omitempty"`
	Phone       *string    `bson:"phone,omitempty"`
	DisplayName string     `bson:"display_name"`
	Bio         *string    `bson:"bio,omitempty"`
	AvatarURL   *string    `bson:"avatar_url,omitempty"`
	WhoCanMsg   string     `bson:"who_can_message"`
	WhoCanSeen  string     `bson:"who_can_see_last_seen"`
	WhoCanProf  string     `bson:"who_can_see_profile"`
	Language    string     `bson:"language"`
	Timezone    string     `bson:"timezone"`
	EmailVer    bool       `bson:"email_verified"`
	PhoneVer    bool       `bson:"phone_verified"`
	IsActive    bool       `bson:"is_active"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	LastSeenAt  *time.Time `bson:"last_seen_at,omitempty"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty"`
	Version     int        `bson:"version"`
}

func FromEntity(u *entity.User) *User {
	var email, phone *string
	if u.Email() != nil {
		s := u.Email().String()
		email = &s
	}
	if u.Phone() != nil {
		s := u.Phone().String()
		phone = &s
	}
	return &User{
		ID:          u.ID(),
		Username:    u.Username().String(),
		Email:       email,
		Phone:       phone,
		DisplayName: u.Profile().DisplayName,
		Bio:         u.Profile().Bio,
		AvatarURL:   u.Profile().AvatarURL,
		WhoCanMsg:   string(u.Settings().Privacy.WhoCanMessage),
		WhoCanSeen:  string(u.Settings().Privacy.WhoCanSeeLastSeen),
		WhoCanProf:  string(u.Settings().Privacy.WhoCanSeeProfile),
		Language:    u.Settings().Language,
		Timezone:    u.Settings().Timezone,
		EmailVer:    u.Verification().EmailVerified,
		PhoneVer:    u.Verification().PhoneVerified,
		IsActive:    u.IsActive(),
		CreatedAt:   u.CreatedAt(),
		UpdatedAt:   u.UpdatedAt(),
		LastSeenAt:  u.LastSeenAt(),
		DeletedAt:   u.DeletedAt(),
		Version:     u.Version(),
	}
}

func (d *User) ConvertTo() *entity.User {
	username := vo.MustNewUsername(d.Username)
	var email *vo.Email
	if d.Email != nil {
		e := vo.MustNewEmail(*d.Email)
		email = &e
	}
	var phone *vo.Phone
	if d.Phone != nil {
		p := vo.MustNewPhone(*d.Phone)
		phone = &p
	}
	return entity.Reconstruct(
		d.ID, username, email, phone,
		entity.Profile{DisplayName: d.DisplayName, Bio: d.Bio, AvatarURL: d.AvatarURL},
		entity.Settings{
			Privacy: entity.PrivacySettings{
				WhoCanMessage:     entity.PrivacyLevel(d.WhoCanMsg),
				WhoCanSeeLastSeen: entity.PrivacyLevel(d.WhoCanSeen),
				WhoCanSeeProfile:  entity.PrivacyLevel(d.WhoCanProf),
			},
			Language: d.Language,
			Timezone: d.Timezone,
		},
		entity.Verification{EmailVerified: d.EmailVer, PhoneVerified: d.PhoneVer},
		d.IsActive, d.CreatedAt, d.UpdatedAt, d.LastSeenAt, d.DeletedAt, d.Version,
	)
}
