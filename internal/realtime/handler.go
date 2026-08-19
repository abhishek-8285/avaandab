package realtime

import (
	"encoding/json"
	"net/http"

	"transport-app/internal/events"
)

// StreamHandler returns an http.HandlerFunc for SSE streaming (Spec 04 §1.2, §7).
// Sets SSE headers, flushes headers immediately, and exits on client disconnect
// or slow consumer drop. Optional ?trip_id= / ?vehicle_id= query filters.
// If sseEnabled is provided and false, returns HTTP 503 Service Unavailable.
func StreamHandler(h *Hub, sseEnabled ...bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(sseEnabled) > 0 && !sseEnabled[0] {
			http.Error(w, `{"error":"SSE streaming is disabled"}`, http.StatusServiceUnavailable)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// Optional filter based on query params
		tripID := r.URL.Query().Get("trip_id")
		vehicleID := r.URL.Query().Get("vehicle_id")

		var f filter
		if tripID != "" || vehicleID != "" {
			f = func(e events.Event) bool {
				if e.Payload == nil {
					return false
				}
				var m map[string]interface{}
				switch p := e.Payload.(type) {
				case map[string]interface{}:
					m = p
				default:
					b, err := json.Marshal(e.Payload)
					if err != nil {
						return false
					}
					if err := json.Unmarshal(b, &m); err != nil {
						return false
					}
				}
				if tripID != "" {
					tid, ok := m["trip_id"].(string)
					if !ok || tid != tripID {
						return false
					}
				}
				if vehicleID != "" {
					vid, ok := m["vehicle_id"].(string)
					if !ok || vid != vehicleID {
						return false
					}
				}
				return true
			}
		}

		ch, unsub := h.Subscribe(r.Context(), f)
		defer unsub()

		// Flush headers immediately
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case frame, ok := <-ch:
				if !ok {
					return // channel closed (slow consumer dropped or hub shutdown)
				}
				if _, err := w.Write(frame); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
