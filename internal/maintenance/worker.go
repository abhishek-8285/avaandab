package maintenance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"transport-app/internal/events"
	"transport-app/internal/maintenance/domain"
	maintsql "transport-app/internal/maintenance/infrastructure/sql"
)

// Worker evaluates maintenance schedules, ingests DTC alerts, and manages the vehicle maintenance due flag (Spec 04 §6, §12).
type Worker struct {
	db              *sql.DB
	repo            *maintsql.MaintenanceRepository
	bus             events.EventBus
	logger          *slog.Logger
	checkInterval   time.Duration
	fallbackCritDTC string
}

// NewWorker constructs a new PM Worker instance.
func NewWorker(db *sql.DB, bus events.EventBus, logger *slog.Logger, intervalMin int, fallbackCritDTC string) *Worker {
	if intervalMin <= 0 {
		intervalMin = 15
	}
	if fallbackCritDTC == "" {
		fallbackCritDTC = "P0A0F,P1602"
	}
	if logger == nil {
		logger = slog.Default()
	}
	w := &Worker{
		db:              db,
		repo:            maintsql.NewMaintenanceRepository(db),
		bus:             bus,
		logger:          logger,
		checkInterval:   time.Duration(intervalMin) * time.Minute,
		fallbackCritDTC: fallbackCritDTC,
	}

	if bus != nil {
		w.subscribeDTC(bus)
	}

	return w
}

// Run starts the background evaluation loop.
func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("preventive maintenance worker started", "interval", w.checkInterval)

	// Run initial evaluation on startup
	w.EvaluateSchedules(ctx)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("preventive maintenance worker stopped")
			return
		case <-ticker.C:
			w.EvaluateSchedules(ctx)
		}
	}
}

// EvaluateSchedules checks odometer and date thresholds for all active schedules (Spec 04 §6).
func (w *Worker) EvaluateSchedules(ctx context.Context) {
	schedules, err := w.repo.ListActiveSchedules(ctx, "")
	if err != nil {
		w.logger.Error("failed to list active maintenance schedules", "error", err)
		return
	}

	now := time.Now().UTC()

	for _, s := range schedules {
		isDue := false
		var reason string

		// 1. Odometer-based evaluation
		latestOdo, err := w.repo.GetLatestOdometer(ctx, s.VehicleID)
		if err == nil && latestOdo > 0 {
			if s.DueKM != nil && latestOdo >= *s.DueKM {
				isDue = true
				reason = fmt.Sprintf("odometer %.0f km >= due threshold %.0f km", latestOdo, *s.DueKM)
			} else if s.IntervalKM != nil && *s.IntervalKM > 0 {
				lastKM := 0.0
				if s.LastDoneKM != nil {
					lastKM = *s.LastDoneKM
				}
				if latestOdo >= lastKM+*s.IntervalKM {
					isDue = true
					reason = fmt.Sprintf("odometer %.0f km exceeded interval %.0f km from last service (%.0f km)", latestOdo, *s.IntervalKM, lastKM)
				}
			}
		}

		// 2. Date-based evaluation
		if !isDue {
			if s.DueAt != nil && (now.After(*s.DueAt) || now.Equal(*s.DueAt)) {
				isDue = true
				reason = fmt.Sprintf("service date %s reached", s.DueAt.Format("2006-01-02"))
			} else if s.IntervalDays != nil && *s.IntervalDays > 0 {
				lastAt := s.CreatedAt
				if s.LastDoneAt != nil {
					lastAt = *s.LastDoneAt
				}
				dueDate := lastAt.Add(time.Duration(*s.IntervalDays) * 24 * time.Hour)
				if now.After(dueDate) || now.Equal(dueDate) {
					isDue = true
					reason = fmt.Sprintf("interval %d days exceeded since %s", *s.IntervalDays, lastAt.Format("2006-01-02"))
				}
			}
		}

		if isDue {
			w.markVehicleDue(ctx, s.VehicleID, s.ServiceType, reason, now)
		}
	}
}

