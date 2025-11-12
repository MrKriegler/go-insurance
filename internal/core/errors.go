package core

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidState = errors.New("invalid state transition")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden operation")
)
