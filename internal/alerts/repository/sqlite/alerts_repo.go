package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"transport-app/internal/alerts/domain"
	"transport-app/internal/alerts/repository"
)

type sqlAlertRepository struct {
	db *sql.DB
}

// NewAlertRepository creates a new SQLite-backed AlertRepository.
func NewAlertRepository(db *sql.DB) repository.AlertRepository {
	return &sqlAlertRepository{db: db}
}

func (r *sqlAlertRepository) FindOpenByDedupKey(ctx context.Context, dedupKey string) (*domain.Alert, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, rule_id, source, alert_type, severity, status, dedup_key,
		       entity_type, entity_id, user_id, title, message, occurrences,
		       first_seen_at, last_seen_at, next_escalation_at, escalation_step,
		       latitude, longitude, metadata, acked_by, acked_at, resolved_by, resolved_at,
		       created_at, updated_at
		FROM alerts
		WHERE dedup_key = ? AND status IN ('open', 'acknowledged', 'escalated')
		ORDER BY last_seen_at DESC
		LIMIT 1`, dedupKey)

	var a domain.Alert
	var ruleID, entityType, entityID, userID, metadata sql.NullString
	var ackedBy, resolvedBy sql.NullString
	var nextEscalationAt, ackedAt, resolvedAt sql.NullTime
	var lat, lng sql.NullFloat64

	err := row.Scan(
		&a.ID, &ruleID, &a.Source, &a.AlertType, &a.Severity, &a.Status, &a.DedupKey,
		&entityType, &entityID, &userID, &a.Title, &a.Message, &a.Occurrences,
		&a.FirstSeenAt, &a.LastSeenAt, &nextEscalationAt, &a.EscalationStep,
		&lat, &lng, &metadata, &ackedBy, &ackedAt, &resolvedBy, &resolvedAt,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if ruleID.Valid {
		a.RuleID = &ruleID.String
	}
	if entityType.Valid {
		a.EntityType = &entityType.String
	}
	if entityID.Valid {
		a.EntityID = &entityID.String
	}
	if userID.Valid {
		a.UserID = &userID.String
	}
	if metadata.Valid {
		a.Metadata = metadata.String
	}
	if ackedBy.Valid {
		a.AckedBy = &ackedBy.String
	}
	if resolvedBy.Valid {
		a.ResolvedBy = &resolvedBy.String
	}
	if nextEscalationAt.Valid {
		a.NextEscalationAt = &nextEscalationAt.Time
	}
	if ackedAt.Valid {
		a.AckedAt = &ackedAt.Time
	}
	if resolvedAt.Valid {
		a.ResolvedAt = &resolvedAt.Time
	}
	if lat.Valid {
		a.Latitude = &lat.Float64
	}
	if lng.Valid {
		a.Longitude = &lng.Float64
	}

	return &a, nil
}

func (r *sqlAlertRepository) CreateAlert(ctx context.Context, a *domain.Alert) error {
	now := time.Now().UTC()
	if a.FirstSeenAt.IsZero() {
		a.FirstSeenAt = now
	}
	if a.LastSeenAt.IsZero() {
		a.LastSeenAt = now
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	if a.Occurrences == 0 {
		a.Occurrences = 1
	}
	if a.Status == "" {
		a.Status = domain.StatusOpen
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO alerts (
			id, rule_id, source, alert_type, severity, status, dedup_key,
			entity_type, entity_id, user_id, title, message, occurrences,
			first_seen_at, last_seen_at, next_escalation_at, escalation_step,
			latitude, longitude, metadata, acked_by, acked_at, resolved_by, resolved_at,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?
		)`,
		a.ID, a.RuleID, a.Source, a.AlertType, a.Severity, a.Status, a.DedupKey,
		a.EntityType, a.EntityID, a.UserID, a.Title, a.Message, a.Occurrences,
		a.FirstSeenAt, a.LastSeenAt, a.NextEscalationAt, a.EscalationStep,
		a.Latitude, a.Longitude, a.Metadata, a.AckedBy, a.AckedAt, a.ResolvedBy, a.ResolvedAt,
		a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *sqlAlertRepository) IncrementOccurrences(ctx context.Context, alertID string, lastSeen time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE alerts
		SET occurrences = occurrences + 1,
		    last_seen_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, lastSeen, alertID)
	return err
}

