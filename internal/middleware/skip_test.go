package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSkipForPaths(t *testing.T) {
	calledMiddleware := false
	dummyMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calledMiddleware = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := SkipForPaths(dummyMiddleware, "/api/v1/telemetry/stream")(handler)

	// Case 1: Route not matching prefix -> middleware must run
	calledMiddleware = false
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)
	if !calledMiddleware {
		t.Errorf("expected middleware to be called for /api/v1/trips")
	}

	// Case 2: Route matching prefix -> middleware must be bypassed
	calledMiddleware = false
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/stream", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)
	if calledMiddleware {
		t.Errorf("expected middleware to be skipped for /api/v1/telemetry/stream")
	}

	// Case 3: Route with subpath or query params on prefix
	calledMiddleware = false
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/stream?trip_id=123", nil)
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)
	if calledMiddleware {
		t.Errorf("expected middleware to be skipped for /api/v1/telemetry/stream?trip_id=123")
	}
}
