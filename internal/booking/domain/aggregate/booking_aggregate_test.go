package aggregate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"transport-app/internal/shared"
)

func TestNewBookingAggregate(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")
	cargo := float64(12.5)
	price, _ := shared.NewMoney(15000, "USD")

	agg := NewBookingAggregate(
		"bk-123",
		tenantID,
		"BK-0001",
		"cust-123",
		"route-123",
		now.Add(24*time.Hour),
		"Truck",
		2,
		&cargo,
		price,
		"Fragile items",
		now,
	)

	assert.Equal(t, BookingID("bk-123"), agg.ID)
	assert.Equal(t, BookingPending, agg.Status)
	assert.Len(t, agg.Events(), 1)

	_, ok := agg.Events()[0].(BookingCreatedEvent)
	assert.True(t, ok)
}

func TestBookingAggregate_Confirm(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")
	price, _ := shared.NewMoney(15000, "USD")

	agg := NewBookingAggregate(
		"bk-123",
		tenantID,
		"BK-0001",
		"cust-123",
		"route-123",
		now.Add(24*time.Hour),
		"Truck",
		2,
		nil,
		price,
		"",
		now,
	)

	agg.ClearEvents()

	err := agg.Confirm(now)
	assert.NoError(t, err)
	assert.Equal(t, BookingConfirmed, agg.Status)
	assert.Len(t, agg.Events(), 1)

	_, ok := agg.Events()[0].(BookingConfirmedEvent)
	assert.True(t, ok)

	// Can't confirm twice
	err = agg.Confirm(now)
	assert.Error(t, err)
}

func TestBookingAggregate_Cancel(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")
	price, _ := shared.NewMoney(15000, "USD")

	agg := NewBookingAggregate(
		"bk-123",
		tenantID,
		"BK-0001",
		"cust-123",
		"route-123",
		now.Add(24*time.Hour),
		"Truck",
		2,
		nil,
		price,
		"",
		now,
	)

	err := agg.Cancel(now)
	assert.NoError(t, err)
	assert.Equal(t, BookingCancelled, agg.Status)

	// Cannot confirm cancelled booking
	err = agg.Confirm(now)
	assert.Error(t, err)
}

func TestBookingAggregate_Complete(t *testing.T) {
	now := time.Now()
	tenantID := shared.TenantID("1")
	price, _ := shared.NewMoney(15000, "USD")

	agg := NewBookingAggregate(
		"bk-123",
		tenantID,
		"BK-0001",
		"cust-123",
		"route-123",
		now.Add(24*time.Hour),
		"Truck",
		2,
		nil,
		price,
		"",
		now,
	)

	// Cannot complete pending booking
	err := agg.Complete(now)
	assert.Error(t, err)

	// Confirm first
	agg.ClearEvents()
	err = agg.Confirm(now)
	assert.NoError(t, err)

	// Now can complete
	agg.ClearEvents()
	err = agg.Complete(now)
	assert.NoError(t, err)
	assert.Equal(t, BookingCompleted, agg.Status)
	assert.Len(t, agg.Events(), 1)

	_, ok := agg.Events()[0].(BookingCompletedEvent)
	assert.True(t, ok)

	// Cannot complete twice
	err = agg.Complete(now)
	assert.Error(t, err)
}
