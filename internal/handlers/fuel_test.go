package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestFuelAuditRoutes_ForbiddenForViewer verifies every /fuel/audit route
// returns 403 for a user without fuel:* permissions (Spec 03 §8 RBAC).
func TestFuelAuditRoutes_ForbiddenForViewer(t *testing.T) {
	app := newTelemetryTestApp(t, denyAuthSvc{})
	fa := &FuelAuditHandlers{App: app}

	r := chi.NewRouter()
	r.Route("/fuel", fa.Routes)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"dashboard", http.MethodGet, "/fuel/audit"},
		{"queue", http.MethodGet, "/fuel/audit/queue"},
		{"run", http.MethodPost, "/fuel/audit/run"},
		{"detail", http.MethodGet, "/fuel/audit/e-1"},
		{"review", http.MethodPost, "/fuel/audit/e-1/review"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := withSession(httptest.NewRequest(tc.method, tc.path, nil), "viewer-1", "viewer")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// TestFuelAuditRoutes_RedirectWithoutSession verifies the routes bounce
// anonymous traffic to /login.
func TestFuelAuditRoutes_RedirectWithoutSession(t *testing.T) {
	app := newTelemetryTestApp(t, denyAuthSvc{})
	fa := &FuelAuditHandlers{App: app}

	r := chi.NewRouter()
	r.Route("/fuel", fa.Routes)

	req := httptest.NewRequest(http.MethodGet, "/fuel/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}
