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
	assert.Nil(t, agg.StartedAt)
	assert.Nil(t, agg.ReachedPickupAt)
	assert.Nil(t, agg.InTransitAt)
	assert.Nil(t, agg.DeliveredAt)
	assert.Nil(t, agg.CompletedAt)
	assert.Len(t, agg.Events(), 1)
	_, ok := agg.Events()[0].(TripCreatedEvent)
	assert.True(t, ok)
}

func TestTripAggregate_ExecutionWorkflow(t *testing.T) {
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
	agg.AssignDriver("driver-1", now)
	assert.Equal(t, TripAssigned, agg.Status)

	// Assigned -> Started (timeline: started_at set)
	err = agg.Start(now)
	assert.NoError(t, err)
	assert.Equal(t, TripStarted, agg.Status)
	assert.NotNil(t, agg.StartedAt)

	// Started -> Reached Pickup
	err = agg.ReachPickup(now)
	assert.NoError(t, err)
	assert.Equal(t, TripReachedPickup, agg.Status)
	assert.NotNil(t, agg.ReachedPickupAt)

	// Reached Pickup -> In Transit
	err = agg.StartTransit(now)
	assert.NoError(t, err)
	assert.Equal(t, TripInTransit, agg.Status)
	assert.NotNil(t, agg.InTransitAt)

	// In Transit -> Delivered
	err = agg.Deliver(now)
	assert.NoError(t, err)
	assert.Equal(t, TripDelivered, agg.Status)
	assert.NotNil(t, agg.DeliveredAt)

	// Delivered -> Completed (timeline: completed_at set, arrival_time set)
	err = agg.Complete(now)
	assert.NoError(t, err)
	assert.Equal(t, TripCompleted, agg.Status)
	assert.NotNil(t, agg.CompletedAt)
	assert.NotNil(t, agg.ArrivalTime)

	// Verify all timeline events were recorded (7 workflow step events after ClearEvents)
	assert.Len(t, agg.Events(), 7)
}

func TestTripAggregate_TransitionErrors(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")

	agg := NewTripAggregate(
		"tr-123",
		tenantID,
		"TR-0001",
		nil,
		"route-123",
		now.Add(2*time.Hour),
		"",
		now,
	)
	agg.ClearEvents()

	// Cannot reach pickup from draft
	err := agg.ReachPickup(now)
	assert.Error(t, err)

	// Cannot start transit from draft
	err = agg.StartTransit(now)
	assert.Error(t, err)

	// Cannot deliver from draft
	err = agg.Deliver(now)
	assert.Error(t, err)

	// Cannot complete from draft
	err = agg.Complete(now)
	assert.Error(t, err)

	// Start without driver/vehicle assignment is allowed (from assigned or scheduled)
	// Schedule -> Start -> Complete should fail (must go through full chain)
	agg.Schedule(now)
	agg.AssignDriver("driver-1", now)
	agg.Start(now)

	// Started -> Complete should fail (must reach pickup, go in transit, deliver first)
	err = agg.Complete(now)
	assert.Error(t, err)

	// Started -> In Transit should fail (must reach pickup first)
	err = agg.StartTransit(now)
	assert.Error(t, err)

	// Started -> Deliver should fail
	err = agg.Deliver(now)
	assert.Error(t, err)
}

func TestTripAggregate_TimelineEvents(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")

	agg := NewTripAggregate(
		"tr-123",
		tenantID,
		"TR-0001",
		nil,
		"route-123",
		now.Add(2*time.Hour),
		"",
		now,
	)
	agg.ClearEvents()

	agg.Schedule(now)
	agg.AssignDriver("driver-1", now)
	agg.Start(now)
	agg.ReachPickup(now)
	agg.StartTransit(now)
	agg.Deliver(now)
	agg.Complete(now)

	events := agg.Events()
	assert.Len(t, events, 7)

	// Verify event types in order (0 to 6)
	_, ok1 := events[0].(TripScheduledEvent)
	assert.True(t, ok1)
	_, ok2 := events[1].(TripAssignedEvent)
	assert.True(t, ok2)
	_, ok3 := events[2].(TripStartedEvent)
	assert.True(t, ok3)
	_, ok4 := events[3].(TripReachedPickupEvent)
	assert.True(t, ok4)
	_, ok5 := events[4].(TripInTransitEvent)
	assert.True(t, ok5)
	_, ok6 := events[5].(TripDeliveredEvent)
	assert.True(t, ok6)
	_, ok7 := events[6].(TripCompletedEvent)
	assert.True(t, ok7)
}
