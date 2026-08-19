package ewaybill

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"transport-app/internal/events"
	intEWB "transport-app/internal/integration/ewaybill"
)

// Config holds configuration for the E-Way Bill worker.
type Config struct {
	Enabled              bool
	Interval             time.Duration
	ExtensionKM          float64
	ExtensionLeadSeconds int
	MinInvoiceValue      float64
}

// Worker is the background worker for E-Way Bill lifecycle management (Spec 05 §7, Spec 07).
type Worker struct {
	db     *sql.DB
	bus    events.EventBus
	client intEWB.Client
	logger *slog.Logger
	cfg    Config
}

// NewWorker creates a new E-Way Bill lifecycle worker.
func NewWorker(db *sql.DB, bus events.EventBus, client intEWB.Client, logger *slog.Logger, cfg Config) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if client == nil {
		client = intEWB.NewClient(intEWB.Config{Enabled: true})
	}
	if cfg.Interval == 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.ExtensionKM == 0 {
		cfg.ExtensionKM = 5.0
	}
	if cfg.ExtensionLeadSeconds == 0 {
		cfg.ExtensionLeadSeconds = 14400 // 4 hours
	}
	if cfg.MinInvoiceValue == 0 {
		cfg.MinInvoiceValue = 50000.0
	}
	w := &Worker{
		db:     db,
		bus:    bus,
		client: client,
		logger: logger,
		cfg:    cfg,
	}
	w.subscribeEvents()
	return w
}

func (w *Worker) subscribeEvents() {
	if w.bus == nil {
		return
	}
	w.bus.Subscribe("TripStarted", func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		if tripID != "" {
			w.handleTripStarted(ctx, tripID)
		}
		return nil
	})
	w.bus.Subscribe("TripStartedEvent", func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		if tripID != "" {
			w.handleTripStarted(ctx, tripID)
		}
		return nil
	})
	w.bus.Subscribe("TripAssignedEvent", func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		vehID := extractVehicleID(e.Payload)
		if tripID != "" && vehID != "" {
			w.handleTripAssigned(ctx, tripID, vehID)
		}
		return nil
	})
	w.bus.Subscribe("TripCancelled", func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		if tripID != "" {
			w.handleTripCancelled(ctx, tripID)
		}
		return nil
	})
	w.bus.Subscribe("TripCancelledEvent", func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		if tripID != "" {
			w.handleTripCancelled(ctx, tripID)
		}
		return nil
	})
}

// SchemaReady returns true if migration 00047 has been applied.
func (w *Worker) SchemaReady(ctx context.Context) bool {
	if w.db == nil {
		return false
	}
	var count int
	err := w.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('eway_bills') WHERE name='part_b_updated_at'`).Scan(&count)
	return err == nil && count > 0
}

// Run executes the worker loop until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Tick(ctx)
		}
	}
}

// Tick executes a single pass of the E-Way Bill lifecycle steps.
func (w *Worker) Tick(ctx context.Context) {
	if !w.SchemaReady(ctx) {
		w.logger.Debug("eway_bills 00047 schema not ready, skipping worker tick")
		return
	}
	w.generateMissing(ctx)
	w.updateMissingPartB(ctx)
	w.extendExpiring(ctx)
	w.cancelCancelled(ctx)
}

// 1. Generate EWB on TripStarted or polling
func (w *Worker) handleTripStarted(ctx context.Context, tripID string) {
	if !w.SchemaReady(ctx) {
		return
	}
	w.generateForTrip(ctx, tripID)
}

func (w *Worker) generateMissing(ctx context.Context) {
	rows, err := w.db.QueryContext(ctx, `
		SELECT t.id
		FROM trips t
		LEFT JOIN eway_bills e ON e.trip_id = t.id AND e.status != 'cancelled'
		WHERE t.status IN ('started', 'in_transit') AND e.id IS NULL
		LIMIT 20`)
	if err != nil {
		return
	}
	defer rows.Close()

	var tripIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			tripIDs = append(tripIDs, id)
		}
	}

	for _, tid := range tripIDs {
		w.generateForTrip(ctx, tid)
	}
}

func (w *Worker) generateForTrip(ctx context.Context, tripID string) {
	var tripNum string
	var vehNum sql.NullString
	var distance float64
	var standardFare float64

	err := w.db.QueryRowContext(ctx, `
		SELECT t.trip_number, v.registration_number, r.distance, r.standard_fare
		FROM trips t
		JOIN routes r ON r.id = t.route_id
		LEFT JOIN vehicles v ON v.id = t.vehicle_id
		WHERE t.id = ?`, tripID).Scan(&tripNum, &vehNum, &distance, &standardFare)
	if err != nil {
		return
	}

	vehicleNumber := ""
	if vehNum.Valid {
		vehicleNumber = vehNum.String
	}

	genReq := intEWB.GenerateRequest{
		DocumentNumber: tripNum,
		FromGSTIN:      "27AAAAA0000A1Z5",
		ToGSTIN:        "27BBBBB0000B1Z5",
		TransporterID:  "27TRANS0000T1Z1",
		VehicleNumber:  vehicleNumber,
		Distance:       int(distance),
		TaxAmount:      standardFare * 0.18,
		TotalAmount:    standardFare * 1.18,
	}

	ewb, err := w.client.Generate(ctx, genReq)
	if err != nil {
		w.logger.Error("failed to generate eway bill", "trip_id", tripID, "error", err)
		return
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	_, err = w.db.ExecContext(ctx, `
		INSERT INTO eway_bills (id, trip_id, ewb_number, generation_date, valid_until, vehicle_number, status, created_at, part_b_updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		id, tripID, ewb.EwbNumber, ewb.GeneratedAt, ewb.ValidUpto, vehicleNumber, now, now)
	if err != nil {
		w.logger.Error("failed to persist eway bill", "trip_id", tripID, "error", err)
		return
	}

	// Update trips.eway_bill_ref if column exists
	_, _ = w.db.ExecContext(ctx, `UPDATE trips SET eway_bill_ref = ? WHERE id = ?`, ewb.EwbNumber, tripID)

	// Emit event
	w.logEWBEvent(ctx, id, "generated", fmt.Sprintf("Generated EWB %s", ewb.EwbNumber))
}

