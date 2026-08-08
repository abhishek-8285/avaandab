package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	invoiceApp "transport-app/internal/invoice/application"
	"transport-app/internal/domain/types"
	"transport-app/internal/domain"
	"transport-app/internal/middleware"
	"transport-app/internal/service"
	clock "transport-app/internal/shared/clock"
	id "transport-app/internal/shared/id"
	uow "transport-app/internal/shared/uow"
	tripapp "transport-app/internal/trip/application"
	tripagg "transport-app/internal/trip/domain/aggregate"
)

// TripHandlers handles trip management.
type TripHandlers struct {
	*App
	createUC          *tripapp.CreateTripUseCase
	startUC           *tripapp.StartTripUseCase
	reachPickupUC     *tripapp.ReachPickupUseCase
	startTransitUC    *tripapp.StartTransitUseCase
	deliverUC         *tripapp.DeliverUseCase
	completeUC        *tripapp.CompleteTripUseCase
	cancelUC          *tripapp.CancelTripUseCase
	getUC             *tripapp.GetTripUseCase
	listUC            *tripapp.ListTripsUseCase
	scheduleUC        *tripapp.ScheduleTripUseCase
	assignDriverUC    *tripapp.AssignDriverUseCase
	assignVehicleUC   *tripapp.AssignVehicleUseCase
	generateInvoiceUC *invoiceApp.GenerateInvoiceUseCase
}

func (h *TripHandlers) init() {
	if h.createUC == nil {
		uowImpl := uow.NewSQLUnitOfWork(h.DB)
		clockImpl := clock.NewRealClock()
		idGenImpl := id.NewUUIDGenerator()

		h.createUC = tripapp.NewCreateTripUseCase(uowImpl, idGenImpl, clockImpl)
		h.startUC = tripapp.NewStartTripUseCase(uowImpl, clockImpl)
		h.reachPickupUC = tripapp.NewReachPickupUseCase(uowImpl, clockImpl)
		h.startTransitUC = tripapp.NewStartTransitUseCase(uowImpl, clockImpl)
		h.deliverUC = tripapp.NewDeliverUseCase(uowImpl, clockImpl)
		h.completeUC = tripapp.NewCompleteTripUseCase(uowImpl, clockImpl)
		h.cancelUC = tripapp.NewCancelTripUseCase(uowImpl, clockImpl)
		h.getUC = tripapp.NewGetTripUseCase(uowImpl)
		h.listUC = tripapp.NewListTripsUseCase(uowImpl)
		h.scheduleUC = tripapp.NewScheduleTripUseCase(uowImpl, clockImpl)
		h.assignDriverUC = tripapp.NewAssignDriverUseCase(uowImpl, clockImpl)
		h.assignVehicleUC = tripapp.NewAssignVehicleUseCase(uowImpl, clockImpl)
		h.generateInvoiceUC = invoiceApp.NewGenerateInvoiceUseCase(uowImpl, idGenImpl, clockImpl)
	}
}

func (h *TripHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "delete")).Post("/{id}/delete", h.Delete)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/schedule", h.Schedule)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "assign")).Post("/{id}/assign-driver", h.AssignDriver)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "assign")).Post("/{id}/assign-vehicle", h.AssignVehicle)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/start", h.StartTrip)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/reach-pickup", h.ReachPickup)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/in-transit", h.StartTransit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/deliver", h.Deliver)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/complete", h.CompleteTrip)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "cancel")).Post("/{id}/cancel", h.CancelTrip)
}

