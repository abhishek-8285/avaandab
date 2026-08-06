package driver

import (
	"time"

	"transport-app/internal/domain/types"
)

// DriverCreated is emitted when a new driver is created.
type DriverCreated struct {
	DriverID    types.DriverID
	FirstName   string
	LastName    string
	Phone       string
	OccurredAt  time.Time
}

// DriverStatusChanged is emitted when driver status changes.
type DriverStatusChanged struct {
	DriverID    types.DriverID
	OldStatus   DriverStatus
	NewStatus   DriverStatus
	OccurredAt  time.Time
}
