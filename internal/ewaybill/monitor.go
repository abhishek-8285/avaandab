package ewaybill

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// Monitor periodically scans active E-Way Bills approaching expiration.
type Monitor struct {
	svc      *EWayBillService
	interval time.Duration
	leadSec  int
	logger   *slog.Logger
}

// NewMonitor creates a new E-Way Bill expiry monitor.
func NewMonitor(svc *EWayBillService, cfg Config) *Monitor {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	leadSec := cfg.ExtensionLeadSeconds
	if leadSec <= 0 {
		leadSec = 14400 // 4 hours
	}
	return &Monitor{
		svc:      svc,
		interval: interval,
		leadSec:  leadSec,
		logger:   svc.logger,
	}
}

// Run starts the monitor loop until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Tick(ctx)
		}
	}
}

// Tick performs a single scan pass.
func (m *Monitor) Tick(ctx context.Context) {
	query := fmt.Sprintf(`
		SELECT e.ewb_number, e.trip_id, t.status
		FROM eway_bills e
		LEFT JOIN trips t ON e.trip_id = t.id
		WHERE e.status IN ('active', 'part_a')
		  AND e.valid_until <= datetime('now', '+%d seconds')
		LIMIT 50
	`, m.leadSec)

	rows, err := m.svc.db.QueryContext(ctx, query)
	if err != nil {
		m.logger.Warn("ewaybill monitor query failed", "error", err)
		return
	}
	defer rows.Close()

	type expiringItem struct {
		ewbNumber  string
		tripID     string
		tripStatus string
	}
	var items []expiringItem
	for rows.Next() {
		var it expiringItem
		var tripID, tripStatus sql.NullString
		if err := rows.Scan(&it.ewbNumber, &tripID, &tripStatus); err == nil {
			it.tripID = tripID.String
			it.tripStatus = tripStatus.String
			items = append(items, it)
		}
	}

	for _, it := range items {
		switch it.tripStatus {
		case "in_transit", "started", "reached_pickup":
			// Try to extend if within geofence
			_, err := m.svc.Extend(ctx, it.ewbNumber, ExtendRequest{
				EwbNumber: it.ewbNumber,
				Reason:    "auto_expiry_monitor_extension",
			})
			if err != nil {
				m.logger.Debug("auto extend skipped/denied", "ewb", it.ewbNumber, "reason", err)
			}
		case "completed", "cancelled":
			// Cancel if trip is finished
			_, _ = m.svc.Cancel(ctx, it.ewbNumber, "trip_completed_or_cancelled")
		}
	}
}