func (r *sqlAlertRepository) ListRulesBySource(ctx context.Context, source string) ([]domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, source, alert_type, name, severity, threshold, threshold_unit,
		       dedup_key_expr, cooldown_seconds, storm_window_seconds, storm_batch_min,
		       channel_routing, escalation_schedule, is_active, created_at, updated_at
		FROM alert_rules
		WHERE source = ? AND is_active = 1`, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.Rule
	for rows.Next() {
		var rule domain.Rule
		var threshold sql.NullFloat64
		var thresholdUnit, escalationSchedule sql.NullString
		var isActive int

		err := rows.Scan(
			&rule.ID, &rule.Source, &rule.AlertType, &rule.Name, &rule.Severity,
			&threshold, &thresholdUnit, &rule.DedupKeyExpr, &rule.CooldownSeconds,
			&rule.StormWindowSeconds, &rule.StormBatchMin, &rule.ChannelRouting,
			&escalationSchedule, &isActive, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if threshold.Valid {
			rule.Threshold = &threshold.Float64
		}
		if thresholdUnit.Valid {
			rule.ThresholdUnit = &thresholdUnit.String
		}
		if escalationSchedule.Valid {
			rule.EscalationSchedule = &escalationSchedule.String
		}
		rule.IsActive = isActive == 1
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *sqlAlertRepository) GetRule(ctx context.Context, ruleID string) (*domain.Rule, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, source, alert_type, name, severity, threshold, threshold_unit,
		       dedup_key_expr, cooldown_seconds, storm_window_seconds, storm_batch_min,
		       channel_routing, escalation_schedule, is_active, created_at, updated_at
		FROM alert_rules
		WHERE id = ?`, ruleID)

	var rule domain.Rule
	var threshold sql.NullFloat64
	var thresholdUnit, escalationSchedule sql.NullString
	var isActive int

	err := row.Scan(
		&rule.ID, &rule.Source, &rule.AlertType, &rule.Name, &rule.Severity,
		&threshold, &thresholdUnit, &rule.DedupKeyExpr, &rule.CooldownSeconds,
		&rule.StormWindowSeconds, &rule.StormBatchMin, &rule.ChannelRouting,
		&escalationSchedule, &isActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if threshold.Valid {
		rule.Threshold = &threshold.Float64
	}
	if thresholdUnit.Valid {
		rule.ThresholdUnit = &thresholdUnit.String
	}
	if escalationSchedule.Valid {
		rule.EscalationSchedule = &escalationSchedule.String
	}
	rule.IsActive = isActive == 1
	return &rule, nil
}

func (r *sqlAlertRepository) GetActiveRuleForType(ctx context.Context, source, alertType string, entityID string) (*domain.Rule, *domain.RuleOverride, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, source, alert_type, name, severity, threshold, threshold_unit,
		       dedup_key_expr, cooldown_seconds, storm_window_seconds, storm_batch_min,
		       channel_routing, escalation_schedule, is_active, created_at, updated_at
		FROM alert_rules
		WHERE source = ? AND alert_type = ? AND is_active = 1
		LIMIT 1`, source, alertType)

	var rule domain.Rule
	var threshold sql.NullFloat64
	var thresholdUnit, escalationSchedule sql.NullString
	var isActive int

	err := row.Scan(
		&rule.ID, &rule.Source, &rule.AlertType, &rule.Name, &rule.Severity,
		&threshold, &thresholdUnit, &rule.DedupKeyExpr, &rule.CooldownSeconds,
		&rule.StormWindowSeconds, &rule.StormBatchMin, &rule.ChannelRouting,
		&escalationSchedule, &isActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if threshold.Valid {
		rule.Threshold = &threshold.Float64
	}
	if thresholdUnit.Valid {
		rule.ThresholdUnit = &thresholdUnit.String
	}
	if escalationSchedule.Valid {
		rule.EscalationSchedule = &escalationSchedule.String
	}
	rule.IsActive = isActive == 1

	if entityID != "" {
		override, _ := r.GetOverrides(ctx, rule.ID, entityID)
		return &rule, override, nil
	}

	return &rule, nil, nil
}

func (r *sqlAlertRepository) GetOverrides(ctx context.Context, ruleID string, entityID string) (*domain.RuleOverride, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, rule_id, entity_id, severity, threshold, cooldown_seconds, channels, is_active, created_at
		FROM rule_overrides
		WHERE rule_id = ? AND (entity_id = ? OR entity_id IS NULL) AND is_active = 1
		ORDER BY entity_id DESC
		LIMIT 1`, ruleID, entityID)

	var o domain.RuleOverride
	var entityIDVal, severity, channels sql.NullString
	var threshold sql.NullFloat64
	var cooldown sql.NullInt64
	var isActive int

	err := row.Scan(&o.ID, &o.RuleID, &entityIDVal, &severity, &threshold, &cooldown, &channels, &isActive, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if entityIDVal.Valid {
		o.EntityID = &entityIDVal.String
	}
	if severity.Valid {
		o.Severity = &severity.String
	}
	if threshold.Valid {
		o.Threshold = &threshold.Float64
	}
	if cooldown.Valid {
		c := int(cooldown.Int64)
		o.CooldownSeconds = &c
	}
	if channels.Valid {
		o.Channels = &channels.String
	}
	o.IsActive = isActive == 1
	return &o, nil
}

