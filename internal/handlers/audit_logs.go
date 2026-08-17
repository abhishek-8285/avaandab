package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
	"transport-app/internal/repository"
)

// AuditLogHandlers handles audit log views.
type AuditLogHandlers struct {
	*App
}

func (h *AuditLogHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "audit_logs", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "audit_logs", "read")).Post("/read-all", h.MarkAllRead)
}

func (h *AuditLogHandlers) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getUserFromContext(r)
	if ok && session != nil {
		uid := domain.UserID(session.UserID)
		_ = h.Services.Audit.LogAction(r.Context(), &uid, "mark_notifications_read", "notifications", string(uid), nil, nil)
	}
	// Set a cookie so the next page render knows all notifications are read
	http.SetCookie(w, &http.Cookie{
		Name:     "notif_read_at",
		Value:    "1",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30, // 30 days
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

func (h *AuditLogHandlers) List(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	logs, total, err := h.Services.Audit.ListAuditLogs(r.Context(), pp.Limit, pp.Offset)
	if err != nil {
		logs = []repository.AuditLogWithUser{}
		total = 0
	}

	pd := newPaginationData(pp, total, "/audit-logs")

	h.renderPage(w, r, "audit_logs_list.html", PageData{
		Title: "Audit Logs",
		User:  session,
		Extra: map[string]interface{}{
			"AuditLogs":  logs,
			"Pagination": pd,
			"Query":      pp.Query,
		},
	})
}
