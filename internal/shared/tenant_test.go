package shared

import (
	"context"
	"testing"
)

func TestTenantIDFromContext_FailClosed(t *testing.T) {
	ctx := context.Background()
	if got := TenantIDFromContext(ctx); got != "" {
		t.Fatalf("expected empty tenant for bare context, got %q", got)
	}
}

func TestTenantIDFromContext_ReturnsSetTenant(t *testing.T) {
	ctx := ContextWithTenantID(context.Background(), TenantID("7"))
	if got := TenantIDFromContext(ctx); got != "7" {
		t.Fatalf("expected tenant 7, got %q", got)
	}
}

func TestTenantIDFromContext_IgnoresEmptyTenant(t *testing.T) {
	ctx := ContextWithTenantID(context.Background(), "")
	if got := TenantIDFromContext(ctx); got != "" {
		t.Fatalf("expected empty tenant, got %q", got)
	}
}

func TestTenantRequired(t *testing.T) {
	if _, err := TenantRequired(context.Background()); err == nil {
		t.Fatal("expected error for bare context")
	}

	ctx := ContextWithTenantID(context.Background(), TenantID("7"))
	got, err := TenantRequired(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "7" {
		t.Fatalf("expected tenant 7, got %q", got)
	}
}

func TestDefaultTenant_SingleTenantBootstrap(t *testing.T) {
	if DefaultTenant != "1" {
		t.Fatalf("DefaultTenant must be %q, got %q", "1", DefaultTenant)
	}
}
