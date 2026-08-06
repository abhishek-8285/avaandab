package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/middleware"
)

// SettingsHandlers handles company settings management.
type SettingsHandlers struct {
	*App
}

func (h *SettingsHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "settings", "read")).Get("/", h.Index)
	r.With(middleware.ResourcePermission(h.AuthSrv, "settings", "update")).Post("/update", h.Update)
}

func (h *SettingsHandlers) Index(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	settings, err := h.Services.Settings.GetSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderPage(w, "settings.html", PageData{
		Title: "Settings",
		User:  session,
		Extra: map[string]interface{}{"Settings": settings},
	})
}

func (h *SettingsHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gstEnabled := r.PostFormValue("gst_enabled") == "1"
	gstRate, _ := parseDecimal(r.PostFormValue("gst_rate"))

	_, err := h.Services.Settings.UpdateSettings(
		r.Context(),
		r.PostFormValue("company_name"),
		r.PostFormValue("currency"),
		r.PostFormValue("timezone"),
		gstEnabled,
		gstRate,
		r.PostFormValue("booking_prefix"),
		r.PostFormValue("trip_prefix"),
		r.PostFormValue("invoice_prefix"),
		r.PostFormValue("address"),
		r.PostFormValue("phone"),
		r.PostFormValue("email"),
		r.PostFormValue("gst_number"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func parseDecimal(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
