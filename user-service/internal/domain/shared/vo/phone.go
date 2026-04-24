package vo

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrPhoneEmpty   = errors.New("phone: empty")
	ErrPhoneInvalid = errors.New("phone: must be in E.164 format (e.g. +79991234567)")
)

// E.164: "+" + до 15 цифр
var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type Phone struct {
	value string
}

func NewPhone(raw string) (Phone, error) {
	raw = strings.TrimSpace(raw)
	// Убираем возможные разделители
	raw = strings.ReplaceAll(raw, " ", "")
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, "(", "")
	raw = strings.ReplaceAll(raw, ")", "")

	if raw == "" {
		return Phone{}, ErrPhoneEmpty
	}
	if !phoneRegex.MatchString(raw) {
		return Phone{}, ErrPhoneInvalid
	}
	return Phone{value: raw}, nil
}

func MustNewPhone(raw string) Phone {
	p, err := NewPhone(raw)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Phone) String() string { return p.value }
func (p Phone) Equals(other Phone) bool { return p.value == other.value }