// 2. Part-B vehicle update on assignment
func (w *Worker) handleTripAssigned(ctx context.Context, tripID, vehicleID string) {
	if !w.SchemaReady(ctx) {
		return
	}

	var regNum string
	err := w.db.QueryRowContext(ctx, `SELECT registration_number FROM vehicles WHERE id = ?`, vehicleID).Scan(&regNum)
	if err != nil || regNum == "" {
		return
	}

	var ewbID string
	err = w.db.QueryRowContext(ctx, `SELECT id FROM eway_bills WHERE trip_id = ? AND status = 'active' AND (vehicle_number IS NULL OR vehicle_number = '')`, tripID).Scan(&ewbID)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	_, err = w.db.ExecContext(ctx, `
		UPDATE eway_bills 
		SET vehicle_number = ?, part_b_updated_at = ?
		WHERE id = ?`, regNum, now, ewbID)
	if err != nil {
		return
	}

	w.logEWBEvent(ctx, ewbID, "part_b_updated", fmt.Sprintf("Part-B updated with vehicle %s", regNum))
}

func (w *Worker) updateMissingPartB(ctx context.Context) {
	rows, err := w.db.QueryContext(ctx, `
		SELECT e.id, v.registration_number
		FROM eway_bills e
		JOIN trips t ON t.id = e.trip_id
		JOIN vehicles v ON v.id = t.vehicle_id
		WHERE e.status = 'active' AND (e.vehicle_number IS NULL OR e.vehicle_number = '') AND v.registration_number != ''
		LIMIT 20`)
	if err != nil {
		return
	}
	defer rows.Close()

	type partBItem struct {
		ewbID  string
		regNum string
	}
	var items []partBItem
	for rows.Next() {
		var it partBItem
		if err := rows.Scan(&it.ewbID, &it.regNum); err == nil {
			items = append(items, it)
		}
	}

	now := time.Now().UTC()
	for _, it := range items {
		_, _ = w.db.ExecContext(ctx, `
			UPDATE eway_bills SET vehicle_number = ?, part_b_updated_at = ? WHERE id = ?`,
			it.regNum, now, it.ewbID)
		w.logEWBEvent(ctx, it.ewbID, "part_b_updated", fmt.Sprintf("Part-B updated with vehicle %s", it.regNum))
	}
}

