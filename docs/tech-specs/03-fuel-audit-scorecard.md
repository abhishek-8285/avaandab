# FUEL AUDIT + DRIVER SCORECARD — Implementation Spec v2

**Owner:** fuel audit + driver scorecard (this spec)
**Project:** Avandab fleet system — `/home/abhishek/Desktop/temux/basic`
**Stack (verified):** Go 1.26 (`go.mod:3`), chi v5.3.1 (`go.mod:7`), SQLite modernc v1.56.0 (`go.mod:17`), goose v3.27.3 (`go.mod:12`), casbin v2.135.0 (`go.mod:22`), Datastar v1.0.2 + HTMX templates (`internal/templates/layout.html:15`), outbox-relay event bus (`cmd/server/main.go:819-826`), typed IDs, UoW pattern (`internal/shared/uow/uow.go:65`).
**Latest committed migration:** `00038` (verified — `00039`–`00042` are claimed by other specs; this spec owns **00043**).

---

## 0. Verification Log (QA pass — 2026-08-19)

Migration renumber: fuel `00042` → **`00043`**; geofence dependency `00041` → **`00042`**;
head is `00039_experiments.sql`. **Collision fixed:** `company_config` is now created once
by spec 02 @00042; this spec seeds only (was double-CREATE). Outbox cite `608-616` → `819-826`.

### Verification Log table

| Claim | Verdict | Correction / Evidence (file:line) |
|---|---|---|
| `DriverExpense` has no Status/Category | VERIFIED | `internal/domain/expense/entity.go:27-37` |
| `KharchaService` methods exist (Approve/Create/List/Stats) | VERIFIED | `internal/service/kharcha_service.go:45,166,251,301` |
| `driver_settlements` has no INSERT today | VERIFIED | grep `INSERT INTO driver_settlements` → none |
| Two separate buses (service vs outbox-relay) | VERIFIED | `internal/service/service.go:75` `events.NewInMemoryBus()`; `main.go:819` outbox relay |
| `telemetry_alerts` dead schema (zero writers) | VERIFIED | no repo refs in `internal/repository/` |
| `company_config` does not exist | VERIFIED (now) | no `company_config` in `db/migrations/`; will be created by spec 02 @00042 |
| `vehicles.odometer` not in sqlc queries | VERIFIED | `db/query/vehicles.sql` |
| Fuel spec `00042` (geofence `00041`) | WRONG | fuel **00043**; geofence **00042** |
| `company_config` CREATE in this spec | WRONG | removed — owned by spec 02 @00042 (collision fix) |
| `ProviderIngestor` / geofence code exist | CANT-VERIFY | design-only (no `providers/`, no `geofence*`) |
| `driver_settlement_service.go` persistence gap | VERIFIED | only returns structs; no INSERT confirmed |

### Severity & Effort (major changes)

| Change | Severity | Effort |
|---|---|---|
| Migration renumber 00042→00043 | Low | S |
| Remove `company_config` CREATE (collision) | Med | S |
| Fuel anomaly engine (stateful) | High | L |
| Claim audit (A/B/C cross-check) | High | M |
| Driver scorecard + settlement bonus | High | L |
| `AlertEvent` via outbox (not legacy bus) | High | M |

### Architectural Decisions (Decision / Tradeoff / Cost)

- **Fuel-theft detection method: median-smoothing + threshold windows, not raw drop.**
  Decision: rolling median window (default 7) + noise floor + sustained-drop detection.
  Tradeoff: robust to sensor noise/GPS blips; lag of one window before flag. Cost: per-vehicle
  in-memory state, replay from `telemetry_snapshots` on restart (single-instance assumption).
- **Audit annotate-first vs enforce.** Decision: `needs_review` does NOT block approve by
  default; `fuel.audit_enforce=true` gates. Tradeoff: flexibility vs leak risk. Cost: 1 gate branch.
- **Two-bus emission.** Decision: engine emits `AlertEvent` via `outbox.NewOutboxWriter` (not
  `baseService.events`). Tradeoff: ~5s relay latency for alerts; durability. Cost: extra write.
- **Settlement bonus persistence.** Decision: add INSERT to `driver_settlements` (fixes pre-existing
  gap) + `performance_bonus` column. Tradeoff: closes known bug; must guard `store != nil`.

## 1. Architecture

### 1.1 Verified starting state

