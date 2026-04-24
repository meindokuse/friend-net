package vo

import (
	"errors"
	"net/mail"
	"strings"
)

var (
	ErrEmailEmpty   = errors.New("email: empty")
	ErrEmailInvalid = errors.New("email: invalid format")
	ErrEmailTooLong = errors.New("email: too long (max 254)")
)

type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return Email{}, ErrEmailEmpty
	}
	if len(raw) > 254 {
		return Email{}, ErrEmailTooLong
	}
	if _, err := mail.ParseAddress(raw); err != nil {
		return Email{}, ErrEmailInvalid
	}
	return Email{value: raw}, nil
}

func MustNewEmail(raw string) Email {
	e, err := NewEmail(raw)
	if err != nil {
		panic(err)
	}
	return e
}

func (e Email) String() string { return e.value }
func (e Email) Equals(other Email) bool { return e.value == other.value }