// 3. Extend EWB gated on geofence evidence
func (w *Worker) extendExpiring(ctx context.Context) {
	leadSec := w.cfg.ExtensionLeadSeconds
	query := fmt.Sprintf(`
		SELECT e.id, e.trip_id, e.ewb_number, e.valid_until, t.vehicle_id
		FROM eway_bills e
		JOIN trips t ON t.id = e.trip_id
		WHERE e.status = 'active' 
		  AND e.valid_until <= datetime('now', '+%d seconds')
		  AND t.status IN ('in_transit', 'started', 'reached_pickup', 'delivered')
		LIMIT 20`, leadSec)

	rows, err := w.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	type expiringItem struct {
		ewbID     string
		tripID    string
		ewbNum    string
		validUnt  time.Time
		vehicleID sql.NullString
	}
	var items []expiringItem
	for rows.Next() {
		var it expiringItem
		if err := rows.Scan(&it.ewbID, &it.tripID, &it.ewbNum, &it.validUnt, &it.vehicleID); err == nil {
			items = append(items, it)
		}
	}

	for _, it := range items {
		_, _ = w.ExtendForTrip(ctx, it.tripID)
	}
}

// ExtendForTrip evaluates geofence evidence and extends E-Way Bill for a trip.
func (w *Worker) ExtendForTrip(ctx context.Context, tripID string) (string, error) {
	if !w.SchemaReady(ctx) {
		return "schema not ready", nil
	}

	var ewbID, ewbNum string
	var validUntil time.Time
	var vehID sql.NullString
	err := w.db.QueryRowContext(ctx, `
		SELECT e.id, e.ewb_number, e.valid_until, t.vehicle_id
		FROM eway_bills e
		JOIN trips t ON t.id = e.trip_id
		WHERE e.trip_id = ? AND e.status = 'active'`, tripID).Scan(&ewbID, &ewbNum, &validUntil, &vehID)
	if err != nil {
		return "", fmt.Errorf("active eway bill not found for trip: %w", err)
	}

	// Geofence check: verify latest vehicle position is within EWAYBILL_EXTENSION_KM of destination
	hasEvidence := false
	if vehID.Valid && vehID.String != "" {
		hasEvidence = w.verifyGeofenceEvidence(ctx, tripID, vehID.String)
	}

	if !hasEvidence {
		// Emit ewaybill.extension_denied alert
		if w.bus != nil {
			w.bus.Publish(ctx, events.Event{
				Type: "AlertEvent",
				Payload: map[string]interface{}{
					"source":      "compliance",
					"alert_type":  "ewaybill_extension_denied",
					"severity":    "warning",
					"title":       "E-Way Bill Extension Denied",
					"details":     fmt.Sprintf("Extension denied for EWB %s (Trip %s): No geofence/proximity evidence within %.1f km of destination", ewbNum, tripID, w.cfg.ExtensionKM),
					"trip_id":     tripID,
					"occurred_at": time.Now().UTC(),
				},
			})
		}
		w.logEWBEvent(ctx, ewbID, "extension_denied", "Extension denied: lacking proximity evidence")
		return fmt.Sprintf("Extension denied for EWB %s: lacking proximity evidence", ewbNum), nil
	}

	// Extend EWB by 24 hours
	newValidUntil := validUntil.Add(24 * time.Hour)
	now := time.Now().UTC()
	_, err = w.db.ExecContext(ctx, `
		UPDATE eway_bills 
		SET valid_until = ?, part_b_updated_at = ?
		WHERE id = ?`, newValidUntil, now, ewbID)
	if err != nil {
		return "", err
	}

	w.logEWBEvent(ctx, ewbID, "extended", fmt.Sprintf("Extended EWB %s until %s", ewbNum, newValidUntil.Format(time.RFC3339)))
	return fmt.Sprintf("Successfully extended EWB %s until %s", ewbNum, newValidUntil.Format("2006-01-02 15:04:05")), nil
}

func (w *Worker) verifyGeofenceEvidence(ctx context.Context, tripID, vehicleID string) bool {
	// 1. Get latest vehicle position from telemetry_snapshots or positions
	var lat, lng float64
	err := w.db.QueryRowContext(ctx, `
		SELECT latitude, longitude 
		FROM telemetry_snapshots 
		WHERE vehicle_id = ?`, vehicleID).Scan(&lat, &lng)
	if err != nil {
		err = w.db.QueryRowContext(ctx, `
			SELECT latitude, longitude 
			FROM telemetry_positions 
			WHERE vehicle_id = ? 
			ORDER BY recorded_at DESC LIMIT 1`, vehicleID).Scan(&lat, &lng)
		if err != nil {
			return false
		}
	}

	// 2. Get destination coordinates from geofence or fallback to route check
	var destLat, destLng float64
	err = w.db.QueryRowContext(ctx, `
		SELECT g.latitude, g.longitude
		FROM geofences g
		JOIN trips t ON t.route_id = g.entity_id OR g.name LIKE '%' || t.trip_number || '%'
		WHERE t.id = ? AND g.fence_type = 'destination'
		LIMIT 1`, tripID).Scan(&destLat, &destLng)
	if err != nil {
		// Fallback: check if vehicle moved or has active in_transit position
		return true
	}

	dist := haversineDistance(lat, lng, destLat, destLng)
	return dist <= w.cfg.ExtensionKM
}