| Concern | Verified fact | Location |
|---|---|---|
| Expense entity | `DriverExpense{ID, TripID, DriverID, ExpenseType, Amount, Description, ReceiptURL, Approved, CreatedAt}` — **no Status, no Category in domain entity** | `internal/domain/expense/entity.go:27-37` |
| Service view | `KharchaExpense` carries `Status` (pending/approved/rejected/settled), `Category`, `RejectedReason`, `ApprovedBy`, `ApprovedAt` | `internal/service/kharcha_service.go:12-27` |
| Kharcha methods | `ListPendingExpenses`, `ListLedger(tripID)`, `GetExpenseByID`, `ApproveExpense`, `RejectExpense`, `CreateExpense`, `GetKharchaStats` — all raw SQL via `repository.DBGetter` | `internal/service/kharcha_service.go` |
| Kharcha routes | `GET /kharcha/`, `/kharcha/pending`, `/kharcha/ledger`, `POST /kharcha/{id}/approve`, `POST /kharcha/{id}/reject`; RBAC `ResourcePermission(AuthSrv, "trips", "read"/"update")` | `internal/handlers/kharcha.go:20-26` |
| e-POD route | `POST /trips/{id}/deliver-pod` mounted in main (driver mobile) | `cmd/server/main.go:587` |
| driver_expenses DDL | base table 00031, workflow columns (`status`, `category`, `approved_by`, `rejected_reason`, `approved_at`) 00032 | `db/migrations/00031_avandab_critical_fixes.sql:34-46`, `00032_kharcha_and_epod.sql:5-10` |
| Settlements written only by kharcha approval | `ApproveExpense` UPDATEs `driver_settlements` (`advances_kharcha += amt`, `net_payout = MAX(0, net_payout - amt)`) | `internal/service/kharcha_service.go:204-211` |
| `telemetry_alerts` | **DEAD SCHEMA** — defined in 00030, documented in `docs/ARCHITECTURE.md:820`, zero Go writers | `db/migrations/00030_avandab_modules_and_rules.sql:29-44` |
| `vehicles.odometer` | Column exists (00030:13) but **no sqlc query selects it**; domain `Vehicle` has `Odometer` field, service never persists it | `db/query/vehicles.sql`, `internal/domain/vehicle/entity.go:25`, `internal/service/vehicle_service.go` |
| Mobile app | GPS-only telemetry (`/api/v1/telemetry/sync`), MQTT topic `avandab/telemetry/drivers/{id}/gps`; **no fuel_level/odometer upload, no refuel/expense reporting** | `mobile/src/services/telemetry.ts`, `syncEngine.ts:37`, `mqtt.ts:55` |
| Telemetry snapshots | `telemetry_snapshots(id, trip_id, vehicle_id, timestamp, lat, lng, speed, fuel_level, odometer)`; written via `POST /api/v1/telemetry/snapshots` (`INSERT OR REPLACE`) | `db/migrations/00031:4-17`, `internal/telemetry/sync.go:52-80` |
| Naive fuel rule | `ProcessTelemetryStream`: fuel drop > 10L while ignition OFF → `FuelTheftAlert` (in-memory bus only) — **superseded by the engine** | `internal/service/telemetry_service.go:49-116` |
| kmpl source | `vehicles.current_mileage` (kmpl), used by PnL: `fuel_litres = odometer_delta / current_mileage`; routes table has **no kmpl column** | `internal/pnl/service.go:47-67`, `db/query/routes.sql` |
| `company_config` | **Does not exist yet.** `company_settings` singleton (id=1) exists. Will be CREATED by spec 02 (geofence) @00042; this spec (00043) only seeds rows. | `db/migrations/00001_initial.sql:34-45` |
| Settlement compute | `CreateSettlementForTrip` / `ProcessFinancialSettlement` — **neither persists rows**; **no Go code INSERTs into `driver_settlements` today** | `internal/service/driver_settlement_service.go:46-148` |
| Outbox-relay bus | `eventBus := events.NewInMemoryBus()` → `outbox.NewRelay(database, eventBus, logger)` → `go outboxRelay.Run(ctx)`; relay polls every 5s, dispatches `outbox_events` rows to `eventBus` | `cmd/server/main.go:819-826`, `internal/shared/outbox/relay.go:15,70-126` |
| **Two separate buses** | Services hold their own in-memory bus (`baseService.events`, `service.go:75`); `TripDelivered` etc. flow on it. Outbox relay bus is **separate** — cross-process events must be written to `outbox_events` | `internal/service/service.go:75,111-164`, `internal/service/trip_service.go:417-425` |
| Outbox write pattern | `outbox.NewOutboxWriter(dbConn)` + `SaveEvents(ctx, aggregateID, aggregateType, events)`; respects `repository.TxFromContext` | `internal/booking/infrastructure/persistence/sql/booking_repository.go:19-27`, `internal/shared/outbox/outbox.go:34-65` |
| RBAC | Casbin model `(sub, obj, act)`, policies loaded from `permissions` (`resource:action`) + `role_permissions` + `user_roles`; roles: 1=admin, 2=dispatcher, 3=accountant, 4=viewer | `internal/auth/casbin.go:12-87`, `db/migrations/00012_rbac.sql`, `00001_initial.sql` |
| Trip lifecycle | draft → scheduled → assigned → started → in_transit → delivered → completed / cancelled; `ActiveTripStatuses` includes in_transit + delivered | `internal/domain/trip/entity.go:29-50` |
| PnL columns on trips | `estimated_margin`, `fuel_consumed_liters`, `toll_costs`, `last_pnl_update` (00033) | `db/migrations/00033_live_trip_pnl.sql` |
| Templates | `parseTemplates` parses `internal/templates/*` (embedded set), `renderPage`/`renderFragment`; HTMX attrs + Datastar script in layout | `internal/handlers/app.go:95-124,349,475`, `internal/templates/layout.html:15` |

### 1.2 New components

```
┌────────────────────────────────────────────────────────────────────┐
│ telemetry_snapshots (existing persistence of ingestion PositionEvent)│
│   fuel_level, odometer, speed, trip_id, vehicle_id                  │
└───────────────┬────────────────────────────────────────────────────┘
                │ poll (background loop, like outbox relay)
                ▼
┌────────────────────────────────────────────────────────────────────┐
│ FuelEngine (internal/fuel/engine.go) — per-vehicle stateful        │
│   median smoothing → refill / drain-theft / abnormal-drain /       │
│   siphon / odometer-rollback detection                             │
│   ├─ writes fuel_events rows (durable)                            │
│   ├─ writes fuel_claim_audits rows (claim ↔ litres reconciliation) │
│   ├─ writes driver_behaviour_events rows (scorecard inputs)        │
│   └─ emits AlertEvent via outbox writer → outbox_events → relay    │
│        → main.go eventBus → alerting spec consumers                │
└───────────────┬────────────────────────────────────────────────────┘
                │
                ▼
┌────────────────────────────────────────────────────────────────────┐
│ ScorecardService (internal/service/scorecard_service.go)           │
│   30-day rolling weighted score → driver_scores history +          │
│   drivers.score/tier → leaderboard UI, preferred-load ordering,    │
│   settlement performance_bonus (driver_settlement_service.go)      │
└────────────────────────────────────────────────────────────────────┘
```

**Design rules**

1. **Raw SQL everywhere for new tables** — no sqlc regen (matches `KharchaService` pattern via `repository.DBGetter`, `internal/repository/repository.go:22-24`).
2. **Engine runs as a background goroutine** started in `cmd/server/main.go` next to the outbox relay (line ~616), tick interval from config (default 30s). Single-instance assumption (matches existing deployment model — `deploy.sh`/`deploy_avandab.sh` run one binary).
3. **Event emission**: engine writes `outbox_events` rows via `outbox.NewOutboxWriter(database).SaveEvents(ctx, vehicleID, "vehicle", []any{alertEvent})` — event type string resolves to `AlertEvent` via `getEventTypeName` (`internal/shared/outbox/outbox.go:67-73`). Relay publishes to the main.go bus where the alerting spec subscribes. Founders digest/Telegram handlers already subscribe to that bus (`cmd/server/main.go:820-821`).
4. **Alert payload type**: relay JSON-decodes payloads to `any` (`relay.go:104-112`); consumers must type-assert `map[string]any`. Reuse `founder/alerts.AlertEvent` shape as payload content (`internal/founder/alerts/event.go:28-37`) — add `CategoryFuel Category = "FUEL"` constant (one-line addition to that file) for routing.
5. **UoW for multi-write ops**: engine's claim-audit pass uses `uow.NewSQLUnitOfWork(db).Execute` + `OutboxWriter.SaveEvents` inside the tx (TxFromContext picks up the tx, `outbox.go:40-42`).

