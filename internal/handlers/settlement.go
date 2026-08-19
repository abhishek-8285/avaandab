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
	"transport-app/internal/service"
)

// SettlementHandlers exposes REST APIs for driver settlements, rate calculations, TDS, and dispute workflows.
type SettlementHandlers struct {
	*App
	settleSvc *service.DriverSettlementService
	authSrv   auth.AuthorizationService
}

// NewSettlementHandlers constructs a new SettlementHandlers instance.
func NewSettlementHandlers(app *App, settleSvc *service.DriverSettlementService, authSrv auth.AuthorizationService) *SettlementHandlers {
	return &SettlementHandlers{
		App:       app,
		settleSvc: settleSvc,
		authSrv:   authSrv,
	}
}

// Mount registers settlement endpoints on the router.
func (h *SettlementHandlers) Mount(r chi.Router) {
	setupRoutes := func(sub chi.Router) {
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "write")).Post("/generate", h.Generate)
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "read")).Get("/", h.List)
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "read")).Get("/{id}", h.GetByID)
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "read")).Get("/{id}/deductions", h.GetDeductions)
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "approve")).Post("/{id}/mark-paid", h.MarkPaid)
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "read")).Post("/{id}/confirm", h.Confirm)
		sub.With(middleware.RequirePermission(h.authSrv, "settlements", "read")).Post("/{id}/dispute", h.Dispute)
	}

	r.Route("/api/settlements", setupRoutes)
	r.Route("/api/v1/settlements", setupRoutes)
}

// Generate handles settlement generation (and recalculation if force_recompute=true).
func (h *SettlementHandlers) Generate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TripID         string `json:"trip_id"`
		ForceRecompute bool   `json:"force_recompute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TripID == "" {
		http.Error(w, `{"error":"invalid_request","message":"trip_id is required"}`, http.StatusBadRequest)
		return
	}

	rec, err := h.settleSvc.GenerateSettlement(r.Context(), req.TripID, req.ForceRecompute)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case errors.Is(err, service.ErrTripNotFound):
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "trip_not_found", "message": "Trip not found"})
		case errors.Is(err, service.ErrDriverNotAssigned):
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "driver_not_assigned", "message": "Driver not assigned to trip"})
		case errors.Is(err, service.ErrBookingPriceMissing):
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "booking_price_missing", "message": "Booking price missing for commission rate calculation"})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "settlement_failed", "message": err.Error()})
		}
		return
	}

	var rateBasis map[string]interface{}
	if rec.RateBasisJSON != "" {
		_ = json.Unmarshal([]byte(rec.RateBasisJSON), &rateBasis)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"settlement_id":       rec.ID,
		"trip_id":             rec.TripID,
		"driver_id":           rec.DriverID,
		"gross_fare":          rec.GrossFare,
		"rate_model":          rec.RateModel,
		"rate_basis":          rateBasis,
		"commission_amount":   rec.CommissionAmount,
		"advances_kharcha":    rec.AdvancesKharcha,
		"approved_deductions": rec.Deductions,
		"performance_bonus":   rec.PerformanceBonus,
		"tds_rate":            rec.TDSRate,
		"tds_amount":          rec.TDSAmount,
		"net_payout":          rec.NetPayout,
		"status":              rec.Status,
		"lines":               rec.Lines,
	})
}

// List returns driver settlements matching query filters.
func (h *SettlementHandlers) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	driverID := r.URL.Query().Get("driver_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	results, err := h.settleSvc.ListSettlements(r.Context(), status, driverID, limit, offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"settlements": results,
		"count":       len(results),
	})
}

// GetByID returns detailed settlement breakdown by settlement ID.
func (h *SettlementHandlers) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := h.settleSvc.GetSettlement(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, service.ErrSettlementNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "settlement_not_found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

// GetDeductions returns kharcha advances and TDS deductions for a settlement.
func (h *SettlementHandlers) GetDeductions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := h.settleSvc.GetSettlement(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "settlement_not_found"})
		return
	}

	var deductionLines []service.SettlementLine
	for _, l := range rec.Lines {
		if l.LineType == "advances" || l.LineType == "deduction" || l.LineType == "tds" {
			deductionLines = append(deductionLines, l)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"settlement_id":       rec.ID,
		"advances_kharcha":    rec.AdvancesKharcha,
		"approved_deductions": rec.Deductions,
		"tds_rate":            rec.TDSRate,
		"tds_amount":          rec.TDSAmount,
		"lines":               deductionLines,
	})
}

// MarkPaid marks a settlement as paid with transaction reference.
func (h *SettlementHandlers) MarkPaid(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		PaymentRef string `json:"payment_ref"`
		PaidAt     string `json:"paid_at"`
		Mode       string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PaymentRef == "" {
		http.Error(w, `{"error":"invalid_request","message":"payment_ref is required"}`, http.StatusBadRequest)
		return
	}

	paidAt := time.Now()
	if req.PaidAt != "" {
		if t, err := time.Parse(time.RFC3339, req.PaidAt); err == nil {
			paidAt = t
		}
	}

	rec, err := h.settleSvc.MarkPaid(r.Context(), id, req.PaymentRef, paidAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, service.ErrSettlementNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "settlement_not_found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

// Confirm handles driver confirmation of received settlement payout.
func (h *SettlementHandlers) Confirm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := h.settleSvc.ConfirmSettlement(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "settlement_not_found"})
		return
	}

	confTime := ""
	if rec.ConfirmedAt != nil {
		confTime = rec.ConfirmedAt.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       rec.Status,
		"confirmed_at": confTime,
	})
}

// Dispute handles driver dispute against settlement payout.
func (h *SettlementHandlers) Dispute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Reason      string  `json:"reason"`
		ExpectedNet float64 `json:"expected_net"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		http.Error(w, `{"error":"invalid_request","message":"reason is required"}`, http.StatusBadRequest)
		return
	}

	rec, err := h.settleSvc.DisputeSettlement(r.Context(), id, req.Reason, req.ExpectedNet)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "settlement_not_found"})
		return
	}

	dispReason := ""
	if rec.DisputeReason != nil {
		dispReason = *rec.DisputeReason
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         rec.Status,
		"dispute_reason": dispReason,
	})
}
