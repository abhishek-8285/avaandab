// Package worker runs the geofence dwell engine as a background loop
// (Spec 02 §4).
package worker

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"transport-app/internal/events"
	"transport-app/internal/geofence/application"
	"transport-app/internal/geofence/domain"
	sqlrepo "transport-app/internal/geofence/infrastructure/persistence/sql"
	"transport-app/internal/shared"
	"transport-app/internal/shared/id"
	"transport-app/internal/shared/outbox"
	"transport-app/internal/shared/ports"
	tripapp "transport-app/internal/trip/application"
	tripaggregate "transport-app/internal/trip/domain/aggregate"
)

// maxFixesPerSweep caps the number of fixes processed in one tick.
const maxFixesPerSweep = 500

// DwellWorker polls telemetry_snapshots, runs fixes through the DwellEngine
// and persists state transitions + events inside UnitOfWork transactions.
type DwellWorker struct {
	uow      ports.UnitOfWork
	config   *application.ConfigReader
	bus      events.EventBus
	outbox   *outbox.OutboxWriter
	idGen    ports.IDGenerator
	log      *slog.Logger
	tenantID string
	db       *sql.DB

	fixes     domain.FixRepository
	geofences domain.GeofenceRepository
	states    domain.EngineStateRepository
	logs      domain.EventLogRepository

	reachPickupUC  *tripapp.ReachPickupUseCase
	startTransitUC *tripapp.StartTransitUseCase
}

// NewDwellWorker constructs the worker over the app database. `bus` receives
// post-commit alert publication; may be nil.
func NewDwellWorker(db *sql.DB, uow ports.UnitOfWork, cfg *application.ConfigReader,
	bus events.EventBus, log *slog.Logger) *DwellWorker {
	return &DwellWorker{
		uow:       uow,
		config:    cfg,
		bus:       bus,
		outbox:    outbox.NewOutboxWriter(db),
		idGen:     id.NewUUIDGenerator(),
		log:       log,
		tenantID:  string(shared.DefaultTenant),
		db:        db,
		fixes:     sqlrepo.NewSnapshotRepository(db),
		geofences: sqlrepo.NewGeofenceRepository(db),
		states:    sqlrepo.NewEngineStateRepository(db),
		logs:      sqlrepo.NewEventLogRepository(db),
	}
}

// WithTripTransitions wires the auto-transition use cases (Spec 02 §5).
// They are invoked only when the company_config gates are enabled.
func (w *DwellWorker) WithTripTransitions(reachPickup *tripapp.ReachPickupUseCase, startTransit *tripapp.StartTransitUseCase) *DwellWorker {
	w.reachPickupUC = reachPickup
	w.startTransitUC = startTransit
	return w
}

// Run blocks until ctx is cancelled, sweeping on poll_interval_seconds.
func (w *DwellWorker) Run(ctx context.Context) {
	interval, err := w.config.GetDurationSeconds(ctx, w.tenantID,
		application.ConfigPollIntervalSeconds, application.DefaultPollInterval)
	if err != nil {
		w.log.Warn("dwell worker: poll interval fallback", "error", err)
		interval = application.DefaultPollInterval
	}

	w.log.Info("dwell worker started", "interval", interval.String())
	if _, err := w.Tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
		w.log.Error("dwell worker initial sweep failed", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.log.Info("dwell worker stopped")
			return
		case <-ticker.C:
			if _, err := w.Tick(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					w.log.Error("dwell worker sweep failed", "error", err)
				}
			}
		}
	}
}

