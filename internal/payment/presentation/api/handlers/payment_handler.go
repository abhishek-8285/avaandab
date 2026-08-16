package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/auth"
	"transport-app/internal/middleware"
	"transport-app/internal/payment/application"
	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
)

// APIPaymentHandler handles REST endpoints for the payment vertical slice.
type APIPaymentHandler struct {
	recordUC        *application.RecordPaymentUseCase
	getUC           *application.GetPaymentUseCase
	listUC          *application.ListPaymentsUseCase
	reverseUC       *application.ReversePaymentUseCase
	listByInvoiceUC *application.ListPaymentsByInvoiceUseCase
	authSrv         auth.AuthorizationService
}

// NewAPIPaymentHandler constructs an APIPaymentHandler.
func NewAPIPaymentHandler(
	recordUC *application.RecordPaymentUseCase,
	getUC *application.GetPaymentUseCase,
	listUC *application.ListPaymentsUseCase,
	reverseUC *application.ReversePaymentUseCase,
	listByInvoiceUC *application.ListPaymentsByInvoiceUseCase,
	authSrv auth.AuthorizationService,
) *APIPaymentHandler {
	return &APIPaymentHandler{
		recordUC:        recordUC,
		getUC:           getUC,
		listUC:          listUC,
		reverseUC:       reverseUC,
		listByInvoiceUC: listByInvoiceUC,
		authSrv:         authSrv,
	}
}

// Register mounts all payment routes.
func (h *APIPaymentHandler) Register(r chi.Router) {
	r.Route("/api/v1/payments", func(r chi.Router) {
		r.With(middleware.RequirePermission(h.authSrv, "payments", "create")).Post("/", h.Record)
		r.With(middleware.RequirePermission(h.authSrv, "payments", "read")).Get("/", h.List)
		r.With(middleware.RequirePermission(h.authSrv, "payments", "read")).Get("/{id}", h.Get)
		r.With(middleware.RequirePermission(h.authSrv, "payments", "read")).Get("/by-invoice/{invoiceID}", h.ListByInvoice)
		r.With(middleware.RequirePermission(h.authSrv, "payments", "create")).Post("/{id}/reverse", h.Reverse)
	})
}

func (h *APIPaymentHandler) Record(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InvoiceID   string  `json:"invoice_id"`
		PaymentDate string  `json:"payment_date"`
		Amount      float64 `json:"amount"`
		Method      string  `json:"method"`
		Reference   *string `json:"reference"`
		Remarks     *string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payDate, err := time.Parse(time.RFC3339, req.PaymentDate)
	if err != nil {
		http.Error(w, "payment_date must be RFC3339", http.StatusBadRequest)
		return
	}

	id, err := h.recordUC.Execute(r.Context(), application.RecordPaymentCommand{
		TenantID:    shared.TenantIDFromContext(r.Context()),
		InvoiceID:   req.InvoiceID,
		PaymentDate: payDate,
		Amount:      req.Amount,
		Method:      aggregate.PaymentMethod(req.Method),
		Reference:   req.Reference,
		Remarks:     req.Remarks,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvoiceNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

func (h *APIPaymentHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := h.listUC.Execute(r.Context(), application.ListPaymentsQuery{
		TenantID: shared.TenantIDFromContext(r.Context()),
		Page:     page,
		Limit:    limit,
		Method:   r.URL.Query().Get("method"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"payments": res.Payments, "total": res.Total})
}

func (h *APIPaymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.getUC.Execute(r.Context(), application.GetPaymentQuery{
		ID:       aggregate.PaymentID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *APIPaymentHandler) ListByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "invoiceID")
	payments, err := h.listByInvoiceUC.Execute(r.Context(), application.ListPaymentsByInvoiceQuery{
		TenantID:  shared.TenantIDFromContext(r.Context()),
		InvoiceID: invoiceID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"payments": payments})
}

func (h *APIPaymentHandler) Reverse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OriginalPaymentID string `json:"original_payment_id"`
		Reason            string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.reverseUC.Execute(r.Context(), application.ReversePaymentCommand{
		TenantID:      shared.TenantIDFromContext(r.Context()),
		OriginalPayID: aggregate.PaymentID(req.OriginalPaymentID),
		Reason:        req.Reason,
	})
	if err != nil {
		if errors.Is(err, application.ErrPaymentNotFound) || errors.Is(err, application.ErrInvoiceNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}
