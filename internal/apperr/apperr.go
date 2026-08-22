// Package apperr provides the application-wide typed error model.
//
// Every error surfaced to an HTTP client is an *AppError carrying a stable
// machine-readable Code, an HTTP status, a short Title, a jargon-free
// UserMsg (safe to render verbatim), and optional Detail. Internal causes
// ride along via Unwrap and never reach the client for 5xx responses
// (enforced by internal/httpx).
package apperr

import (
	"errors"
	"fmt"
)

type AppError struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"-"`
	Title      string `json:"title"`
	UserMsg    string `json:"user_message"`
	Detail     string `json:"detail,omitempty"`
	cause      error
}

func (e *AppError) Error() string {
	switch {
	case e.cause != nil:
		return fmt.Sprintf("%s: %v", e.Code, e.cause)
	case e.Detail != "":
		return fmt.Sprintf("%s: %s", e.Code, e.Detail)
	default:
		return e.Code
	}
}

func (e *AppError) Unwrap() error { return e.cause }

func (e *AppError) WithCause(cause error) *AppError {
	e.cause = cause
	return e
}

func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

func New(code string) *AppError {
	if def, ok := registry[code]; ok {
		clone := *def
		return &clone
	}
	fallback := *registry[CodeInternal]
	fallback.Code = code
	return &fallback
}

func Wrap(code string, cause error) *AppError {
	return New(code).WithCause(cause)
}

func From(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
