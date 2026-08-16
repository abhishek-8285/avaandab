package apiversion

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// httpDateFormat is the IMF-fixdate format defined in RFC 7231.
const httpDateFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// V1Registrar is implemented by handlers that mount routes under /api/v1.
type V1Registrar interface {
	Register(r chi.Router)
}

// DeprecationMiddleware adds Deprecation and Sunset headers to every response.
// It is intended for deprecated version aliases such as /api/v2.
func DeprecationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", Sunset.UTC().Format(httpDateFormat))
		next.ServeHTTP(w, r)
	})
}

// MountV2 mounts /api/v2 routes that delegate to existing v1 handlers.
//
// The supplied registrars are used to build an isolated v1 router at
// /api/v1/..., and every /api/v2 request is rewritten to the equivalent v1
// path before being served. All v2 responses include Deprecation and Sunset
// headers. A dedicated /api/v2/health endpoint is also provided; if healthH is
// nil a simple JSON {"status":"ok"} response is returned.
//
// If authMiddleware is non-nil, it is applied to the aliased /api/v2/* routes
// (but not to /api/v2/health) so that v2 inherits the same protection as v1.
func MountV2(r chi.Router, authMiddleware func(http.Handler) http.Handler, healthH http.Handler, registrars ...V1Registrar) {
	v1 := chi.NewRouter()
	for _, reg := range registrars {
		if reg != nil {
			reg.Register(v1)
		}
	}

	if healthH == nil {
		healthH = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})
	}

	r.Get("/api/v2/health", DeprecationMiddleware(healthH).ServeHTTP)

	alias := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := WithVersion(req.Context(), V2)
		// The outer router has already populated a chi route context. Remove it
		// so the isolated v1 router can dispatch the rewritten path cleanly.
		ctx = context.WithValue(ctx, chi.RouteCtxKey, nil)
		req = req.WithContext(ctx)
		// Replace the /api/v2 prefix with /api/v1 so the v1 router can find
		// the equivalent route.
		req.URL.Path = strings.Replace(req.URL.Path, "/api/v2", "/api/v1", 1)
		v1.ServeHTTP(w, req)
	})

	if authMiddleware != nil {
		r.With(authMiddleware).Mount("/api/v2", DeprecationMiddleware(alias))
	} else {
		r.Mount("/api/v2", DeprecationMiddleware(alias))
	}
}
