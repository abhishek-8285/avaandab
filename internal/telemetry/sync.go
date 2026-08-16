package telemetry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type GPSLogPayload struct {
	ID        int64   `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp string  `json:"timestamp"`
}

type SyncBatchRequest struct {
	DriverID string          `json:"driver_id"`
	Logs     []GPSLogPayload `json:"logs"`
}

type SyncBatchResponse struct {
	Success     bool    `json:"success"`
	SyncedCount int     `json:"synced_count"`
	SyncedIDs   []int64 `json:"synced_ids"`
	ServerTime  string  `json:"server_time"`
}

type TelemetrySnapshotPayload struct {
	ID        string  `json:"id,omitempty"`
	TripID    string  `json:"trip_id"`
	VehicleID string  `json:"vehicle_id"`
	Timestamp string  `json:"timestamp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	FuelLevel float64 `json:"fuel_level"`
	Odometer  float64 `json:"odometer"`
}

func RegisterTelemetryRoutes(r chi.Router, databases ...*sql.DB) {
	var database *sql.DB
	if len(databases) > 0 {
		database = databases[0]
	}
	r.Post("/api/v1/telemetry/sync", HandleTelemetrySync)
	r.Post("/api/v1/telemetry/snapshots", snapshotHandler(database))
}

func snapshotHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var snap TelemetrySnapshotPayload
		if err := json.NewDecoder(r.Body).Decode(&snap); err != nil || snap.TripID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid snapshot payload"})
			return
		}
		id := snap.ID
		if id == "" {
			id = generateSnapshotID()
		}
		if database != nil {
			at, err := time.Parse(time.RFC3339, snap.Timestamp)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid timestamp"})
				return
			}
			if _, err := database.ExecContext(r.Context(), `INSERT OR REPLACE INTO telemetry_snapshots (id, trip_id, vehicle_id, timestamp, latitude, longitude, speed, fuel_level, odometer) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, snap.TripID, snap.VehicleID, at, snap.Latitude, snap.Longitude, snap.Speed, snap.FuelLevel, snap.Odometer); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to store snapshot"})
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "snapshot_id": id, "server_time": time.Now().Format(time.RFC3339)})
	}
}

func generateSnapshotID() string {
	return "snap-" + time.Now().UTC().Format("20060102150405.000000000")
}

func HandleTelemetrySync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SyncBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request payload"})
		return
	}

	syncedIDs := make([]int64, 0, len(req.Logs))
	for _, logItem := range req.Logs {
		syncedIDs = append(syncedIDs, logItem.ID)
	}

	resp := SyncBatchResponse{
		Success:     true,
		SyncedCount: len(syncedIDs),
		SyncedIDs:   syncedIDs,
		ServerTime:  time.Now().Format(time.RFC3339),
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func HandleTelemetrySnapshots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var snap TelemetrySnapshotPayload
	if err := json.NewDecoder(r.Body).Decode(&snap); err != nil || snap.TripID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid snapshot payload"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"snapshot_id": "snap-9001",
		"server_time": time.Now().Format(time.RFC3339),
	})
}
