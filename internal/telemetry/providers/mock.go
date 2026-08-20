package providers

import (
	"context"
	"net/http"
	"time"
)

// MockProvider returns canned frames for tests and local dev.
// Registered only when no real provider secrets are configured.
type MockProvider struct{}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) VerifySignature(rawBody []byte, header http.Header) error {
	return nil
}

func (m *MockProvider) HandleWebhook(ctx context.Context, rawBody []byte) ([]RawFrame, error) {
	return nil, nil // no-op for mock
}

func (m *MockProvider) Poll(ctx context.Context, since time.Time) ([]RawFrame, error) {
	return nil, nil // no-op for mock
}

var _ TelematicsProvider = (*MockProvider)(nil)
