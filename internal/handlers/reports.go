package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
	"transport-app/internal/repository"
)

// ReportHandlers handles report-related requests.
type ReportHandlers struct {
	*App
}

func (h *ReportHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/", h.Index)
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/revenue", h.RevenueReport)
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/trips", h.TripReport)
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/drivers", h.DriverReport)
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/vehicles", h.VehicleReport)
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/customers", h.CustomerReport)
	r.With(middleware.ResourcePermission(h.AuthSrv, "reports", "read")).Get("/pending-payments", h.PendingPaymentsReport)
}

func (h *ReportHandlers) Index(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	h.renderPage(w, "reports_index.html", PageData{
		Title: "Reports",
		User:  session,
	})
}

func (h *ReportHandlers) RevenueReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	monthlyRev, _ := h.Services.Payments.GetMonthlyRevenue(r.Context())
	totalRev, _ := h.Services.Payments.GetTotalRevenue(r.Context())

	h.renderPage(w, "report_revenue.html", PageData{
		Title: "Revenue Report",
		User:  session,
		Extra: map[string]interface{}{
			"MonthlyRevenue": monthlyRev,
			"TotalRevenue":   totalRev,
		},
	})
}

func (h *ReportHandlers) TripReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	trips, total, err := h.Services.Trips.ListTrips(r.Context(), pp.Query, pp.Status, pp.Limit, pp.Offset)
	if err != nil {
		trips = []repository.TripWithJoins{}
		total = 0
	}

	today := time.Now().Format("2006-01-02")
	rawCounts, _ := h.Services.Dashboard.GetTodayTripsSummary(r.Context(), today)
	statusCounts := make(map[string]int64)
	for k, v := range rawCounts {
		statusCounts[string(k)] = v
	}

	pd := newPaginationData(pp, total, "/reports/trips")

	h.renderPage(w, "report_trips.html", PageData{
		Title: "Trip Performance Report",
		User:  session,
		Extra: map[string]interface{}{
			"Trips":        trips,
			"TotalTrips":   total,
			"StatusCounts": statusCounts,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		},
	})
}

func (h *ReportHandlers) DriverReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	drivers, total, err := h.Services.Drivers.ListDrivers(r.Context(), pp.Query, pp.Status, pp.Limit, pp.Offset)
	if err != nil {
		drivers = []domain.Driver{}
		total = 0
	}

	availableCount, _ := h.Services.Dashboard.GetAvailableDriversForDashboard(r.Context())

	pagination := newPaginationData(pp, total, "/reports/drivers")

	h.renderPage(w, "report_drivers.html", PageData{
		Title: "Driver Roster Report",
		User:  session,
		Extra: map[string]interface{}{
			"Drivers":          drivers,
			"TotalDrivers":     total,
			"AvailableDrivers": availableCount,
			"Pagination":       pagination,
			"Query":            pp.Query,
			"StatusFilter":     pp.Status,
		},
	})
}

func (h *ReportHandlers) VehicleReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	vehicles, total, err := h.Services.Vehicles.ListVehicles(r.Context(), pp.Query, pp.Status, pp.Limit, pp.Offset)
	if err != nil {
		vehicles = []domain.Vehicle{}
		total = 0
	}

	availableCount, _ := h.Services.Dashboard.GetAvailableVehiclesForDashboard(r.Context())

	pagination := newPaginationData(pp, total, "/reports/vehicles")

	h.renderPage(w, "report_vehicles.html", PageData{
		Title: "Vehicle Fleet Report",
		User:  session,
		Extra: map[string]interface{}{
			"Vehicles":          vehicles,
			"TotalVehicles":     total,
			"AvailableVehicles": availableCount,
			"Pagination":        pagination,
			"Query":             pp.Query,
			"StatusFilter":      pp.Status,
		},
	})
}

func (h *ReportHandlers) CustomerReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	customers, total, err := h.Services.Customers.ListCustomers(r.Context(), pp.Query, pp.Limit, pp.Offset)
	if err != nil {
		customers = []domain.Customer{}
		total = 0
	}
	pagination := newPaginationData(pp, total, "/reports/customers")

	h.renderPage(w, "report_customers.html", PageData{
		Title: "Customer Directory Report",
		User:  session,
		Extra: map[string]interface{}{
			"Customers":      customers,
			"TotalCustomers": total,
			"Pagination":     pagination,
			"Query":          pp.Query,
		},
	})
}

func (h *ReportHandlers) PendingPaymentsReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	pending, err := h.Services.Invoices.GetPendingInvoices(r.Context())
	if err != nil {
		pending = []repository.InvoiceWithJoins{}
	}

	var totalOutstanding float64
	for _, inv := range pending {
		totalOutstanding += inv.Total
	}

	h.renderPage(w, "report_pending_payments.html", PageData{
		Title: "Outstanding Payments Audit Report",
		User:  session,
		Extra: map[string]interface{}{
			"Invoices":         pending,
			"TotalInvoices":    len(pending),
			"TotalOutstanding": totalOutstanding,
		},
	})
}
