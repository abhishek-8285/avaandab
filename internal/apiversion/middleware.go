package apiversion

import (
	"net/http"
	"regexp"
)

var (
	pathVersionRE = regexp.MustCompile(`^/api/(v\d+)/`)
	acceptRE      = regexp.MustCompile(`application/vnd\.transport\.(v\d+)\+json`)
)

// Middleware extracts the API version from the request path (e.g. /api/v1/...,
// /api/v2/...) and injects it into the request context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := extractPathVersion(r.URL.Path); v != "" {
			r = r.WithContext(WithVersion(r.Context(), v))
		}
		next.ServeHTTP(w, r)
	})
}

// Negotiate returns the API version requested by the client. It first inspects
// the Accept header for a vendor media type such as
// application/vnd.transport.v2+json; if no version is found there, it falls back
// to the version embedded in the URL path.
func Negotiate(r *http.Request) string {
	if accept := r.Header.Get("Accept"); accept != "" {
		if m := acceptRE.FindStringSubmatch(accept); len(m) > 1 {
			return m[1]
		}
	}
	return extractPathVersion(r.URL.Path)
}

func extractPathVersion(path string) string {
	if m := pathVersionRE.FindStringSubmatch(path); len(m) > 1 {
		return m[1]
	}
	return ""
}
