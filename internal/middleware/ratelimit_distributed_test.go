package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"transport-app/internal/cache"
	"transport-app/internal/middleware"
)

type fakeSettings struct{}

func (fakeSettings) GetDriver() string            { return "memory" }
func (fakeSettings) GetRedisAddr() string         { return "" }
func (fakeSettings) GetRedisPassword() string     { return "" }
func (fakeSettings) GetRedisDB() int              { return 0 }
func (fakeSettings) GetDefaultTTL() time.Duration { return time.Minute }
func (fakeSettings) GetKeyPrefix() string         { return "test:" }

func nil2ctx() context.Context { return context.Background() }

func TestRateLimitDistributed_FallsBackToLocalWithoutIncrementer(t *testing.T) {
	mw := middleware.RateLimitDistributed(cache.Noop{}, 2)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	codes := []int{}
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	// Local limiter allows the first `limit` requests from one IP.
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 {
		t.Errorf("codes = %v, want [200 200 429]", codes)
	}
}

func TestRateLimitDistributed_MemoryCacheCountsAcrossRequests(t *testing.T) {
	mc, err := cache.New(nil2ctx(), &fakeSettings{}, nil)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	if _, ok := mc.(cache.Incrementer); !ok {
		t.Fatalf("memory cache must implement Incrementer for distributed limiting")
	}

	mw := middleware.RateLimitDistributed(mc, 3)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != 200 {
			t.Fatalf("request %d = %d, want 200", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 429 {
		t.Errorf("4th request = %d, want 429", rec.Code)
	}
	if ra := rec.Header().Get("Retry-After"); ra != "60" {
		t.Errorf("Retry-After = %q, want 60", ra)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	mw := middleware.RateLimit(1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("first IP request = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "8.8.8.8:1234"
	handler.ServeHTTP(rec, req2)
	if rec.Code != 200 {
		t.Errorf("second IP request = %d, want 200", rec.Code)
	}
}
