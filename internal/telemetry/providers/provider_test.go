package providers

import (
	"testing"
)

func TestRegistry_RoundTrip(t *testing.T) {
	m := &MockProvider{}
	Register(m)

	got, ok := Get("mock")
	if !ok {
		t.Fatal("expected provider 'mock' to be registered")
	}
	if got.Name() != "mock" {
		t.Fatalf("expected name 'mock', got %q", got.Name())
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	if _, ok := Get("does-not-exist"); ok {
		t.Fatal("expected unknown provider to not be found")
	}
}

func TestRegistry_All(t *testing.T) {
	all := All()
	if _, ok := all["mock"]; !ok {
		t.Fatal("expected 'mock' to be present in All()")
	}
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	// Register a second provider under the same name; the latest wins.
	second := &MockProvider{}
	Register(second)
	got, ok := Get("mock")
	if !ok {
		t.Fatal("expected provider 'mock' to be registered")
	}
	if got != TelematicsProvider(second) {
		t.Fatal("expected latest registration to win")
	}
}

// compile-time interface check
var _ TelematicsProvider = (*MockProvider)(nil)
