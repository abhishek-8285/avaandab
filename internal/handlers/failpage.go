package handlers

import (
	"log/slog"
	"net/http"

	"transport-app/internal/apperr"
	"transport-app/internal/logging"
)

func (a *App) failPage(w http.ResponseWriter, r *http.Request, err error, fallbackStatus int, title string) {
	status := fallbackStatus
	msg := "Something went wrong. Please try again, or contact support if it keeps happening."
	if ae, ok := apperr.From(err); ok {
		status = ae.HTTPStatus
		msg = ae.UserMsg
	}
	slog.ErrorContext(r.Context(), "handler error",
		slog.String("title", title),
		slog.String("path", r.URL.Path),
		slog.String("error", logging.Redact(err.Error())),
	)
	session, _ := a.getUserFromContext(r)
	a.renderError(w, status, title, msg, session)
}
