package application

import (
	"context"
	"time"

	"transport-app/internal/events"
	"transport-app/internal/shared/outbox"
)

// Severity levels for geofence alerts.
const (
	SeverityHigh   = "high"
	SeverityMedium = "medium"
	SeverityLow    = "low"
)

// GeofenceAlertEvent is emitted to the outbox when a vehicle breaches a
// restricted/no_entry zone (Spec 02 §4). Registered in the canonical event
// catalog as events.GeofenceZoneBreach.
type GeofenceAlertEvent struct {
	TenantID   string    `json:"tenant_id"`
	VehicleID  string    `json:"vehicle_id"`
	TripID     string    `json:"trip_id"`
	GeofenceID string    `json:"geofence_id"`
	ZoneKind   string    `json:"zone_kind"`
	AlertType  string    `json:"alert_type"`
	Severity   string    `json:"severity"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
}

func init() {
	// Register in the canonical catalog so the outbox persists the exact
	// string subscribers listen on (Spec 09 §5.1). Done lazily here to
	// avoid an import cycle (events → application → outbox → events).
	events.EventTypeOf[GeofenceAlertEvent{}] = events.GeofenceZoneBreach
}

// SeverityFor maps a zone kind to an alert severity.
func SeverityFor(kind string) string {
	switch kind {
	case "no_entry":
		return SeverityHigh
	case "restricted":
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// EmitGeofenceAlert writes the alert to outbox_events. Call inside the same
// UnitOfWork transaction as the geofence_events row so both commit together
// (the outbox writer honours repository.TxFromContext).
func EmitGeofenceAlert(ctx context.Context, w *outbox.OutboxWriter, alert GeofenceAlertEvent) error {
	return w.SaveEvents(ctx, alert.VehicleID, "Vehicle", []any{alert})
}
