package dao

import (
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// Account is the database representation of entity.Account.
type Account struct {
	ID           uuid.UUID  `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	IsActive     bool       `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	LastLoginAt  *time.Time `db:"last_login_at"`
}

// ToEntity converts the DAO to a domain entity.
func (d *Account) ToEntity() *entity.Account {
	return &entity.Account{
		ID:           d.ID,
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		IsActive:     d.IsActive,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
		LastLoginAt:  d.LastLoginAt,
	}
}
