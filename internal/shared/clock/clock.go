package clock

import (
	"time"
	"transport-app/internal/shared/ports"
)

type realClock struct{}

// NewRealClock returns a Clock instance using the system time.
func NewRealClock() ports.Clock {
	return &realClock{}
}

func (c *realClock) Now() time.Time {
	return time.Now()
}
