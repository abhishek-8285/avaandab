package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	bookingapp "transport-app/internal/booking/application"
	bookingagg "transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/middleware"
	"transport-app/internal/shared"
	clock "transport-app/internal/shared/clock"
	id "transport-app/internal/shared/id"
	uow "transport-app/internal/shared/uow"
)

// BookingHandlers handles booking management.
type BookingHandlers struct {
	*App
	createUC   *bookingapp.CreateBookingUseCase
	confirmUC  *bookingapp.ConfirmBookingUseCase
	cancelUC   *bookingapp.CancelBookingUseCase
	getUC      *bookingapp.GetBookingUseCase
	listUC     *bookingapp.ListBookingsUseCase
	updateUC   *bookingapp.UpdateBookingUseCase
	deleteUC   *bookingapp.DeleteBookingUseCase
	completeUC *bookingapp.CompleteBookingUseCase
}

func (h *BookingHandlers) init() {
	if h.createUC == nil {
		uowImpl := uow.NewSQLUnitOfWork(h.DB)
		clockImpl := clock.NewRealClock()
		idGenImpl := id.NewUUIDGenerator()

		h.createUC = bookingapp.NewCreateBookingUseCase(uowImpl, idGenImpl, clockImpl)
		h.confirmUC = bookingapp.NewConfirmBookingUseCase(uowImpl, clockImpl)
		h.cancelUC = bookingapp.NewCancelBookingUseCase(uowImpl, clockImpl)
		h.getUC = bookingapp.NewGetBookingUseCase(uowImpl)
		h.listUC = bookingapp.NewListBookingsUseCase(uowImpl)
		h.updateUC = bookingapp.NewUpdateBookingUseCase(uowImpl)
		h.deleteUC = bookingapp.NewDeleteBookingUseCase(uowImpl)
		h.completeUC = bookingapp.NewCompleteBookingUseCase(uowImpl, clockImpl)
	}
}

func (h *BookingHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "delete")).Post("/{id}/delete", h.Delete)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "approve")).Post("/{id}/confirm", h.Confirm)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "cancel")).Post("/{id}/cancel", h.Cancel)
	r.With(middleware.ResourcePermission(h.AuthSrv, "bookings", "update")).Post("/{id}/complete", h.Complete)
}

