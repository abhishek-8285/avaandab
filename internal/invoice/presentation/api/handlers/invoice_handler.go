package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/auth"
	"transport-app/internal/invoice/application"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/middleware"
	"transport-app/internal/shared"
)

// APIInvoiceHandler handles REST endpoints for the invoice vertical slice.
type APIInvoiceHandler struct {
	generateUC *application.GenerateInvoiceUseCase
	getUC      *application.GetInvoiceUseCase
	listUC     *application.ListInvoicesUseCase
	voidUC     *application.VoidInvoiceUseCase
	authSrv    auth.AuthorizationService
}

// NewAPIInvoiceHandler constructs an APIInvoiceHandler.
func NewAPIInvoiceHandler(
	generateUC *application.GenerateInvoiceUseCase,
	getUC *application.GetInvoiceUseCase,
	listUC *application.ListInvoicesUseCase,
	voidUC *application.VoidInvoiceUseCase,
	authSrv auth.AuthorizationService,
) *APIInvoiceHandler {
	return &APIInvoiceHandler{generateUC: generateUC, getUC: getUC, listUC: listUC, voidUC: voidUC, authSrv: authSrv}
}

// Register mounts all invoice routes.
func (h *APIInvoiceHandler) Register(r chi.Router) {
	r.Route("/api/v1/invoices", func(r chi.Router) {
		r.With(middleware.RequirePermission(h.authSrv, "invoices", "create")).Post("/", h.Generate)
		r.With(middleware.RequirePermission(h.authSrv, "invoices", "read")).Get("/", h.List)
		r.With(middleware.RequirePermission(h.authSrv, "invoices", "read")).Get("/{id}", h.Get)
		r.With(middleware.RequirePermission(h.authSrv, "invoices", "delete")).Post("/{id}/void", h.Void)
	})
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
		TenantID:   shared.TenantIDFromContext(r.Context()),
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
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	res, err := h.listUC.Execute(r.Context(), application.ListInvoicesQuery{
		TenantID: shared.TenantIDFromContext(r.Context()),
		Page:     page,
		Limit:    limit,
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
	})
	if err != nil {
		http.Error(w, "Failed to list invoices", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"invoices": res.Invoices, "total": res.Total})
}

func (h *APIInvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.getUC.Execute(r.Context(), application.GetInvoiceQuery{
		ID:       aggregate.InvoiceID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *APIInvoiceHandler) Void(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.voidUC.Execute(r.Context(), application.VoidInvoiceCommand{
		TenantID: shared.TenantIDFromContext(r.Context()),
		ID:       aggregate.InvoiceID(id),
	})
	if err != nil {
		switch err {
		case application.ErrInvoiceAlreadyCancelled:
			http.Error(w, err.Error(), http.StatusConflict)
		case application.ErrInvoiceCannotVoid:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			if err.Error() == "invoice not found" {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
