package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/invoice/application"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
)

// APIInvoiceHandler handles REST endpoints for the invoice vertical slice.
type APIInvoiceHandler struct {
	generateUC *application.GenerateInvoiceUseCase
	getUC      *application.GetInvoiceUseCase
	listUC     *application.ListInvoicesUseCase
}

// NewAPIInvoiceHandler constructs an APIInvoiceHandler.
func NewAPIInvoiceHandler(
	generateUC *application.GenerateInvoiceUseCase,
	getUC *application.GetInvoiceUseCase,
	listUC *application.ListInvoicesUseCase,
) *APIInvoiceHandler {
	return &APIInvoiceHandler{generateUC: generateUC, getUC: getUC, listUC: listUC}
}

// Register mounts all invoice routes.
func (h *APIInvoiceHandler) Register(r chi.Router) {
	r.Route("/api/v1/invoices", func(r chi.Router) {
		r.Post("/", h.Generate)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
	})
}

func invoiceTenantID(r *http.Request) shared.TenantID {
	return "1"
}

func (h *APIInvoiceHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BookingID  string  `json:"booking_id"`
		CustomerID string  `json:"customer_id"`
		TripID     *string `json:"trip_id"`
		Subtotal   float64 `json:"subtotal"`
		Tax        float64 `json:"tax"`
		Discount   float64 `json:"discount"`
		Total      float64 `json:"total"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.generateUC.Execute(r.Context(), application.GenerateInvoiceCommand{
		TenantID:   invoiceTenantID(r),
		BookingID:  req.BookingID,
		CustomerID: req.CustomerID,
		TripID:     req.TripID,
		Subtotal:   req.Subtotal,
		Tax:        req.Tax,
		Discount:   req.Discount,
		Total:      req.Total,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

func (h *APIInvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := h.listUC.Execute(r.Context(), application.ListInvoicesQuery{
		TenantID: invoiceTenantID(r),
		Page:     page,
		Limit:    limit,
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"invoices": res.Invoices, "total": res.Total})
}

func (h *APIInvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.getUC.Execute(r.Context(), application.GetInvoiceQuery{
		ID:       aggregate.InvoiceID(id),
		TenantID: invoiceTenantID(r),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}
