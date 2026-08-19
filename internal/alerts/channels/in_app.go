package channels

import "context"

// InAppProvider handles notifications within the web application.
type InAppProvider struct{}

// NewInAppProvider creates a new InAppProvider.
func NewInAppProvider() *InAppProvider {
	return &InAppProvider{}
}

func (p *InAppProvider) Name() string { return "in_app" }

func (p *InAppProvider) Send(ctx context.Context, msg Message) error {
	// The DB row in alerts table is the source of truth for in_app notifications.
	// UI refresh / Datastar polling displays the badge and notification items.
	return nil
}
