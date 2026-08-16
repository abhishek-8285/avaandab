package integration

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/auth"
	"transport-app/internal/integration/accounting"
	"transport-app/internal/integration/ewaybill"
	"transport-app/internal/integration/fastag"
	"transport-app/internal/integration/gstn"
	"transport-app/internal/middleware"
)

// Handler exposes stub integration endpoints.
type Handler struct {
	ewaybill   ewaybill.Client
	gstn       gstn.Client
	fastag     fastag.Client
	accounting accounting.Client
	authSrv    auth.AuthorizationService
}

// NewHandler builds a Handler with stub clients created from cfg.
func NewHandler(cfg Config, authSrv auth.AuthorizationService) *Handler {
	return &Handler{
		ewaybill:   ewaybill.NewClient(cfg.EWayBill),
		gstn:       gstn.NewClient(cfg.GSTN),
		fastag:     fastag.NewClient(cfg.FASTag),
		accounting: accounting.NewClient(cfg.Accounting),
		authSrv:    authSrv,
	}
}

// Register mounts integration routes under /api/v1/integrations.
func (h *Handler) Register(r chi.Router) {
	r.Route("/api/v1/integrations", func(r chi.Router) {
		r.Route("/ewaybill", func(r chi.Router) {
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "ewaybill")).Post("/generate", h.GenerateEWayBill)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "ewaybill")).Get("/get/{ewbNumber}", h.GetEWayBill)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "ewaybill")).Post("/cancel", h.CancelEWayBill)
		})
		r.Route("/gstn", func(r chi.Router) {
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "gstn")).Get("/validate/{gstin}", h.ValidateGSTIN)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "gstn")).Get("/gstr1-summary", h.GSTR1Summary)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "gstn")).Get("/gstr3b-summary", h.GSTR3BSummary)
		})
		r.Route("/fastag", func(r chi.Router) {
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "fastag")).Get("/balance", h.GetFASTagBalance)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "fastag")).Post("/deduct", h.DeductToll)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "fastag")).Get("/transactions", h.ListFASTagTransactions)
		})
		r.Route("/accounting", func(r chi.Router) {
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "accounting")).Post("/export-invoice", h.ExportInvoice)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "accounting")).Post("/sync-contacts", h.SyncContacts)
			r.With(middleware.RequirePermission(h.authSrv, "integrations", "accounting")).Post("/push-journal-entry", h.PushJournalEntry)
		})
	})
}

func (h *Handler) GenerateEWayBill(w http.ResponseWriter, r *http.Request) {
	var req ewaybill.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.ewaybill.Generate(r.Context(), req)
	if err != nil {
		http.Error(w, "E-Way Bill service unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) GetEWayBill(w http.ResponseWriter, r *http.Request) {
	ewbNumber := chi.URLParam(r, "ewbNumber")
	res, err := h.ewaybill.Get(r.Context(), ewbNumber)
	if err != nil {
		http.Error(w, "E-Way Bill service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) CancelEWayBill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EwbNumber string `json:"ewb_number"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.ewaybill.Cancel(r.Context(), req.EwbNumber, req.Reason)
	if err != nil {
		http.Error(w, "E-Way Bill service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) ValidateGSTIN(w http.ResponseWriter, r *http.Request) {
	gstin := chi.URLParam(r, "gstin")
	res, err := h.gstn.ValidateGSTIN(r.Context(), gstin)
	if err != nil {
		http.Error(w, "GSTN service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) GSTR1Summary(w http.ResponseWriter, r *http.Request) {
	gstin := r.URL.Query().Get("gstin")
	period := r.URL.Query().Get("period")
	res, err := h.gstn.FetchGSTR1Summary(r.Context(), gstin, period)
	if err != nil {
		http.Error(w, "GSTN service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) GSTR3BSummary(w http.ResponseWriter, r *http.Request) {
	gstin := r.URL.Query().Get("gstin")
	period := r.URL.Query().Get("period")
	res, err := h.gstn.FetchGSTR3BSummary(r.Context(), gstin, period)
	if err != nil {
		http.Error(w, "GSTN service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) GetFASTagBalance(w http.ResponseWriter, r *http.Request) {
	vehicle := r.URL.Query().Get("vehicle_number")
	tagID := r.URL.Query().Get("tag_id")
	res, err := h.fastag.GetBalance(r.Context(), vehicle, tagID)
	if err != nil {
		http.Error(w, "FASTag service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) DeductToll(w http.ResponseWriter, r *http.Request) {
	var req fastag.DeductTollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.fastag.DeductToll(r.Context(), req)
	if err != nil {
		http.Error(w, "FASTag service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) ListFASTagTransactions(w http.ResponseWriter, r *http.Request) {
	vehicle := r.URL.Query().Get("vehicle_number")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	res, err := h.fastag.ListTransactions(r.Context(), vehicle, limit)
	if err != nil {
		http.Error(w, "FASTag service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"transactions": res})
}

func (h *Handler) ExportInvoice(w http.ResponseWriter, r *http.Request) {
	var req accounting.ExportedInvoice
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.accounting.ExportInvoice(r.Context(), req)
	if err != nil {
		http.Error(w, "Accounting service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) SyncContacts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contacts []accounting.Contact `json:"contacts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.accounting.SyncContacts(r.Context(), req.Contacts)
	if err != nil {
		http.Error(w, "Accounting service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) PushJournalEntry(w http.ResponseWriter, r *http.Request) {
	var req accounting.JournalEntry
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.accounting.PushJournalEntry(r.Context(), req)
	if err != nil {
		http.Error(w, "Accounting service unavailable", http.StatusServiceUnavailable)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}
