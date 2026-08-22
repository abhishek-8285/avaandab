package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"transport-app/internal/apperr"
	"transport-app/internal/auth"
)

func ctxWithRequestID(id string) (ctx context.Context) {
	return context.WithValue(context.Background(), auth.ContextReqID, id)
}

func TestErrorRendersAppErrorEnvelope(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/kharcha", nil)
	r = r.WithContext(ctxWithRequestID("req-123"))
	w := httptest.NewRecorder()

	Error(w, r, apperr.Wrap(apperr.CodeKharchaApprovalDenied, errors.New("role=driver")))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("content-type = %q", ct)
	}

	var p Problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != apperr.CodeKharchaApprovalDenied || p.Status != 403 {
		t.Fatalf("unexpected envelope: %+v", p)
	}
	if p.RequestID != "req-123" {
		t.Fatalf("request_id = %q, want req-123", p.RequestID)
	}
	if p.Instance != "/api/v1/kharcha" {
		t.Fatalf("instance = %q", p.Instance)
	}
	if p.Message == "" || p.Title == "" || p.Type == "" {
		t.Fatalf("title/message/type must be set: %+v", p)
	}
	if strings.Contains(w.Body.String(), "role=driver") {
		t.Fatal("internal cause must never leak for client errors either when Detail unset")
	}
}

func TestErrorMasksUnknownErrorsAs500(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	r = r.WithContext(ctxWithRequestID("req-9"))
	w := httptest.NewRecorder()

	secret := "sql: SELECT password_hash FROM users"
	Error(w, r, errors.New(secret))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, secret) {
		t.Fatal("raw error leaked to client")
	}
	var p Problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != apperr.CodeInternal || p.Detail != "" {
		t.Fatalf("500 must be generic: %+v", p)
	}
	if p.RequestID != "req-9" {
		t.Fatalf("request_id = %q", p.RequestID)
	}
}

func TestErrorStripsDetailOn5xxAppError(t *testing.T) {
	r := httptest.NewRequest("GET", "/x", nil)
	r = r.WithContext(ctxWithRequestID("r1"))
	w := httptest.NewRecorder()

	ae := apperr.New(apperr.CodeServiceUnavailable).WithDetail("redis at 10.0.0.5 refused")
	Error(w, r, ae)

	var p Problem
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	if p.Detail != "" {
		t.Fatalf("detail must be stripped on 5xx, got %q", p.Detail)
	}
}

func TestFieldProblemIncludesFieldErrors(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	FieldProblem(w, r, apperr.CodeValidation, []FieldError{
		{Field: "email", Message: "Must be a valid email address."},
	})

	var p Problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.FieldErrs) != 1 || p.FieldErrs[0].Field != "email" {
		t.Fatalf("field errors missing: %+v", p)
	}
	if p.Status != 400 {
		t.Fatalf("status = %d", p.Status)
	}
}

func TestJSONHelper(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, 200, map[string]string{"ok": "true"})
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("ct = %q", ct)
	}
}
