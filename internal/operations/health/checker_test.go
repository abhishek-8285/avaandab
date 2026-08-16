package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandlerWritesJSONBody(t *testing.T) {
	c := NewChecker(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	c.HealthHandler(rr, req)

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"status"`) {
		t.Fatalf("body missing status field: %s", body)
	}
	var resp HealthCheckResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not valid JSON: %v (body: %s)", err, body)
	}
	if resp.Status != "DOWN" {
		t.Fatalf("expected status DOWN with nil db, got %s", resp.Status)
	}
	if len(resp.Components) == 0 {
		t.Fatal("expected at least one component in response")
	}
}

func TestHealthHandlerDownStatusWithNilDB(t *testing.T) {
	c := NewChecker(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	c.HealthHandler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 with nil db, got %d", rr.Code)
	}
}

func TestLivenessHandler(t *testing.T) {
	c := NewChecker(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	c.LivenessHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "OK" {
		t.Fatalf("expected body OK, got %q", rr.Body.String())
	}
}
