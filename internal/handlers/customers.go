package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
)

// CustomerHandlers handles customer management.
type CustomerHandlers struct {
	*App
}

func (h *CustomerHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "customers", "delete")).Post("/{id}/delete", h.Delete)
}

func (h *CustomerHandlers) List(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	list, total, err := h.Services.Customers.ListCustomers(r.Context(), pp.Query, pp.Limit, pp.Offset)
	if err != nil {
		http.Error(w, "Failed to list customers", http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, total, "/customers")

	if isDatastarRequest(r) {
		h.renderFragment(w, "customer_list_table.html", map[string]interface{}{
			"Customers":    list,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, r, "customer_list.html", PageData{
		Title: "Customers",
		User:  session,
		Extra: map[string]interface{}{"Customers": list, "Pagination": pd, "Query": pp.Query},
	})
}

func (h *CustomerHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	h.renderForm(w, r, "customer_edit.html", PageData{Title: "New Customer", User: session})
}

func (h *CustomerHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.Services.Customers.CreateCustomer(
		r.Context(),
		r.PostFormValue("name"),
		r.PostFormValue("company"),
		r.PostFormValue("phone"),
		r.PostFormValue("email"),
		r.PostFormValue("gst"),
		r.PostFormValue("address"),
		r.PostFormValue("notes"),
	)
	if err != nil {
		h.renderForm(w, r, "customer_edit.html", PageData{Title: "New Customer", FlashError: err.Error()})
		return
	}

	if isDatastarRequest(r) {
		w.Header().Set("Location", "/customers")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *CustomerHandlers) View(w http.ResponseWriter, r *http.Request) {
	id := domain.CustomerID(chi.URLParam(r, "id"))
	customer, err := h.Services.Customers.GetCustomer(r.Context(), id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}
	h.renderPage(w, r, "customer_view.html", PageData{Title: "View Customer", Extra: map[string]interface{}{"Customer": customer}})
}

func (h *CustomerHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	id := domain.CustomerID(chi.URLParam(r, "id"))
	customer, err := h.Services.Customers.GetCustomer(r.Context(), id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}
	h.renderForm(w, r, "customer_edit.html", PageData{Title: "Edit Customer", Extra: map[string]interface{}{"Customer": customer}})
}

func (h *CustomerHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := domain.CustomerID(chi.URLParam(r, "id"))
	_, err := h.Services.Customers.UpdateCustomer(
		r.Context(), id,
		r.PostFormValue("name"),
		r.PostFormValue("company"),
		r.PostFormValue("phone"),
		r.PostFormValue("email"),
		r.PostFormValue("gst"),
		r.PostFormValue("address"),
		r.PostFormValue("notes"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/customers/"+id.String(), http.StatusSeeOther)
}

func (h *CustomerHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := domain.CustomerID(chi.URLParam(r, "id"))
	if err := h.Services.Customers.DeleteCustomer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

var _ = strconv.Itoa
