package shared

import (
	"context"
	"errors"
)

// TenantID represents a company/tenant identifier in a multi-tenant system.
type TenantID string

// NewTenantID validates and creates a TenantID.
func NewTenantID(id string) (TenantID, error) {
	if id == "" {
		return "", errors.New("tenant ID cannot be empty")
	}
	return TenantID(id), nil
}

type contextKey string

const TenantIDKey contextKey = "tenant_id"

// ContextWithTenantID returns a new context containing the TenantID.
func ContextWithTenantID(ctx context.Context, id TenantID) context.Context {
	return context.WithValue(ctx, TenantIDKey, id)
}

// TenantIDFromContext retrieves TenantID from the context, defaulting to "1" if not found.
func TenantIDFromContext(ctx context.Context) TenantID {
	if val, ok := ctx.Value(TenantIDKey).(TenantID); ok {
		return val
	}
	return "1" // Fallback default tenant
}