func (w *Worker) markVehicleDue(ctx context.Context, vehicleID, serviceType, reason string, dueDate time.Time) {
	err := w.repo.SetMaintenanceDue(ctx, vehicleID, dueDate)
	if err != nil {
		w.logger.Error("failed to set maintenance_due on vehicle", "vehicle_id", vehicleID, "error", err)
		return
	}

	// Insert in-app notification
	notifID := uuid.NewString()
	title := fmt.Sprintf("Maintenance Due: %s", serviceType)
	msg := fmt.Sprintf("Vehicle %s requires %s maintenance: %s", vehicleID, serviceType, reason)
	_, _ = w.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, title, message, channel, status, created_at)
		VALUES (?, 'system', ?, ?, 'in_app', 'unread', CURRENT_TIMESTAMP)`,
		notifID, title, msg,
	)

	// Publish maintenance.due event on bus
	if w.bus != nil {
		evt := events.Event{
			Type: "maintenance.due",
			Payload: map[string]interface{}{
				"vehicle_id":   vehicleID,
				"service_type": serviceType,
				"reason":       reason,
				"due_date":     dueDate.Format("2006-01-02"),
			},
		}
		w.bus.Publish(ctx, evt)
	}
}

// EvaluateResolution checks if all due conditions for a vehicle have been cleared.
func (w *Worker) EvaluateResolution(ctx context.Context, vehicleID string) {
	// 1. Check unresolved critical DTCs
	critDTCs, err := w.repo.ListUnresolvedCriticalDtc(ctx, vehicleID)
	if err == nil && len(critDTCs) > 0 {
		return // Critical DTCs still unresolved
	}

	// 2. Check active schedules
	schedules, err := w.repo.ListActiveSchedules(ctx, vehicleID)
	if err == nil {
		now := time.Now().UTC()
		latestOdo, _ := w.repo.GetLatestOdometer(ctx, vehicleID)
		for _, s := range schedules {
			if s.DueKM != nil && latestOdo >= *s.DueKM {
				return
			}
			if s.IntervalKM != nil && *s.IntervalKM > 0 {
				lastKM := 0.0
				if s.LastDoneKM != nil {
					lastKM = *s.LastDoneKM
				}
				if latestOdo >= lastKM+*s.IntervalKM {
					return
				}
			}
			if s.DueAt != nil && (now.After(*s.DueAt) || now.Equal(*s.DueAt)) {
				return
			}
			if s.IntervalDays != nil && *s.IntervalDays > 0 {
				lastAt := s.CreatedAt
				if s.LastDoneAt != nil {
					lastAt = *s.LastDoneAt
				}
				if now.After(lastAt.Add(time.Duration(*s.IntervalDays) * 24 * time.Hour)) {
					return
				}
			}
		}
	}

	// Clear maintenance due
	if err := w.repo.ClearMaintenanceDue(ctx, vehicleID); err == nil {
		if w.bus != nil {
			evt := events.Event{
				Type: "maintenance.cleared",
				Payload: map[string]interface{}{
					"vehicle_id": vehicleID,
					"cleared_at": time.Now().UTC().Format(time.RFC3339),
				},
			}
			w.bus.Publish(ctx, evt)
		}
	}
}

// subscribeDTC listens for DTC alerts from the event bus.
func (w *Worker) subscribeDTC(bus events.EventBus) {
	bus.Subscribe("alert.dtc", func(ctx context.Context, evt events.Event) error {
		return w.HandleDtcEvent(ctx, evt.Payload)
	})

	bus.Subscribe("AlertEvent", func(ctx context.Context, evt events.Event) error {
		return w.HandleDtcEvent(ctx, evt.Payload)
	})
}

// HandleDtcEvent processes a DTC payload and inserts it with minute-granularity dedup (Spec 04 §6).
func (w *Worker) HandleDtcEvent(ctx context.Context, payload interface{}) error {
	var m map[string]interface{}
	switch p := payload.(type) {
	case map[string]interface{}:
		m = p
	case []byte:
		_ = json.Unmarshal(p, &m)
	case string:
		_ = json.Unmarshal([]byte(p), &m)
	default:
		data, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		_ = json.Unmarshal(data, &m)
	}

	if m == nil {
		return nil
	}

	vehicleID, _ := m["vehicle_id"].(string)
	if vehicleID == "" {
		return nil
	}

	tripID, _ := m["trip_id"].(string)
	var tripIDPtr *string
	if tripID != "" {
		tripIDPtr = &tripID
	}

	dtcCode, _ := m["dtc_code"].(string)
	if dtcCode == "" {
		if codes, ok := m["dtc_codes"].([]interface{}); ok && len(codes) > 0 {
			dtcCode = fmt.Sprint(codes[0])
		}
	}
	if dtcCode == "" {
		return nil
	}
	dtcCode = strings.ToUpper(strings.TrimSpace(dtcCode))

	severity, _ := m["severity"].(string)
	if severity == "" {
		severity = "info"
	}
	severity = strings.ToLower(severity)
	if severity != "info" && severity != "warning" && severity != "critical" {
		severity = "info"
	}

	desc, _ := m["description"].(string)
	var descPtr *string
	if desc != "" {
		descPtr = &desc
	}

	rawBytes, _ := json.Marshal(m)
	rawStr := string(rawBytes)

	occurredAt := time.Now().UTC()
	if occStr, ok := m["occurred_at"].(string); ok && occStr != "" {
		if t, err := time.Parse(time.RFC3339, occStr); err == nil {
			occurredAt = t.UTC()
		}
	}

	// Minute dedup insert
	dtc := domain.DtcEvent{
		ID:          uuid.NewString(),
		VehicleID:   vehicleID,
		TripID:      tripIDPtr,
		DtcCode:     dtcCode,
		Severity:    severity,
		Description: descPtr,
		RawPayload:  &rawStr,
		OccurredAt:  occurredAt,
	}

	inserted, err := w.repo.InsertDtcEvent(ctx, dtc)
	if err != nil {
		w.logger.Error("failed to insert DTC event", "vehicle_id", vehicleID, "dtc_code", dtcCode, "error", err)
		return err
	}

	// Check if this DTC is critical
	critCodes := w.repo.GetCriticalDtcCodes(ctx, w.fallbackCritDTC)
	isCritical := severity == "critical"
	if !isCritical {
		for _, c := range critCodes {
			if dtcCode == c {
				isCritical = true
				break
			}
		}
	}

	if inserted && isCritical {
		w.markVehicleDue(ctx, vehicleID, "engine", fmt.Sprintf("Critical DTC %s detected", dtcCode), occurredAt)
	}

	return nil
}
