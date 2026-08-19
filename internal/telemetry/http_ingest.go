package telemetry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/telemetry/providers"
)

// HTTPIngestHandler handles REST GPS ingestion for own devices.
type HTTPIngestHandler struct {
	ingestor *Ingestor
	store    *DeviceStore
	pepper   string // TELEMETRY_DEVICE_SECRET_PEPPER
}

// NewHTTPIngestHandler constructs an HTTPIngestHandler.
func NewHTTPIngestHandler(ingestor *Ingestor, store *DeviceStore, pepper string) *HTTPIngestHandler {
	return &HTTPIngestHandler{ingestor: ingestor, store: store, pepper: pepper}
}

// RegisterRoutes mounts the device GPS endpoint. Must be called on a public
// router (no RequireAPIAuth) — devices authenticate via X-Device-Token.
func (h *HTTPIngestHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/telemetry/devices/{imei}/gps", h.HandleDeviceGPS)
}

// HandleDeviceGPS processes POST /api/v1/telemetry/devices/{imei}/gps
// Auth: X-Device-Token header (HMAC-SHA256 of device secret).
func (h *HTTPIngestHandler) HandleDeviceGPS(w http.ResponseWriter, r *http.Request) {
	imei := chi.URLParam(r, "imei")
	if imei == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing imei"})
		return
	}

	token := r.Header.Get("X-Device-Token")
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing device token"})
		return
	}

	device, err := h.store.GetByIMEI(r.Context(), imei)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "device lookup failed"})
		return
	}
	if device == nil {
		// Unknown IMEI — quarantine server-side, return 404.
		_ = h.ingestor.quarantineUnknown(r.Context(), imei, providers.RawFrame{IMEI: imei})
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown imei"})
		return
	}

	if device.Status != DeviceStatusActive {
		// Retired / quarantined / inventory devices are quarantined.
		_ = h.ingestor.quarantineFrame(r.Context(), providers.RawFrame{IMEI: imei},
			QuarantineReasonRetiredDevice)
		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"accepted":    true,
			"quarantined": true,
			"reason":      QuarantineReasonRetiredDevice,
		})
		return
	}

	if !h.validateToken(token, device.DeviceSecretHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "bad token"})
		return
	}

	// Parse body into RawFrame.
	var frame providers.RawFrame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// Override identity fields from the authenticated context.
	frame.IMEI = imei
	frame.Provider = "own"

	result, err := h.ingestor.IngestRawFrame(r.Context(), frame)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pipeline failed"})
		return
	}

	if result.Quarantined {
		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"accepted":    true,
			"quarantined": true,
			"reason":      result.Reason,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accepted": true,
		"deduped":  result.Deduped,
	})
}

// validateToken checks the device token against the stored hash using
// HMAC-SHA256 with a server-side pepper (Decision D9).
func (h *HTTPIngestHandler) validateToken(token string, storedHash *string) bool {
	if storedHash == nil || *storedHash == "" {
		return false // no secret provisioned
	}
	computed := hmacSHA256(h.pepper, token)
	return hmac.Equal([]byte(computed), []byte(*storedHash))
}

func hmacSHA256(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
