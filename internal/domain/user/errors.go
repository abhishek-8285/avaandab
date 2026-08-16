package user

import "errors"

// Auth errors
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserEmailExists    = errors.New("user with this email already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session has expired")
	ErrUserEmailRequired  = errors.New("email is required")
	ErrUserPhoneRequired  = errors.New("phone number is required")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
)

// MinPasswordLength is the minimum accepted password length.
const MinPasswordLength = 8
