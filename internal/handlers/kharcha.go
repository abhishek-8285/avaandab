package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
	"transport-app/internal/service"
)

// KharchaHandlers handles the driver expense (kharcha) approval dashboard.
type KharchaHandlers struct {
	*App
}

func (h *KharchaHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "read")).Get("/", h.Dashboard)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "read")).Get("/pending", h.PendingQueue)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "read")).Get("/ledger", h.Ledger)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/approve", h.Approve)
	r.With(middleware.ResourcePermission(h.AuthSrv, "trips", "update")).Post("/{id}/reject", h.Reject)
}

// GET /kharcha — full dashboard (pending queue + ledger).
func (h *KharchaHandlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	ctx := r.Context()

	pending, _ := h.Services.Kharcha.ListPendingExpenses(ctx)
	ledger, _ := h.Services.Kharcha.ListLedger(ctx, "")
	stats, _ := h.Services.Kharcha.GetKharchaStats(ctx)
	trips, _, _ := h.Services.Trips.ListTrips(ctx, "", "in_transit", 100, 0)

	h.renderPage(w, r, "kharcha_dashboard.html", PageData{
		Title: "Kharcha Ledger",
		User:  session,
		Extra: map[string]interface{}{
			"PendingExpenses": pending,
			"LedgerEntries":   ledger,
			"Stats":           stats,
			"ActiveTrips":     trips,
		},
	})
}

// GET /kharcha/pending — HTMX partial: live-refresh the queue every 30s.
func (h *KharchaHandlers) PendingQueue(w http.ResponseWriter, r *http.Request) {
	pending, _ := h.Services.Kharcha.ListPendingExpenses(r.Context())
	h.renderFragment(w, "kharcha_queue.html", map[string]interface{}{
		"PendingExpenses": pending,
	})
}

// GET /kharcha/ledger?trip_id= — HTMX partial: filtered ledger rows.
func (h *KharchaHandlers) Ledger(w http.ResponseWriter, r *http.Request) {
	tripID := r.URL.Query().Get("trip_id")
	entries, _ := h.Services.Kharcha.ListLedger(r.Context(), tripID)
	h.renderFragment(w, "kharcha_ledger_rows.html", map[string]interface{}{
		"LedgerEntries": entries,
	})
}

// POST /kharcha/{id}/approve — HTMX inline swap: approve and replace the row.
func (h *KharchaHandlers) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expenseID := chi.URLParam(r, "id")
	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.Services.Kharcha.ApproveExpense(ctx, expenseID, session.UserID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `<div class="px-6 py-4 bg-red-50 text-red-600 text-sm font-semibold border-l-4 border-red-500">Error: %s</div>`, template.HTMLEscapeString(err.Error()))
		return
	}

	expense, err := h.Services.Kharcha.GetExpenseByID(ctx, expenseID)
	if err != nil {
		// Silently replace row with approved confirmation
		_, _ = fmt.Fprintf(w, `<div class="px-6 py-4 flex items-center gap-3 bg-emerald-50/60"><span class="w-2 h-2 rounded-full bg-emerald-500"></span><span class="text-sm font-semibold text-emerald-700">Expense approved successfully.</span></div>`)
		return
	}

	h.renderFragment(w, "kharcha_row_approved.html", expense)
}

// POST /kharcha/{id}/reject — form post with reason field.
func (h *KharchaHandlers) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expenseID := chi.URLParam(r, "id")
	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	reason := r.FormValue("reason")

	if err := h.Services.Kharcha.RejectExpense(ctx, expenseID, session.UserID, reason); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `<div class="px-6 py-4 bg-red-50 text-red-600 text-sm font-semibold border-l-4 border-red-500">Error: %s</div>`, template.HTMLEscapeString(err.Error()))
		return
	}

	expense, err := h.Services.Kharcha.GetExpenseByID(ctx, expenseID)
	if err != nil {
		_, _ = fmt.Fprintf(w, `<div class="px-6 py-4 flex items-center gap-3 bg-rose-50/60"><span class="w-2 h-2 rounded-full bg-rose-500"></span><span class="text-sm font-semibold text-rose-700">Expense rejected.</span></div>`)
		return
	}

	h.renderFragment(w, "kharcha_row_rejected.html", expense)
}

// DeliverWithPOD handles e-POD submission from driver mobile (multipart form).
// POST /trips/{id}/deliver-pod
func (h *KharchaHandlers) DeliverWithPOD(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tripID := chi.URLParam(r, "id")

	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "form parse error", http.StatusBadRequest)
		return
	}

	trip, err := h.Services.Trips.GetTrip(ctx, domain.TripID(tripID))
	if err != nil {
		http.Error(w, "trip not found", http.StatusNotFound)
		return
	}

	if trip.DriverID == nil || string(*trip.DriverID) != session.UserID {
		http.Error(w, "forbidden: trip is not assigned to this user", http.StatusForbidden)
		return
	}

	consigneeName := r.FormValue("consignee_name")
	consigneePhone := r.FormValue("consignee_phone")
	notes := r.FormValue("notes")

	// Upload POD photo using existing UploadFile if provided
	var podPhotoURL string
	if _, fh, err := r.FormFile("pod_photo"); err == nil {
		if fileRec, saveErr := h.Services.Files.UploadFile(ctx, fh, "trip_pod", tripID); saveErr == nil {
			podPhotoURL = "/files/" + string(fileRec.ID)
		}
	}

	req := service.DeliverWithPODRequest{
		ConsigneeName:  consigneeName,
		ConsigneePhone: consigneePhone,
		Notes:          notes,
		PODPhotoURL:    podPhotoURL,
	}

	tripNum, err := h.Services.Trips.DeliverWithPOD(ctx, tripID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.renderFragment(w, "epod_success.html", map[string]interface{}{
		"TripNumber": tripNum,
	})
}