// Tick processes one sweep of new fixes. Returns the number of fixes handled.
func (w *DwellWorker) Tick(ctx context.Context) (int, error) {
	fixes, err := w.fixes.LoadNewFixes(ctx, maxFixesPerSweep)
	if err != nil {
		return 0, err
	}

	engine := application.NewDwellEngine(application.EngineConfig{
		Debounce:         mustDuration(ctx, w, application.ConfigDwellDebounceSeconds, application.DefaultDwellDebounce),
		BufferMetres:     mustFloat(ctx, w, application.ConfigBufferMetres, application.DefaultBufferMetres),
		HysteresisMetres: mustFloat(ctx, w, application.ConfigHysteresisMetres, application.DefaultHysteresisMetres),
	})

	handled := 0
	for _, fix := range fixes {
		select {
		case <-ctx.Done():
			return handled, ctx.Err()
		default:
		}

		// Trip context gates zone evaluation for pickup/drop zones and
		// enables auto-transitions (Spec 02 §5).
		tripStatus, routeSource, routeDest := "", "", ""
		if fix.TripID != nil {
			tripStatus, routeSource, routeDest = w.tripContext(ctx, *fix.TripID)
		}

		zones, err := w.zonesFor(ctx, fix.VehicleID, fix.TripID, tripStatus, routeSource, routeDest)
		if err != nil {
			w.log.Error("dwell worker: zones load failed", "vehicle", fix.VehicleID, "error", err)
			continue
		}

		current, err := w.states.GetByVehicle(ctx, w.tenantID, fix.VehicleID)
		if err != nil {
			if !errors.Is(err, sqlrepo.ErrNoEngineState) {
				w.log.Error("dwell worker: state load failed", "vehicle", fix.VehicleID, "error", err)
				continue
			}
			current = &domain.EngineState{
				VehicleID: fix.VehicleID,
				TenantID:  w.tenantID,
				State:     domain.StateOutside,
			}
		}

		next, zoneEvents := engine.Evaluate(*current, fix, zones)
		if err := w.persist(ctx, *current, next, fix, zoneEvents); err != nil {
			w.log.Error("dwell worker: persist failed", "vehicle", fix.VehicleID, "error", err)
			continue
		}
		w.applyTransitions(ctx, fix, zoneEvents, tripStatus)
		handled++
	}
	return handled, nil
}

// tripContext reads the trip status and route endpoints via plain SQL so the
// aggregate guard prerequisites are checked before invoking use cases
// (Spec 02 §5 note). Returns empty strings when the trip is missing.
func (w *DwellWorker) tripContext(ctx context.Context, tripID string) (status, routeSource, routeDest string) {
	row := w.db.QueryRowContext(ctx,
		`SELECT t.status, COALESCE(r.source, ''), COALESCE(r.destination, '')
		 FROM trips t
		 LEFT JOIN routes r ON r.id = t.route_id
		 WHERE t.id = ? AND t.tenant_id = ?`, tripID, w.tenantID)
	if err := row.Scan(&status, &routeSource, &routeDest); err != nil {
		return "", "", ""
	}
	return status, routeSource, routeDest
}

// zonesFor loads active zones for a vehicle (explicit bindings first, then
// the tenant-wide set). Pickup/drop zones are only evaluated when the trip is
// in an eligible status and (if route_name is set) the trip's route endpoint
// matches — LOWER(TRIM()) comparison (Spec 02 §5 route fallback).
func (w *DwellWorker) zonesFor(ctx context.Context, vehicleID string, tripID *string,
	tripStatus, routeSource, routeDest string) ([]domain.Geofence, error) {
	bound, err := w.geofences.ListBoundForVehicle(ctx, w.tenantID, vehicleID)
	if err != nil {
		return nil, err
	}
	active, err := w.geofences.ListActiveByTenant(ctx, w.tenantID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(bound)+len(active))
	var zones []domain.Geofence
	for _, z := range bound {
		if !w.zoneApplies(z, tripID, tripStatus, routeSource, routeDest) {
			continue
		}
		seen[z.ID] = true
		zones = append(zones, z)
	}
	for _, z := range active {
		if seen[z.ID] {
			continue
		}
		if !w.zoneApplies(z, tripID, tripStatus, routeSource, routeDest) {
			continue
		}
		zones = append(zones, z)
	}
	return zones, nil
}