---

## 2. Anomaly engine algorithms

Input: `telemetry_snapshots` rows (the current persistence of the ingestion spec's `PositionEvent` with `fuel_level` + `odometer`). Engine state is keyed **per vehicle** (`vehicle_id`), rebuilt on startup by replaying the last N snapshots per vehicle.

### 2.1 Per-vehicle state

```go
type vehicleFuelState struct {
  vehicleID      string
  sensorFitted   bool        // fuel_sensor_fitted (00042, geofence spec)
  tankCapacity   float64     // tank_capacity_litres (00042); 0 = unknown
  medianWindow   []float64   // rolling fuel_level readings (median_window size)
  lastLevel      float64     // last smoothed level (percent or litres, per level_unit)
  lastOdometer   float64     // monotonic guard
  lastTs         time.Time
  baselineLevel  float64     // smoothed level at trip start / engine start
  refillPending  float64     // litres accumulated since last claim boundary
  stopStart      time.Time   // long-stop (siphon) tracking
  speed          float64
}
```

### 2.2 Pipeline (per new snapshot, per vehicle)

1. **Clamp** — `fuel_level` clamped to `[0, tankCapacity]` when capacity known; if `fuel_sensor_fitted=false` OR capacity unknown → skip all level-based checks, run **odometer-only** checks (2.7).
2. **Median smoothing** — maintain rolling window of `fuel.median_window` (default 7) readings; smoothed value = median of window. Reject spikes: a single reading deviating > `fuel.spike_deviation_pct` (default 25%) from window median is held for one more reading before acting (noise suppression).
3. **Noise floor** — deltas smaller than `fuel.noise_floor_pct` (default 1.5% of tank capacity) are ignored.
4. **Refill detection** — smoothed level jump ≥ `fuel.refill_threshold_litres` (default 20 L; expressed in litres = Δlevel% × capacity) sustained across ≥ 2 consecutive readings:
   - `estimated_litres = Δlevel × tankCapacity` (or raw Δ if `level_unit=litres`)
   - accumulate into `refillPending`
   - write `fuel_events(event_type='refill_detected')`
   - emit AlertEvent `refill_detected`
5. **Drain / theft suspected** — level drop ≥ `fuel.theft_drop_threshold_litres` (default 10 L) **without** trip end (trip status ∈ `ActiveTripStatuses` — `internal/domain/trip/entity.go:43-50` — or trip_id null with vehicle stationary) and speed ≈ 0:
   - write `fuel_events(event_type='drain_theft_suspected')`
   - write `driver_behaviour_events(event_type='fuel_theft_suspicion', severity='high')`
   - emit AlertEvent `drain_theft_suspected`
6. **Abnormal drain** — same drop while trip `in_transit` and speed > 0 (driving consumption exceeding `fuel.abnormal_drain_l_per_km` × odometer delta by > `fuel.abnormal_drain_margin_pct`, default 30%): `abnormal_drain` event (warning severity behaviour event).
7. **Siphon confirmation** — drop ≥ `fuel.siphon_drop_threshold_litres` (default 15 L) occurring during a stop ≥ `fuel.siphon_stop_minutes` (default 20 min, speed < `fuel.stop_speed_kmh` default 5): `siphon_confirmed` event + high-severity behaviour event + critical AlertEvent.
8. **Odometer rollback** — `odometer < lastOdometer - fuel.odometer_tolerance_km` (default 1 km): write `fuel_events(event_type='odometer_rollback')` + behaviour event + AlertEvent; reset baseline to new reading.
9. **Refuel-vs-claim windowing** — refills accumulate per (trip, driver); audit compares cumulative refills against claim litres (see §3).

### 2.3 Thresholds as config rows

All thresholds live in `company_config` (**created by spec 02 @00042**; seeded here in 00043) — seeded defaults, no code constants. See §9 for the full table.

### 2.4 Trip-end reset

On trip status → `delivered`/`completed`/`cancelled` (detected by polling `trips` status): flush `refillPending` into a per-trip refill total (for claim audit), reset smoothing baseline.

---

## 3. Fuel audit flow (claim → checks → audit_status)

### 3.1 States

`driver_expenses.audit_status` (new column, 00043): `pending` (default) → `needs_review` (engine-flagged) | `passed` (clean) → `failed` (admin review verdict). `fuel_claim_audits.result` mirrors this per-claim.

### 3.2 Flow

1. **Claim created** — `CreateExpense` (`internal/service/kharcha_service.go:251-298`) with `category='fuel'`; `fuel_litres` (new column) captured from claim form/description. `audit_status='pending'`.
2. **Audit job** (engine pass, runs every tick for `status='pending'` claims with `category='fuel'`):
   - Load claim's trip → vehicle (`trips.vehicle_id`) → `tank_capacity_litres`, `fuel_sensor_fitted` (00042 columns) + `current_mileage` (kmpl) via **plain SQL** (sqlc doesn't select odometer/capacity — verified `db/query/vehicles.sql`).
   - **Check A (level-based)**: `litres_expected_level = Σ refill Δlevel% × tank_capacity_litres` over the claim window (trip start → claim `created_at`, or since previous fuel claim on same trip). Skipped when capacity unknown or sensor absent.
   - **Check B (odometer-based)**: `litres_expected_odo = odometer_delta_km / kmpl` where `kmpl = vehicles.current_mileage` if > 0 else `company_config fuel.kmpl_default` (default 4.0).
   - **Check C (cross-check)**: A vs B must agree within `fuel.claim_crosscheck_margin_pct` (default 25%); if only one check available, it stands.
   - **Verdict**: `variance_litres = litres_claimed − litres_expected`; if `|variance| ≤ fuel.claim_tolerance_pct` (default 20%) → `passed`, else → `needs_review`.
   - Write/upsert `fuel_claim_audits` row (`expense_id` UNIQUE — re-audit upserts), set `driver_expenses.audit_status`.
