package route

import (
	"time"

	"transport-app/internal/domain/types"
)

// Route represents a transport route between a source and destination.
type Route struct {
	ID             types.RouteID
	Source         string
	Destination    string
	Distance       float64
	EstimatedHours float64
	StandardFare   float64
	Remarks        *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
