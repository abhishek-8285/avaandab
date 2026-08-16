package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// ErrWeakPassword is returned when a password fails the complexity policy.
var ErrWeakPassword = errors.New("password must be at least 8 characters and include at least one letter and one digit")

// ValidatePassword enforces the password policy: minimum length of 8 and a
// mix of letters and digits. Register and change-password flows should call
// this before hashing.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return ErrWeakPassword
	}
	return nil
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