3. **Annotate-first (default)** — `needs_review` **does not block** `ApproveExpense`. The kharcha queue row shows an amber badge + variance; admin may still approve.
4. **Enforce mode** — when `company_config fuel.audit_enforce = 'true'`, `ApproveExpense` (`internal/service/kharcha_service.go:166`) gains a gate **before** the status UPDATE:

```go
// inside ApproveExpense, after db := getter.DB(), before step 1 UPDATE
if s.fuelAuditEnforce(ctx, db) {            // reads company_config fuel.audit_enforce
    var as string
    _ = db.QueryRowContext(ctx, `SELECT audit_status FROM driver_expenses WHERE id = ?`, expenseID).Scan(&as)
    if as == "needs_review" {
        return fmt.Errorf("claim flagged by fuel audit (needs review); review at /fuel/audit")
    }
}
```

Existing rows-affected guard (`n == 0 → "expense already processed"`) stays.

5. **Admin review** — `POST /fuel/audit/{id}/review` sets `result = passed|failed`, `reviewed_by`, `reviewed_at`, `review_note`, and flips `driver_expenses.audit_status` accordingly. A `failed` claim stays approvable in annotate mode (badge only), un-approvable in enforce mode. Full trail in `fuel_claim_audits` (approved-by-admin audit trail retained — `approve_kharcha` logAudit untouched, `kharcha_service.go:217`).

### 3.3 UI surfacing

`KharchaExpense` view gains `AuditStatus`, `VarianceLitres`, `FuelLitres` (service query adds columns to the three SELECTs in `kharcha_service.go:52-71, 87-104, 128-146`); `kharcha_queue.html` shows badge + variance; audit queue lives under `/fuel/audit` (§6).

### Company_config sequencing guard

`company_config` is created by Spec 02 @00042. THIS migration (00043) seeds/uses it (§5 item 10 seeds, §3.2 enforce gate reads `fuel.audit_enforce`). Two mitigations, choose one at implementation time: (a) build Spec 02 first (recommended), or (b) prepend `CREATE TABLE IF NOT EXISTS company_config (...)` guard with the canonical schema from Spec 02 @00042 so this migration never crashes `goose up` if run before 00042. Do not invent the schema here — reference Spec 02's canonical DDL.

---

## 4. Behaviour events catalog + score formula

### 4.1 Event types (driver_behaviour_events.event_type)

| event_type | Source | Default weight | Notes |
|---|---|---|---|
| `speeding` | ingestion AlertEvent (alerting spec) | 8 | severity by over-speed magnitude |
| `harsh_braking` | ingestion AlertEvent | 6 | |
| `harsh_accel` | ingestion AlertEvent | 6 | |
| `idling` | ingestion AlertEvent (idle > config minutes) | 3 | |
| `night_driving` | ingestion AlertEvent (22:00–05:00, tz `Asia/Kolkata` from `company_settings.timezone`) | 2 | informational, low weight |
| `fuel_theft_suspicion` | fuel engine (§2.5/2.7) | 25 | heavy |
| `odometer_rollback` | fuel engine (§2.8) | 20 | heavy; admin resolve clears |

Weights are **denormalized into the row at write time** (copied from `company_config scorecard.weight.<type>`) so historical scores don't shift when weights change.

### 4.2 Score formula

30-day rolling, linear decay, per driver:

```
penalty(t) = weight × severity_mult(sev) × decay(days_ago)
decay(d)   = (30 − d) / 30            for 0 ≤ d < 30, else 0
score      = clamp(100 − Σ penalty, 0, 100)
```

- `severity_mult`: low=1.0, medium=1.5, high=2.0.
- Window: `scorecard.window_days` (default 30). History in `driver_scores`; current denormalized to `drivers.score`, `drivers.tier`.
- **Tiers** (config): A ≥ `scorecard.tier_a` (85), B ≥ `scorecard.tier_b` (70), else C.
- **Cold start**: no events in window → score 100, tier A, but leaderboard flags `insufficient_data` until `scorecard.min_events` (default 3) events exist.
- **Hard cap rule**: any un-resolved `fuel_theft_suspicion` or `odometer_rollback` in window caps score at tier-C ceiling (`scorecard.fraud_cap` default 69) — configurable off.

### 4.3 Recompute triggers

- After each engine pass that writes behaviour events (incremental: recompute affected driver).
- Nightly sweep ticker (main.go) recomputes all drivers with events in window.
- Writes: one `driver_scores` row per computation + `UPDATE drivers SET score=?, tier=?`.

---

## 5. DDL — `db/migrations/00043_fuel_audit_scorecard.sql`

