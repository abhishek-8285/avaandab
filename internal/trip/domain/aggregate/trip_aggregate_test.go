package aggregate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"transport-app/internal/shared"
)

func TestNewTripAggregate(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")
	bookingID := "bk-123"

	agg := NewTripAggregate(
		"tr-123",
		tenantID,
		"TR-0001",
		&bookingID,
		"route-123",
		now.Add(2*time.Hour),
		"Standard Remarks",
		now,
	)

	assert.Equal(t, TripID("tr-123"), agg.ID)
	assert.Equal(t, TripDraft, agg.Status)
	assert.Len(t, agg.Events(), 1)
	_, ok := agg.Events()[0].(TripCreatedEvent)
	assert.True(t, ok)
}

func TestTripAggregate_Transitions(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")
	bookingID := "bk-123"

	agg := NewTripAggregate(
		"tr-123",
		tenantID,
		"TR-0001",
		&bookingID,
		"route-123",
		now.Add(2*time.Hour),
		"",
		now,
	)

	agg.ClearEvents()

	// Draft -> Scheduled
	err := agg.Schedule(now)
	assert.NoError(t, err)
	assert.Equal(t, TripScheduled, agg.Status)

	// Scheduled -> Assigned
	err = agg.AssignDriver("driver-1", now)
	assert.NoError(t, err)
	assert.Equal(t, TripAssigned, agg.Status)
	assert.Equal(t, "driver-1", *agg.DriverID)

	// Assigned -> Started
	err = agg.Start(now)
	assert.NoError(t, err)
	assert.Equal(t, TripStarted, agg.Status)

	// Started -> Completed
	err = agg.Complete(now)
	assert.NoError(t, err)
	assert.Equal(t, TripCompleted, agg.Status)
	assert.NotNil(t, agg.ArrivalTime)
}
