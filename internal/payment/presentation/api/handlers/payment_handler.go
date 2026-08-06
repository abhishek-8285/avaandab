package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/payment/application"
	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
)

// APIPaymentHandler handles REST endpoints for the payment vertical slice.
type APIPaymentHandler struct {
	recordUC *application.RecordPaymentUseCase
	getUC    *application.GetPaymentUseCase
	listUC   *application.ListPaymentsUseCase
}

// NewAPIPaymentHandler constructs an APIPaymentHandler.
func NewAPIPaymentHandler(
	recordUC *application.RecordPaymentUseCase,
	getUC *application.GetPaymentUseCase,
	listUC *application.ListPaymentsUseCase,
) *APIPaymentHandler {
	return &APIPaymentHandler{recordUC: recordUC, getUC: getUC, listUC: listUC}
}

// Register mounts all payment routes.
func (h *APIPaymentHandler) Register(r chi.Router) {
	r.Route("/api/v1/payments", func(r chi.Router) {
		r.Post("/", h.Record)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
	})
}

func paymentTenantID(r *http.Request) shared.TenantID {
	return "1"
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
		TenantID:    paymentTenantID(r),
		InvoiceID:   req.InvoiceID,
		PaymentDate: payDate,
		Amount:      req.Amount,
		Method:      aggregate.PaymentMethod(req.Method),
		Reference:   req.Reference,
		Remarks:     req.Remarks,
	})
	if err != nil {
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
		TenantID: paymentTenantID(r),
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
		TenantID: paymentTenantID(r),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}
