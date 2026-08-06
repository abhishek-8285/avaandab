package handlers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"transport-app/internal/booking/application"
	driverapp "transport-app/internal/driver/application"
	invoiceapp "transport-app/internal/invoice/application"
	paymentapp "transport-app/internal/payment/application"
	tripapp "transport-app/internal/trip/application"
)

// mockAuthSvc provides a mock AuthorizationService for template testing.
type mockAuthSvc struct{}

func (m *mockAuthSvc) Can(userID, resource, action string) bool { return true }
func (m *mockAuthSvc) Reload() error                           { return nil }

func TestAllTemplatesRenderCleanly(t *testing.T) {
	// Change working directory to project root if running from internal/handlers
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "handlers" {
		_ = os.Chdir("../..")
	}

	tmpl := parseTemplates(&mockAuthSvc{})

	sampleTripDTO := tripapp.TripResponseDTO{
		ID:                        "trip-1",
		TripNumber:                "TRIP-001",
		VehicleRegistrationNumber: "KA-01-HH-1234",
		VehicleNumber:             "V1",
		DriverFirstName:           "John",
		DriverLastName:            "Doe",
		RouteSource:               "Source City",
		RouteDestination:          "Dest City",
		DepartureTime:             time.Now(),
		Status:                    "scheduled",
	}

	sampleBookingDTO := application.BookingResponseDTO{
		ID:               "book-1",
		BookingNumber:    "BK-001",
		CustomerName:     "ACME Corp",
		RouteSource:      "Source",
		RouteDestination: "Destination",
		PickupDate:       time.Now(),
		Price:            1000.00,
		Status:           "confirmed",
	}

	sampleInvoiceDTO := invoiceapp.InvoiceResponseDTO{
		ID:            "inv-1",
		InvoiceNumber: "INV-001",
		BookingID:     "book-1",
		CustomerID:    "cust-1",
		Subtotal:      1000.00,
		Tax:           180.00,
		Total:         1180.00,
		PaymentStatus: "pending",
		CreatedAt:     time.Now(),
	}

	samplePaymentDTO := paymentapp.PaymentResponseDTO{
		ID:          "pay-1",
		InvoiceID:   "inv-1",
		PaymentDate: time.Now(),
		Amount:      500.00,
		Method:      "cash",
		CreatedAt:   time.Now(),
	}

	sampleDriverDTO := driverapp.DriverResponseDTO{
		ID:              "drv-1",
		DriverDisplayID: "D1",
		FirstName:       "John",
		LastName:        "Doe",
		Phone:           "555-1234",
		LicenseNumber:   "LIC-001",
		Status:          "available",
	}

	dummyPagination := PaginationData{
		Page:       1,
		PerPage:    10,
		Total:      1,
		TotalPages: 1,
		HasPrev:    false,
		HasNext:    false,
		BasePath:   "/test",
	}

	testCases := []struct {
		name string
		data interface{}
	}{
		{"trip_list.html", map[string]interface{}{
			"Trips":        []tripapp.TripResponseDTO{sampleTripDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"booking_list.html", map[string]interface{}{
			"Bookings":     []application.BookingResponseDTO{sampleBookingDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"invoice_list.html", map[string]interface{}{
			"Invoices":     []invoiceapp.InvoiceResponseDTO{sampleInvoiceDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"payment_list.html", map[string]interface{}{
			"Payments":   []paymentapp.PaymentResponseDTO{samplePaymentDTO},
			"Pagination": dummyPagination,
		}},
		{"driver_list.html", map[string]interface{}{
			"Drivers":      []driverapp.DriverResponseDTO{sampleDriverDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"driver_view.html", buildTemplateData(PageData{
			Title: "View Driver",
			Extra: map[string]interface{}{"Driver": sampleDriverDTO},
		})},
		{"trip_view.html", buildTemplateData(PageData{
			Title: "View Trip",
			Extra: map[string]interface{}{"Trip": sampleTripDTO},
		})},
		{"invoice_view.html", buildTemplateData(PageData{
			Title: "View Invoice",
			Extra: map[string]interface{}{"Invoice": sampleInvoiceDTO},
		})},
		{"payment_view.html", buildTemplateData(PageData{
			Title: "View Payment",
			Extra: map[string]interface{}{"Payment": samplePaymentDTO},
		})},
		{"booking_view.html", buildTemplateData(PageData{
			Title: "View Booking",
			Extra: map[string]interface{}{"Booking": sampleBookingDTO},
		})},
		{"report_trips.html", buildTemplateData(PageData{
			Title: "Trip Report",
			Extra: map[string]interface{}{
				"Trips":        []tripapp.TripResponseDTO{sampleTripDTO},
				"TotalTrips":   int64(1),
				"StatusCounts": map[string]int64{"scheduled": 1, "started": 0, "completed": 0},
				"Pagination":   dummyPagination,
			},
		})},
		{"report_drivers.html", buildTemplateData(PageData{
			Title: "Driver Report",
			Extra: map[string]interface{}{"Drivers": []driverapp.DriverResponseDTO{sampleDriverDTO}, "Pagination": dummyPagination},
		})},
		{"report_vehicles.html", buildTemplateData(PageData{
			Title: "Vehicle Report",
			Extra: map[string]interface{}{"Vehicles": []map[string]interface{}{{"RegistrationNumber": "KA-01-HH-1234", "VehicleNumber": "V1", "VehicleType": "truck", "Capacity": "10 ton", "Status": "available"}}, "Pagination": dummyPagination},
		})},
		{"report_customers.html", buildTemplateData(PageData{
			Title: "Customer Report",
			Extra: map[string]interface{}{"Customers": []map[string]interface{}{{"Name": "ACME Corp", "Company": strPtr("ACME Ltd"), "Email": "acme@example.com", "Phone": "555-0100"}}, "Pagination": dummyPagination},
		})},
		{"report_pending_payments.html", buildTemplateData(PageData{
			Title: "Pending Payments",
			Extra: map[string]interface{}{"Invoices": []invoiceapp.InvoiceResponseDTO{sampleInvoiceDTO}},
		})},
		{"dashboard.html", buildTemplateData(PageData{
			Title: "Dashboard",
			Extra: map[string]interface{}{
				"Stats": map[string]interface{}{
					"TodaysTripsCount":       1,
					"ActiveTripsCount":       1,
					"CompletedTripsCount":    0,
					"CancelledTripsCount":    0,
					"AvailableVehiclesCount": 5,
					"AvailableDriversCount":  3,
					"PendingPaymentsCount":   2,
					"MonthlyRevenue":         15000.0,
					"UpcomingTrips":          []tripapp.TripResponseDTO{sampleTripDTO},
					"RecentBookings":         []application.BookingResponseDTO{sampleBookingDTO},
					"RecentPayments":         []paymentapp.PaymentResponseDTO{samplePaymentDTO},
				},
			},
		})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tTmpl := tmpl.Lookup(tc.name)
			if tTmpl == nil {
				t.Fatalf("Template %s not found", tc.name)
			}
			var buf bytes.Buffer
			if err := tTmpl.Execute(&buf, tc.data); err != nil {
				t.Fatalf("Failed to execute template %s: %v", tc.name, err)
			}
		})
	}
}
