package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

func NewAccount(email, passwordHash string) (*Account,error) {
	now := time.Now().UTC()
	id,err := uuid.NewUUID()
	if err != nil {
		return nil,fmt.Errorf("account NewAccount: error create uuid: %w",err)
	} 
	return &Account{
		ID:			  id,
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	},nil
}

type Login struct {
	Email string
}

type Register struct {
	Email    string
	Password string
	DisplayName string
}
