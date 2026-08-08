package dashboard

import (
	"encoding/json"
	"net/http"
	"time"

	"transport-app/internal/operations/audit"
	"transport-app/internal/operations/errors"
)

type DashboardHandler struct {
	reporter   *errors.Reporter
	loginAudit *audit.LoginAuditService
}

func NewDashboardHandler(reporter *errors.Reporter, loginAudit *audit.LoginAuditService) *DashboardHandler {
	return &DashboardHandler{
		reporter:   reporter,
		loginAudit: loginAudit,
	}
}

type SummaryResponse struct {
	TotalErrors     int                  `json:"total_errors"`
	ActiveIncidents int                  `json:"active_incidents"`
	RecentErrors    []errors.ErrorReport `json:"recent_errors"`
	RecentIncidents []errors.Incident    `json:"recent_incidents"`
	Timestamp       time.Time            `json:"timestamp"`
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	errs := h.reporter.ListErrors()
	incidents := h.reporter.ListIncidents()

	openIncidents := 0
	for _, inc := range incidents {
		if inc.Status == "OPEN" {
			openIncidents++
		}
	}

	summary := SummaryResponse{
		TotalErrors:     len(errs),
		ActiveIncidents: openIncidents,
		RecentErrors:    errs,
		RecentIncidents: incidents,
		Timestamp:       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}
