package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	driverapp "transport-app/internal/driver/application"
	driveragg "transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/middleware"
	"transport-app/internal/shared"
	clock "transport-app/internal/shared/clock"
	id "transport-app/internal/shared/id"
	uow "transport-app/internal/shared/uow"
)

// DriverHandlers handles driver management.
type DriverHandlers struct {
	*App
	createUC *driverapp.CreateDriverUseCase
	updateUC *driverapp.UpdateDriverUseCase
	getUC    *driverapp.GetDriverUseCase
	listUC   *driverapp.ListDriversUseCase
}

func (h *DriverHandlers) init() {
	if h.createUC == nil {
		uowImpl := uow.NewSQLUnitOfWork(h.DB)
		clockImpl := clock.NewRealClock()
		idGenImpl := id.NewUUIDGenerator()

		h.createUC = driverapp.NewCreateDriverUseCase(uowImpl, idGenImpl, clockImpl)
		h.updateUC = driverapp.NewUpdateDriverUseCase(uowImpl, clockImpl)
		h.getUC = driverapp.NewGetDriverUseCase(uowImpl)
		h.listUC = driverapp.NewListDriversUseCase(uowImpl)
	}
}

func (h *DriverHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "delete")).Post("/{id}/delete", h.Delete)
	r.With(middleware.ResourcePermission(h.AuthSrv, "drivers", "update")).Post("/{id}/status", h.UpdateStatus)
}

