package errors_test

import (
	"context"
	"testing"

	"transport-app/internal/operations/errors"
	"transport-app/internal/operations/notifications"
)

func TestErrorReporterAndIncidents(t *testing.T) {
	notifSvc := notifications.NewService()
	reporter := errors.NewReporter(notifSvc, "test", "v1.0.0")

	ctx := context.Background()
	report, err := reporter.Report(ctx, errors.ErrorReport{
		URL:        "/api/v1/test",
		Method:     "POST",
		Message:    "Database deadlock",
		Severity:   errors.SeverityCritical,
		StackTrace: "main.go:42",
	})

	if err != nil {
		t.Fatalf("expected no error reporting, got: %v", err)
	}

	if report.ID == "" {
		t.Fatalf("expected generated error ID")
	}

	errs := reporter.ListErrors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error logged, got %d", len(errs))
	}

	incidents := reporter.ListIncidents()
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident created for Critical error, got %d", len(incidents))
	}
}
