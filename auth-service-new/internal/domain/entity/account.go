package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Account represents a user account entity
type Account struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

// NewAccount creates a new Account entity
func NewAccount(email, passwordHash string) (*Account, error) {
	now := time.Now().UTC()
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("account NewAccount: error create uuid: %w", err)
	}
	return &Account{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// UpdateLastLogin sets the last login time
func (a *Account) UpdateLastLogin() {
	now := time.Now().UTC()
	a.LastLoginAt = &now
	a.UpdatedAt = now
}
