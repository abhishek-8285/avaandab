package logging_test

import (
	"testing"

	"transport-app/internal/logging"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		level string
		env   string
	}{
		{"debug", "development"},
		{"warn", "production"},
		{"error", "production"},
		{"info", "development"},
		{"unknown", "staging"},
	}

	for _, tt := range tests {
		logging.Setup(tt.level, tt.env)
	}
}
