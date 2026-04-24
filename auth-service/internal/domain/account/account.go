package domain

import (
	"time"
)

type Account struct {
	ID           string
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

func NewAccount(email, passwordHash string) *Account {
	now := time.Now().UTC()
	return &Account{
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

type Login struct {
	Email string
}

type Register struct {
	Email    string
	Password string
}
