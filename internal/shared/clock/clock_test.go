package clock_test

import (
	"testing"
	"time"

	"transport-app/internal/shared/clock"
)

func TestRealClock_Now(t *testing.T) {
	c := clock.NewRealClock()
	before := time.Now()
	now := c.Now()
	after := time.Now()

	if now.Before(before) || now.After(after) {
		t.Fatalf("realClock.Now() returned time out of bounds")
	}
}
