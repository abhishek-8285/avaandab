package telemetry_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/telemetry"
)

func TestHandleTelemetrySync_Success(t *testing.T) {
	r := chi.NewRouter()
	telemetry.RegisterTelemetryRoutes(r)

	reqPayload := telemetry.SyncBatchRequest{
		DriverID: "drv-101",
		Logs: []telemetry.GPSLogPayload{
			{ID: 1, Latitude: 19.076, Longitude: 72.877, Timestamp: "2026-08-13T00:00:00Z"},
			{ID: 2, Latitude: 19.080, Longitude: 72.880, Timestamp: "2026-08-13T00:01:00Z"},
		},
	}

	body, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest("POST", "/api/v1/telemetry/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp telemetry.SyncBatchResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success || resp.SyncedCount != 2 {
		t.Fatalf("expected synced_count=2, got %d", resp.SyncedCount)
	}
}

func TestHandleTelemetrySync_InvalidBody(t *testing.T) {
	r := chi.NewRouter()
	telemetry.RegisterTelemetryRoutes(r)

	req := httptest.NewRequest("POST", "/api/v1/telemetry/sync", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid body, got %d", w.Code)
	}
}

func TestHandleTelemetrySnapshots_SuccessAndInvalid(t *testing.T) {
	r := chi.NewRouter()
	telemetry.RegisterTelemetryRoutes(r)

	snap := telemetry.TelemetrySnapshotPayload{
		TripID:    "trp-701",
		VehicleID: "vh-301",
		Timestamp: "2026-08-13T00:15:00Z",
		Latitude:  19.076,
		Longitude: 72.877,
		Speed:     60.0,
		FuelLevel: 50.0,
		Odometer:  1000.0,
	}
	body, _ := json.Marshal(snap)
	req := httptest.NewRequest("POST", "/api/v1/telemetry/snapshots", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	reqBad := httptest.NewRequest("POST", "/api/v1/telemetry/snapshots", bytes.NewReader([]byte("{}")))
	wBad := httptest.NewRecorder()

	r.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing trip_id, got %d", wBad.Code)
	}
}
