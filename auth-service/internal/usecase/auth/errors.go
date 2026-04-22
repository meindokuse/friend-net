package usecase

import (
	"errors"

	pg "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/postgresql"
	rd "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/redis"
)

var (
	// Credentials
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserAlreadyExists  = errors.New("user already exists")

	// Session
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExpired      = errors.New("session expired")
	ErrSessionRevoked      = errors.New("session revoked")
	ErrMaxSessionsReached  = errors.New("max sessions reached")

	// Token
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrTokenReuse          = errors.New("refresh token reuse detected")
	ErrRefreshNotFound     = errors.New("refresh token not found")

	// Fingerprint
	ErrFingerprintMismatch = errors.New("fingerprint mismatch")

	// Internal
	ErrInternal            = errors.New("internal error")
)

func mapPostgresError(err error) error {
	switch {
	case errors.Is(err, pg.ErrNoRows):
		return ErrInvalidCredentials
	case errors.Is(err, pg.ErrUserAlreadyExists):
		return ErrUserAlreadyExists
	case errors.Is(err, pg.ErrOAuthAccountExists):
		return ErrUserAlreadyExists
	case errors.Is(err, pg.ErrUnavailable):
		return ErrInternal
	case errors.Is(err, pg.ErrDatabaseRead):
		return ErrInternal
	case errors.Is(err, pg.ErrDatabaseWrite):
		return ErrInternal
	default:
		return ErrInternal
	}
}

func mapRedisError(err error) error {
	switch {
	case errors.Is(err, rd.ErrSessionNotFound):
		return ErrSessionNotFound
	case errors.Is(err, rd.ErrRefreshNotFound):
		return ErrRefreshNotFound
	case errors.Is(err, rd.ErrRedisUnavailable):
		return ErrInternal
	case errors.Is(err, rd.ErrRedisRead):
		return ErrInternal
	case errors.Is(err, rd.ErrRedisWrite):
		return ErrInternal
	default:
		return ErrInternal
	}
}