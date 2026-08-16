package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"transport-app/internal/auth"
)

// CSRFProtect blocks state-changing requests that carry a session cookie
// but originate from a different site (CSRF). Requests authenticated with a
// Bearer token (non-browser clients) are exempt. Safe methods and requests
// without a session cookie are allowed through.
//
// This complements SameSite=Lax session cookies: Lax already blocks
// cross-site POST cookies in modern browsers; this adds a defense-in-depth
// origin check for older clients and SameSite=None deployments.
func CSRFProtect(store *auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			// Non-browser clients authenticate with Bearer tokens and are
			// not vulnerable to cookie-based CSRF.
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			// No session cookie: not a browser-authenticated request.
			if !store.HasSession(r) {
				next.ServeHTTP(w, r)
				return
			}

			if !sameOrigin(r) {
				http.Error(w, "Forbidden: cross-site request rejected", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

// sameOrigin verifies that the request's Origin (or, failing that, Referer)
// header, when present, matches the request Host. A missing Origin and
// Referer is allowed: non-browser clients (curl, server-to-server) and
// same-origin navigation requests may omit both.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}
