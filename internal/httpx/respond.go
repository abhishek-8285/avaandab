// Package httpx renders canonical HTTP error responses.
//
// Error(w, r, err) writes an RFC 7807-style problem+json envelope for any
// error. *apperr.AppError values map to their registered status, code and
// user message; anything else becomes a generic 500 whose real cause is
// logged server-side with request correlation fields but never sent to the
// client.
package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"transport-app/internal/apperr"
	"transport-app/internal/auth"
	domainuser "transport-app/internal/domain/user"
	"transport-app/internal/logging"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Problem struct {
	Type      string       `json:"type"`
	Code      string       `json:"code"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Message   string       `json:"message"`
	Detail    string       `json:"detail,omitempty"`
	Instance  string       `json:"instance,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
	Timestamp string       `json:"timestamp"`
	FieldErrs []FieldError `json:"errors,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func FieldProblem(w http.ResponseWriter, r *http.Request, code string, fieldErrs []FieldError) {
	ae := apperr.New(code)
	p := problemFromAppError(r, ae)
	p.FieldErrs = fieldErrs
	writeProblem(w, p)
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	ae, ok := apperr.From(err)
	if !ok {
		reqID := RequestID(r.Context())
		slog.ErrorContext(r.Context(), "request failed",
			slog.String("request_id", reqID),
			slog.String("user_id", userID(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", logging.Redact(err.Error())),
		)
		ae = apperr.New(apperr.CodeInternal).WithCause(err)
	}
	writeProblem(w, problemFromAppError(r, ae))
}

func writeProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func problemFromAppError(r *http.Request, ae *apperr.AppError) Problem {
	detail := ae.Detail
	if ae.HTTPStatus >= 500 {
		detail = ""
	}
	return Problem{
		Type:      "/errors/" + slug(ae.Code),
		Code:      ae.Code,
		Title:     ae.Title,
		Status:    ae.HTTPStatus,
		Message:   ae.UserMsg,
		Detail:    detail,
		Instance:  r.URL.Path,
		RequestID: RequestID(r.Context()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func slug(code string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "_", "-"))
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(auth.ContextReqID).(string)
	return id
}

func userID(ctx context.Context) string {
	switch v := ctx.Value(auth.ContextUser).(type) {
	case *auth.SessionData:
		if v != nil {
			return v.UserID
		}
	case auth.SessionData:
		return v.UserID
	case domainuser.User:
		return string(v.ID)
	case *domainuser.User:
		if v != nil {
			return string(v.ID)
		}
	}
	return ""
}
