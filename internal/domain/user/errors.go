package user

import "errors"

// Auth errors
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserEmailExists    = errors.New("user with this email already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session has expired")
)