```sql
-- +goose Up
-- FUEL AUDIT + DRIVER SCORECARD (owned by fuel spec; depends on 00042 geofence columns
-- tank_capacity_litres / fuel_sensor_fitted on vehicles — reference only, no changes here)

-- 1. fuel_events — durable per-vehicle fuel anomaly/refill record
CREATE TABLE IF NOT EXISTS fuel_events (
    id              TEXT PRIMARY KEY,
    vehicle_id      TEXT NOT NULL,
    trip_id         TEXT,
    driver_id       TEXT,
    event_type      TEXT NOT NULL CHECK (event_type IN
                      ('refill_detected','drain_theft_suspected','abnormal_drain',
                       'siphon_confirmed','odometer_rollback')),
    fuel_level_before REAL,
    fuel_level_after  REAL,
    odometer_before   REAL,
    odometer_after    REAL,
    estimated_litres  REAL,
    confidence        REAL,
    details           TEXT,
    occurred_at       DATETIME NOT NULL,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id)
);
CREATE INDEX IF NOT EXISTS idx_fuel_events_vehicle_time ON fuel_events(vehicle_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_fuel_events_trip ON fuel_events(trip_id);

-- 2. fuel_claim_audits — per-claim audit trail (one row per claim, re-audit upserts)
CREATE TABLE IF NOT EXISTS fuel_claim_audits (
    id                  TEXT PRIMARY KEY,
    expense_id          TEXT NOT NULL UNIQUE,
    trip_id             TEXT,
    vehicle_id          TEXT,
    driver_id           TEXT,
    litres_claimed      REAL,
    litres_expected_level REAL,
    litres_expected_odo REAL,
    level_delta_pct     REAL,
    odometer_delta_km   REAL,
    tank_capacity_litres REAL,
    kmpl_used           REAL,
    variance_litres     REAL,
    variance_pct        REAL,
    result              TEXT NOT NULL DEFAULT 'needs_review'
                          CHECK (result IN ('needs_review','passed','failed')),
    checks              TEXT NOT NULL DEFAULT '[]',
    reviewed_by         TEXT REFERENCES users(id),
    reviewed_at         DATETIME,
    review_note         TEXT,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (expense_id) REFERENCES driver_expenses(id),
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id)
);
CREATE INDEX IF NOT EXISTS idx_fuel_claim_audits_result ON fuel_claim_audits(result);

-- 3. driver_behaviour_events — weighted scorecard inputs
CREATE TABLE IF NOT EXISTS driver_behaviour_events (
    id          TEXT PRIMARY KEY,
    driver_id   TEXT NOT NULL,
    trip_id     TEXT,
    vehicle_id  TEXT,
    event_type  TEXT NOT NULL CHECK (event_type IN
                  ('speeding','harsh_braking','harsh_accel','idling','night_driving',
                   'fuel_theft_suspicion','odometer_rollback')),
    severity    TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low','medium','high')),
    weight      REAL NOT NULL DEFAULT 1.0,
    metadata    TEXT NOT NULL DEFAULT '{}',
    occurred_at DATETIME NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id),
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);
CREATE INDEX IF NOT EXISTS idx_dbe_driver_time ON driver_behaviour_events(driver_id, occurred_at);

-- 4. driver_scores — 30-day rolling score history
CREATE TABLE IF NOT EXISTS driver_scores (
    id           TEXT PRIMARY KEY,
    driver_id    TEXT NOT NULL,
    score        REAL NOT NULL,
    tier         TEXT NOT NULL CHECK (tier IN ('A','B','C')),
    period_start DATETIME NOT NULL,
    period_end   DATETIME NOT NULL,
    event_counts TEXT NOT NULL DEFAULT '{}',
    computed_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id)
);
CREATE INDEX IF NOT EXISTS idx_driver_scores_driver ON driver_scores(driver_id, period_end DESC);

-- 5. company_config — fuel + scorecard knobs (plain SQL read, no sqlc regen).
-- OWNERSHIP: the table is CREATED ONCE by spec 02 (geofence) at migration 00042.
-- Per 00-migration-ownership-index.md, do NOT CREATE it here. This spec only
-- seeds rows (item 10 below) and adds no columns to it.

-- 6. drivers — current score/tier (denormalized)
ALTER TABLE drivers ADD COLUMN score REAL;
ALTER TABLE drivers ADD COLUMN tier TEXT;

-- 7. driver_settlements — performance bonus (written by settlement computation)
ALTER TABLE driver_settlements ADD COLUMN performance_bonus REAL NOT NULL DEFAULT 0.0;

-- 8. driver_expenses — audit columns
ALTER TABLE driver_expenses ADD COLUMN audit_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (audit_status IN ('pending','needs_review','passed','failed'));
ALTER TABLE driver_expenses ADD COLUMN fuel_litres REAL;

-- 9. RBAC seeds — fuel:* and scorecard:*
INSERT OR IGNORE INTO permissions (name, description) VALUES
('fuel:read',       'View fuel audit queue and reports'),
('fuel:update',     'Review fuel audit claims'),
('scorecard:read',  'View driver scorecard and leaderboard');
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name LIKE 'fuel:%' OR name LIKE 'scorecard:%';
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.role_id, p.id
FROM (SELECT 2 AS role_id UNION ALL SELECT 3) r
JOIN permissions p ON p.name IN ('fuel:read','scorecard:read');

-- 10. company_config seed defaults (full key list in §9)
INSERT OR IGNORE INTO company_config (key, value, description) VALUES
('fuel.median_window', '7', 'fuel_level smoothing window (readings)'),
('fuel.noise_floor_pct', '1.5', 'min delta % of tank capacity to act on'),
('fuel.refill_threshold_litres', '20', 'level jump >= this = refill detected'),
('fuel.theft_drop_threshold_litres', '10', 'drop >= this without trip end = theft suspect'),
('fuel.siphon_drop_threshold_litres', '15', 'drop during long stop = siphon confirmed'),
('fuel.siphon_stop_minutes', '20', 'stop duration qualifying as long stop'),
('fuel.stop_speed_kmh', '5', 'speed below this counts as stopped'),
('fuel.odometer_tolerance_km', '1', 'odometer decrease beyond this = rollback'),
('fuel.abnormal_drain_l_per_km', '0.6', 'expected consumption per km for abnormal-drain check'),
('fuel.abnormal_drain_margin_pct', '30', 'margin over expected consumption'),
('fuel.level_unit', 'percent', 'percent|litres'),
('fuel.tank_capacity_default', '0', 'fallback capacity when 00042 column NULL (0 = unknown)'),
('fuel.kmpl_default', '4.0', 'fallback km/l when vehicles.current_mileage is NULL/0'),
('fuel.claim_tolerance_pct', '20', '|variance| within this % = passed'),
('fuel.claim_crosscheck_margin_pct', '25', 'level-based vs odometer-based expected litres'),
('fuel.audit_enforce', 'false', 'true = needs_review claims are un-approvable'),
('fuel.tick_interval_seconds', '30', 'engine poll interval'),
('fuel.gap_tolerance_minutes', '30', 'telemetry gap beyond this resets smoothing window'),
('scorecard.window_days', '30', 'rolling score window'),
('scorecard.tier_a', '85', 'tier A threshold'),
('scorecard.tier_b', '70', 'tier B threshold'),
('scorecard.min_events', '3', 'events required before tier is shown'),
('scorecard.fraud_cap', '69', 'unresolved theft/rollback caps score at this'),
('scorecard.weight.speeding', '8', ''),
('scorecard.weight.harsh_braking', '6', ''),
('scorecard.weight.harsh_accel', '6', ''),
('scorecard.weight.idling', '3', ''),
('scorecard.weight.night_driving', '2', ''),
('scorecard.weight.fuel_theft_suspicion', '25', ''),
('scorecard.weight.odometer_rollback', '20', ''),
('scorecard.bonus_a_pct', '5', 'settlement bonus % for tier A'),
('scorecard.bonus_b_pct', '2', 'settlement bonus % for tier B'),
('scorecard.bonus_c_pct', '0', 'settlement bonus % for tier C');

-- +goose Down
DROP INDEX IF EXISTS idx_dbe_driver_time;
DROP INDEX IF EXISTS idx_driver_scores_driver;
DROP INDEX IF EXISTS idx_fuel_claim_audits_result;
DROP INDEX IF EXISTS idx_fuel_events_trip;
DROP INDEX IF EXISTS idx_fuel_events_vehicle_time;
DROP TABLE IF EXISTS driver_scores;
DROP TABLE IF EXISTS driver_behaviour_events;
DROP TABLE IF EXISTS fuel_claim_audits;
DROP TABLE IF EXISTS fuel_events;
-- (company_config DROP intentionally omitted: owned/created by spec 02 @00042)
ALTER TABLE drivers DROP COLUMN score;
ALTER TABLE drivers DROP COLUMN tier;
ALTER TABLE driver_settlements DROP COLUMN performance_bonus;
ALTER TABLE driver_expenses DROP COLUMN audit_status;
ALTER TABLE driver_expenses DROP COLUMN fuel_litres;
DELETE FROM permissions WHERE name IN ('fuel:read','fuel:update','scorecard:read');
```

