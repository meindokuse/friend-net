package valueobject

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type Username struct{ value string }

func NewUsername(raw string) (Username, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return Username{}, errors.New("username: empty")
	case len(raw) < 3:
		return Username{}, errors.New("username: too short (min 3)")
	case len(raw) > 32:
		return Username{}, errors.New("username: too long (max 32)")
	case !usernameRegex.MatchString(raw):
		return Username{}, errors.New("username: only letters, digits and underscore allowed")
	}
	return Username{value: strings.ToLower(raw)}, nil
}

func MustNewUsername(raw string) Username { u, _ := NewUsername(raw); return u }
func (u Username) String() string         { return u.value }

type Email struct{ value string }

func NewEmail(raw string) (Email, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch {
	case raw == "":
		return Email{}, errors.New("email: empty")
	case len(raw) > 254:
		return Email{}, errors.New("email: too long (max 254)")
	}
	if _, err := mail.ParseAddress(raw); err != nil {
		return Email{}, errors.New("email: invalid format")
	}
	return Email{value: raw}, nil
}

func MustNewEmail(raw string) Email { e, _ := NewEmail(raw); return e }
func (e Email) String() string      { return e.value }

type Phone struct{ value string }

func NewPhone(raw string) (Phone, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, " ", "")
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, "(", "")
	raw = strings.ReplaceAll(raw, ")", "")
	if raw == "" {
		return Phone{}, errors.New("phone: empty")
	}
	if !phoneRegex.MatchString(raw) {
		return Phone{}, errors.New("phone: must be in E.164 format")
	}
	return Phone{value: raw}, nil
}

func MustNewPhone(raw string) Phone { p, _ := NewPhone(raw); return p }
func (p Phone) String() string      { return p.value }
