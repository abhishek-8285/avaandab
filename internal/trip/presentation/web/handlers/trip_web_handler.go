package handlers

import (
	"net/http"
	"strconv"

	"transport-app/internal/shared"
	"transport-app/internal/trip/application"
	"transport-app/internal/trip/domain/aggregate"
	"transport-app/internal/trip/presentation/web/viewmodels"
)

type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data any)
}

type TripWebHandler struct {
	listUC     *application.ListTripsUseCase
	getUC      *application.GetTripUseCase
	tmplRender TemplateRenderer
}

func NewTripWebHandler(listUC *application.ListTripsUseCase, getUC *application.GetTripUseCase, tmplRender TemplateRenderer) *TripWebHandler {
	return &TripWebHandler{listUC: listUC, getUC: getUC, tmplRender: tmplRender}
}

func (h *TripWebHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	res, err := h.listUC.Execute(r.Context(), application.ListTripsQuery{
		TenantID: shared.TenantID("1"),
		Page:     page,
		Limit:    limit,
		Search:   search,
		Status:   status,
	})
	if err != nil {
		http.Error(w, "Failed to load trips: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.tmplRender.Render(w, "trip_list.html", map[string]any{
		"Trips":        viewmodels.FromDTOs(res.Trips),
		"Total":        res.Total,
		"Page":         page,
		"Limit":        limit,
		"Search":       search,
		"StatusFilter": status,
	})
}

func (h *TripWebHandler) View(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	dto, err := h.getUC.Execute(r.Context(), application.GetTripQuery{
		TripID:   aggregate.TripID(id),
		TenantID: shared.TenantID("1"),
	})
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	h.tmplRender.Render(w, "trip_view.html", map[string]any{
		"Trip": viewmodels.FromDTO(dto),
	})
}
