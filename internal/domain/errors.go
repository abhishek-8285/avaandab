package domain

import "errors"

// Shared domain errors not specific to any entity.
var (
	// ErrInvalidStatus is a general error for invalid state transitions.
	ErrInvalidStatus = errors.New("invalid status transition")
)
