package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/shared"
	"transport-app/internal/trip/application"
	"transport-app/internal/trip/domain/aggregate"
)

// APITripHandler handles REST endpoints for the trip vertical slice.
type APITripHandler struct {
	createUC        *application.CreateTripUseCase
	assignDriverUC  *application.AssignDriverUseCase
	assignVehicleUC *application.AssignVehicleUseCase
	scheduleUC      *application.ScheduleTripUseCase
	startUC         *application.StartTripUseCase
	reachPickupUC   *application.ReachPickupUseCase
	startTransitUC  *application.StartTransitUseCase
	deliverUC       *application.DeliverUseCase
	completeUC      *application.CompleteTripUseCase
	cancelUC        *application.CancelTripUseCase
	getUC           *application.GetTripUseCase
	listUC          *application.ListTripsUseCase
}

// NewAPITripHandler constructs an APITripHandler.
func NewAPITripHandler(
	createUC *application.CreateTripUseCase,
	assignDriverUC *application.AssignDriverUseCase,
	assignVehicleUC *application.AssignVehicleUseCase,
	scheduleUC *application.ScheduleTripUseCase,
	startUC *application.StartTripUseCase,
	reachPickupUC *application.ReachPickupUseCase,
	startTransitUC *application.StartTransitUseCase,
	deliverUC *application.DeliverUseCase,
	completeUC *application.CompleteTripUseCase,
	cancelUC *application.CancelTripUseCase,
	getUC *application.GetTripUseCase,
	listUC *application.ListTripsUseCase,
) *APITripHandler {
	return &APITripHandler{
		createUC:        createUC,
		assignDriverUC:  assignDriverUC,
		assignVehicleUC: assignVehicleUC,
		scheduleUC:      scheduleUC,
		startUC:         startUC,
		reachPickupUC:   reachPickupUC,
		startTransitUC:  startTransitUC,
		deliverUC:       deliverUC,
		completeUC:      completeUC,
		cancelUC:        cancelUC,
		getUC:           getUC,
		listUC:          listUC,
	}
}

// Register mounts all trip routes.
func (h *APITripHandler) Register(r chi.Router) {
	r.Route("/api/v1/trips", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Post("/{id}/assign-driver", h.AssignDriver)
		r.Post("/{id}/assign-vehicle", h.AssignVehicle)
		r.Post("/{id}/schedule", h.Schedule)
		r.Post("/{id}/start", h.Start)
		r.Post("/{id}/reach-pickup", h.ReachPickup)
		r.Post("/{id}/in-transit", h.StartTransit)
		r.Post("/{id}/deliver", h.Deliver)
		r.Post("/{id}/complete", h.Complete)
		r.Post("/{id}/cancel", h.Cancel)
	})
}

func (h *APITripHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BookingID     *string `json:"booking_id"`
		RouteID       string  `json:"route_id"`
		DepartureTime string  `json:"departure_time"`
		Remarks       string  `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	depTime, err := time.Parse(time.RFC3339, req.DepartureTime)
	if err != nil {
		http.Error(w, "departure_time must be RFC3339", http.StatusBadRequest)
		return
	}

	id, err := h.createUC.Execute(r.Context(), application.CreateTripCommand{
		TenantID:      shared.TenantIDFromContext(r.Context()),
		BookingID:     req.BookingID,
		RouteID:       req.RouteID,
		DepartureTime: depTime,
		Remarks:       req.Remarks,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

func (h *APITripHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := h.listUC.Execute(r.Context(), application.ListTripsQuery{
		TenantID: shared.TenantIDFromContext(r.Context()),
		Page:     page,
		Limit:    limit,
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"trips": res.Trips, "total": res.Total})
}

func (h *APITripHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.getUC.Execute(r.Context(), application.GetTripQuery{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *APITripHandler) AssignDriver(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.assignDriverUC.Execute(r.Context(), application.AssignDriverCommand{
		TripID:   aggregate.TripID(id),
		DriverID: req.DriverID,
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "driver_assigned"})
}

func (h *APITripHandler) AssignVehicle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.assignVehicleUC.Execute(r.Context(), application.AssignVehicleCommand{
		TripID:    aggregate.TripID(id),
		VehicleID: req.VehicleID,
		TenantID:  shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "vehicle_assigned"})
}

func (h *APITripHandler) Schedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.scheduleUC.Execute(r.Context(), application.ScheduleTripCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "scheduled"})
}

func (h *APITripHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.startUC.Execute(r.Context(), application.StartTripCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (h *APITripHandler) ReachPickup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.reachPickupUC.Execute(r.Context(), application.ReachPickupCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "reached_pickup"})
}

func (h *APITripHandler) StartTransit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.startTransitUC.Execute(r.Context(), application.StartTransitCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "in_transit"})
}

func (h *APITripHandler) Deliver(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.deliverUC.Execute(r.Context(), application.DeliverCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "delivered"})
}

func (h *APITripHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.completeUC.Execute(r.Context(), application.CompleteTripCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

func (h *APITripHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.cancelUC.Execute(r.Context(), application.CancelTripCommand{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}