func (r *sqlAlertRepository) ListAlerts(ctx context.Context, status string, limit, offset int) ([]domain.Alert, error) {
	query := `
		SELECT id, rule_id, source, alert_type, severity, status, dedup_key,
		       entity_type, entity_id, user_id, title, message, occurrences,
		       first_seen_at, last_seen_at, next_escalation_at, escalation_step,
		       latitude, longitude, metadata, acked_by, acked_at, resolved_by, resolved_at,
		       created_at, updated_at
		FROM alerts `
	var args []interface{}
	if status != "" {
		query += "WHERE status = ? "
		args = append(args, status)
	}
	query += "ORDER BY last_seen_at DESC "
	if limit > 0 {
		query += "LIMIT ? OFFSET ? "
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var ruleID, entityType, entityID, userID, metadata sql.NullString
		var ackedBy, resolvedBy sql.NullString
		var nextEscalationAt, ackedAt, resolvedAt sql.NullTime
		var lat, lng sql.NullFloat64

		err := rows.Scan(
			&a.ID, &ruleID, &a.Source, &a.AlertType, &a.Severity, &a.Status, &a.DedupKey,
			&entityType, &entityID, &userID, &a.Title, &a.Message, &a.Occurrences,
			&a.FirstSeenAt, &a.LastSeenAt, &nextEscalationAt, &a.EscalationStep,
			&lat, &lng, &metadata, &ackedBy, &ackedAt, &resolvedBy, &resolvedAt,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if ruleID.Valid {
			a.RuleID = &ruleID.String
		}
		if entityType.Valid {
			a.EntityType = &entityType.String
		}
		if entityID.Valid {
			a.EntityID = &entityID.String
		}
		if userID.Valid {
			a.UserID = &userID.String
		}
		if metadata.Valid {
			a.Metadata = metadata.String
		}
		if ackedBy.Valid {
			a.AckedBy = &ackedBy.String
		}
		if resolvedBy.Valid {
			a.ResolvedBy = &resolvedBy.String
		}
		if nextEscalationAt.Valid {
			a.NextEscalationAt = &nextEscalationAt.Time
		}
		if ackedAt.Valid {
			a.AckedAt = &ackedAt.Time
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		if lat.Valid {
			a.Latitude = &lat.Float64
		}
		if lng.Valid {
			a.Longitude = &lng.Float64
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *sqlAlertRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM alerts
		WHERE status IN ('open', 'acknowledged', 'escalated')
		  AND (user_id = ? OR user_id IS NULL)`, userID).Scan(&count)
	return count, err
}

func (r *sqlAlertRepository) Recent(ctx context.Context, userID string, limit int) ([]domain.Alert, error) {
	if limit <= 0 {
		limit = 5
	}
	return r.ListAlerts(ctx, "", limit, 0)
}

func (r *sqlAlertRepository) Ack(ctx context.Context, alertID string, userID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE alerts
		SET status = 'acknowledged',
		    next_escalation_at = NULL,
		    acked_by = ?,
		    acked_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN ('open', 'escalated')`, userID, now, alertID)
	return err
}

func (r *sqlAlertRepository) Resolve(ctx context.Context, alertID string, userID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE alerts
		SET status = 'resolved',
		    next_escalation_at = NULL,
		    resolved_by = ?,
		    resolved_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN ('open', 'acknowledged', 'escalated')`, userID, now, alertID)
	return err
}

func (r *sqlAlertRepository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE alerts
		SET status = 'acknowledged',
		    acked_by = ?,
		    acked_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE status = 'open' AND (user_id = ? OR user_id IS NULL)`, userID, now, userID)
	return err
}

func (r *sqlAlertRepository) ListPendingEscalations(ctx context.Context, now time.Time) ([]domain.Alert, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, rule_id, source, alert_type, severity, status, dedup_key,
		       entity_type, entity_id, user_id, title, message, occurrences,
		       first_seen_at, last_seen_at, next_escalation_at, escalation_step,
		       latitude, longitude, metadata, acked_by, acked_at, resolved_by, resolved_at,
		       created_at, updated_at
		FROM alerts
		WHERE status IN ('open', 'escalated')
		  AND next_escalation_at IS NOT NULL
		  AND next_escalation_at <= ?
		ORDER BY next_escalation_at ASC`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var ruleID, entityType, entityID, userID, metadata sql.NullString
		var ackedBy, resolvedBy sql.NullString
		var nextEscalationAt, ackedAt, resolvedAt sql.NullTime
		var lat, lng sql.NullFloat64

		err := rows.Scan(
			&a.ID, &ruleID, &a.Source, &a.AlertType, &a.Severity, &a.Status, &a.DedupKey,
			&entityType, &entityID, &userID, &a.Title, &a.Message, &a.Occurrences,
			&a.FirstSeenAt, &a.LastSeenAt, &nextEscalationAt, &a.EscalationStep,
			&lat, &lng, &metadata, &ackedBy, &ackedAt, &resolvedBy, &resolvedAt,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if ruleID.Valid {
			a.RuleID = &ruleID.String
		}
		if entityType.Valid {
			a.EntityType = &entityType.String
		}
		if entityID.Valid {
			a.EntityID = &entityID.String
		}
		if userID.Valid {
			a.UserID = &userID.String
		}
		if metadata.Valid {
			a.Metadata = metadata.String
		}
		if ackedBy.Valid {
			a.AckedBy = &ackedBy.String
		}
		if resolvedBy.Valid {
			a.ResolvedBy = &resolvedBy.String
		}
		if nextEscalationAt.Valid {
			a.NextEscalationAt = &nextEscalationAt.Time
		}
		if ackedAt.Valid {
			a.AckedAt = &ackedAt.Time
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		if lat.Valid {
			a.Latitude = &lat.Float64
		}
		if lng.Valid {
			a.Longitude = &lng.Float64
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *sqlAlertRepository) UpdateEscalation(ctx context.Context, alertID string, nextStep int, nextAt *time.Time, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE alerts
		SET escalation_step = ?,
		    next_escalation_at = ?,
		    status = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, nextStep, nextAt, status, alertID)
	return err
}

func (r *sqlAlertRepository) ListUnflushedStormAlerts(ctx context.Context, windowCutoff time.Time) ([]domain.Alert, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, rule_id, source, alert_type, severity, status, dedup_key,
		       entity_type, entity_id, user_id, title, message, occurrences,
		       first_seen_at, last_seen_at, next_escalation_at, escalation_step,
		       latitude, longitude, metadata, acked_by, acked_at, resolved_by, resolved_at,
		       created_at, updated_at
		FROM alerts
		WHERE status IN ('open', 'escalated')
		  AND occurrences > 1
		  AND last_seen_at <= ?
		  AND (metadata NOT LIKE '%"flushed":true%' OR metadata IS NULL)
		ORDER BY last_seen_at ASC`, windowCutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var ruleID, entityType, entityID, userID, metadata sql.NullString
		var ackedBy, resolvedBy sql.NullString
		var nextEscalationAt, ackedAt, resolvedAt sql.NullTime
		var lat, lng sql.NullFloat64

		err := rows.Scan(
			&a.ID, &ruleID, &a.Source, &a.AlertType, &a.Severity, &a.Status, &a.DedupKey,
			&entityType, &entityID, &userID, &a.Title, &a.Message, &a.Occurrences,
			&a.FirstSeenAt, &a.LastSeenAt, &nextEscalationAt, &a.EscalationStep,
			&lat, &lng, &metadata, &ackedBy, &ackedAt, &resolvedBy, &resolvedAt,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if ruleID.Valid {
			a.RuleID = &ruleID.String
		}
		if entityType.Valid {
			a.EntityType = &entityType.String
		}
		if entityID.Valid {
			a.EntityID = &entityID.String
		}
		if userID.Valid {
			a.UserID = &userID.String
		}
		if metadata.Valid {
			a.Metadata = metadata.String
		}
		if ackedBy.Valid {
			a.AckedBy = &ackedBy.String
		}
		if resolvedBy.Valid {
			a.ResolvedBy = &resolvedBy.String
		}
		if nextEscalationAt.Valid {
			a.NextEscalationAt = &nextEscalationAt.Time
		}
		if ackedAt.Valid {
			a.AckedAt = &ackedAt.Time
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		if lat.Valid {
			a.Latitude = &lat.Float64
		}
		if lng.Valid {
			a.Longitude = &lng.Float64
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *sqlAlertRepository) UpdateMetadata(ctx context.Context, alertID string, metadata string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE alerts
		SET metadata = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, metadata, alertID)
	return err
}