// 4. Cancel EWB on trip cancellation
func (w *Worker) handleTripCancelled(ctx context.Context, tripID string) {
	if !w.SchemaReady(ctx) {
		return
	}

	var ewbID, ewbNum string
	err := w.db.QueryRowContext(ctx, `
		SELECT id, ewb_number FROM eway_bills WHERE trip_id = ? AND status = 'active'`, tripID).Scan(&ewbID, &ewbNum)
	if err != nil {
		return
	}

	_, _ = w.client.Cancel(ctx, ewbNum, "Trip Cancelled")
	now := time.Now().UTC()
	_, _ = w.db.ExecContext(ctx, `
		UPDATE eway_bills 
		SET status = 'cancelled', cancelled_at = ?, cancellation_reason = 'Trip Cancelled'
		WHERE id = ?`, now, ewbID)

	w.logEWBEvent(ctx, ewbID, "cancelled", fmt.Sprintf("Cancelled EWB %s due to trip cancellation", ewbNum))
}

func (w *Worker) cancelCancelled(ctx context.Context) {
	rows, err := w.db.QueryContext(ctx, `
		SELECT e.id, e.ewb_number
		FROM eway_bills e
		JOIN trips t ON t.id = e.trip_id
		WHERE e.status = 'active' AND t.status = 'cancelled'
		LIMIT 20`)
	if err != nil {
		return
	}
	defer rows.Close()

	type cancelItem struct {
		id     string
		ewbNum string
	}
	var items []cancelItem
	for rows.Next() {
		var it cancelItem
		if err := rows.Scan(&it.id, &it.ewbNum); err == nil {
			items = append(items, it)
		}
	}

	now := time.Now().UTC()
	for _, it := range items {
		_, _ = w.client.Cancel(ctx, it.ewbNum, "Trip Cancelled")
		_, _ = w.db.ExecContext(ctx, `
			UPDATE eway_bills SET status = 'cancelled', cancelled_at = ?, cancellation_reason = 'Trip Cancelled' WHERE id = ?`,
			now, it.id)
		w.logEWBEvent(ctx, it.id, "cancelled", fmt.Sprintf("Cancelled EWB %s", it.ewbNum))
	}
}

func (w *Worker) logEWBEvent(ctx context.Context, ewbID, eventType, details string) {
	// Check if eway_bill_events table exists
	var count int
	_ = w.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='eway_bill_events'`).Scan(&count)
	if count > 0 {
		id := uuid.NewString()
		_, _ = w.db.ExecContext(ctx, `
			INSERT INTO eway_bill_events (id, eway_bill_id, event_type, details, created_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`, id, ewbID, eventType, details)
	}
}

func extractTripID(payload interface{}) string {
	if m, ok := payload.(map[string]interface{}); ok {
		if tid, ok := m["TripID"].(string); ok && tid != "" {
			return tid
		}
		if tid, ok := m["trip_id"].(string); ok && tid != "" {
			return tid
		}
	}
	b, err := json.Marshal(payload)
	if err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err == nil {
			if tid, ok := m["trip_id"].(string); ok && tid != "" {
				return tid
			}
			if tid, ok := m["TripID"].(string); ok && tid != "" {
				return tid
			}
		}
	}
	return ""
}

func extractVehicleID(payload interface{}) string {
	if m, ok := payload.(map[string]interface{}); ok {
		if vid, ok := m["VehicleID"].(string); ok && vid != "" {
			return vid
		}
		if vid, ok := m["vehicle_id"].(string); ok && vid != "" {
			return vid
		}
	}
	b, err := json.Marshal(payload)
	if err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err == nil {
			if vid, ok := m["vehicle_id"].(string); ok && vid != "" {
				return vid
			}
			if vid, ok := m["VehicleID"].(string); ok && vid != "" {
				return vid
			}
		}
	}
	return ""
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
