package dispatch_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/dispatch"
	"transport-app/internal/domain/types"
)

func TestDispatch_AssignResources(t *testing.T) {
	d := &dispatch.Dispatch{
		ID:           types.DispatchID("dsp-1"),
		DispatcherID: types.UserID("usr-1"),
		BookingID:    types.BookingID("bk-1"),
		ScheduledAt:  time.Now(),
		Status:       dispatch.DispatchDraft,
	}

	driverID := types.DriverID("drv-1")
	vehicleID := types.VehicleID("vh-1")

	err := d.AssignResources(driverID, vehicleID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if d.Status != dispatch.DispatchAssigned {
		t.Errorf("expected status %s, got %s", dispatch.DispatchAssigned, d.Status)
	}

	if d.DriverID == nil || *d.DriverID != driverID {
		t.Errorf("expected driver %s, got %v", driverID, d.DriverID)
	}

	if d.VehicleID == nil || *d.VehicleID != vehicleID {
		t.Errorf("expected vehicle %s, got %v", vehicleID, d.VehicleID)
	}
}

func TestDispatch_ConvertToTrip(t *testing.T) {
	d := &dispatch.Dispatch{
		ID:     types.DispatchID("dsp-1"),
		Status: dispatch.DispatchDraft,
	}

	// Converting draft without assignment should fail
	err := d.ConvertToTrip(types.TripID("trp-1"))
	if err == nil {
		t.Fatalf("expected error when converting draft dispatch without assignments")
	}

	driverID := types.DriverID("drv-1")
	vehicleID := types.VehicleID("vh-1")
	_ = d.AssignResources(driverID, vehicleID)

	tripID := types.TripID("trp-1")
	err = d.ConvertToTrip(tripID)
	if err != nil {
		t.Fatalf("expected successful trip conversion, got %v", err)
	}

	if d.Status != dispatch.DispatchConverted {
		t.Errorf("expected status %s, got %s", dispatch.DispatchConverted, d.Status)
	}

	if d.TripID == nil || *d.TripID != tripID {
		t.Errorf("expected trip ID %s, got %v", tripID, d.TripID)
	}
}

func TestDispatch_Cancel(t *testing.T) {
	d := &dispatch.Dispatch{
		ID:     types.DispatchID("dsp-cancel"),
		Status: dispatch.DispatchDraft,
	}

	if err := d.Cancel(); err != nil {
		t.Fatalf("expected successful cancellation of draft dispatch: %v", err)
	}
	if d.Status != dispatch.DispatchCancelled {
		t.Fatalf("expected status cancelled, got %s", d.Status)
	}

	dConverted := &dispatch.Dispatch{
		ID:     types.DispatchID("dsp-conv"),
		Status: dispatch.DispatchConverted,
	}
	if err := dConverted.Cancel(); err == nil {
		t.Fatalf("expected error when cancelling converted dispatch")
	}
}
