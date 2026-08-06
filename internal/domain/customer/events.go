package customer

import (
	"time"

	"transport-app/internal/domain/types"
)

// CustomerCreated is emitted when a new customer is created.
type CustomerCreated struct {
	CustomerID types.CustomerID
	Name       string
	Phone      string
	OccurredAt time.Time
}
