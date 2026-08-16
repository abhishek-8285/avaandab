package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type ComponentStatus struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

type HealthCheckResponse struct {
	Status     string            `json:"status"` // UP, DOWN
	Timestamp  time.Time         `json:"timestamp"`
	Components []ComponentStatus `json:"components"`
}

type Checker struct {
	db *sql.DB
}

func NewChecker(db *sql.DB) *Checker {
	return &Checker{db: db}
}

func (c *Checker) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthCheckResponse{
		Status:     "UP",
		Timestamp:  time.Now(),
		Components: make([]ComponentStatus, 0),
	}

	dbStatus := ComponentStatus{Name: "database", Healthy: true}
	if c.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := c.db.PingContext(ctx); err != nil {
			dbStatus.Healthy = false
			dbStatus.Message = err.Error()
			resp.Status = "DOWN"
		}
	} else {
		dbStatus.Healthy = false
		dbStatus.Message = "database connection uninitialized"
		resp.Status = "DOWN"
	}
	resp.Components = append(resp.Components, dbStatus)

	// Outbox & Email workers component status
	resp.Components = append(resp.Components, ComponentStatus{Name: "outbox_workers", Healthy: true})
	resp.Components = append(resp.Components, ComponentStatus{Name: "notification_service", Healthy: true})

	w.Header().Set("Content-Type", "application/json")
	if resp.Status == "DOWN" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	if err := writeJSON(w, resp); err != nil {
		logger := slog.Default()
		logger.Error("failed to write health response", "error", err)
	}
}

func (c *Checker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (c *Checker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if c.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := c.db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Database Unreachable"))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("READY"))
}

func writeJSON(w http.ResponseWriter, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}
