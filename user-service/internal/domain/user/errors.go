package user

import "errors"

// ─── Ошибки бизнес-логики (валидация, инварианты) ────────────────────────────
// Бросаются самой доменной сущностью.

var (
	ErrEmailOrPhoneRequired = errors.New("user: email or phone is required")
	ErrDisplayNameRequired  = errors.New("user: display name is required")
	ErrDisplayNameTooLong   = errors.New("user: display name too long (max 64)")
	ErrBioTooLong           = errors.New("user: bio too long (max 500)")
	ErrInvalidPrivacyLevel  = errors.New("user: invalid privacy level")
	ErrAlreadyDeleted       = errors.New("user: already deleted")
)

// ─── Ошибки контракта репозитория ────────────────────────────────────────────
// Определены здесь как sentinel values — домен владеет контрактом.
// Бросаются реализацией репозитория (адаптером), обрабатываются usecase.
// Usecase импортирует только этот пакет — никакой зависимости от mongo/sql/etc.

var (
	ErrUserNotFound         = errors.New("user: not found")
	ErrUsernameAlreadyTaken = errors.New("user: username already taken")
	ErrEmailAlreadyTaken    = errors.New("user: email already taken")
	ErrPhoneAlreadyTaken    = errors.New("user: phone already taken")
	ErrVersionConflict      = errors.New("user: version conflict (optimistic lock)")
)
