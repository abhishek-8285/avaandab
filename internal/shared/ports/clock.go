package ports

import "time"

// Clock provides a mockable interface for retrieving the current system time.
type Clock interface {
	Now() time.Time
}