func (h *TripHandlers) List(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	res, err := h.listUC.Execute(r.Context(), tripapp.ListTripsQuery{
		TenantID: "1",
		Page:     pp.Page,
		Limit:    pp.Limit,
		Search:   pp.Query,
		Status:   pp.Status,
	})
	if err != nil {
		fmt.Printf("[Trips Error] Failed to list trips: %v\n", err)
		http.Error(w, "Failed to load trips: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, res.Total, "/trips")

	if isDatastarRequest(r) {
		h.renderFragment(w, "trip_list.html", map[string]interface{}{
			"Trips":        res.Trips,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, "trip_list.html", PageData{
		Title: "Trips",
		User:  session,
		Extra: map[string]interface{}{"Trips": res.Trips, "Pagination": pd, "Query": pp.Query, "StatusFilter": pp.Status},
	})
}

func (h *TripHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	bookingID := r.URL.Query().Get("booking")
	drivers, _, _ := h.Services.Drivers.ListDrivers(r.Context(), "", "available", 1000, 0)
	vehicles, _, _ := h.Services.Vehicles.ListVehicles(r.Context(), "", "available", 1000, 0)
	routes, _, _ := h.Services.Routes.ListRoutes(r.Context(), "", 1000, 0)
	h.renderForm(w, r, "trip_edit.html", PageData{
		Title: "New Trip",
		User:  session,
		Extra: map[string]interface{}{"Drivers": drivers, "Vehicles": vehicles, "Routes": routes, "SelectedBookingID": bookingID},
	})
}

func (h *TripHandlers) Create(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var bookingID *string
	if b := r.PostFormValue("booking_id"); b != "" {
		bookingID = &b
	}

	departureTime, err := time.Parse("2006-01-02T15:04", r.PostFormValue("departure_time"))
	if err != nil {
		departureTime, err = time.Parse("2006-01-02 15:04", r.PostFormValue("departure_time"))
		if err != nil {
			departureTime = time.Now()
		}
	}

	id, err := h.createUC.Execute(r.Context(), tripapp.CreateTripCommand{
		TenantID:      "1",
		BookingID:     bookingID,
		RouteID:       r.PostFormValue("route_id"),
		DepartureTime: departureTime,
		Remarks:       r.PostFormValue("remarks"),
	})
	if err != nil {
		h.renderForm(w, r, "trip_edit.html", PageData{Title: "New Trip", FlashError: err.Error()})
		return
	}

	// Assign driver/vehicle if supplied in form post
	if dID := r.PostFormValue("driver_id"); dID != "" {
		_ = h.assignDriverUC.Execute(r.Context(), tripapp.AssignDriverCommand{
			TripID:   id,
			DriverID: dID,
			TenantID: "1",
		})
	}
	if vID := r.PostFormValue("vehicle_id"); vID != "" {
		_ = h.assignVehicleUC.Execute(r.Context(), tripapp.AssignVehicleCommand{
			TripID:    id,
			VehicleID: vID,
			TenantID:  "1",
		})
	}

	http.Redirect(w, r, "/trips", http.StatusSeeOther)
}

func (h *TripHandlers) View(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	id := chi.URLParam(r, "id")
	trip, err := h.getUC.Execute(r.Context(), tripapp.GetTripQuery{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		h.renderError(w, http.StatusNotFound, "Trip Not Found", fmt.Sprintf("No trip found with ID %q.", id), session)
		return
	}

	var availableDrivers, availableVehicles interface{}
	if trip.DriverID == nil || trip.VehicleID == nil {
		availableDrivers, _, _ = h.Services.Drivers.ListDrivers(r.Context(), "", "available", 1000, 0)
		availableVehicles, _, _ = h.Services.Vehicles.ListVehicles(r.Context(), "", "available", 1000, 0)
	}

	h.renderPage(w, "trip_view.html", PageData{
		Title: "View Trip",
		Extra: map[string]interface{}{
			"Trip":              trip,
			"AvailableDrivers":  availableDrivers,
			"AvailableVehicles": availableVehicles,
		},
	})
}

func (h *TripHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	session, _ := h.getUserFromContext(r)
	trip, err := h.getUC.Execute(r.Context(), tripapp.GetTripQuery{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}
	drivers, _, _ := h.Services.Drivers.ListDrivers(r.Context(), "", "", 1000, 0)
	vehicles, _, _ := h.Services.Vehicles.ListVehicles(r.Context(), "", "", 1000, 0)
	routes, _, _ := h.Services.Routes.ListRoutes(r.Context(), "", 1000, 0)

	var selDriverID, selVehicleID string
	if trip.DriverID != nil {
		selDriverID = *trip.DriverID
	}
	if trip.VehicleID != nil {
		selVehicleID = *trip.VehicleID
	}

	h.renderForm(w, r, "trip_edit.html", PageData{
		Title: "Edit Trip",
		User:  session,
		Extra: map[string]interface{}{
			"Trip":              trip,
			"Drivers":           drivers,
			"Vehicles":          vehicles,
			"Routes":            routes,
			"SelectedDriverID":  selDriverID,
			"SelectedVehicleID": selVehicleID,
		},
	})
}

func (h *TripHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := domain.TripID(chi.URLParam(r, "id"))
	var driverID *domain.DriverID
	if d := r.PostFormValue("driver_id"); d != "" {
		did := domain.DriverID(d)
		driverID = &did
	}
	var vehicleID *domain.VehicleID
	if v := r.PostFormValue("vehicle_id"); v != "" {
		vid := domain.VehicleID(v)
		vehicleID = &vid
	}

	_, err := h.Services.Trips.UpdateTrip(r.Context(), id, service.CreateTripRequest{
		RouteID:       domain.RouteID(r.PostFormValue("route_id")),
		DriverID:      driverID,
		VehicleID:     vehicleID,
		DepartureTime: r.PostFormValue("departure_time"),
		ArrivalTime:   r.PostFormValue("arrival_time"),
		Remarks:       r.PostFormValue("remarks"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id.String(), http.StatusSeeOther)
}

func (h *TripHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := domain.TripID(chi.URLParam(r, "id"))
	if err := h.Services.Trips.DeleteTrip(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips", http.StatusSeeOther)
}

func (h *TripHandlers) Schedule(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.scheduleUC.Execute(r.Context(), tripapp.ScheduleTripCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (h *TripHandlers) AssignDriver(w http.ResponseWriter, r *http.Request) {
	h.init()
	tripID := chi.URLParam(r, "id")
	driverID := r.FormValue("driver_id")

	err := h.assignDriverUC.Execute(r.Context(), tripapp.AssignDriverCommand{
		TripID:   tripagg.TripID(tripID),
		DriverID: driverID,
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (h *TripHandlers) AssignVehicle(w http.ResponseWriter, r *http.Request) {
	h.init()
	tripID := chi.URLParam(r, "id")
	vehicleID := r.FormValue("vehicle_id")

	err := h.assignVehicleUC.Execute(r.Context(), tripapp.AssignVehicleCommand{
		TripID:    tripagg.TripID(tripID),
		VehicleID: vehicleID,
		TenantID:  "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (h *TripHandlers) StartTrip(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.startUC.Execute(r.Context(), tripapp.StartTripCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (h *TripHandlers) ReachPickup(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.reachPickupUC.Execute(r.Context(), tripapp.ReachPickupCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (h *TripHandlers) StartTransit(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.startTransitUC.Execute(r.Context(), tripapp.StartTransitCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (h *TripHandlers) Deliver(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.deliverUC.Execute(r.Context(), tripapp.DeliverCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (h *TripHandlers) CompleteTrip(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.completeUC.Execute(r.Context(), tripapp.CompleteTripCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Auto-generate invoice upon trip completion if associated with a booking
	if trip, err := h.getUC.Execute(r.Context(), tripapp.GetTripQuery{TripID: tripagg.TripID(id), TenantID: "1"}); err == nil && trip.BookingID != nil {
		bookingID := types.BookingID(*trip.BookingID)
		if booking, err := h.Services.Bookings.GetBooking(r.Context(), bookingID); err == nil {
			subtotal := booking.Price
			tax := subtotal * 0.18 // 18% GST standard rate
			total := subtotal + tax
			tripIDStr := string(trip.ID)
			_, _ = h.generateInvoiceUC.Execute(r.Context(), invoiceApp.GenerateInvoiceCommand{
				TenantID:   "1",
				BookingID:  string(booking.ID),
				CustomerID: string(booking.CustomerID),
				TripID:     &tripIDStr,
				Subtotal:   subtotal,
				Tax:        tax,
				Discount:   0,
				Total:      total,
			})
		}
	}

	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (h *TripHandlers) CancelTrip(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	err := h.cancelUC.Execute(r.Context(), tripapp.CancelTripCommand{
		TripID:   tripagg.TripID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}
