package shared

import "errors"

// Общие ошибки, которые могут возникать в нескольких доменах.
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrConflict         = errors.New("conflict")
	ErrNotFound         = errors.New("not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrOptimisticLock   = errors.New("optimistic lock: record was modified")
)