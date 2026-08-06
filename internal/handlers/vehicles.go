package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
	clock "transport-app/internal/shared/clock"
	id "transport-app/internal/shared/id"
	uow "transport-app/internal/shared/uow"
	vehicleapp "transport-app/internal/vehicle/application"
	vehicleagg "transport-app/internal/vehicle/domain/aggregate"
)

// VehicleHandlers handles vehicle management.
type VehicleHandlers struct {
	*App
	createUC *vehicleapp.CreateVehicleUseCase
	updateUC *vehicleapp.UpdateVehicleUseCase
	getUC    *vehicleapp.GetVehicleUseCase
	listUC   *vehicleapp.ListVehiclesUseCase
}

func (h *VehicleHandlers) init() {
	if h.createUC == nil {
		uowImpl := uow.NewSQLUnitOfWork(h.DB)
		clockImpl := clock.NewRealClock()
		idGenImpl := id.NewUUIDGenerator()

		h.createUC = vehicleapp.NewCreateVehicleUseCase(uowImpl, idGenImpl, clockImpl)
		h.updateUC = vehicleapp.NewUpdateVehicleUseCase(uowImpl, clockImpl)
		h.getUC = vehicleapp.NewGetVehicleUseCase(uowImpl)
		h.listUC = vehicleapp.NewListVehiclesUseCase(uowImpl)
	}
}

func (h *VehicleHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "delete")).Post("/{id}/delete", h.Delete)
	r.With(middleware.ResourcePermission(h.AuthSrv, "vehicles", "update")).Post("/{id}/status", h.UpdateStatus)
}

