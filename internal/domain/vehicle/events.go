package vehicle

import (
	"time"

	"transport-app/internal/domain/types"
)

// VehicleCreated is emitted when a new vehicle is created.
type VehicleCreated struct {
	VehicleID     types.VehicleID
	Registration  string
	VehicleNumber string
	OccurredAt    time.Time
}

// VehicleStatusChanged is emitted when vehicle status changes.
type VehicleStatusChanged struct {
	VehicleID  types.VehicleID
	OldStatus  VehicleStatus
	NewStatus  VehicleStatus
	OccurredAt time.Time
}
