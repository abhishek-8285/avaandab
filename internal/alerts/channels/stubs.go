package channels

import (
	"context"
	"log/slog"
)

// StubProvider is a log-only channel adapter for email/sms/whatsapp.
type StubProvider struct {
	name   string
	logger *slog.Logger
}

// NewStubProvider creates a new StubProvider for the given channel name.
func NewStubProvider(name string, logger *slog.Logger) *StubProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &StubProvider{name: name, logger: logger}
}

func (s *StubProvider) Name() string { return s.name }

func (s *StubProvider) Send(ctx context.Context, msg Message) error {
	s.logger.Info("stub alert channel message",
		"channel", s.name,
		"alert_id", msg.AlertID,
		"title", msg.Title,
		"severity", msg.Severity,
	)
	return nil
}

// NewStubProviders returns a map of stub providers for email, whatsapp, and sms.
func NewStubProviders(logger *slog.Logger) map[string]Provider {
	return map[string]Provider{
		"email":    NewStubProvider("email", logger),
		"whatsapp": NewStubProvider("whatsapp", logger),
		"sms":      NewStubProvider("sms", logger),
	}
}
