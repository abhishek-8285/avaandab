package module

import (
	"fmt"
	"net/http"
)

// AppError is a unified error type that maps to HTTP responses.
// Services return it to signal structured errors that handlers can translate to the correct status code.
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// ErrorCode categorizes errors for HTTP response mapping.
type ErrorCode int

const (
	CodeValidation ErrorCode = iota
	CodeNotFound
	CodeConflict
	CodeUnauthorized
	CodeForbidden
	CodeInternal
)

// New creates an AppError with the given code and a formatted message.
func New(code ErrorCode, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error into an AppError.
func Wrap(code ErrorCode, err error, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap allows errors.Is and errors.As to traverse the chain.
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus maps the error code to an HTTP status code.
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case CodeValidation:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// WriteError writes an AppError (or any error) as an HTTP response.
func WriteError(w http.ResponseWriter, err error) {
	if ae, ok := err.(*AppError); ok {
		http.Error(w, ae.Message, ae.HTTPStatus())
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