func (h *VehicleHandlers) List(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	res, err := h.listUC.Execute(r.Context(), vehicleapp.ListVehiclesQuery{
		TenantID: "1",
		Page:     pp.Page,
		Limit:    pp.Limit,
		Search:   pp.Query,
		Status:   pp.Status,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, res.Total, "/vehicles")

	if isDatastarRequest(r) {
		h.renderFragment(w, "vehicle_list_table.html", map[string]interface{}{
			"Vehicles":     res.Vehicles,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, "vehicle_list.html", PageData{
		Title: "Vehicles",
		User:  session,
		Extra: map[string]interface{}{"Vehicles": res.Vehicles, "Pagination": pd, "Query": pp.Query, "StatusFilter": pp.Status},
	})
}

func (h *VehicleHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	h.renderForm(w, r, "vehicle_edit.html", PageData{Title: "New Vehicle", User: session})
}

func (h *VehicleHandlers) Create(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	capacity, _ := strconv.ParseInt(r.PostFormValue("capacity"), 10, 64)

	insExp, err := time.Parse("2006-01-02", r.PostFormValue("insurance_expiry"))
	if err != nil {
		insExp = time.Now().AddDate(1, 0, 0)
	}

	fitExp, err := time.Parse("2006-01-02", r.PostFormValue("fitness_expiry"))
	if err != nil {
		fitExp = time.Now().AddDate(1, 0, 0)
	}

	perExp, err := time.Parse("2006-01-02", r.PostFormValue("permit_expiry"))
	if err != nil {
		perExp = time.Now().AddDate(1, 0, 0)
	}

	var currentMileage *float64
	if milStr := r.PostFormValue("current_mileage"); milStr != "" {
		if mil, err := strconv.ParseFloat(milStr, 64); err == nil {
			currentMileage = &mil
		}
	}

	_, err = h.createUC.Execute(r.Context(), vehicleapp.CreateVehicleCommand{
		TenantID:           "1",
		RegistrationNumber: r.PostFormValue("registration_number"),
		VehicleNumber:      r.PostFormValue("vehicle_number"),
		VehicleType:        vehicleagg.VehicleType(r.PostFormValue("vehicle_type")),
		Capacity:           capacity,
		FuelType:           vehicleagg.FuelType(r.PostFormValue("fuel_type")),
		InsuranceExpiry:    insExp,
		FitnessExpiry:      fitExp,
		PermitExpiry:       perExp,
		CurrentMileage:     currentMileage,
	})
	if err != nil {
		h.renderForm(w, r, "vehicle_edit.html", PageData{Title: "New Vehicle", FlashError: err.Error()})
		return
	}

	if isDatastarRequest(r) {
		w.Header().Set("Location", "/vehicles")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/vehicles", http.StatusSeeOther)
}

func (h *VehicleHandlers) View(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	vehicle, err := h.getUC.Execute(r.Context(), vehicleapp.GetVehicleQuery{
		ID:       vehicleagg.VehicleID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}
	files, _ := h.Services.Files.GetFilesByEntity(r.Context(), "vehicle_insurance", id)
	h.renderPage(w, "vehicle_view.html", PageData{Title: "View Vehicle", Extra: map[string]interface{}{"Vehicle": vehicle, "Files": files}})
}

func (h *VehicleHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	vehicle, err := h.getUC.Execute(r.Context(), vehicleapp.GetVehicleQuery{
		ID:       vehicleagg.VehicleID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}
	h.renderForm(w, r, "vehicle_edit.html", PageData{Title: "Edit Vehicle", Extra: map[string]interface{}{"Vehicle": vehicle}})
}

func (h *VehicleHandlers) Update(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	capacity, _ := strconv.ParseInt(r.PostFormValue("capacity"), 10, 64)
	status := vehicleagg.VehicleStatus(r.PostFormValue("status"))
	if status == "" {
		status = vehicleagg.VehicleAvailable
	}

	insExp, err := time.Parse("2006-01-02", r.PostFormValue("insurance_expiry"))
	if err != nil {
		insExp = time.Now().AddDate(1, 0, 0)
	}

	fitExp, err := time.Parse("2006-01-02", r.PostFormValue("fitness_expiry"))
	if err != nil {
		fitExp = time.Now().AddDate(1, 0, 0)
	}

	perExp, err := time.Parse("2006-01-02", r.PostFormValue("permit_expiry"))
	if err != nil {
		perExp = time.Now().AddDate(1, 0, 0)
	}

	var currentMileage *float64
	if milStr := r.PostFormValue("current_mileage"); milStr != "" {
		if mil, err := strconv.ParseFloat(milStr, 64); err == nil {
			currentMileage = &mil
		}
	}

	err = h.updateUC.Execute(r.Context(), vehicleapp.UpdateVehicleCommand{
		ID:                 vehicleagg.VehicleID(id),
		TenantID:           "1",
		RegistrationNumber: r.PostFormValue("registration_number"),
		VehicleNumber:      r.PostFormValue("vehicle_number"),
		VehicleType:        vehicleagg.VehicleType(r.PostFormValue("vehicle_type")),
		Capacity:           capacity,
		FuelType:           vehicleagg.FuelType(r.PostFormValue("fuel_type")),
		InsuranceExpiry:    insExp,
		FitnessExpiry:      fitExp,
		PermitExpiry:       perExp,
		Status:             status,
		CurrentMileage:     currentMileage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/vehicles/"+id, http.StatusSeeOther)
}

func (h *VehicleHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := domain.VehicleID(chi.URLParam(r, "id"))
	if err := h.Services.Vehicles.DeleteVehicle(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isDatastarRequest(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/vehicles", http.StatusSeeOther)
}

func (h *VehicleHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	status := r.PostFormValue("status")

	vehicle, err := h.getUC.Execute(r.Context(), vehicleapp.GetVehicleQuery{
		ID:       vehicleagg.VehicleID(id),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.updateUC.Execute(r.Context(), vehicleapp.UpdateVehicleCommand{
		ID:                 vehicleagg.VehicleID(id),
		TenantID:           "1",
		RegistrationNumber: vehicle.RegistrationNumber,
		VehicleNumber:      vehicle.VehicleNumber,
		VehicleType:        vehicleagg.VehicleType(vehicle.VehicleType),
		Capacity:           vehicle.Capacity,
		FuelType:           vehicleagg.FuelType(vehicle.FuelType),
		InsuranceExpiry:    vehicle.InsuranceExpiry,
		FitnessExpiry:      vehicle.FitnessExpiry,
		PermitExpiry:       vehicle.PermitExpiry,
		Status:             vehicleagg.VehicleStatus(status),
		CurrentMileage:     vehicle.CurrentMileage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if isDatastarRequest(r) {
		h.renderFragment(w, "vehicle_view.html", nil)
		return
	}
	http.Redirect(w, r, "/vehicles/"+id, http.StatusSeeOther)
}
