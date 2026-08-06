package driver

import "errors"

var (
	ErrDriverNotFound    = errors.New("driver not found")
	ErrDriverUnavailable = errors.New("driver is not available")
	ErrDriverOnTrip      = errors.New("driver is currently on a trip")
)
