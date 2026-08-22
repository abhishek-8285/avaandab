package ewaybill

import (
	"context"
	"database/sql"
	"strings"

	"transport-app/internal/events"
)

// SubscribeTripEvents registers listeners for trip lifecycle events.
func (s *EWayBillService) SubscribeTripEvents(bus events.EventBus) {
	if bus == nil {
		return
	}

	handler := func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		if tripID == "" {
			return nil
		}

		// Check company_config for ewaybill_auto_generate
		autoGen := s.isAutoGenerateEnabled(ctx)
		if !autoGen {
			s.logger.Debug("ewaybill auto-generate skipped (disabled in config)", "trip_id", tripID)
			return nil
		}

		// Load trip goods_value
		var goodsVal float64
		err := s.db.QueryRowContext(ctx, `
			SELECT b.price
			FROM trips t
			JOIN bookings b ON t.booking_id = b.id
			WHERE t.id = ?
		`, tripID).Scan(&goodsVal)
		if err != nil {
			s.logger.Warn("could not resolve goods value for trip auto-generate", "trip_id", tripID, "error", err)
			return nil
		}

		if goodsVal <= s.cfg.MinInvoiceValue {
			s.logger.Debug("ewaybill auto-generate skipped (goods value <= 50k)", "trip_id", tripID, "goods_value", goodsVal)
			return nil
		}

		_, err = s.GeneratePartA(ctx, GeneratePartARequest{
			TripID:     tripID,
			GoodsValue: goodsVal,
			GenMode:    "AUTO",
		})
		if err != nil {
			s.logger.Warn("failed to auto-generate ewaybill", "trip_id", tripID, "error", err)
		}
		return nil
	}

	bus.Subscribe("TripConfirmedEvent", handler)
	bus.Subscribe("trip.confirmed", handler)
	bus.Subscribe("TripAssignedEvent", func(ctx context.Context, e events.Event) error {
		tripID := extractTripID(e.Payload)
		vehID := extractVehicleID(e.Payload)
		if tripID == "" || vehID == "" {
			return nil
		}

		var regNum string
		err := s.db.QueryRowContext(ctx, `SELECT registration_number FROM vehicles WHERE id = ?`, vehID).Scan(&regNum)
		if err != nil || regNum == "" {
			return nil
		}

		var ewbNum string
		err = s.db.QueryRowContext(ctx, `SELECT ewb_number FROM eway_bills WHERE trip_id = ? AND (vehicle_number IS NULL OR vehicle_number = '')`, tripID).Scan(&ewbNum)
		if err == nil && ewbNum != "" {
			_, _ = s.AttachPartB(ctx, ewbNum, regNum, "")
		}
		return nil
	})
}

func (s *EWayBillService) isAutoGenerateEnabled(ctx context.Context) bool {
	var val string
	// company_config columns are `key`/`value` (migration 00042) — an earlier
	// revision queried config_key/config_value, which always failed and fell
	// back to "enabled".
	err := s.db.QueryRowContext(ctx, `SELECT value FROM company_config WHERE key = 'ewaybill_auto_generate' LIMIT 1`).Scan(&val)
	if err != nil {
		if err == sql.ErrNoRows {
			return true // default true per spec
		}
		return true
	}
	val = strings.TrimSpace(strings.ToLower(val))
	return val == "true" || val == "1" || val == "yes"
}
