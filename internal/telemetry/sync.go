package telemetry

import (
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

func RegisterTelemetryRoutes(r chi.Router) {
	r.Post("/api/v1/telemetry/sync", HandleTelemetrySync)
}

func HandleTelemetrySync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SyncBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request payload"})
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

	json.NewEncoder(w).Encode(resp)
}
