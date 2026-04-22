package postgresql

import "errors"

var (
	ErrNoRows             = errors.New("no rows found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrOAuthAccountExists = errors.New("oauth account already exists")
	ErrDatabaseRead       = errors.New("database read error")
	ErrDatabaseWrite      = errors.New("database write error")
	ErrUnavailable        = errors.New("database unavailable")
)
