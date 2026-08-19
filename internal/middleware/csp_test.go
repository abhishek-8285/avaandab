package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentSecurityPolicy_Enabled(t *testing.T) {
	handler := ContentSecurityPolicy(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/tracking", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "script-src 'self' 'unsafe-inline'")
	assert.Contains(t, csp, "https://mt1.google.com")
	assert.Contains(t, csp, "https://tile.openstreetmap.org")
	assert.Contains(t, csp, "https://nominatim.openstreetmap.org")
}

func TestContentSecurityPolicy_Disabled(t *testing.T) {
	handler := ContentSecurityPolicy(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/tracking", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")
	assert.Empty(t, csp)
}