func (h *BookingHandlers) List(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	tenantID := shared.TenantIDFromContext(r.Context())

	res, err := h.listUC.Execute(r.Context(), bookingapp.ListBookingsQuery{
		TenantID: tenantID,
		Page:     pp.Page,
		Limit:    pp.Limit,
		Search:   pp.Query,
		Status:   pp.Status,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, res.Total, "/bookings")

	if isDatastarRequest(r) {
		h.renderFragment(w, "booking_list_table.html", map[string]interface{}{
			"Bookings":     res.Bookings,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, "booking_list.html", PageData{
		Title: "Bookings",
		User:  session,
		Extra: map[string]interface{}{"Bookings": res.Bookings, "Pagination": pd, "Query": pp.Query, "StatusFilter": pp.Status},
	})
}

func (h *BookingHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	customers, _, _ := h.Services.Customers.ListCustomers(r.Context(), "", 1000, 0)
	routes, _, _ := h.Services.Routes.ListRoutes(r.Context(), "", 1000, 0)
	h.renderForm(w, r, "booking_edit.html", PageData{
		Title: "New Booking",
		User:  session,
		Extra: map[string]interface{}{"Customers": customers, "Routes": routes},
	})
}

func (h *BookingHandlers) Create(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	customerID := r.PostFormValue("customer_id")
	routeID := r.PostFormValue("route_id")
	passengers, _ := strconv.ParseInt(r.PostFormValue("passengers"), 10, 64)
	cargoWeight, _ := strconv.ParseFloat(r.PostFormValue("cargo_weight"), 64)
	price, _ := strconv.ParseFloat(r.PostFormValue("price"), 64)
	tenantID := shared.TenantIDFromContext(r.Context())

	_, err := h.createUC.Execute(r.Context(), bookingapp.CreateBookingCommand{
		TenantID:    tenantID,
		CustomerID:  customerID,
		RouteID:     routeID,
		PickupDate:  r.PostFormValue("pickup_date"),
		VehicleType: r.PostFormValue("vehicle_type"),
		Passengers:  passengers,
		CargoWeight: &cargoWeight,
		Price:       price,
		Notes:       r.PostFormValue("notes"),
	})
	if err != nil {
		h.renderForm(w, r, "booking_edit.html", PageData{Title: "New Booking", FlashError: err.Error()})
		return
	}

	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

func (h *BookingHandlers) View(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	id := chi.URLParam(r, "id")
	tenantID := shared.TenantIDFromContext(r.Context())

	booking, err := h.getUC.Execute(r.Context(), bookingapp.GetBookingQuery{
		BookingID: bookingagg.BookingID(id),
		TenantID:  tenantID,
	})
	if err != nil {
		h.renderError(w, http.StatusNotFound, "Booking Not Found", fmt.Sprintf("No booking found with ID %q.", id), session)
		return
	}

	h.renderPage(w, "booking_view.html", PageData{
		Title: "View Booking",
		User:  session,
		Extra: map[string]interface{}{"Booking": booking},
	})
}

func (h *BookingHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	session, _ := h.getUserFromContext(r)
	tenantID := shared.TenantIDFromContext(r.Context())

	booking, err := h.getUC.Execute(r.Context(), bookingapp.GetBookingQuery{
		BookingID: bookingagg.BookingID(id),
		TenantID:  tenantID,
	})
	if err != nil {
		h.renderError(w, http.StatusNotFound, "Booking Not Found", fmt.Sprintf("No booking found with ID %q.", id), session)
		return
	}
	customers, _, _ := h.Services.Customers.ListCustomers(r.Context(), "", 1000, 0)
	routes, _, _ := h.Services.Routes.ListRoutes(r.Context(), "", 1000, 0)
	h.renderForm(w, r, "booking_edit.html", PageData{
		Title: "Edit Booking",
		User:  session,
		Extra: map[string]interface{}{"Booking": booking, "Customers": customers, "Routes": routes},
	})
}

func (h *BookingHandlers) Update(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	customerID := r.PostFormValue("customer_id")
	routeID := r.PostFormValue("route_id")
	passengers, _ := strconv.ParseInt(r.PostFormValue("passengers"), 10, 64)
	cargoWeight, _ := strconv.ParseFloat(r.PostFormValue("cargo_weight"), 64)
	price, _ := strconv.ParseFloat(r.PostFormValue("price"), 64)
	tenantID := shared.TenantIDFromContext(r.Context())

	err := h.updateUC.Execute(r.Context(), bookingapp.UpdateBookingCommand{
		BookingID:   bookingagg.BookingID(id),
		TenantID:    tenantID,
		CustomerID:  customerID,
		RouteID:     routeID,
		PickupDate:  r.PostFormValue("pickup_date"),
		VehicleType: r.PostFormValue("vehicle_type"),
		Passengers:  passengers,
		CargoWeight: &cargoWeight,
		Price:       price,
		Notes:       r.PostFormValue("notes"),
	})
	if err != nil {
		booking, _ := h.getUC.Execute(r.Context(), bookingapp.GetBookingQuery{
			BookingID: bookingagg.BookingID(id),
			TenantID:  tenantID,
		})
		customers, _, _ := h.Services.Customers.ListCustomers(r.Context(), "", 1000, 0)
		routes, _, _ := h.Services.Routes.ListRoutes(r.Context(), "", 1000, 0)
		h.renderForm(w, r, "booking_edit.html", PageData{
			Title:      "Edit Booking",
			FlashError: err.Error(),
			Extra:      map[string]interface{}{"Booking": booking, "Customers": customers, "Routes": routes},
		})
		return
	}
	http.Redirect(w, r, "/bookings/"+id, http.StatusSeeOther)
}

func (h *BookingHandlers) Complete(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	tenantID := shared.TenantIDFromContext(r.Context())

	err := h.completeUC.Execute(r.Context(), bookingapp.CompleteBookingCommand{
		BookingID: bookingagg.BookingID(id),
		TenantID:  tenantID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bookings/"+id, http.StatusSeeOther)
}

func (h *BookingHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	tenantID := shared.TenantIDFromContext(r.Context())

	if err := h.deleteUC.Execute(r.Context(), bookingapp.DeleteBookingCommand{
		BookingID: bookingagg.BookingID(id),
		TenantID:  tenantID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

func (h *BookingHandlers) Confirm(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	tenantID := shared.TenantIDFromContext(r.Context())

	err := h.confirmUC.Execute(r.Context(), bookingapp.ConfirmBookingCommand{
		BookingID: bookingagg.BookingID(id),
		TenantID:  tenantID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bookings/"+id, http.StatusSeeOther)
}

func (h *BookingHandlers) Cancel(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	tenantID := shared.TenantIDFromContext(r.Context())

	err := h.cancelUC.Execute(r.Context(), bookingapp.CancelBookingCommand{
		BookingID: bookingagg.BookingID(id),
		TenantID:  tenantID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bookings/"+id, http.StatusSeeOther)
}
