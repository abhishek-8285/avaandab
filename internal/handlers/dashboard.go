package handlers

import (
	"net/http"
)

// DashboardHandlers handles dashboard-related requests.
type DashboardHandlers struct {
	*App
}

// Index renders the dashboard page.
func (h *DashboardHandlers) Index(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	data, err := h.Services.Dashboard.GetDashboardData(r.Context())
	if err != nil {
		http.Error(w, "Failed to load dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	company, _ := h.Services.Settings.GetSettings(r.Context())

	h.renderPage(w, "dashboard.html", PageData{
		Title:      "Dashboard",
		User:       session,
		Settings:   company,
		FlashError: "",
		Extra: map[string]interface{}{
			"Stats": data,
		},
	})
}