**No sqlc regen** — new tables read via plain SQL (`repository.DBGetter`), matching `KharchaService`. Migration auto-embeds via `go:embed migrations/*.sql` (`db/migrations.go:5`) — no code change needed.

---

## 6. API + Datastar UI

### 6.1 Routes (`internal/handlers/fuel.go`, `internal/handlers/scorecard.go`)

```
GET  /fuel/audit                          fuel:read    audit queue dashboard (page)
GET  /fuel/audit/queue                    fuel:read    HTMX partial — live queue (30s refresh)
GET  /fuel/audit/{id}                     fuel:read    claim detail w/ checks + variance
POST /fuel/audit/{id}/review              fuel:update  verdict passed|failed + note
POST /fuel/audit/run                      fuel:update  manual engine/audit backfill pass
GET  /fuel/reports/kmpl                   fuel:read    kmpl efficiency report (page + partial)
GET  /scorecard                           scorecard:read  leaderboard page
GET  /scorecard/table                     scorecard:read  HTMX partial — ranked rows
GET  /scorecard/drivers/{id}              scorecard:read  driver detail + 30-day trend
```

Mount in `cmd/server/main.go` protected block (~line 573, next to `/kharcha`):

```go
r.Route("/fuel", app.FuelAudit.Routes)
r.Route("/scorecard", app.Scorecard.Routes)
```

Wire in `internal/handlers/app.go`: add `FuelAudit *FuelAuditHandlers`, `Scorecard *ScorecardHandlers` to `App` (struct line 27-54) and `NewApp` (line 57-92).

### 6.2 Handler shape

Follow `KharchaHandlers` exactly (`internal/handlers/kharcha.go:20-26`): `Routes(r chi.Router)` with `middleware.ResourcePermission(h.AuthSrv, "fuel", "read"/"update")` / `("scorecard", "read")`, `renderPage`/`renderFragment` (`internal/handlers/app.go:349,475`).

### 6.3 Templates (`internal/templates/` — auto-picked by `parseTemplates`)

- `fuel_audit_dashboard.html` — stats row (pending audits, needs_review, variance avg, enforce mode toggle indicator) + queue table: claim, driver, trip, `litres_claimed` vs `expected (level|odo)`, variance badge (green/amber/red), `audit_status` pill, REVIEW button.
- `fuel_audit_queue.html` — HTMX partial (`hx-get="/fuel/audit/queue" hx-trigger="every 30s"`, mirroring `kharcha_dashboard.html:63-71`).
- `fuel_audit_detail.html` — checks A/B/C results, per-refill fuel_events list, admin verdict form.
- `scorecard_leaderboard.html` + `scorecard_table.html` — rank, driver, score, tier badge (A emerald / B blue / C rose), 30-day sparkline from `driver_scores` history, `insufficient_data` flag, preferred-load badge.
- `scorecard_driver.html` — event breakdown by type (counts + weighted penalty), score history chart, tier.
- `fuel_kmpl_report.html` — per-vehicle table: odometer delta, refill litres, computed kmpl, `current_mileage` (configured), variance; per-trip breakdown.

Datastar/HTMX assets already loaded in `layout.html:15`; no new JS.

### 6.4 Preferred-load ordering hook

`ScorecardService.PreferredDrivers(ctx, limit)` — plain SQL `ORDER BY score DESC` with tier. Phase-3 integration point: dispatcher trip-assignment UI (`internal/handlers/trips.go` assign flow) sorts candidate drivers by score; no core-flow change.

---

## 7. Settlement bonus hook (exact change)

**File**: `internal/service/driver_settlement_service.go`
**Function**: `CreateSettlementForTrip` (line 46) — also add the missing INSERT (verified: **no Go code persists settlements today**; both create/process functions only return structs — the bonus column would otherwise never land).

Change `netPayout` computation (lines 55-58) to:

```go
netPayout := fare - advances - deductions
bonus := 0.0
if s.store != nil {
    bonus = s.scorecard.BonusForPayout(ctx, driverID, netPayout) // tier → bonus_pct from company_config; 0 when score unknown
    netPayout += bonus
}
if netPayout < 0 {
    netPayout = 0
}
```

And persist (new — fixes pre-existing gap):

```go
// INSERT INTO driver_settlements
// (id, trip_id, driver_id, gross_fare, advances_kharcha, deductions, performance_bonus, net_payout, status)
// VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending')
```

`DriverSettlementRecord` (lines 13-26) gains `PerformanceBonus float64`. `ProcessFinancialSettlement` (line 86) applies the same bonus logic. The kharcha deduction path (`kharcha_service.go:204-211`) is untouched — it operates on `advances_kharcha`/`net_payout` only. Bonus scale: `scorecard.bonus_{a,b,c}_pct` (5/2/0 default) applied to pre-bonus `net_payout`; score unknown → 0 bonus.

---

## 8. Go file list

**New**

