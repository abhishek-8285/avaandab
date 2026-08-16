package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
)

// RouteHandlers handles route management.
type RouteHandlers struct {
	*App
}

func (h *RouteHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "routes", "delete")).Post("/{id}/delete", h.Delete)
}

func (h *RouteHandlers) List(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	list, total, err := h.Services.Routes.ListRoutes(r.Context(), pp.Query, pp.Limit, pp.Offset)
	if err != nil {
		http.Error(w, "Failed to list routes", http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, total, "/routes")

	if isDatastarRequest(r) {
		h.renderFragment(w, "route_list_table.html", map[string]interface{}{
			"Routes":     list,
			"Pagination": pd,
			"Query":      pp.Query,
		})
		return
	}

	h.renderPage(w, r, "route_list.html", PageData{
		Title: "Routes",
		User:  session,
		Extra: map[string]interface{}{"Routes": list, "Pagination": pd, "Query": pp.Query},
	})
}

func (h *RouteHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	h.renderForm(w, r, "route_edit.html", PageData{Title: "New Route", User: session})
}

func (h *RouteHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	distance, _ := strconv.ParseFloat(r.PostFormValue("distance"), 64)
	estHours, _ := strconv.ParseFloat(r.PostFormValue("estimated_hours"), 64)
	fare, _ := strconv.ParseFloat(r.PostFormValue("standard_fare"), 64)

	_, err := h.Services.Routes.CreateRoute(
		r.Context(),
		r.PostFormValue("source"),
		r.PostFormValue("destination"),
		distance, estHours, fare,
		r.PostFormValue("remarks"),
	)
	if err != nil {
		h.renderForm(w, r, "route_edit.html", PageData{Title: "New Route", FlashError: err.Error()})
		return
	}

	if isDatastarRequest(r) {
		w.Header().Set("Location", "/routes")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/routes", http.StatusSeeOther)
}

func (h *RouteHandlers) View(w http.ResponseWriter, r *http.Request) {
	id := domain.RouteID(chi.URLParam(r, "id"))
	route, err := h.Services.Routes.GetRoute(r.Context(), id)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}
	h.renderPage(w, r, "route_view.html", PageData{Title: "View Route", Extra: map[string]interface{}{"Route": route}})
}

func (h *RouteHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	id := domain.RouteID(chi.URLParam(r, "id"))
	route, err := h.Services.Routes.GetRoute(r.Context(), id)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}
	h.renderForm(w, r, "route_edit.html", PageData{Title: "Edit Route", Extra: map[string]interface{}{"Route": route}})
}

func (h *RouteHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := domain.RouteID(chi.URLParam(r, "id"))
	distance, _ := strconv.ParseFloat(r.PostFormValue("distance"), 64)
	estHours, _ := strconv.ParseFloat(r.PostFormValue("estimated_hours"), 64)
	fare, _ := strconv.ParseFloat(r.PostFormValue("standard_fare"), 64)

	_, err := h.Services.Routes.UpdateRoute(
		r.Context(), id,
		r.PostFormValue("source"),
		r.PostFormValue("destination"),
		distance, estHours, fare,
		r.PostFormValue("remarks"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/routes/"+id.String(), http.StatusSeeOther)
}

func (h *RouteHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := domain.RouteID(chi.URLParam(r, "id"))
	if err := h.Services.Routes.DeleteRoute(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/routes", http.StatusSeeOther)
}
