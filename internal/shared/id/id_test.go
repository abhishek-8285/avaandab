package id_test

import (
	"strings"
	"testing"

	"transport-app/internal/shared/id"
)

func TestUUIDGenerator_Generate(t *testing.T) {
	g := id.NewUUIDGenerator()

	u := g.GenerateUUID()
	if len(u) == 0 {
		t.Fatalf("expected non-empty UUID")
	}

	disp := g.GenerateDisplayID("BK")
	if !strings.HasPrefix(disp, "BK-") {
		t.Fatalf("expected display ID with prefix BK-, got %s", disp)
	}
}
