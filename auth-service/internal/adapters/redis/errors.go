package redis

import "errors"

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrRedisWrite      = errors.New("redis write error")
	ErrRefreshNotFound = errors.New("session: refresh pair not found")
	ErrRedisRead       = errors.New("redis read error")
	ErrRedisUnavailable = errors.New("redis unavailable")
)


