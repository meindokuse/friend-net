package user

import "errors"

// Ошибки входных данных usecase.
// Доменные ошибки (ErrUserNotFound и т.п.) берём из domain/user.
var (
	ErrInvalidInput = errors.New("usecase: invalid input")
)