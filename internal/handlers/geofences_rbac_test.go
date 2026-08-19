package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestGeofenceRoutes_ForbiddenForViewer verifies every /geofences route
// returns 403 for a user without geofences:* permissions (Spec 02 §8 RBAC).
func TestGeofenceRoutes_ForbiddenForViewer(t *testing.T) {
	app := newTelemetryTestApp(t, denyAuthSvc{})
	gf := NewGeofenceHandlers(app, nil)

	r := chi.NewRouter()
	r.Route("/geofences", gf.Routes)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"list", http.MethodGet, "/geofences"},
		{"new form", http.MethodGet, "/geofences/new"},
		{"create", http.MethodPost, "/geofences/new"},
		{"edit form", http.MethodGet, "/geofences/zone-1/edit"},
		{"update", http.MethodPost, "/geofences/zone-1/edit"},
		{"delete", http.MethodPost, "/geofences/zone-1/delete"},
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

// TestGeofenceRoutes_RedirectWithoutSession verifies the routes bounce
// anonymous traffic to /login.
func TestGeofenceRoutes_RedirectWithoutSession(t *testing.T) {
	app := newTelemetryTestApp(t, denyAuthSvc{})
	gf := NewGeofenceHandlers(app, nil)

	r := chi.NewRouter()
	r.Route("/geofences", gf.Routes)

	req := httptest.NewRequest(http.MethodGet, "/geofences", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}
