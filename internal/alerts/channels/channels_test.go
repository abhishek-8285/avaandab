package channels

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/config"
)

func TestInAppProvider(t *testing.T) {
	p := NewInAppProvider()
	assert.Equal(t, "in_app", p.Name())

	msg := Message{
		AlertID:  "al-1",
		Title:    "Overspeeding",
		Body:     "Vehicle exceeded speed limit",
		Severity: "warning",
		UserID:   "u-1",
	}
	err := p.Send(context.Background(), msg)
	assert.NoError(t, err)
}

func TestTelegramProvider_GracefulFallback(t *testing.T) {
	cfg := &config.Config{
		Alerts: config.AlertConfig{
			TelegramBotToken: "", // unconfigured
			TelegramChatID:   "",
		},
	}
	p := NewTelegramProvider(cfg, nil)
	assert.Equal(t, "telegram", p.Name())

	msg := Message{
		AlertID:  "al-2",
		Title:    "GPS Deviation",
		Body:     "Vehicle off route by 6km",
		Severity: "critical",
		Meta: map[string]any{
			"entity_type": "vehicle",
			"entity_id":   "v-101",
		},
	}
	err := p.Send(context.Background(), msg)
	assert.NoError(t, err, "unconfigured telegram provider must not error")
}

func TestStubProviders(t *testing.T) {
	stubs := NewStubProviders(nil)
	require.Contains(t, stubs, "email")
	require.Contains(t, stubs, "sms")
	require.Contains(t, stubs, "whatsapp")

	for name, p := range stubs {
		assert.Equal(t, name, p.Name())
		err := p.Send(context.Background(), Message{
			AlertID:  "al-3",
			Title:    "Test",
			Body:     "Body",
			Severity: "info",
		})
		assert.NoError(t, err)
	}
}
