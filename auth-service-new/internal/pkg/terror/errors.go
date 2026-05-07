package terror

import (
	"errors"
	"fmt"
)

// Common error types
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrBadRequest   = errors.New("bad request")
	ErrInternal     = errors.New("internal error")
	ErrForbidden    = errors.New("forbidden")
)

// Error represents a structured error with type and context
type Error struct {
	Type    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// NewNotFoundErr creates a not found error
func NewNotFoundErr(message string, cause error) error {
	return &Error{
		Type:    "NOT_FOUND",
		Message: message,
		Cause:   cause,
	}
}

// NewConflictErr creates a conflict error
func NewConflictErr(message string, cause error) error {
	return &Error{
		Type:    "CONFLICT",
		Message: message,
		Cause:   cause,
	}
}

// NewUnauthorizedErr creates an unauthorized error
func NewUnauthorizedErr(message string, cause error) error {
	return &Error{
		Type:    "UNAUTHORIZED",
		Message: message,
		Cause:   cause,
	}
}

// NewBadRequestErr creates a bad request error
func NewBadRequestErr(message string, cause error) error {
	return &Error{
		Type:    "BAD_REQUEST",
		Message: message,
		Cause:   cause,
	}
}

// NewInternalErr creates an internal error
func NewInternalErr(message string, cause error) error {
	return &Error{
		Type:    "INTERNAL",
		Message: message,
		Cause:   cause,
	}
}

// NewForbiddenErr creates a forbidden error
func NewForbiddenErr(message string, cause error) error {
	return &Error{
		Type:    "FORBIDDEN",
		Message: message,
		Cause:   cause,
	}
}

// IsNotFound checks if error is not found
func IsNotFound(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Type == "NOT_FOUND"
	}
	return false
}

// IsConflict checks if error is conflict
func IsConflict(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Type == "CONFLICT"
	}
	return false
}

// IsUnauthorized checks if error is unauthorized
func IsUnauthorized(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Type == "UNAUTHORIZED"
	}
	return false
}
