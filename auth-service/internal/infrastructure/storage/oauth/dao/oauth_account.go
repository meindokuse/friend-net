package dao

import (
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// OAuthAccount is the database representation of entity.OAuthAccount.
type OAuthAccount struct {
	ID           uuid.UUID `db:"id"`
	AccountID    string    `db:"account_id"`
	Provider     string    `db:"provider"`
	ProviderID   string    `db:"provider_id"`
	Email        string    `db:"email"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	Expiry       time.Time `db:"expiry"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// ToEntity converts the DAO to a domain entity.
func (d *OAuthAccount) ToEntity() *entity.OAuthAccount {
	return &entity.OAuthAccount{
		ID:           d.ID.String(),
		AccountID:    d.AccountID,
		Provider:     entity.OAuthProvider(d.Provider),
		ProviderID:   d.ProviderID,
		Email:        d.Email,
		AccessToken:  d.AccessToken,
		RefreshToken: d.RefreshToken,
		Expiry:       d.Expiry,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