// zoneApplies decides whether a zone generates events for a fix. Pickup/drop
// zones need an active trip; depot/restricted/no_entry always apply. Route
// alignment (Spec 02 §4) uses LOWER/TRIM against routes.source/destination
// when route_name is set. The trip-status PHASE gate (pickup → started/
// reached_pickup, drop → reached_pickup/in_transit) is enforced by
// applyTransitions — coverage-based detection and dwell billing (Spec 02 §6)
// must keep working for any active trip.
func (w *DwellWorker) zoneApplies(z domain.Geofence, tripID *string, tripStatus, routeSource, routeDest string) bool {
	if z.Kind == domain.KindPickup || z.Kind == domain.KindDrop {
		if tripID == nil {
			return false
		}
	}

	if z.RouteName == "" {
		return true
	}
	route := strings.ToLower(strings.TrimSpace(z.RouteName))
	return route == strings.ToLower(strings.TrimSpace(routeSource)) ||
		route == strings.ToLower(strings.TrimSpace(routeDest))
}

// applyTransitions runs the auto ReachPickup / StartTransit use cases AFTER
// the persist transaction commits. Each use case opens its own UnitOfWork,
// so it must never be invoked inside persist (nested transactions would
// deadlock — 1E lesson). Errors are logged, never fatal to the sweep.
func (w *DwellWorker) applyTransitions(ctx context.Context, fix domain.Fix, zoneEvents []application.ZoneEvent, tripStatus string) {
	if fix.TripID == nil {
		return
	}
	autoReach, err := w.config.GetBool(ctx, w.tenantID, application.ConfigAutoReachPickup, false)
	if err != nil {
		autoReach = false
	}
	autoTransit, err := w.config.GetBool(ctx, w.tenantID, application.ConfigAutoStartTransit, false)
	if err != nil {
		autoTransit = false
	}

	for _, ev := range zoneEvents {
		switch {
		case ev.EventType == domain.EventEntering && ev.Zone.Kind == domain.KindPickup &&
			tripStatus == string(tripaggregate.TripStarted) && autoReach && w.reachPickupUC != nil:
			if err := w.reachPickupUC.Execute(ctx, tripapp.ReachPickupCommand{
				TripID:   tripaggregate.TripID(*fix.TripID),
				TenantID: shared.TenantID(w.tenantID),
			}); err != nil {
				w.log.Warn("dwell worker: ReachPickup rejected by aggregate", "trip", *fix.TripID, "error", err)
			}

		case ev.EventType == domain.EventEntering && ev.Zone.Kind == domain.KindDrop &&
			tripStatus == string(tripaggregate.TripReachedPickup) && autoTransit && w.startTransitUC != nil:
			if err := w.startTransitUC.Execute(ctx, tripapp.StartTransitCommand{
				TripID:   tripaggregate.TripID(*fix.TripID),
				TenantID: shared.TenantID(w.tenantID),
			}); err != nil {
				w.log.Warn("dwell worker: StartTransit rejected by aggregate", "trip", *fix.TripID, "error", err)
			}

		case ev.EventType == domain.EventLeaving && ev.Zone.Kind == domain.KindPickup &&
			tripStatus == string(tripaggregate.TripReachedPickup) && autoTransit && w.startTransitUC != nil:
			if err := w.startTransitUC.Execute(ctx, tripapp.StartTransitCommand{
				TripID:   tripaggregate.TripID(*fix.TripID),
				TenantID: shared.TenantID(w.tenantID),
			}); err != nil {
				w.log.Warn("dwell worker: StartTransit rejected by aggregate", "trip", *fix.TripID, "error", err)
			}
		}
	}
}

