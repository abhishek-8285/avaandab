package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/middleware"
	"transport-app/internal/repository"
)

// AuditLogHandlers handles audit log views.
type AuditLogHandlers struct {
	*App
}

func (h *AuditLogHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "audit_logs", "read")).Get("/", h.List)
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

	h.renderPage(w, "audit_logs_list.html", PageData{
		Title: "Audit Logs",
		User:  session,
		Extra: map[string]interface{}{
			"AuditLogs":  logs,
			"Pagination": pd,
		},
	})
}
