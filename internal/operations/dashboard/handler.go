package dashboard

import (
	"encoding/json"
	"net/http"
	"time"

	"transport-app/internal/httpx"
	"transport-app/internal/operations/audit"
	"transport-app/internal/operations/errors"
	"transport-app/internal/shared"
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

type RedactedErrorReport struct {
	ID          string          `json:"id"`
	Timestamp   time.Time       `json:"timestamp"`
	RequestID   string          `json:"request_id"`
	URL         string          `json:"url"`
	Method      string          `json:"method"`
	StatusCode  int             `json:"status_code"`
	Message     string          `json:"message"`
	Severity    errors.Severity `json:"severity"`
	Environment string          `json:"environment"`
	AppVersion  string          `json:"app_version"`
}

type SummaryResponse struct {
	TotalErrors     int                   `json:"total_errors"`
	ActiveIncidents int                   `json:"active_incidents"`
	RecentErrors    []RedactedErrorReport `json:"recent_errors"`
	RecentIncidents []errors.Incident     `json:"recent_incidents"`
	Timestamp       time.Time             `json:"timestamp"`
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := string(shared.TenantIDFromContext(r.Context()))
	errs, err := h.reporter.ListErrors(r.Context(), errors.ErrorFilter{TenantID: tenantID, Limit: 50})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	incidents, err := h.reporter.ListIncidents(r.Context(), errors.IncidentFilter{TenantID: tenantID, Limit: 50})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	openIncidents := 0
	for _, inc := range incidents {
		if inc.Status == "OPEN" {
			openIncidents++
		}
	}

	redactedErrors := make([]RedactedErrorReport, len(errs))
	for i, e := range errs {
		redactedErrors[i] = RedactedErrorReport{
			ID:          e.ID,
			Timestamp:   e.Timestamp,
			RequestID:   e.RequestID,
			URL:         e.URL,
			Method:      e.Method,
			StatusCode:  e.StatusCode,
			Message:     e.Message,
			Severity:    e.Severity,
			Environment: e.Environment,
			AppVersion:  e.AppVersion,
		}
	}

	summary := SummaryResponse{
		TotalErrors:     len(errs),
		ActiveIncidents: openIncidents,
		RecentErrors:    redactedErrors,
		RecentIncidents: incidents,
		Timestamp:       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}
