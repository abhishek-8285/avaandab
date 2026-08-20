package providers

import (
	"testing"
)

func isolateRegistry(t *testing.T) {
	t.Helper()
	orig := All()
	t.Cleanup(func() {
		Reset()
		for _, p := range orig {
			Register(p)
		}
	})
}

func TestRegistry_RoundTrip(t *testing.T) {
	isolateRegistry(t)
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
	isolateRegistry(t)
	if _, ok := Get("does-not-exist"); ok {
		t.Fatal("expected unknown provider to not be found")
	}
}

func TestRegistry_All(t *testing.T) {
	isolateRegistry(t)
	m := &MockProvider{}
	Register(m)
	all := All()
	if _, ok := all["mock"]; !ok {
		t.Fatal("expected 'mock' to be present in All()")
	}
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	isolateRegistry(t)
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
