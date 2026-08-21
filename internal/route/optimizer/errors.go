package optimizer

import "errors"

var (
	ErrNoShipments      = errors.New("at least one shipment required")
	ErrTooManyShipments = errors.New("too many shipments: max 50 per job")
	ErrNoVehicles       = errors.New("at least one vehicle required")
	ErrProviderFailed   = errors.New("routing provider failed")
)
