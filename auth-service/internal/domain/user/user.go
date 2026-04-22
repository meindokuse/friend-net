package domain

import (
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	MFAEnabled   bool
	MFASecret    string
	CreatedAt    time.Time
}

func NewUser(email, passwordHash string) *User {
	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
}

type Login struct {
	Email        string
	PasswordHash string
}

type Register struct {
	Email    string
	Password string
}
