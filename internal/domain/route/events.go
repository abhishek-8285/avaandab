package route

import (
	"time"

	"transport-app/internal/domain/types"
)

// RouteCreated is emitted when a new route is created.
type RouteCreated struct {
	RouteID     types.RouteID
	Source      string
	Destination string
	OccurredAt  time.Time
}
