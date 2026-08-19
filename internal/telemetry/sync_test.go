package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// newTestRouter builds a chi router with the telemetry routes wired to a
// real Ingestor over a migrated in-memory DB.
func newTestRouter(t *testing.T) (chi.Router, *Ingestor) {
	t.Helper()
	db := newTestIngestorDB(t)
	ing := newTestIngestor(t, db, nil)
	r := chi.NewRouter()
	RegisterTelemetryRoutes(r, ing, db, 15*time.Minute)
	return r, ing
}

// newTestRouterWithDevice registers an active device + vehicle and returns
// the router plus the device IMEI and vehicle ID.
func newTestRouterWithDevice(t *testing.T) (chi.Router, string, string) {
	t.Helper()
	db := newTestIngestorDB(t)
	ing := newTestIngestor(t, db, nil)
	vID := "vh-sync-1"
	insertTestVehicle(t, db, vID)
	imei := "IMEI-SYNC-1"
	insertTestDevice(t, db, imei, DeviceStatusActive, &vID)
	r := chi.NewRouter()
	RegisterTelemetryRoutes(r, ing, db, 15*time.Minute)
	return r, imei, vID
}

func TestHandleTelemetrySync_Success(t *testing.T) {
	r, imei, _ := newTestRouterWithDevice(t)

	reqPayload := SyncBatchRequest{
		DeviceID: imei,
		Logs: []GPSLogPayload{
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

	var resp SyncBatchResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success || resp.SyncedCount != 2 {
		t.Fatalf("expected synced_count=2, got %d", resp.SyncedCount)
	}
	// Real synced_ids returned (not echoed)
	assert.Equal(t, []int64{1, 2}, resp.SyncedIDs)
}

func TestHandleTelemetrySync_InvalidBody(t *testing.T) {
	r, _ := newTestRouter(t)

	req := httptest.NewRequest("POST", "/api/v1/telemetry/sync", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid body, got %d", w.Code)
	}
}

func TestHandleTelemetrySnapshots_SuccessAndInvalid(t *testing.T) {
	r, _, vID := newTestRouterWithDevice(t)

	// Success: a real vehicle with a registered device → pipeline accepts.
	snap := TelemetrySnapshotPayload{
		TripID:    "trp-701",
		VehicleID: vID,
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
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid: missing vehicle_id
	reqBad := httptest.NewRequest("POST", "/api/v1/telemetry/snapshots", bytes.NewReader([]byte("{}")))
	wBad := httptest.NewRecorder()

	r.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing trip_id, got %d", wBad.Code)
	}
}
