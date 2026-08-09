package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/domain/invoice"
	"transport-app/internal/domain/types"
	invoiceapp "transport-app/internal/invoice/application"
	invoiceagg "transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/middleware"
	pdfgen "transport-app/internal/pdf"
	clock "transport-app/internal/shared/clock"
	id "transport-app/internal/shared/id"
	uow "transport-app/internal/shared/uow"
)

// InvoiceHandlers handles invoice management.
type InvoiceHandlers struct {
	*App
	getUC      *invoiceapp.GetInvoiceUseCase
	listUC     *invoiceapp.ListInvoicesUseCase
	generateUC *invoiceapp.GenerateInvoiceUseCase
}

func (h *InvoiceHandlers) init() {
	if h.getUC == nil {
		uowImpl := uow.NewSQLUnitOfWork(h.DB)
		clockImpl := clock.NewRealClock()
		idGenImpl := id.NewUUIDGenerator()

		h.getUC = invoiceapp.NewGetInvoiceUseCase(uowImpl)
		h.listUC = invoiceapp.NewListInvoicesUseCase(uowImpl)
		h.generateUC = invoiceapp.NewGenerateInvoiceUseCase(uowImpl, idGenImpl, clockImpl)
	}
}

func (h *InvoiceHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "invoices", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "invoices", "read")).Get("/{id}/pdf", h.DownloadPDF)
	r.With(middleware.ResourcePermission(h.AuthSrv, "invoices", "read")).Get("/{id}", h.View)
	r.With(middleware.ResourcePermission(h.AuthSrv, "invoices", "delete")).Post("/{id}/delete", h.Delete)
	r.With(middleware.ResourcePermission(h.AuthSrv, "invoices", "read")).Get("/number/{number}", h.ViewByNumber)
}

func (h *InvoiceHandlers) List(w http.ResponseWriter, r *http.Request) {
	h.init()
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	res, err := h.listUC.Execute(r.Context(), invoiceapp.ListInvoicesQuery{
		TenantID: "1",
		Page:     pp.Page,
		Limit:    pp.Limit,
		Search:   pp.Query,
		Status:   pp.Status,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, res.Total, "/invoices")

	if isDatastarRequest(r) {
		h.renderFragment(w, "invoice_list_table.html", map[string]interface{}{
			"Invoices":     res.Invoices,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, r, "invoice_list.html", PageData{
		Title: "Invoices",
		User:  session,
		Extra: map[string]interface{}{"Invoices": res.Invoices, "Pagination": pd, "Query": pp.Query, "StatusFilter": pp.Status},
	})
}

func (h *InvoiceHandlers) View(w http.ResponseWriter, r *http.Request) {
	h.init()
	idParam := chi.URLParam(r, "id")
	invoice, err := h.getUC.Execute(r.Context(), invoiceapp.GetInvoiceQuery{
		ID:       invoiceagg.InvoiceID(idParam),
		TenantID: "1",
	})
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	// Retrieve payments using the legacy services for now until the payment module vertical slice is ready
	payments, _ := h.Services.Invoices.GetPaymentsForInvoice(r.Context(), domain.InvoiceID(idParam))
	balance, _ := h.Services.Invoices.GetBalance(r.Context(), domain.InvoiceID(idParam))

	h.renderPage(w, r, "invoice_view.html", PageData{
		Title: "View Invoice",
		Extra: map[string]interface{}{
			"Invoice":  invoice,
			"Payments": payments,
			"Balance":  balance,
		},
	})
}

func (h *InvoiceHandlers) ViewByNumber(w http.ResponseWriter, r *http.Request) {
	h.init()
	// Fallback to legacy get since query by number is a read operation on db
	invoice, err := h.Services.Invoices.GetInvoiceByNumber(r.Context(), chi.URLParam(r, "number"))
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}
	h.renderPage(w, r, "invoice_view.html", PageData{
		Title: "View Invoice",
		Extra: map[string]interface{}{"Invoice": invoice},
	})
}

func (h *InvoiceHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.Services.Invoices.DeleteInvoice(r.Context(), domain.InvoiceID(chi.URLParam(r, "id"))); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/invoices", http.StatusSeeOther)
}

func (h *InvoiceHandlers) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	h.init()
	idParam := chi.URLParam(r, "id")
	invDTO, err := h.getUC.Execute(r.Context(), invoiceapp.GetInvoiceQuery{
		ID:       invoiceagg.InvoiceID(idParam),
		TenantID: "1",
	})
	if err != nil {
		fmt.Printf("[DownloadPDF Error] Invoice query failed for ID %s: %v\n", idParam, err)
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	balance, _ := h.Services.Invoices.GetBalance(r.Context(), domain.InvoiceID(idParam))
	paidAmount := invDTO.Total - balance
	if paidAmount < 0 {
		paidAmount = 0
	}

	invEntity := invoice.Invoice{
		ID:            types.InvoiceID(invDTO.ID),
		InvoiceNumber: invDTO.InvoiceNumber,
		BookingID:     types.BookingID(invDTO.BookingID),
		CustomerID:    types.CustomerID(invDTO.CustomerID),
		Subtotal:      invDTO.Subtotal,
		Tax:           invDTO.Tax,
		Discount:      invDTO.Discount,
		Total:         invDTO.Total,
		PaidAmount:    paidAmount,
		Status:        invoice.InvoiceStatus(invDTO.PaymentStatus),
		CreatedAt:     invDTO.CreatedAt,
	}

	pdfBytes, err := pdfgen.GenerateInvoicePDF(invEntity, "Apex Transport Ltd")
	if err != nil {
		fmt.Printf("[DownloadPDF Error] PDF generation failed: %v\n", err)
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, invDTO.InvoiceNumber))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
