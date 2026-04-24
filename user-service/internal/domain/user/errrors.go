package user

import "errors"

// Ошибки валидации при создании / изменении User.
var (
	ErrEmailOrPhoneRequired = errors.New("user: email or phone is required")
	ErrDisplayNameRequired  = errors.New("user: display name is required")
	ErrDisplayNameTooLong   = errors.New("user: display name too long (max 64)")
	ErrBioTooLong           = errors.New("user: bio too long (max 500)")
	ErrInvalidPrivacyLevel  = errors.New("user: invalid privacy level")
)

// Ошибки жизненного цикла.
var (
	ErrAlreadyDeleted = errors.New("user: already deleted")
)

// Ошибки, которые возвращает Repository (их знает домен, бросает БД-адаптер).
var (
	ErrUserNotFound         = errors.New("user: not found")
	ErrUsernameAlreadyTaken = errors.New("user: username already taken")
	ErrEmailAlreadyTaken    = errors.New("user: email already taken")
	ErrPhoneAlreadyTaken    = errors.New("user: phone already taken")
	ErrVersionConflict      = errors.New("user: version conflict (optimistic lock)")
)