// persist writes the state transition, zone events, detentions and outbox
// alerts atomically via the UnitOfWork (Spec 02 §4 — outbox in same tx).
func (w *DwellWorker) persist(ctx context.Context, current, next domain.EngineState,
	fix domain.Fix, zoneEvents []application.ZoneEvent) error {

	return w.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		if err := w.states.Upsert(txCtx, next); err != nil {
			return err
		}

		for _, ev := range zoneEvents {
			eventID := w.idGen.GenerateUUID()
			lat := fix.Latitude
			lng := fix.Longitude
			if err := w.logs.InsertEvent(txCtx, domain.GeofenceEvent{
				ID:         eventID,
				TenantID:   w.tenantID,
				VehicleID:  &fix.VehicleID,
				TripID:     fix.TripID,
				GeofenceID: &ev.Zone.ID,
				ZoneKind:   &ev.Zone.Kind,
				EventType:  ev.EventType,
				Latitude:   &lat,
				Longitude:  &lng,
				Details:    strPtr(ev.Details),
				CreatedAt:  ev.At,
			}); err != nil {
				return err
			}

			switch ev.EventType {
			case domain.EventEntering:
				if err := w.openDetention(txCtx, fix, ev); err != nil {
					return err
				}
			case domain.EventLeaving:
				if err := w.closeDetention(txCtx, fix, ev); err != nil {
					return err
				}
			case domain.EventBreach:
				sev := application.SeverityFor(ev.Zone.Kind)
				alert := application.GeofenceAlertEvent{
					TenantID:   w.tenantID,
					VehicleID:  fix.VehicleID,
					TripID:     strOrEmpty(fix.TripID),
					GeofenceID: ev.Zone.ID,
					ZoneKind:   ev.Zone.Kind,
					AlertType:  "zone_breach",
					Severity:   sev,
					Latitude:   fix.Latitude,
					Longitude:  fix.Longitude,
					Details:    ev.Details,
					Timestamp:  ev.At,
				}
				if err := application.EmitGeofenceAlert(txCtx, w.outbox, alert); err != nil {
					return err
				}
				if w.bus != nil {
					w.bus.Publish(ctx, events.Event{
						Type:    events.GeofenceZoneBreach,
						Payload: alert,
					})
				}
			}
		}
		return nil
	})
}

// openDetention starts a pickup/drop dwell window on confirmed entry.
func (w *DwellWorker) openDetention(txCtx context.Context, fix domain.Fix, ev application.ZoneEvent) error {
	if ev.Zone.Kind != domain.KindPickup && ev.Zone.Kind != domain.KindDrop {
		return nil
	}
	if fix.TripID == nil {
		return nil
	}
	return w.logs.OpenDetention(txCtx, domain.Detention{
		ID:         w.idGen.GenerateUUID(),
		TenantID:   w.tenantID,
		TripID:     *fix.TripID,
		VehicleID:  &fix.VehicleID,
		GeofenceID: &ev.Zone.ID,
		ZoneKind:   ev.Zone.Kind,
		EnteredAt:  ev.At,
		Status:     domain.DetentionOpen,
	})
}

// closeDetention closes the matching open window with its dwell duration and
// computes the billable seconds + amount (Spec 02 §5/§6: amount is rounded to
// 2 decimals = billable_hours × rate_per_hour).
func (w *DwellWorker) closeDetention(txCtx context.Context, fix domain.Fix, ev application.ZoneEvent) error {
	if ev.Zone.Kind != domain.KindPickup && ev.Zone.Kind != domain.KindDrop {
		return nil
	}
	if fix.TripID == nil {
		return nil
	}
	d, err := w.logs.FindOpenDetention(txCtx, w.tenantID, *fix.TripID, ev.Zone.Kind)
	if err != nil || d == nil {
		return err
	}
	dwell := int64(ev.At.Sub(d.EnteredAt).Seconds())
	if dwell < 0 {
		dwell = 0
	}
	free := int64(mustDuration(txCtx, w, application.ConfigDetentionFreeSeconds, application.DefaultDetentionFree).Seconds())
	rate := mustFloat(txCtx, w, application.ConfigDetentionRatePerHour, application.DefaultDetentionRate)
	amount := math.Round((float64(dwell-free)/3600.0*rate)*100) / 100
	if amount < 0 {
		amount = 0
	}
	return w.logs.CloseDetention(txCtx, d.ID, ev.At, dwell, free, rate, amount)
}

func mustDuration(ctx context.Context, w *DwellWorker, key string, def time.Duration) time.Duration {
	d, err := w.config.GetDurationSeconds(ctx, w.tenantID, key, def)
	if err != nil {
		return def
	}
	return d
}

func mustFloat(ctx context.Context, w *DwellWorker, key string, def float64) float64 {
	f, err := w.config.GetFloat(ctx, w.tenantID, key, def)
	if err != nil {
		return def
	}
	return f
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