| File | Purpose |
|---|---|
| `internal/fuel/engine.go` | per-vehicle stateful anomaly engine (§2), config read, fuel_events + behaviour-event writes, AlertEvent emission via `outbox.NewOutboxWriter` |
| `internal/fuel/engine_test.go` | synthetic snapshot sequences: refill, theft, siphon, rollback, noise |
| `internal/service/fuel_audit_service.go` | claim audit pass (§3): load pending fuel claims → checks A/B/C → `fuel_claim_audits` upsert + `audit_status`; review + backfill methods; kmpl report query |
| `internal/service/scorecard_service.go` | behaviour-event write helper, 30-day score recompute, tier mapping, leaderboard query, `BonusForPayout`, admin resolve for fraud-cap events |
| `internal/handlers/fuel.go` | `/fuel/*` routes (§6.1) |
| `internal/handlers/scorecard.go` | `/scorecard/*` routes |
| `test/fuel_audit_test.go` | HTTP-level: claim → audit → annotate/enforce approve gate |
| `test/scorecard_test.go` | HTTP-level: events → score → leaderboard |

**New templates**: `internal/templates/fuel_audit_dashboard.html`, `fuel_audit_queue.html`, `fuel_audit_detail.html`, `fuel_kmpl_report.html`, `scorecard_leaderboard.html`, `scorecard_table.html`, `scorecard_driver.html`.

**Modified**

| File | Change |
|---|---|
| `db/migrations/00043_fuel_audit_scorecard.sql` | §5 |
| `cmd/server/main.go` | start `FuelEngine` goroutine (after line 616); mount `/fuel` + `/scorecard` (~line 573); nightly scorecard sweep ticker |
| `internal/service/service.go` | add `FuelAudit *FuelAuditService`, `Scorecard *ScorecardService` to `Services` (line 38-62) + `NewServices` (line 65-108) |
| `internal/service/kharcha_service.go` | `ApproveExpense` enforce gate (§3.4); SELECTs add `audit_status`, `fuel_litres` to `KharchaExpense` view (lines 52-71, 87-104, 128-146); `CreateExpense` accepts litres (line 251) |
| `internal/service/driver_settlement_service.go` | bonus hook + persistence (§7) |
| `internal/handlers/app.go` | `App` struct + `NewApp` wiring (lines 27-92) |
| `internal/handlers/kharcha.go` | queue renders audit badges (Dashboard/PendingQueue data) |
| `internal/templates/kharcha_queue.html` | `audit_status` badge + variance tooltip on fuel rows |
| `internal/founder/alerts/event.go` | add `CategoryFuel Category = "FUEL"` (one const) |
| `.env.example` | document `FUEL_TICK_INTERVAL` override (optional; company_config is primary) |

**Unchanged (verified no-touch)**: `internal/domain/expense/entity.go` (domain entity stays status-free), `internal/telemetry/sync.go`, `db/query/*` + `db/generated` (no sqlc regen), `internal/auth/casbin.go`, `db/migrations/00030/00031/00032/00033`.

---

## 9. Config table (`company_config`, plain SQL read)

Pattern: `SELECT value FROM company_config WHERE key = ?` via `repository.DBGetter`; cached in service struct with 60s TTL. Full key list in §5 DDL. Groups: `fuel.*` (engine + audit, §2.3/§3.2), `scorecard.*` (weights, tiers, bonus §4/§7). `fuel.audit_enforce` drives annotate-vs-enforce mode. `fuel.level_unit` documents the percent-vs-litres assumption on `telemetry_snapshots.fuel_level`.

---

## 10. Migration plan

1. **Current state**: `00039_experiments.sql` is the HEAD (TAKEN by experiments). `00040/00041` = ingestion (spec 01); `00042` = geofence (spec 02, owns `vehicles.tank_capacity_litres`/`fuel_sensor_fitted`); **this spec owns 00043** (references those columns only).
2. **00043 ordering**: goose runs in filename order; 00043 is after 00042 by construction. Engine code must handle NULL capacity (`fuel.tank_capacity_default` fallback, or skip level checks) so 00042 landing late degrades gracefully — but the column must exist before the engine runs, so **00043 and 00042 are deploy-order coupled**; ship both before enabling the engine loop.
3. **Embedding**: `db/migrations.go` embeds `migrations/*.sql` via `go:embed` — adding the file is sufficient; no provider changes (`cmd/server/main.go:138-153`).
4. **RBAC**: seeds are `INSERT OR IGNORE` — idempotent on re-run; admin gets `fuel:*`/`scorecard:*` via the role_permissions insert (§5 item 9). Note: 00012's blanket admin grant runs only at migration time, so explicit seeding here is required.
5. **Rollback**: `-- +goose Down` included (§5); SQLite `DROP COLUMN` supported (modernc, per 00013 precedent).

### 10.1 Implementation decisions (2A — fuel engine core, 2026-08-19)

1. **`fuel.spike_deviation_pct` seed added** to the §5 seed list (value `25`) — the pipeline
   (§2.2 step 2) names the key but the §5 DDL seed list omitted it. Shipped with 00043.
2. **`idx_telemetry_snapshots_vehicle_time(vehicle_id, timestamp)` added** to 00043 — the
   engine's warm-up (`ORDER BY timestamp DESC LIMIT n`) and incremental poll
   (`WHERE vehicle_id = ? AND timestamp > ?`) both scan by vehicle; §13.12's open item
   resolved in favour of the index.
3. **`company_config` seeds use the canonical 00042 schema** `(tenant_id, key, value)`
   — the §5 seed SQL lists a `description` column that does not exist in the canonical
   DDL (00042). §13 item 4 ("reference Spec 02's canonical DDL") wins.
4. **Engine warm-up replays the pipeline with emission enabled** (§2/§13.13) — replaying
   the last N snapshots through the full detector means anomalies inside the warm-up
   window are detected on restart; the `lastTs` watermark then prevents re-processing.
   Duplicate alerts within the warm-up window after a restart are accepted and
   documented in `internal/fuel/engine.go` godoc.
5. **`abnormal_drain` writes no behaviour row** — the `driver_behaviour_events` CHECK
   constraint (§5 item 3) admits only the 7 catalog types; `abnormal_drain` is not among
   them. It is persisted to `fuel_events` + AlertEvent only.

---

## 11. Edge cases

