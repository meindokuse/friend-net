package vo

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrUsernameEmpty     = errors.New("username: empty")
	ErrUsernameTooShort  = errors.New("username: too short (min 3)")
	ErrUsernameTooLong   = errors.New("username: too long (max 32)")
	ErrUsernameInvalid   = errors.New("username: only letters, digits and underscore allowed")
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Username — value object. Case-insensitive (нормализуется в lowercase при создании).
type Username struct {
	value string
}

func NewUsername(raw string) (Username, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Username{}, ErrUsernameEmpty
	}
	if len(raw) < 3 {
		return Username{}, ErrUsernameTooShort
	}
	if len(raw) > 32 {
		return Username{}, ErrUsernameTooLong
	}
	if !usernameRegex.MatchString(raw) {
		return Username{}, ErrUsernameInvalid
	}
	return Username{value: strings.ToLower(raw)}, nil
}

// MustNewUsername — для тестов и реконструкции из БД (где данные уже валидные).
func MustNewUsername(raw string) Username {
	u, err := NewUsername(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func (u Username) String() string { return u.value }
func (u Username) Equals(other Username) bool { return u.value == other.value }