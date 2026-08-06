package handlers

import (
	"net/http"

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

	pd := newPaginationData(pp, total, "/reports/trips")

	h.renderPage(w, "report_trips.html", PageData{
		Title: "Trip Report",
		User:  session,
		Extra: map[string]interface{}{"Trips": trips, "Pagination": pd, "Query": pp.Query},
	})
}

func (h *ReportHandlers) DriverReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pd := parsePaginationParams(r)

	drivers, total, err := h.Services.Drivers.ListDrivers(r.Context(), pd.Query, pd.Status, pd.Limit, pd.Offset)
	if err != nil {
		drivers = []domain.Driver{}
		total = 0
	}
	pagination := newPaginationData(pd, total, "/reports/drivers")

	h.renderPage(w, "report_drivers.html", PageData{
		Title: "Driver Report",
		User:  session,
		Extra: map[string]interface{}{"Drivers": drivers, "Pagination": pagination},
	})
}

func (h *ReportHandlers) VehicleReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pd := parsePaginationParams(r)

	vehicles, total, err := h.Services.Vehicles.ListVehicles(r.Context(), pd.Query, pd.Status, pd.Limit, pd.Offset)
	if err != nil {
		vehicles = []domain.Vehicle{}
		total = 0
	}
	pagination := newPaginationData(pd, total, "/reports/vehicles")

	h.renderPage(w, "report_vehicles.html", PageData{
		Title: "Vehicle Report",
		User:  session,
		Extra: map[string]interface{}{"Vehicles": vehicles, "Pagination": pagination},
	})
}

func (h *ReportHandlers) CustomerReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pd := parsePaginationParams(r)

	customers, total, err := h.Services.Customers.ListCustomers(r.Context(), pd.Query, pd.Limit, pd.Offset)
	if err != nil {
		customers = []domain.Customer{}
		total = 0
	}
	pagination := newPaginationData(pd, total, "/reports/customers")

	h.renderPage(w, "report_customers.html", PageData{
		Title: "Customer Report",
		User:  session,
		Extra: map[string]interface{}{"Customers": customers, "Pagination": pagination},
	})
}

func (h *ReportHandlers) PendingPaymentsReport(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	pending, err := h.Services.Invoices.GetPendingInvoices(r.Context())
	if err != nil {
		pending = []repository.InvoiceWithJoins{}
	}

	h.renderPage(w, "report_pending_payments.html", PageData{
		Title: "Pending Payments",
		User:  session,
		Extra: map[string]interface{}{"Invoices": pending},
	})
}
