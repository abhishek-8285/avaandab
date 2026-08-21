package eta

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RecordHistory stores a completed trip segment for future ETA prediction.
// Called after trip arrival; tenant-scoped.
func (s *EtaService) RecordHistory(ctx context.Context, tenantID, tripID, segmentStart, segmentEnd string, actualMinutes int, trafficTag string) error {
	if s.db == nil {
		return fmt.Errorf("eta: no db")
	}
	if actualMinutes <= 0 {
		return fmt.Errorf("eta: invalid minutes")
	}
	now := time.Now().UTC()
	dayOfWeek := int(now.Weekday())
	hourOfDay := now.Hour()
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO eta_history (id, tenant_id, trip_id, segment_start, segment_end, actual_minutes, traffic_tag, day_of_week, hour_of_day, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, tripID, segmentStart, segmentEnd, actualMinutes, trafficTag, dayOfWeek, hourOfDay, now)
	return err
}

// PredictFromHistory returns historical average for a segment (90-day window).
// Returns avgMinutes and sampleCount.
func (s *EtaService) PredictFromHistory(ctx context.Context, tenantID, segmentStart, segmentEnd string) (float64, int, error) {
	var avg sql.NullFloat64
	var cnt int
	err := s.db.QueryRowContext(ctx,
		`SELECT AVG(actual_minutes), COUNT(*) FROM eta_history
		 WHERE tenant_id=? AND segment_start=? AND segment_end=? AND created_at > datetime('now','-90 days')`,
		tenantID, segmentStart, segmentEnd).Scan(&avg, &cnt)
	if err != nil {
		return 0, 0, err
	}
	if !avg.Valid || cnt == 0 {
		return 0, 0, fmt.Errorf("no history")
	}
	return avg.Float64, cnt, nil
}

// CleanupOldHistory deletes raw rows older than 90 days. Run daily via cron.
func (s *EtaService) CleanupOldHistory(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM eta_history WHERE created_at < datetime('now','-90 days')`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// AggregateMonthly rolls up raw history into eta_history_monthly (call after cleanup).
func (s *EtaService) AggregateMonthly(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO eta_history_monthly (tenant_id, segment_start, segment_end, month, avg_minutes, sample_count)
		SELECT tenant_id, segment_start, segment_end,
		       strftime('%Y-%m-01', created_at) as month,
		       AVG(actual_minutes), COUNT(*)
		FROM eta_history
		WHERE created_at < datetime('now','-90 days')
		GROUP BY tenant_id, segment_start, segment_end, month
	`)
	return err
}
