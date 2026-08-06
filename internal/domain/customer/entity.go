package customer

import (
	"time"

	"transport-app/internal/domain/types"
)

// Customer represents a customer who books transport services.
type Customer struct {
	ID        types.CustomerID
	Name      string
	Company   *string
	Phone     string
	Email     *string
	GST       *string
	Address   *string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
