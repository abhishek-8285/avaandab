package apiversion

import "context"

type ctxKey int

const versionKey ctxKey = iota

// WithVersion returns a new context carrying the given API version.
func WithVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, versionKey, version)
}

// FromContext returns the API version stored in the context, or an empty
// string if none is present.
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(versionKey).(string); ok {
		return v
	}
	return ""
}
