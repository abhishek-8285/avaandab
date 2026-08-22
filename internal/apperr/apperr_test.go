package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNewKnownCodeClonesDefinition(t *testing.T) {
	a := New(CodeValidation)
	b := New(CodeValidation)
	if a == b {
		t.Fatal("expected distinct instances per New call")
	}
	if a.Code != CodeValidation || a.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected definition: %+v", a)
	}
	if a.Title == "" || a.UserMsg == "" {
		t.Fatal("title and user message must be populated from registry")
	}
}

func TestNewUnknownCodeFallsBackToInternal(t *testing.T) {
	e := New("TOTALLY_UNKNOWN")
	if e.Code != "TOTALLY_UNKNOWN" {
		t.Fatalf("code should be preserved, got %q", e.Code)
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("unknown code should map to 500, got %d", e.HTTPStatus)
	}
}

func TestErrorStringVariants(t *testing.T) {
	plain := New(CodeNotFound)
	if got, want := plain.Error(), CodeNotFound; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	withDetail := New(CodeNotFound).WithDetail("booking abc")
	if got, want := withDetail.Error(), "NOT_FOUND: booking abc"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	cause := errors.New("db: connection refused")
	wrapped := Wrap(CodeBookingNotFound, cause)
	if got, want := wrapped.Error(), "BOOKING_NOT_FOUND: db: connection refused"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestUnwrapChainAndFrom(t *testing.T) {
	root := fmt.Errorf("select failed: %w", errors.New("no such table"))
	wrapped := Wrap(CodeBookingNotFound, root)
	triple := fmt.Errorf("handler: %w", wrapped)

	got, ok := From(triple)
	if !ok {
		t.Fatal("From should find AppError through wrap chain")
	}
	if got.Code != CodeBookingNotFound {
		t.Fatalf("unexpected code %q", got.Code)
	}
	if !errors.Is(triple, root) {
		t.Fatal("errors.Is must still traverse to root cause")
	}
	if _, ok := From(errors.New("plain")); ok {
		t.Fatal("plain errors should not match")
	}
}
