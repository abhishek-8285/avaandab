package handlers

import (
	"net/http"
	"strconv"

	"transport-app/internal/invoice/application"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/invoice/presentation/web/viewmodels"
	"transport-app/internal/shared"
)

type InvoiceWebHandler struct {
	listUC     *application.ListInvoicesUseCase
	getUC      *application.GetInvoiceUseCase
	voidUC     *application.VoidInvoiceUseCase
	tmplRender TemplateRenderer
}

type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data any)
}

func NewInvoiceWebHandler(listUC *application.ListInvoicesUseCase, getUC *application.GetInvoiceUseCase, voidUC *application.VoidInvoiceUseCase, tmplRender TemplateRenderer) *InvoiceWebHandler {
	return &InvoiceWebHandler{listUC: listUC, getUC: getUC, voidUC: voidUC, tmplRender: tmplRender}
}

func (h *InvoiceWebHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
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
	vms := viewmodels.FromDTOs(res.Invoices)
	h.tmplRender.Render(w, "invoices_list", map[string]interface{}{
		"invoices": vms,
		"total":    res.Total,
	})
}

func (h *InvoiceWebHandler) View(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	res, err := h.getUC.Execute(r.Context(), application.GetInvoiceQuery{
		ID:       aggregate.InvoiceID(id),
		TenantID: shared.TenantIDFromContext(r.Context()),
	})
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}
	vm := viewmodels.FromDTO(res)
	h.tmplRender.Render(w, "invoice_detail", vm)
}
