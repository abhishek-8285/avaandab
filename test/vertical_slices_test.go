package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bookingApp "transport-app/internal/booking/application"
	invoiceApp "transport-app/internal/invoice/application"
	paymentApp "transport-app/internal/payment/application"
	paymentAggregate "transport-app/internal/payment/domain/aggregate"
	tripApp "transport-app/internal/trip/application"

	"transport-app/internal/shared/clock"
	"transport-app/internal/shared/id"
	"transport-app/internal/shared/uow"
)


// ─────────────────────────────────────────────────────────────────────────────
// Sprint 1: Booking
// ─────────────────────────────────────────────────────────────────────────────

func TestSprint1_CreateBooking(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	route, err := svc.Routes.CreateRoute(ctx, "Mumbai", "Delhi", 1400, 24, 15000, "")
	require.NoError(t, err)

	createUC := bookingApp.NewCreateBookingUseCase(sqlUoW, idGen, realClock)
	id, err := createUC.Execute(ctx, bookingApp.CreateBookingCommand{
		TenantID:    "1",
		CustomerID:  "cust-1",
		RouteID:     string(route.ID),
		PickupDate:  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		VehicleType: "Truck",
		Passengers:  2,
		Price:       15000,
		Notes:       "fragile",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestSprint1_ConfirmAndCancelBooking(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	route, _ := svc.Routes.CreateRoute(ctx, "A", "B", 100, 2, 5000, "")

	createUC := bookingApp.NewCreateBookingUseCase(sqlUoW, idGen, realClock)
	confirmUC := bookingApp.NewConfirmBookingUseCase(sqlUoW, realClock)
	cancelUC := bookingApp.NewCancelBookingUseCase(sqlUoW, realClock)
	getUC := bookingApp.NewGetBookingUseCase(sqlUoW)

	bookingID, err := createUC.Execute(ctx, bookingApp.CreateBookingCommand{
		TenantID:    "1",
		CustomerID:  "cust-1",
		RouteID:     string(route.ID),
		PickupDate:  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		VehicleType: "Van",
		Passengers:  1,
		Price:       5000,
	})
	require.NoError(t, err)

	// Confirm
	require.NoError(t, confirmUC.Execute(ctx, bookingApp.ConfirmBookingCommand{
		BookingID: bookingID,
		TenantID:  "1",
	}))

	res, err := getUC.Execute(ctx, bookingApp.GetBookingQuery{BookingID: bookingID, TenantID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "confirmed", res.Status)

	// Cancel a confirmed booking
	require.NoError(t, cancelUC.Execute(ctx, bookingApp.CancelBookingCommand{
		BookingID: bookingID,
		TenantID:  "1",
	}))

	res, err = getUC.Execute(ctx, bookingApp.GetBookingQuery{BookingID: bookingID, TenantID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "cancelled", res.Status)
}

func TestSprint1_ListBookings(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	route, _ := svc.Routes.CreateRoute(ctx, "X", "Y", 200, 3, 8000, "")

	createUC := bookingApp.NewCreateBookingUseCase(sqlUoW, idGen, realClock)
	listUC := bookingApp.NewListBookingsUseCase(sqlUoW)

	for i := 0; i < 3; i++ {
		_, err := createUC.Execute(ctx, bookingApp.CreateBookingCommand{
			TenantID:    "1",
			CustomerID:  "cust-x",
			RouteID:     string(route.ID),
			PickupDate:  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			VehicleType: "Bus",
			Passengers:  10,
			Price:       8000,
		})
		require.NoError(t, err)
	}

	res, err := listUC.Execute(ctx, bookingApp.ListBookingsQuery{TenantID: "1", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.Total, int64(3))
}

// ─────────────────────────────────────────────────────────────────────────────
// Sprint 2: Trip
// ─────────────────────────────────────────────────────────────────────────────

func TestSprint2_CreateTripAndLifecycle(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	route, _ := svc.Routes.CreateRoute(ctx, "Mumbai", "Pune", 150, 3, 3000, "")
	driver, _ := svc.Drivers.CreateDriver(ctx, "Ali", "Khan", "111", "", "", "LIC999", "2027-01-01", 3, nil, nil, nil)
	vehicle, _ := svc.Vehicles.CreateVehicle(ctx, "MH-01-XX-1234", "V200", "Truck", 15, "Diesel", "2027-01-01", "2027-01-01", "2027-01-01", "0")

	createUC := tripApp.NewCreateTripUseCase(sqlUoW, idGen, realClock)
	assignDriverUC := tripApp.NewAssignDriverUseCase(sqlUoW, realClock)
	assignVehicleUC := tripApp.NewAssignVehicleUseCase(sqlUoW, realClock)
	startUC := tripApp.NewStartTripUseCase(sqlUoW, realClock)
	completeUC := tripApp.NewCompleteTripUseCase(sqlUoW, realClock)
	getUC := tripApp.NewGetTripUseCase(sqlUoW)

	tripID, err := createUC.Execute(ctx, tripApp.CreateTripCommand{
		TenantID:      "1",
		RouteID:       string(route.ID),
		DepartureTime: time.Now().Add(1 * time.Hour),
		Remarks:       "test trip",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tripID)

	require.NoError(t, assignDriverUC.Execute(ctx, tripApp.AssignDriverCommand{
		TripID:   tripID,
		DriverID: string(driver.ID),
		TenantID: "1",
	}))

	require.NoError(t, assignVehicleUC.Execute(ctx, tripApp.AssignVehicleCommand{
		TripID:    tripID,
		VehicleID: string(vehicle.ID),
		TenantID:  "1",
	}))

	require.NoError(t, startUC.Execute(ctx, tripApp.StartTripCommand{TripID: tripID, TenantID: "1"}))

	trip, err := getUC.Execute(ctx, tripApp.GetTripQuery{TripID: tripID, TenantID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "started", trip.Status)

	require.NoError(t, completeUC.Execute(ctx, tripApp.CompleteTripCommand{TripID: tripID, TenantID: "1"}))

	trip, err = getUC.Execute(ctx, tripApp.GetTripQuery{TripID: tripID, TenantID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "completed", trip.Status)
}

func TestSprint2_CancelTrip(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	route, _ := svc.Routes.CreateRoute(ctx, "C", "D", 100, 2, 2000, "")

	createUC := tripApp.NewCreateTripUseCase(sqlUoW, idGen, realClock)
	cancelUC := tripApp.NewCancelTripUseCase(sqlUoW, realClock)
	getUC := tripApp.NewGetTripUseCase(sqlUoW)

	tripID, err := createUC.Execute(ctx, tripApp.CreateTripCommand{
		TenantID:      "1",
		RouteID:       string(route.ID),
		DepartureTime: time.Now().Add(2 * time.Hour),
	})
	require.NoError(t, err)

	require.NoError(t, cancelUC.Execute(ctx, tripApp.CancelTripCommand{TripID: tripID, TenantID: "1"}))

	trip, err := getUC.Execute(ctx, tripApp.GetTripQuery{TripID: tripID, TenantID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "cancelled", trip.Status)
}

// ─────────────────────────────────────────────────────────────────────────────
// Sprint 3: Invoice
// ─────────────────────────────────────────────────────────────────────────────

func TestSprint3_GenerateAndGetInvoice(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	customer, err := svc.Customers.CreateCustomer(ctx, "Acme Corp", "Acme", "555-9000", "acme@example.com", "", "", "")
	require.NoError(t, err)

	generateUC := invoiceApp.NewGenerateInvoiceUseCase(sqlUoW, idGen, realClock)
	getUC := invoiceApp.NewGetInvoiceUseCase(sqlUoW)

	invID, err := generateUC.Execute(ctx, invoiceApp.GenerateInvoiceCommand{
		TenantID:   "1",
		BookingID:  "bk-001",
		CustomerID: string(customer.ID),
		Subtotal:   10000,
		Tax:        1800,
		Discount:   0,
		Total:      11800,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, invID)

	inv, err := getUC.Execute(ctx, invoiceApp.GetInvoiceQuery{ID: invID, TenantID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "bk-001", inv.BookingID)
	assert.InDelta(t, 11800.0, inv.Total, 0.01)
	assert.Equal(t, "pending", inv.PaymentStatus)
}

func TestSprint3_GenerateInvoice_Idempotent(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	svc := NewTestServices(t, db)
	customer, err := svc.Customers.CreateCustomer(ctx, "Beta Ltd", "Beta", "555-1111", "beta@example.com", "", "", "")
	require.NoError(t, err)

	generateUC := invoiceApp.NewGenerateInvoiceUseCase(sqlUoW, idGen, realClock)

	cmd := invoiceApp.GenerateInvoiceCommand{
		TenantID:   "1",
		BookingID:  "bk-idem",
		CustomerID: string(customer.ID),
		Subtotal:   5000,
		Tax:        900,
		Total:      5900,
	}

	id1, err := generateUC.Execute(ctx, cmd)
	require.NoError(t, err)

	id2, err := generateUC.Execute(ctx, cmd)
	require.NoError(t, err)

	// Same booking → same invoice returned (idempotent)
	assert.Equal(t, id1, id2)
}

// ─────────────────────────────────────────────────────────────────────────────
// Sprint 4: Payment
// ─────────────────────────────────────────────────────────────────────────────

func TestSprint4_RecordPaymentAndGet(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	// First generate an invoice to have a valid invoice ID
	genUC := invoiceApp.NewGenerateInvoiceUseCase(sqlUoW, idGen, realClock)
	invID, err := genUC.Execute(ctx, invoiceApp.GenerateInvoiceCommand{
		TenantID:  "1",
		BookingID: "bk-pay",
		Subtotal:  8000,
		Tax:       1440,
		Total:     9440,
	})
	require.NoError(t, err)

	recordUC := paymentApp.NewRecordPaymentUseCase(sqlUoW, idGen, realClock)
	getUC := paymentApp.NewGetPaymentUseCase(sqlUoW)

	payID, err := recordUC.Execute(ctx, paymentApp.RecordPaymentCommand{
		TenantID:    "1",
		InvoiceID:   string(invID),
		PaymentDate: time.Now(),
		Amount:      9440,
		Method:      paymentAggregate.PaymentMethodCash,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, payID)

	pay, err := getUC.Execute(ctx, paymentApp.GetPaymentQuery{ID: payID, TenantID: "1"})
	require.NoError(t, err)
	assert.InDelta(t, 9440.0, pay.Amount, 0.01)
	assert.Equal(t, "cash", pay.Method)
}

func TestSprint4_RecordPayment_InvalidAmount(t *testing.T) {
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	ctx := context.Background()

	recordUC := paymentApp.NewRecordPaymentUseCase(sqlUoW, idGen, realClock)

	_, err := recordUC.Execute(ctx, paymentApp.RecordPaymentCommand{
		TenantID:  "1",
		InvoiceID: "inv-x",
		Amount:    -100,
		Method:    paymentAggregate.PaymentMethodCash,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than zero")
}
