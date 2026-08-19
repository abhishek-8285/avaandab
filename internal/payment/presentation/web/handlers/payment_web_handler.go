package handlers

import (
	"net/http"
	"strconv"

	"transport-app/internal/payment/application"
	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/payment/presentation/web/viewmodels"
	"transport-app/internal/shared"
)

type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data any)
}

type PaymentWebHandler struct {
	listUC          *application.ListPaymentsUseCase
	listByInvoiceUC *application.ListPaymentsByInvoiceUseCase
	getUC           *application.GetPaymentUseCase
	tmplRender      TemplateRenderer
}

func NewPaymentWebHandler(listUC *application.ListPaymentsUseCase, listByInvoiceUC *application.ListPaymentsByInvoiceUseCase, getUC *application.GetPaymentUseCase, tmplRender TemplateRenderer) *PaymentWebHandler {
	return &PaymentWebHandler{listUC: listUC, listByInvoiceUC: listByInvoiceUC, getUC: getUC, tmplRender: tmplRender}
}

func (h *PaymentWebHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	method := r.URL.Query().Get("method")

	res, err := h.listUC.Execute(r.Context(), application.ListPaymentsQuery{
		TenantID: shared.TenantIDFromContext(r.Context()),
		Page:     page,
		Limit:    limit,
		Method:   method,
	})
	if err != nil {
		http.Error(w, "Failed to load payments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.tmplRender.Render(w, "payment_list.html", map[string]any{
		"Payments": viewmodels.FromDTOs(res.Payments),
		"Total":    res.Total,
		"Page":     page,
		"Limit":    limit,
		"Method":   method,
	})
}

func (h *PaymentWebHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	dto, err := h.getUC.Execute(r.Context(), application.GetPaymentQuery{
		ID:       aggregate.PaymentID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	h.tmplRender.Render(w, "payment_view.html", map[string]any{
		"Payment": viewmodels.FromDTO(dto),
	})
}

func (h *PaymentWebHandler) ListByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("invoice_id")
	if invoiceID == "" {
		invoiceID = r.URL.Query().Get("invoice_id")
	}

	dtos, err := h.listByInvoiceUC.Execute(r.Context(), application.ListPaymentsByInvoiceQuery{
		TenantID:  shared.TenantIDFromContext(r.Context()),
		InvoiceID: invoiceID,
	})
	if err != nil {
		http.Error(w, "Failed to load payments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.tmplRender.Render(w, "payment_list.html", map[string]any{
		"Payments":  viewmodels.FromDTOs(dtos),
		"InvoiceID": invoiceID,
	})
}