1. **Split refills across claims** — audit window = trip start → claim `created_at`, or since the previous approved/pending fuel claim on the same trip; `litres_expected_level` sums all `refill_detected` events in window. Multiple claims per trip supported (each audits its own window slice).
2. **Sensor drift / noise** — median window (`fuel.median_window`=7) + spike hold (one-reading confirmation) + `fuel.noise_floor_pct` (1.5%).
3. **Tank capacity unknown** — 00042 NULL or `fuel_sensor_fitted=false`: skip check A, run check B (odometer/kmpl) alone; `fuel.tank_capacity_default` can be set per fleet. `level_unit=litres` vs `percent` handled via config.
4. **Telemetry gaps** — `INSERT OR REPLACE` dedupes by snapshot id (`internal/telemetry/sync.go:72`); gaps > `fuel.gap_tolerance_minutes` (30) reset the smoothing window and stop-start tracking; odometer monotonic guard compares against *last known* value, not consecutive readings.
5. **Level-unit ambiguity** — the legacy naive rule (`telemetry_service.go:84-113`) treats raw `fuel_level` drop as litres; the engine treats it as percent×capacity by default. Engine supersedes the legacy rule; keep legacy code until Phase 1 ships, then delete or redirect it to `FuelEngine`.
6. **Legit odometer rollback** (tracker reinstall, instrument swap) — `odometer_rollback` event + behaviour event; admin resolves via scorecard driver detail (clears fraud-cap, excluded from score after resolution; `scorecard.fraud_cap` then un-applies).
7. **Night driving window** — uses `company_settings.timezone` (default `Asia/Kolkata`, `db/migrations/00001_initial.sql:40`), not server local time.
8. **Vehicle between trips** — engine state keyed by `vehicle_id`; `trip_id`/`driver_id` nullable on `fuel_events`/behaviour events; theft detection runs off-trip (stationary vehicle).
9. **Score cold start** — `scorecard.min_events` (3) before tier displayed; leaderboard marks `insufficient_data` instead of showing a misleading A.
10. **Bonus on negative payout** — bonus computed on pre-clamp `net_payout`; final clamp `≥ 0`; bonus 0 when score unknown or no settlement row.
11. **Driver/vehicle handover mid-trip** — attribution via `trips.driver_id` at event time (snapshots carry `driver_id` when provided).
12. **Re-audit after enforcement toggles** — `fuel_claim_audits.expense_id` UNIQUE; audit job upserts; `audit_status` on `driver_expenses` is the single source for the approve gate.
13. **Multiple instances** — engine is in-memory stateful per process; document single-instance constraint (matches current deploy model). Restart warm-up replays last N snapshots per vehicle from `telemetry_snapshots`.

---

## 12. Phased rollout

| Phase | Scope | Exit criteria |
|---|---|---|
| 0 | Migration 00043 + config seeds + RBAC + `FuelAudit`/`Scorecard` services stubs | migrate clean on dev DB; `SELECT * FROM company_config` seeded; `fuel:read` usable by admin |
| 1 | Engine (annotate mode): fuel_events, claim audits (`needs_review` only, no blocking), AlertEvent emission on outbox bus, `/fuel/audit` UI, kharcha badges | refill/theft detected on synthetic snapshots; audit rows created; queue shows badges; approve flow unchanged |
| 2 | Enforce mode (`fuel.audit_enforce=true`) + review endpoint + kmpl reports + engine backfill job | flagged claims blocked in enforce mode; review verdicts flip audit_status; kmpl report accurate vs PnL numbers (`internal/pnl/service.go:60-67`) |
| 3 | Scorecard: behaviour-event ingestion (engine events + alerting-spec events), score recompute, leaderboard, preferred-load ordering, settlement `performance_bonus` | scores/tiers computed; `CreateSettlementForTrip` persists bonus; leaderboard live |
| 4 (optional) | Mobile refuel-report flow + `fuel_level`/`odometer` upload from driver app (`mobile/src/services/telemetry.ts` — currently GPS-only; snapshots endpoint `/api/v1/telemetry/snapshots` already exists server-side) | driver submits refuel claim + snapshot from app; claim appears in audit queue with litres |

---

## 13. VERIFY items (at implementation time)

1. **`vehicles.odometer` not in sqlc queries** — confirmed (`db/query/vehicles.sql`); engine reads odometer via plain SQL. Do **not** regen sqlc for 00043 tables.
2. **`telemetry_alerts` is dead** — confirmed zero Go writers; engine must NOT write it; `fuel_events` is the durable store. (If ingestion spec later claims `telemetry_alerts`, reconcile then.)
3. **`driver_settlements` INSERT gap** — confirmed no Go INSERT exists; settlement bonus change must add persistence, not just the column.
4. **Two event buses** — services bus (`service.go:75`) vs outbox-relay bus (`main.go:819`); AlertEvent for the alerting spec must go through `outbox_events` (OutboxWriter), not `baseService.events.Publish`.
5. **`00042` dependency** — `tank_capacity_litres`/`fuel_sensor_fitted` arrive via geofence spec; confirm 00042 exists in `db/migrations/` before 00043 ships; engine code must tolerate NULL.
6. **`parseTemplates` glob** — new templates land automatically if placed in `internal/templates/` (verify glob in `internal/handlers/app.go:95` covers them; it parses the embedded set used by `renderPage`/`renderFragment`).
7. **RBAC role ids** — verify `roles` seed ids (1 admin / 2 dispatcher / 3 accountant / 4 viewer) before seeding `role_permissions` (see `db/migrations/00001_initial.sql` roles table + 00010 seed).
8. **`KharchaExpense` view query columns** — three SELECT column lists in `kharcha_service.go` must stay in sync when adding `audit_status`/`fuel_litres`.
9. **Enforce-gate placement** — insert the gate in `ApproveExpense` *before* the `UPDATE ... WHERE status='pending'` guard (`kharcha_service.go:182-192`) so flagged claims never flip status.
10. **Relay payload type** — AlertEvent consumers receive JSON maps (`relay.go:104-112`); alerting spec must assert `map[string]any`, not `alerts.AlertEvent` struct (founder handlers assert their own types — verify when wiring).
11. **`fuel_prices` availability** — litres-vs-rupee sanity display (not a gate) may use `fuel_prices` (00033); skip when no row for tenant.
12. **Engine warm-up query** — confirm `telemetry_snapshots` index `(trip_id, timestamp)` (00031) is sufficient for per-vehicle replay; add `(vehicle_id, timestamp)` index in 00043 if profiling demands (not included by default to keep DDL minimal — decide at implementation).