func (h *DriverHandlers) List(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	res, err := h.listUC.Execute(r.Context(), driverapp.ListDriversQuery{
		TenantID: shared.TenantIDFromContext(r.Context()),
		Page:     pp.Page,
		Limit:    pp.Limit,
		Search:   pp.Query,
		Status:   pp.Status,
	})
	if err != nil {
		http.Error(w, "Failed to list drivers", http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, res.Total, "/drivers")

	if isDatastarRequest(r) {
		h.renderFragment(w, "driver_list_table.html", map[string]interface{}{
			"Drivers":      res.Drivers,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, r, "driver_list.html", PageData{
		Title: "Drivers",
		User:  session,
		Extra: map[string]interface{}{"Drivers": res.Drivers, "Pagination": pd, "Query": pp.Query, "StatusFilter": pp.Status},
	})
}

func (h *DriverHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	h.renderForm(w, r, "driver_edit.html", PageData{Title: "New Driver", User: session})
}

func (h *DriverHandlers) Create(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	licenseExpiry, err := time.Parse("2006-01-02", r.PostFormValue("license_expiry"))
	if err != nil {
		licenseExpiry = time.Now().AddDate(5, 0, 0)
	}
	exp, _ := strconv.ParseInt(r.PostFormValue("experience"), 10, 64)

	var email *string
	if val := r.PostFormValue("email"); val != "" {
		email = &val
	}
	var address *string
	if val := r.PostFormValue("address"); val != "" {
		address = &val
	}

	ecName, ecPhone, notes := r.PostFormValue("emergency_contact_name"), r.PostFormValue("emergency_contact_phone"), r.PostFormValue("notes")

	_, err = h.createUC.Execute(r.Context(), driverapp.CreateDriverCommand{
		TenantID:              shared.TenantIDFromContext(r.Context()),
		FirstName:             r.PostFormValue("first_name"),
		LastName:              r.PostFormValue("last_name"),
		Phone:                 r.PostFormValue("phone"),
		Email:                 email,
		Address:               address,
		LicenseNumber:         r.PostFormValue("license_number"),
		LicenseExpiry:         licenseExpiry,
		ExperienceYears:       exp,
		EmergencyContactName:  strPtr(ecName),
		EmergencyContactPhone: strPtr(ecPhone),
		Notes:                 strPtr(notes),
	})
	if err != nil {
		session, _ := h.getUserFromContext(r)
		h.renderForm(w, r, "driver_edit.html", PageData{Title: "New Driver", User: session, FlashError: err.Error()})
		return
	}

	if isDatastarRequest(r) {
		w.Header().Set("Location", "/drivers")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/drivers", http.StatusSeeOther)
}

func (h *DriverHandlers) View(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	id := chi.URLParam(r, "id")
	driver, err := h.getUC.Execute(r.Context(), driverapp.GetDriverQuery{
		ID:       driveragg.DriverID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		h.renderError(w, http.StatusNotFound, "Driver Not Found", fmt.Sprintf("No driver found with ID %q.", id), session)
		return
	}
	files, _ := h.Services.Files.GetFilesByEntity(r.Context(), "driver_license", id)
	h.renderPage(w, r, "driver_view.html", PageData{Title: "View Driver", User: session, Extra: map[string]interface{}{"Driver": driver, "Files": files}})
}

func (h *DriverHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	id := chi.URLParam(r, "id")
	driver, err := h.getUC.Execute(r.Context(), driverapp.GetDriverQuery{
		ID:       driveragg.DriverID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		h.renderError(w, http.StatusNotFound, "Driver Not Found", fmt.Sprintf("No driver found with ID %q.", id), session)
		return
	}
	h.renderForm(w, r, "driver_edit.html", PageData{Title: "Edit Driver", User: session, Extra: map[string]interface{}{"Driver": driver}})
}

func (h *DriverHandlers) Update(w http.ResponseWriter, r *http.Request) {
	h.init()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	licenseExpiry, err := time.Parse("2006-01-02", r.PostFormValue("license_expiry"))
	if err != nil {
		licenseExpiry = time.Now().AddDate(5, 0, 0)
	}
	exp, _ := strconv.ParseInt(r.PostFormValue("experience"), 10, 64)
	status := driveragg.DriverStatus(r.PostFormValue("status"))
	if status == "" {
		status = driveragg.DriverAvailable
	}

	var email *string
	if val := r.PostFormValue("email"); val != "" {
		email = &val
	}
	var address *string
	if val := r.PostFormValue("address"); val != "" {
		address = &val
	}

	var ecName, ecPhone, notes *string
	if e := r.PostFormValue("emergency_contact_name"); e != "" {
		ecName = &e
	}
	if e := r.PostFormValue("emergency_contact_phone"); e != "" {
		ecPhone = &e
	}
	if e := r.PostFormValue("notes"); e != "" {
		notes = &e
	}

	err = h.updateUC.Execute(r.Context(), driverapp.UpdateDriverCommand{
		ID:                    driveragg.DriverID(id),
		TenantID:              shared.TenantIDFromContext(r.Context()),
		FirstName:             r.PostFormValue("first_name"),
		LastName:              r.PostFormValue("last_name"),
		Phone:                 r.PostFormValue("phone"),
		Email:                 email,
		Address:               address,
		LicenseNumber:         r.PostFormValue("license_number"),
		LicenseExpiry:         licenseExpiry,
		ExperienceYears:       exp,
		Status:                status,
		EmergencyContactName:  ecName,
		EmergencyContactPhone: ecPhone,
		Notes:                 notes,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/drivers/"+id, http.StatusSeeOther)
}

func (h *DriverHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := domain.DriverID(chi.URLParam(r, "id"))
	if err := h.Services.Drivers.DeleteDriver(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete driver", http.StatusInternalServerError)
		return
	}
	if isDatastarRequest(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/drivers", http.StatusSeeOther)
}

func (h *DriverHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	h.init()
	id := chi.URLParam(r, "id")
	status := r.PostFormValue("status")

	driver, err := h.getUC.Execute(r.Context(), driverapp.GetDriverQuery{
		ID:       driveragg.DriverID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.updateUC.Execute(r.Context(), driverapp.UpdateDriverCommand{
		ID:                    driveragg.DriverID(id),
		TenantID:              shared.TenantIDFromContext(r.Context()),
		FirstName:             driver.FirstName,
		LastName:              driver.LastName,
		Phone:                 driver.Phone,
		Email:                 driver.Email,
		Address:               driver.Address,
		LicenseNumber:         driver.LicenseNumber,
		LicenseExpiry:         driver.LicenseExpiry,
		ExperienceYears:       driver.ExperienceYears,
		Status:                driveragg.DriverStatus(status),
		EmergencyContactName:  driver.EmergencyContactName,
		EmergencyContactPhone: driver.EmergencyContactPhone,
		Notes:                 driver.Notes,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if isDatastarRequest(r) {
		h.renderFragment(w, "driver_view.html", nil)
		return
	}
	http.Redirect(w, r, "/drivers/"+id, http.StatusSeeOther)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetMe returns the driver profile for the authenticated user.
// Spec 13 §2.2: GET /api/v1/drivers/me
func (h *DriverHandlers) GetMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session, ok := h.getUserFromContext(r)
	if !ok || session == nil || session.UserID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()
	var d struct {
		ID        string
		DriverID  string
		FirstName string
		LastName  string
		Phone     string
		Status    string
	}

	// Query driver linked to user by ID, user_id (if matches ID), or email
	err := h.DB.QueryRowContext(ctx, `
		SELECT id, driver_id, first_name, last_name, phone, status
		FROM drivers
		WHERE id = ? OR email = (SELECT email FROM users WHERE id = ?)
		LIMIT 1
	`, session.UserID, session.UserID).Scan(&d.ID, &d.DriverID, &d.FirstName, &d.LastName, &d.Phone, &d.Status)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "driver not found"})
		return
	}

	fullName := strings.TrimSpace(d.FirstName + " " + d.LastName)
	driverCode := d.DriverID
	if driverCode == "" {
		driverCode = d.ID
	}

	// Check active trip and vehicle plate
	var vehiclePlate string
	var vehicleID string
	_ = h.DB.QueryRowContext(ctx, `
		SELECT COALESCE(v.registration_number, ''), COALESCE(t.vehicle_id, '')
		FROM trips t
		LEFT JOIN vehicles v ON t.vehicle_id = v.id
		WHERE (t.driver_id = ? OR t.driver_id = ? OR t.driver_id = ?)
		  AND t.status IN ('assigned', 'started', 'reached_pickup', 'in_transit')
		ORDER BY t.departure_time DESC, t.created_at DESC
		LIMIT 1
	`, d.ID, d.DriverID, session.UserID).Scan(&vehiclePlate, &vehicleID)

	// Check current location from latest snapshot or vehicle latest position
	type Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	var curLoc *Location
	if vehicleID != "" {
		var lat, lon float64
		if errLoc := h.DB.QueryRowContext(ctx, `
			SELECT latitude, longitude 
			FROM vehicle_latest_position 
			WHERE vehicle_id = ?
		`, vehicleID).Scan(&lat, &lon); errLoc == nil {
			curLoc = &Location{Latitude: lat, Longitude: lon}
		}
	}
	if curLoc == nil {
		var lat, lon float64
		if errLoc := h.DB.QueryRowContext(ctx, `
			SELECT latitude, longitude
			FROM telemetry_snapshots
			WHERE (vehicle_id = ? AND vehicle_id != '') OR driver_id = ? OR driver_id = ?
			ORDER BY timestamp DESC
			LIMIT 1
		`, vehicleID, d.ID, d.DriverID).Scan(&lat, &lon); errLoc == nil {
			curLoc = &Location{Latitude: lat, Longitude: lon}
		}
	}

	resp := map[string]interface{}{
		"driver_id":        driverCode,
		"user_id":          session.UserID,
		"name":             fullName,
		"phone":            d.Phone,
		"status":           d.Status,
		"vehicle_plate":    vehiclePlate,
		"current_location": curLoc,
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
