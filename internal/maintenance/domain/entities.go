package domain

import (
	"time"
)

// Schedule represents a preventive maintenance schedule for a vehicle (Spec 04 §6, §3).
type Schedule struct {
	ID           string     `json:"id"`
	VehicleID    string     `json:"vehicle_id"`
	ServiceType  string     `json:"service_type"` // oil_change, brake, tyre, battery, engine, fitness, insurance, permit, general
	IntervalKM   *float64   `json:"interval_km,omitempty"`
	IntervalDays *int       `json:"interval_days,omitempty"`
	LastDoneKM   *float64   `json:"last_done_km,omitempty"`
	LastDoneAt   *time.Time `json:"last_done_at,omitempty"`
	DueKM        *float64   `json:"due_km,omitempty"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Record represents a completed maintenance service event (Spec 04 §6, §3).
type Record struct {
	ID          string    `json:"id"`
	VehicleID   string    `json:"vehicle_id"`
	ScheduleID  *string   `json:"schedule_id,omitempty"`
	ServiceType string    `json:"service_type"`
	PerformedAt time.Time `json:"performed_at"`
	OdometerKM  *float64  `json:"odometer_km,omitempty"`
	Cost        *float64  `json:"cost,omitempty"`
	Vendor      *string   `json:"vendor,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	RecordedBy  *string   `json:"recorded_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// DtcEvent represents an OBD-II / J1939 diagnostic trouble code event (Spec 04 §6, §3).
type DtcEvent struct {
	ID          string     `json:"id"`
	VehicleID   string     `json:"vehicle_id"`
	TripID      *string    `json:"trip_id,omitempty"`
	DtcCode     string     `json:"dtc_code"`
	Severity    string     `json:"severity"` // 'info', 'warning', 'critical'
	Description *string    `json:"description,omitempty"`
	RawPayload  *string    `json:"raw_payload,omitempty"`
	OccurredAt  time.Time  `json:"occurred_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ServiceTypes valid enum values
var ServiceTypes = []string{
	"oil_change",
	"brake",
	"tyre",
	"battery",
	"engine",
	"fitness",
	"insurance",
	"permit",
	"general",
}
