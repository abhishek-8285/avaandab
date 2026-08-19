package handlers

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/auth"
	"transport-app/internal/booking/application"
	"transport-app/internal/domain"
	driverapp "transport-app/internal/driver/application"
	"transport-app/internal/ewaybill"
	invoiceapp "transport-app/internal/invoice/application"
	paymentapp "transport-app/internal/payment/application"
	tripapp "transport-app/internal/trip/application"
)

// mockAuthSvc provides a mock AuthorizationService for template testing.
type mockAuthSvc struct {
	allowed map[string]bool
}

func (m *mockAuthSvc) Can(userID, resource, action string) bool {
	if m.allowed == nil {
		return true
	}
	key := userID + ":" + resource + ":" + action
	return m.allowed[key]
}
func (m *mockAuthSvc) Reload() error                            { return nil }
func (m *mockAuthSvc) AddRoleForUser(userID, role string) error { return nil }
func (m *mockAuthSvc) DeleteRolesForUser(userID string) error   { return nil }

func TestAllTemplatesRenderCleanly(t *testing.T) {
	// Change working directory to project root if running from internal/handlers
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "handlers" {
		_ = os.Chdir("../..")
	}

	tmpl, err := parseTemplates(&mockAuthSvc{})
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

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
		CGST:          90.00,
		SGST:          90.00,
		IGST:          0.00,
		IRN:           "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		IRNAckNo:      "ACK-1001",
		IRNAckDate:    "2026-08-19",
		SignedQR:      "data:image/png;base64,sample",
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
		{"trip_list_table.html", map[string]interface{}{
			"Trips":      []tripapp.TripResponseDTO{sampleTripDTO},
			"Pagination": dummyPagination,
		}},
		{"booking_list.html", map[string]interface{}{
			"Bookings":     []application.BookingResponseDTO{sampleBookingDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"booking_list_table.html", map[string]interface{}{
			"Bookings":   []application.BookingResponseDTO{sampleBookingDTO},
			"Pagination": dummyPagination,
		}},
		{"vehicle_list.html", map[string]interface{}{
			"Vehicles":     []domain.Vehicle{{ID: "veh-1", RegistrationNumber: "KA-01-1234", VehicleNumber: "V1", VehicleType: domain.VehicleTypeTruck, Capacity: 5000, FuelType: domain.FuelTypeDiesel, Status: domain.VehicleAvailable}},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"vehicle_list_table.html", map[string]interface{}{
			"Vehicles":   []domain.Vehicle{{ID: "veh-1", RegistrationNumber: "KA-01-1234", VehicleNumber: "V1", VehicleType: domain.VehicleTypeTruck, Capacity: 5000, FuelType: domain.FuelTypeDiesel, Status: domain.VehicleAvailable}},
			"Pagination": dummyPagination,
		}},
		{"customer_list.html", map[string]interface{}{
			"Customers":  []domain.Customer{{ID: "c-1", Name: "Customer 1", Phone: "9999999999"}},
			"Pagination": dummyPagination,
			"Query":      "",
		}},
		{"customer_list_table.html", map[string]interface{}{
			"Customers":  []domain.Customer{{ID: "c-1", Name: "Customer 1", Phone: "9999999999"}},
			"Pagination": dummyPagination,
		}},
		{"invoice_list.html", map[string]interface{}{
			"Invoices":     []invoiceapp.InvoiceResponseDTO{sampleInvoiceDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"invoice_list_table.html", map[string]interface{}{
			"Invoices":   []invoiceapp.InvoiceResponseDTO{sampleInvoiceDTO},
			"Pagination": dummyPagination,
		}},
		{"payment_list.html", map[string]interface{}{
			"Payments":   []paymentapp.PaymentResponseDTO{samplePaymentDTO},
			"Pagination": dummyPagination,
			"Method":     "",
		}},
		{"payment_list_table.html", map[string]interface{}{
			"Payments":   []paymentapp.PaymentResponseDTO{samplePaymentDTO},
			"Pagination": dummyPagination,
		}},
		{"driver_list.html", map[string]interface{}{
			"Drivers":      []driverapp.DriverResponseDTO{sampleDriverDTO},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"driver_list_table.html", map[string]interface{}{
			"Drivers":    []driverapp.DriverResponseDTO{sampleDriverDTO},
			"Pagination": dummyPagination,
		}},
		{"user_list.html", map[string]interface{}{
			"Users":        []domain.User{{ID: "u-1", Name: "User 1", Email: "u1@example.com", Role: domain.Role{Name: domain.RoleAdmin}, Status: domain.UserStatusActive}},
			"Pagination":   dummyPagination,
			"Query":        "",
			"StatusFilter": "",
		}},
		{"user_list_table.html", map[string]interface{}{
			"Users":      []domain.User{{ID: "u-1", Name: "User 1", Email: "u1@example.com", Role: domain.Role{Name: domain.RoleAdmin}, Status: domain.UserStatusActive}},
			"Pagination": dummyPagination,
		}},
		{"route_list.html", map[string]interface{}{
			"Routes":     []domain.Route{{ID: "r-1", Source: "Mumbai", Destination: "Pune", Distance: 150, EstimatedHours: 3.5, StandardFare: 4500}},
			"Pagination": dummyPagination,
			"Query":      "",
		}},
		{"route_list_table.html", map[string]interface{}{
			"Routes":     []domain.Route{{ID: "r-1", Source: "Mumbai", Destination: "Pune", Distance: 150, EstimatedHours: 3.5, StandardFare: 4500}},
			"Pagination": dummyPagination,
		}},
		{"audit_logs_list.html", map[string]interface{}{
			"AuditLogs":  []map[string]interface{}{{"CreatedAt": time.Now(), "UserName": "Admin", "Action": "create", "TableName": "vehicles", "RecordID": "veh-1"}},
			"Pagination": dummyPagination,
			"Query":      "",
		}},
		{"audit_logs_list_table.html", map[string]interface{}{
			"AuditLogs":  []map[string]interface{}{{"CreatedAt": time.Now(), "UserName": "Admin", "Action": "create", "TableName": "vehicles", "RecordID": "veh-1"}},
			"Pagination": dummyPagination,
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
		{"dashboard.html", buildTemplateData(PageData{
			Title: "Dashboard",
			Extra: map[string]interface{}{
				"DashboardVariant": "B",
				"ChartData": map[string]interface{}{
					"variant":       "B",
					"statusCounts":  map[string]int64{"scheduled": 3, "completed": 1},
					"revenueByDay":  []map[string]interface{}{{"Day": "2026-08-18", "Total": 1200.0}},
					"bookingsByDay": []map[string]interface{}{{"Day": "2026-08-18", "Count": 4}},
				},
				"Stats": map[string]interface{}{
					"TodaysTripsCount":       4,
					"ActiveTripsCount":       3,
					"CompletedTripsCount":    1,
					"CancelledTripsCount":    0,
					"AvailableVehiclesCount": 5,
					"AvailableDriversCount":  3,
					"PendingPaymentsCount":   2,
					"MonthlyRevenue":         15000.0,
					"DeltaYesterday":         1,
					"OverdueTrips":           []tripapp.TripResponseDTO{sampleTripDTO},
					"IdleVehicles": []map[string]interface{}{{
						"RegistrationNumber": "KA-01-HH-1234",
						"VehicleType":        "truck",
						"UpdatedAt":          time.Now(),
					}},
					"UpcomingTrips":  []tripapp.TripResponseDTO{sampleTripDTO},
					"RecentBookings": []application.BookingResponseDTO{sampleBookingDTO},
					"RecentPayments": []paymentapp.PaymentResponseDTO{samplePaymentDTO},
				},
			},
		})},
		{"invoice_line_items.html", buildTemplateData(PageData{
			Title: "Invoice Line Items",
			Extra: map[string]interface{}{
				"Invoice":      sampleInvoiceDTO,
				"Customer":     map[string]string{"Name": "ACME", "GST": "27AAACP0000M1Z9", "State": "27"},
				"IsIntraState": true,
				"LineItems": []LineItemRecord{{
					ID:           "li-1",
					InvoiceID:    "inv-1",
					HSNSACCode:   "996511",
					Description:  "Freight",
					Unit:         "NOS",
					Quantity:     1,
					Rate:         1000,
					TaxableValue: 1000,
					CGSTRate:     9,
					SGSTRate:     9,
					CGSTAmount:   90,
					SGSTAmount:   90,
					Total:        1180,
				}},
				"HSNCodes": []HSNSACRecord{{Code: "996511", Description: "Freight", Type: "SAC", Rate: 18}},
				"TaxSplit": TaxSplitSummary{TaxableTotal: 1000, IsIntraState: true, Cgst: 90, Sgst: 90, Total: 1180},
			},
		})},
		{"ewaybill_index.html", buildTemplateData(PageData{
			Title: "E-Way Bills",
			Extra: map[string]interface{}{
				"Stats": EWBStats{Total: 1, Active: 1},
				"EWayBills": []EWBListItem{{
					ID:             "ewb-1",
					TripID:         "trip-1",
					TripNumber:     "TRP-001",
					EwbNumber:      "EWB-12345",
					VehicleNumber:  "MH12AB1234",
					FromPlace:      "Mumbai",
					ToPlace:        "Pune",
					GoodsValue:     60000,
					Status:         "active",
					ValidUntil:     time.Now().Add(24 * time.Hour),
					ExtensionCount: 0,
					CreatedAt:      time.Now(),
				}},
				"Trips": []TripOption{{ID: "trip-1", TripNumber: "TRP-001", Source: "Mumbai", Destination: "Pune"}},
			},
		})},
		{"ewaybill_detail.html", buildTemplateData(PageData{
			Title: "EWB Detail",
			Extra: map[string]interface{}{
				"EWayBill": &ewaybill.EWayBillRecord{
					ID:             "ewb-1",
					TripID:         "trip-1",
					EwbNumber:      "EWB-12345",
					FromPlace:      "Mumbai",
					FromStateCode:  "27",
					ToPlace:        "Pune",
					ToStateCode:    "27",
					GoodsValue:     60000,
					Distance:       150,
					DocType:        "INV",
					DocNo:          "INV-001",
					DocDate:        "2026-08-19",
					Status:         "active",
					GenMode:        "MANUAL",
					ValidUntil:     time.Now().Add(24 * time.Hour),
					ExtensionCount: 0,
					CreatedAt:      time.Now(),
				},
				"Events": []EWBEventRecord{{
					ID:        "ev-1",
					EwbNumber: "EWB-12345",
					TripID:    "trip-1",
					EventType: "PART_A_GENERATED",
					Payload:   "{}",
					CreatedBy: "system",
					CreatedAt: time.Now(),
				}},
			},
		})},
		{"map.html", buildTemplateData(PageData{
			Title: "Live Fleet Map",
			Extra: map[string]interface{}{
				"MapAssets": true,
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

func TestRenderFragment_ListTables(t *testing.T) {
	tmpl, err := parseTemplates(&mockAuthSvc{})
	require.NoError(t, err)

	app := &App{Templates: tmpl}
	tables := []string{
		"vehicle_list_table.html",
		"customer_list_table.html",
		"invoice_list_table.html",
		"driver_list_table.html",
		"user_list_table.html",
		"route_list_table.html",
		"payment_list_table.html",
		"trip_list_table.html",
		"audit_logs_list_table.html",
		"booking_list_table.html",
	}

	for _, tbl := range tables {
		t.Run(tbl, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.renderFragment(w, tbl, map[string]interface{}{})
			assert.Equal(t, 200, w.Code)
			assert.NotEmpty(t, w.Body.String())
		})
	}
}

func TestCanTemplateFunc(t *testing.T) {
	mockAuth := &mockAuthSvc{
		allowed: map[string]bool{
			"admin-1:users:read":       true,
			"dispatcher-1:trips:write": true,
		},
	}

	canFunc := func(user interface{}, resource string, action string) bool {
		if user == nil {
			return false
		}
		var uid string
		switch u := user.(type) {
		case *auth.SessionData:
			if u == nil {
				return false
			}
			uid = u.UserID
		case auth.SessionData:
			uid = u.UserID
		case string:
			uid = u
		default:
			return false
		}
		return mockAuth.Can(uid, resource, action)
	}

	funcMap := template.FuncMap{"can": canFunc}

	// Case 1: SessionData with allowed perm
	var buf1 bytes.Buffer
	t1, err := template.New("test1").Funcs(funcMap).Parse(`{{if can .User "users" "read"}}ALLOWED{{else}}DENIED{{end}}`)
	require.NoError(t, err)
	err = t1.Execute(&buf1, map[string]interface{}{
		"User": &auth.SessionData{UserID: "admin-1", Role: "admin"},
	})
	require.NoError(t, err)
	assert.Equal(t, "ALLOWED", buf1.String())

	// Case 2: SessionData with denied perm
	var buf2 bytes.Buffer
	t2, err := template.New("test2").Funcs(funcMap).Parse(`{{if can .User "users" "write"}}ALLOWED{{else}}DENIED{{end}}`)
	require.NoError(t, err)
	err = t2.Execute(&buf2, map[string]interface{}{
		"User": &auth.SessionData{UserID: "admin-1", Role: "admin"},
	})
	require.NoError(t, err)
	assert.Equal(t, "DENIED", buf2.String())

	// Case 3: string UserID with allowed perm
	var buf3 bytes.Buffer
	t3, err := template.New("test3").Funcs(funcMap).Parse(`{{if can .User "trips" "write"}}ALLOWED{{else}}DENIED{{end}}`)
	require.NoError(t, err)
	err = t3.Execute(&buf3, map[string]interface{}{
		"User": "dispatcher-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "ALLOWED", buf3.String())
}

func TestLayoutNoHardcodedAdminRole(t *testing.T) {
	content, err := os.ReadFile("internal/templates/layout.html")
	require.NoError(t, err)

	assert.NotContains(t, string(content), `eq .User.Role "admin"`, "layout.html should not contain hardcoded role check")
}
