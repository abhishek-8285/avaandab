package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/auth"
	"transport-app/internal/geofence/application"
	"transport-app/internal/middleware"
	"transport-app/internal/shared"
)

// APIGeofenceHandler exposes detention attach/waive endpoints for closed
// detentions (Spec 02 §6). Detention resolution touches both geofence and
// invoice state, so both permissions are required for attach.
type APIGeofenceHandler struct {
	attachUC *application.AttachDetentionUseCase
	waiveUC  *application.WaiveDetentionUseCase
	authSrv  auth.AuthorizationService
}

// NewAPIGeofenceHandler constructs an APIGeofenceHandler.
func NewAPIGeofenceHandler(
	attachUC *application.AttachDetentionUseCase,
	waiveUC *application.WaiveDetentionUseCase,
	authSrv auth.AuthorizationService,
) *APIGeofenceHandler {
	return &APIGeofenceHandler{attachUC: attachUC, waiveUC: waiveUC, authSrv: authSrv}
}

// Register mounts the detention API routes.
func (h *APIGeofenceHandler) Register(r chi.Router) {
	r.Route("/api/v1/geofences/detentions", func(r chi.Router) {
		r.With(
			middleware.RequirePermission(h.authSrv, "geofences", "update"),
			middleware.RequirePermission(h.authSrv, "invoices", "update"),
		).Post("/{id}/attach", h.Attach)
		r.With(middleware.RequirePermission(h.authSrv, "geofences", "update")).Post("/{id}/waive", h.Waive)
	})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Attach bills a closed detention onto its trip's invoice.
func (h *APIGeofenceHandler) Attach(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenant := shared.TenantIDFromContext(r.Context())
	if err := h.attachUC.Execute(r.Context(), application.AttachDetentionCommand{DetentionID: id, TenantID: tenant}); err != nil {
		if errors.Is(err, application.ErrInvoicePaid) {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"attached"}`))
}

// Waive zeroes the amount of an open detention and marks it waived.
func (h *APIGeofenceHandler) Waive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenant := shared.TenantIDFromContext(r.Context())
	if err := h.waiveUC.Execute(r.Context(), application.WaiveDetentionCommand{DetentionID: id, TenantID: tenant}); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"waived"}`))
}
