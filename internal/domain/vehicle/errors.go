package vehicle

import "errors"

var (
	ErrVehicleNotFound    = errors.New("vehicle not found")
	ErrVehicleUnavailable = errors.New("vehicle is not available")
	ErrVehicleAssigned    = errors.New("vehicle is already assigned to another trip")
)
