package trip

import "errors"

var (
	ErrTripNotFound           = errors.New("trip not found")
	ErrTripImmutable          = errors.New("trip cannot be modified")
	ErrCancelledTripImmutable = errors.New("cancelled trips cannot be modified")
	ErrCompletedTripImmutable = errors.New("completed trips cannot be edited")
)
