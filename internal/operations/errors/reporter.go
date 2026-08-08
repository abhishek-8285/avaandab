package errors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"transport-app/internal/shared/ports"
)

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
)

type ErrorReport struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id"`
	UserID      string                 `json:"user_id"`
	TenantID    string                 `json:"tenant_id"`
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	StatusCode  int                    `json:"status_code"`
	StackTrace  string                 `json:"stack_trace"`
	Message     string                 `json:"message"`
	Severity    Severity               `json:"severity"`
	Environment string                 `json:"environment"`
	AppVersion  string                 `json:"app_version"`
	UserAgent   string                 `json:"user_agent"`
	IPAddress   string                 `json:"ip_address"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Incident struct {
	ID         string     `json:"id"`
	ErrorID    string     `json:"error_id"`
	Status     string     `json:"status"` // OPEN, ASSIGNED, RESOLVED
	Severity   Severity   `json:"severity"`
	Created    time.Time  `json:"created"`
	AssignedTo string     `json:"assigned_to"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	RootCause  string     `json:"root_cause,omitempty"`
}

type Reporter struct {
	mu          sync.RWMutex
	errors      []ErrorReport
	incidents   []Incident
	notifSvc    ports.NotificationService
	environment string
	appVersion  string
}

func NewReporter(notifSvc ports.NotificationService, env, appVersion string) *Reporter {
	return &Reporter{
		errors:      make([]ErrorReport, 0),
		incidents:   make([]Incident, 0),
		notifSvc:    notifSvc,
		environment: env,
		appVersion:  appVersion,
	}
}

func (r *Reporter) Report(ctx context.Context, report ErrorReport) (ErrorReport, error) {
	r.mu.Lock()
	if report.ID == "" {
		report.ID = fmt.Sprintf("err_%d", time.Now().UnixNano())
	}
	if report.Timestamp.IsZero() {
		report.Timestamp = time.Now()
	}
	if report.Environment == "" {
		report.Environment = r.environment
	}
	if report.AppVersion == "" {
		report.AppVersion = r.appVersion
	}
	if report.Severity == "" {
		report.Severity = SeverityHigh
	}

	r.errors = append(r.errors, report)

	// Automatically create an incident for Critical/High errors
	if report.Severity == SeverityCritical || report.Severity == SeverityHigh {
		inc := Incident{
			ID:       fmt.Sprintf("inc_%d", time.Now().UnixNano()),
			ErrorID:  report.ID,
			Status:   "OPEN",
			Severity: report.Severity,
			Created:  time.Now(),
		}
		r.incidents = append(r.incidents, inc)
	}
	r.mu.Unlock()

	// Notify tech team on Critical severity
	if report.Severity == SeverityCritical && r.notifSvc != nil {
		_ = r.notifSvc.SendEmail(ctx, ports.NotificationMessage{
			Recipient: "tech-alerts@flyfleet.io",
			Subject:   fmt.Sprintf("[CRITICAL ALERT] %s - %s", report.Method, report.URL),
			Body:      fmt.Sprintf("Error ID: %s\nMessage: %s\nStack Trace:\n%s", report.ID, report.Message, report.StackTrace),
			Type:      ports.NotificationTypeEmail,
		})
	}

	return report, nil
}

func (r *Reporter) ListErrors() []ErrorReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]ErrorReport, len(r.errors))
	copy(copied, r.errors)
	return copied
}

func (r *Reporter) ListIncidents() []Incident {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]Incident, len(r.incidents))
	copy(copied, r.incidents)
	return copied
}
