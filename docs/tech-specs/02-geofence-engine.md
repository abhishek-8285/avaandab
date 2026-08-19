# GEOFENCE ENGINE — Implementation Spec v2

**Module:** `internal/geofence/` (new vertical slice, mirrors `internal/trip/` structure)
**Migration:** `db/migrations/00042_geofence_engine.sql` (next free: latest is `00039_experiments.sql`)
**Project:** Avandab fleet system — `/home/abhishek/Desktop/temux/basic`
**Stack (verified):** Go 1.26, chi v5.3.1, goose v3.27.3, SQLite (modernc), casbin v2.135.0, Datastar v1.0.2 + Tailwind templates, outbox-relay event bus (`cmd/server/main.go:819-826`), typed IDs, UoW pattern.
**Status:** Implementation-ready, updated (full-scope detention invoicing + FlyFleet-style drawing UI).

---

## 0. Verification Log (QA pass — 2026-08-19)

Migration renumber: geofence `00041` → **`00042`** (head is `00039_experiments.sql`, TAKEN;
ingestion owns `00040/00041`). `company_config` is created HERE (corrected index: `00042`);
fuel spec 03 must only seed it (collision flagged). Outbox cite `609-616` → `819-826`.

### Verification Log table

| Claim | Verdict | Correction / Evidence (file:line) |
|---|---|---|
| Geofence package / `DwellWorker` / `geofence_events` exist | CANT-VERIFY | no `geofence*` symbol in `internal/`/`db/` (grep) — design-only |
| `reach_pickup.go` / `start_transit.go` use cases exist | VERIFIED | `internal/trip/application/reach_pickup.go`, `start_transit.go` present |
| Outbox relay at `main.go:609-616` | WRONG | actual `main.go:819-826` (`outbox.NewRelay` @825) |
| Latest migration 00038 (geofence `00041`) | WRONG | head `00039_experiments`; geofence **00042** |
| `company_config` created in this spec | VERIFIED (plan) | but spec 03 also CREATEs it → **collision**, fix spec 03 to seed-only |
| Trip transitions avoid `deliver.go` | VERIFIED (design) | `deliver.go` only via `/trips/{id}/deliver` per handler |
| No lat/lng on `routes` | VERIFIED | `00005/00031/00038` schema confirmed no coord cols |
| `AssignVehicle` records no event | VERIFIED (design) | trip_aggregate.go `AssignVehicle` has no RecordEvent |

### Severity & Effort (major changes)

| Change | Severity | Effort |
|---|---|---|
| Migration renumber 00041 → 00042 | Low | S |
| `company_config` canonical creation here | Med | S |
| Geofence DDL + RBAC seeds | High | M |
| Dwell 4-state machine + geo math | High | L |
| Detention → invoice line items | High | L |
| Leaflet drawing UI | Med | M |
| Auto trip-transition wiring | High | M |

### Architectural Decisions (Decision / Tradeoff / Cost)

- **Geofence algorithm: ray-cast PIP + haversine buffer/hysteresis.** Decision: pure
  functions, unit-tested, equirectangular projection for edge distance (≤5km scale).
  Tradeoff: polygon hysteresis is approximated (boolean containment + edge-distance), not a
  true Minkowski offset — acceptable at GPS noise scale. Cost: none; deterministic, no deps.
- **Dwell state in `engine_state` (DB, per-vehicle).** Decision: persist to survive restart.
  Tradeoff: extra writes per fix vs crash-safe resumption. Cost: 1 row/vehicle.
- **Trip transitions fired by worker (not bus round-trip).** Decision: direct use-case call.
  Tradeoff: tighter coupling to trip aggregate, but avoids event-loop latency. Cost: worker
  must hold no domain state (per spec).
- **Detention billing via `invoice_line_items` + `GenerateInvoiceCommand.LineItems`.**
  Decision: paid invoices never auto-attached (admin-only). Tradeoff: protects billed
  invoices; requires admin endpoint. Cost: extra reconciliation code.
- **`company_config` ownership.** Decision: create once at 00042; all other specs seed rows.
  (See collision fix in spec 03.)

## 1) Architecture + Bus Flow

Three runtime actors:

```
driver app / MQTT stub
        │  POST /api/v1/telemetry/snapshots (RequireAPIAuth)   [internal/telemetry/sync.go]
        ▼
telemetry_snapshots (trip_id, vehicle_id, timestamp, lat, lng, speed, fuel_level, odometer)
        │
        ▼  poll every 10s (config)
DwellWorker (internal/geofence/infrastructure/worker/dwell_worker.go)
        │  per-vehicle 4-state machine, geo math, config, zone lookup
        ├──► ReachPickupUseCase / StartTransitUseCase  (vertical-slice use cases ONLY, UoW)
        ├──► trip_detentions rows (pickup/drop dwell windows)
        ├──► geofence_events rows (durable event/alert log)
        └──► outbox_events  (via outbox.OutboxWriter)
                    │
                    ▼  relay poll 5s  [cmd/server/main.go:825 outbox.NewRelay(database, eventBus, logger)]
              main.go eventBus (events.InMemoryBus)
                    │
                    ├──► alerting spec consumer (subscribes "GeofenceAlertEvent")
                    └──► (founder handlers already subscribed; unaffected)
```

Bus rules (verified):

- **Use only the outbox-relay bus**: vertical-slice repos write `outbox_events` via `outbox.OutboxWriter.SaveEvents(ctx, aggregateID, aggregateType, events)` (`internal/shared/outbox/outbox.go`); `Relay.publish` unmarshals payload into `map[string]any` and dispatches `events.Event{Type: <struct name>, Payload: map}` onto the `eventBus` created at `cmd/server/main.go:819`.
- **Never the old bus**: `internal/service/service.go:59` builds a private `events.NewInMemoryBus()` (`bs.events`) used only by old `TripService`/`InvoiceService` subscribers. Geofence code must not touch it.
- **Event type strings are Go struct names** (`getEventTypeName`, outbox.go:79): vertical-slice trip events arrive as `"TripStartedEvent"`, `"TripReachedPickupEvent"`, `"TripInTransitEvent"`, `"TripCompletedEvent"`, payload keys `"TripID"`, `"TenantID"`, `"OccurredAt"` (no JSON tags on `trip_aggregate.go` events — do not rely on lowercase keys).
- Trip transitions are triggered **directly from the dwell worker** (same process, real-time), not via bus round-trip. Bus consumption is for cross-module reactions only (e.g., future alerting, invoice module listening for `"TripCompletedEvent"`).

New canonical alert event (consumed by the alerting spec):

```go
// internal/geofence/application/alerts.go
type GeofenceAlertEvent struct {
    AlertType    string  `json:"alert_type"`    // night_driving|restricted_zone|unauthorized_movement|off_hours_use|geofence_breach
    Severity     string  `json:"severity"`      // info|warning|critical
    TenantID     string  `json:"tenant_id"`
    VehicleID    string  `json:"vehicle_id"`
    DriverID     *string `json:"driver_id,omitempty"`
    TripID       *string `json:"trip_id,omitempty"`
    GeofenceID   string  `json:"geofence_id"`
    GeofenceName string  `json:"geofence_name"`
    Latitude     float64 `json:"latitude"`
    Longitude    float64 `json:"longitude"`
    OccurredAt   string  `json:"occurred_at"` // RFC3339
}
```

Worker emits via `outbox.NewOutboxWriter(db).SaveEvents(ctx, vehicleID, "Vehicle", []any{ev})` → relay → `eventBus` under type `"GeofenceAlertEvent"`. Same struct is persisted verbatim (JSON) into `geofence_events`.

---

## 2) Data Model DDL — `db/migrations/00042_geofence_engine.sql`

```sql
-- +goose Up
CREATE TABLE geofences (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('pickup','drop','depot','restricted','no_entry')),
    shape       TEXT NOT NULL CHECK (shape IN ('circle','polygon')),
    center_lat  REAL,                          -- circle only
    center_lng  REAL,                          -- circle only
    radius_m    REAL,                          -- circle only
    polygon     TEXT,                          -- polygon only: JSON [[lat,lng],...] (closed ring)
    route_name  TEXT,                          -- fallback: matches routes.source OR routes.destination (LOWER/TRIM)
    priority    INTEGER NOT NULL DEFAULT 0,    -- lower = higher priority; tie-break: smallest radius/area, then id
    is_active   INTEGER NOT NULL DEFAULT 1,    -- soft delete; active dwell keeps state until exit
    created_by  TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_geofences_tenant_active ON geofences(tenant_id, is_active);
CREATE INDEX idx_geofences_kind ON geofences(kind);

CREATE TABLE vehicle_geofences (               -- explicit per-vehicle zone bindings (depot/restricted/no_entry)
    vehicle_id   TEXT NOT NULL,
    geofence_id  TEXT NOT NULL,
    tenant_id    TEXT NOT NULL DEFAULT '1',
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (vehicle_id, geofence_id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (geofence_id) REFERENCES geofences(id)
);

CREATE TABLE geofence_events (                 -- durable event + alert log
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    vehicle_id  TEXT,
    trip_id     TEXT,
    geofence_id TEXT,
    zone_kind   TEXT,
    event_type  TEXT NOT NULL CHECK (event_type IN
        ('entering','inside','leaving','outside','breach','alert')),
    alert_type  TEXT,                          -- set when event_type='alert'
    severity    TEXT,
    latitude    REAL,
    longitude   REAL,
    details     TEXT,                          -- JSON payload (GeofenceAlertEvent / transition)
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_geofence_events_vehicle ON geofence_events(vehicle_id, created_at);
CREATE INDEX idx_geofence_events_trip ON geofence_events(trip_id, created_at);

CREATE TABLE engine_state (                    -- per-vehicle dwell state machine
    vehicle_id          TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL DEFAULT '1',
    state               TEXT NOT NULL DEFAULT 'outside'
                        CHECK (state IN ('outside','entering','inside','leaving')),
    trip_id             TEXT,
    geofence_id         TEXT,                  -- zone currently dwelled / entered
    zone_kind           TEXT,
    zone_entered_at     DATETIME,              -- first fix inside zone (debounce start)
    confirmed_at        DATETIME,              -- debounce confirmed entry
    exit_started_at     DATETIME,              -- leaving debounce start
    last_fix_at         DATETIME,
    last_lat            REAL,
    last_lng            REAL,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);

CREATE TABLE trip_detentions (
    id               TEXT PRIMARY KEY,
    tenant_id        TEXT NOT NULL DEFAULT '1',
    trip_id          TEXT NOT NULL,
    vehicle_id       TEXT,
    geofence_id      TEXT,
    zone_kind        TEXT NOT NULL CHECK (zone_kind IN ('pickup','drop')),
    entered_at       DATETIME NOT NULL,
    exited_at        DATETIME,
    dwell_seconds    INTEGER NOT NULL DEFAULT 0,
    free_seconds     INTEGER NOT NULL DEFAULT 0,     -- snapshot of free window at entry
    billable_seconds INTEGER NOT NULL DEFAULT 0,
    rate_per_hour    REAL NOT NULL DEFAULT 0,
    amount           REAL NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open','closed','attached','invoiced','waived')),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (geofence_id) REFERENCES geofences(id)
);
CREATE INDEX idx_trip_detentions_trip ON trip_detentions(trip_id);
CREATE INDEX idx_trip_detentions_status ON trip_detentions(status);

CREATE TABLE invoice_line_items (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    invoice_id  TEXT NOT NULL,
    trip_id     TEXT,
    line_type   TEXT NOT NULL CHECK (line_type IN ('freight','detention','accessorial')),
    description TEXT NOT NULL,
    quantity    REAL NOT NULL DEFAULT 1,      -- detention: billable hours
    unit_price  REAL NOT NULL DEFAULT 0,      -- detention: rate per hour
    amount      REAL NOT NULL DEFAULT 0,
    ref_id      TEXT,                         -- trip_detentions.id when line_type='detention'
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id),
    FOREIGN KEY (trip_id) REFERENCES trips(id)
);
CREATE INDEX idx_invoice_line_items_invoice ON invoice_line_items(invoice_id);

-- vehicles additions (tank capacity / fuel sensor / maintenance)
ALTER TABLE vehicles ADD COLUMN tank_capacity_litres REAL;
ALTER TABLE vehicles ADD COLUMN fuel_sensor_fitted INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vehicles ADD COLUMN maintenance_due DATE;

-- company_config (shared config table — also used by fuel + alerting specs)
CREATE TABLE company_config (
    tenant_id  TEXT NOT NULL DEFAULT '1',
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, key)
);

-- RBAC seeds
INSERT OR IGNORE INTO permissions (name, description) VALUES
('geofences:create', 'Create geofences'),
('geofences:read', 'Read geofences'),
('geofences:update', 'Update geofences'),
('geofences:delete', 'Delete geofences');
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name LIKE 'geofences:%';                    -- admin
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name LIKE 'geofences:%';                    -- dispatcher
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions WHERE name = 'geofences:read';                    -- viewer

-- +goose Down
DROP TABLE IF EXISTS invoice_line_items;
DROP TABLE IF EXISTS trip_detentions;
DROP TABLE IF EXISTS engine_state;
DROP TABLE IF EXISTS geofence_events;
DROP TABLE IF EXISTS vehicle_geofences;
DROP TABLE IF EXISTS geofences;
DROP TABLE IF EXISTS company_config;
ALTER TABLE vehicles DROP COLUMN tank_capacity_litres;
ALTER TABLE vehicles DROP COLUMN fuel_sensor_fitted;
ALTER TABLE vehicles DROP COLUMN maintenance_due;
DELETE FROM permissions WHERE name LIKE 'geofences:%';
```

Notes:

- Latest migration today is `00039_experiments.sql` (head is now TAKEN by experiments); `00042` is the agreed number (**00040/00041 reserved by ingestion spec 01**).
- RBAC: casbin adapter loads `role_permissions` (internal/auth/casbin.go:34) into `p = role, resource, action`; admin grant pattern copied from `00012_rbac.sql`. Policy loads at startup — restart or `authSvc.Reload()` after seeding.
- No lat/lng on `routes` (verified 00005/00031/00038) → `geofences.route_name` fallback matches `routes.source` / `routes.destination` (LOWER/TRIM, same normalization as `source_normalized`/`dest_normalized`).
- `telemetry_alerts` (00030) has a restrictive CHECK on `alert_type` — do not reuse it; `geofence_events` is the geofence log.

---

## 3) Dwell State Machine (per vehicle)

States: `outside → entering → inside → leaving → outside`.

```
outside ──fix inside zone (pre-debounce)──► entering
entering ──60s continuous inside (debounce)──► inside          [confirm entry]
inside   ──fix outside zone buffer──► leaving
leaving  ──60s continuous outside (debounce)──► outside        [confirm exit]
entering ──fix outside before debounce──► outside              (jitter ignored)
leaving  ──fix back inside before debounce──► inside           (jitter ignored)
```

Timing constants (all overridable in `company_config`, §9):

| Constant | Default | Meaning |
|---|---|---|
| `DwellDebounce` | 60s | fixes must be continuously inside/outside before state confirms |
| `BufferMetres` | 20m | expand zone for entering test (GPS error absorption) |
| `HysteresisMetres` | 25m | shrink zone for exit test (prevents boundary flap) |
| `MaxFixAge` | 30s | reject stale snapshots |
| `PollInterval` | 10s | worker sweep of `telemetry_snapshots` |
| `MaxDwellWindow` | 24h | cap dwell tracking per zone entry |

Rules:

- Entry test: `dist/point-in-zone <= (nominal + BufferMetres)`.
- Exit test: `dist > (nominal − HysteresisMetres)` (circles: `radius − hysteresis`; polygons: inward offset approximated by buffer on the ray-cast edge test — treat polygon containment as boolean, apply buffer/hysteresis via haversine distance to nearest polygon edge).
- State is persisted in `engine_state` (PK vehicle_id) → survives restart; worker resumes from `last_fix_at`.
- Zone selection on each fix: candidate = active zones for tenant ∩ (explicit `vehicle_geofences` ∪ route_name match for current trip) ∪ zones covering the fix. Overlap resolution: lowest `priority` number → smallest radius/area → lexicographic id. `restricted`/`no_entry` zones are matched for every fix of a bound vehicle regardless of trip; `pickup`/`drop` zones match only when they align with the trip's route (see §4 matching).
- `engine_state.trip_id` mismatch vs snapshot `trip_id` ⇒ vehicle reassigned mid-dwell ⇒ reset to `outside`, close any open detention (status `closed`), emit `leaving`/`outside` only if a zone was confirmed. Trip change is detected on the snapshot, not via bus (AssignVehicle on the aggregate records no event — verified trip_aggregate.go:120).

---

## 4) Geo Algorithms

`internal/geofence/domain/geo.go` — pure functions, unit-tested:

- **Haversine** `DistanceMeters(lat1,lng1,lat2,lng2) float64` — great-circle distance, Earth radius 6371008.8 m.
- **Point-in-polygon** — ray casting (even-odd rule) on the closed ring `[[lat,lng],...]`; handle vertex-on-edge with a small epsilon; reject rings with <3 points or non-closed rings (auto-close).
- **Circle contains** — `DistanceMeters(center, fix) <= radius + buffer`.
- **Polygon contains** — ray-cast hit; buffer/hysteresis applied by computing min haversine distance from fix to each edge segment (equirectangular projection for segment distance, adequate ≤ 5 km scale).
- **Zone match by route (pickup/drop)** — geofence has `route_name`; trip has `route_id` → `routes.source`/`routes.destination` (00005/00038). Match if `LOWER(TRIM(geofences.route_name))` equals either endpoint AND zone kind matches phase:
  - `pickup` zone relevant when trip status ∈ {started, reached_pickup}
  - `drop` zone relevant when trip status ∈ {reached_pickup, in_transit}
  - no match → fall back to coverage-only detection for `depot`/`restricted`/`no_entry`.

---

## 5) Trip Auto-Transition Wiring

**Exact use cases (both verified — internal/trip/application/):**

```go
// reach_pickup.go — ReachPickupCommand{TripID aggregate.TripID; TenantID shared.TenantID}
// Execute: uow.Execute → repo.Find → t.ReachPickup(clock.Now()) → repo.Save → logAudit
// start_transit.go — StartTransitCommand{TripID aggregate.TripID; TenantID shared.TenantID}
// Execute: uow.Execute → repo.Find → t.StartTransit(clock.Now()) → repo.Save → logAudit
```

Worker wiring (config-gated, §9; off by default in Phase B):

- **ReachPickup**: vehicle state confirms `inside` a `pickup` zone while trip status == `started` → `NewReachPickupUseCase(uow, clock).Execute(ctx, ReachPickupCommand{TripID: t, TenantID: "1"})`. Guard inside aggregate (`ReachPickup` requires `started`) is the final authority; use-case error is logged, not retried beyond one sweep.
- **StartTransit**: vehicle state confirms `leaving`→`outside` the pickup zone while status == `reached_pickup`, OR confirms `inside` a `drop` zone while status == `reached_pickup` → `NewStartTransitUseCase(uow, clock).Execute(ctx, StartTransitCommand{...})`.
- **GEOFENCE NEVER AUTO-DELIVERS.** `DeliverUseCase` (deliver.go) is only reachable via `POST /trips/{id}/deliver` (handlers/trips.go:384) and `POST /trips/{id}/deliver-pod` (POD, kharcha.go:126 → old `TripService.DeliverWithPOD`). Both remain authoritative; no code path from the dwell worker calls Deliver.
- Trip lookup: dwell worker resolves snapshot `trip_id` → trip status via `GetTripByID` plain SQL (no sqlc); worker holds no trip domain state.
- Tenant: hardcoded `"1"` matches every existing handler (handlers/trips.go:403).

---

## 6) Detention → Invoice (FULL SCOPE)

**Detention capture** (worker, per confirmed `inside` at `pickup`/`drop` zone):

- Open `trip_detentions` row on confirmed entry: `entered_at`, `free_seconds = config.free_minutes*60`, `rate_per_hour = config.rate`.
- Update on each sweep while inside: `dwell_seconds`, `billable_seconds = max(0, dwell_seconds − free_seconds)`, `amount = round2(billable_seconds/3600 * rate_per_hour)`.
- Close on confirmed exit: `exited_at`, status `closed`.
- One open detention per (trip, zone_kind); re-entry after exit opens a new row.

**Invoice line items — exact changes to `GenerateInvoiceCommand`** (`internal/invoice/application/generate_invoice.go:31`):

```go
type InvoiceLineItemInput struct {
    LineType    string  // freight|detention|accessorial
    Description string
    Quantity    float64
    UnitPrice   float64
    RefID       string  // trip_detentions.id
}

type GenerateInvoiceCommand struct {
    TenantID   shared.TenantID
    BookingID  string
    CustomerID string
    TripID     *string
    Subtotal   float64
    Tax        float64
    Discount   float64
    Total      float64
    LineItems  []InvoiceLineItemInput  // NEW — nil-safe; empty = current behaviour
}
```

Use case changes (`generate_invoice.go`):

1. Keep existing dedupe (`FindByBookingID` → return existing). **If the existing invoice is `paid` or `partially_paid`, never auto-attach line items** — return existing untouched (decision: admin-only adjustment for paid invoices, audit-logged).
2. If `existing` is `draft`/`issued`/`outstanding`: load its line items, compute missing detention lines (see below), persist `invoice_line_items`, recompute aggregate totals `Subtotal += Σ detention amounts`, `Total = Subtotal + Tax − Discount`, `repo.Save`.
3. New invoice: persist `LineItems` inside the same UoW `Execute` (invoice repo gains `SaveLineItems(ctx, invoiceID, []LineItem)` — plain SQL in `invoice_repository.go`, no sqlc regen).

**Auto-attach at completion** — `internal/handlers/trips.go` `CompleteTrip` (line 398) already runs `completeUC.Execute` then `generateInvoiceUC.Execute` (hardcoded 18% GST). Change the second call:

1. After `completeUC`, plain-SQL query: `SELECT ... FROM trip_detentions WHERE trip_id=? AND status='closed' AND amount > 0 AND NOT EXISTS (SELECT 1 FROM invoice_line_items WHERE ref_id = trip_detentions.id)`.
2. Build `LineItems` (`line_type='detention'`, `description = "Detention — <zone name> (<duration>)", quantity = billable hours, unit_price = rate_per_hour, ref_id`).
3. Pass into `GenerateInvoiceCommand`. Mark detentions `attached` after invoice save (same transaction).
4. If the trip has no booking (invoice path skipped), detentions stay `closed` — the admin adjust endpoint (below) attaches them later.

**Admin adjust endpoint** (never automatic on paid invoices):

- `POST /api/v1/geofences/detentions/{id}/attach` — attaches a closed detention to the trip's existing invoice as a line item; refuses when invoice `status='paid'` (HTTP 409); writes `audit_logs` row (`action='detention_attach_admin'`, table_name=`invoice_line_items`).
- `POST /api/v1/geofences/detentions/{id}/waive` — status `waived`, audit-logged.
- RBAC: `invoices:update` + `geofences:update` required.

---

## 7) Alerts Emitted

Worker evaluates each confirmed transition + continuous checks; every alert = `GeofenceAlertEvent` (JSON-tagged, §1) written to outbox (`aggregateType "Vehicle"`, event type `"GeofenceAlertEvent"`) AND persisted to `geofence_events` (`event_type='alert'`).

| alert_type | Trigger | severity |
|---|---|---|
| `geofence_breach` | vehicle enters `restricted` or `no_entry` zone (bound via `vehicle_geofences`) | critical |
| `unauthorized_movement` | confirmed exit of `pickup`/`drop` zone while trip status is `started`/`reached_pickup`/`in_transit` but the exit direction contradicts the trip phase (e.g., leaves pickup zone before `ReachPickup` confirmed, or movement detected for a vehicle with no active trip) | warning |
| `night_driving` | vehicle moving (fix-to-fix speed > 5 km/h) inside a `depot` zone during night window; night window from config, timezone from `company_config.geofence.timezone` (default `Asia/Kolkata`) | warning |
| `off_hours_use` | vehicle active (speed > 5 km/h) outside its depot zone during night window, when trip is not in `started`/`in_transit` | info |
| `restricted_zone` | vehicle dwells > 5 min in `restricted` zone (soft variant; `no_entry` uses `geofence_breach`) | warning |

Dedupe: same `(vehicle_id, alert_type, geofence_id)` not re-emitted within 15 min per alert_type. Night window + timezone: read `users.timezone` of the assigned driver (00029, default `Asia/Kolkata`) or `company_config` override.

---

## 8) API + Datastar UI

**Web UI** (Datastar, mirrors handlers/routes.go + internal/templates/route_*.html):

- Mount in authenticated group: `r.Route("/geofences", app.Geofences.Routes)` next to `r.Route("/routes", ...)` (main.go:548).
- `internal/handlers/geofences.go` — `GeofenceHandlers{*App}` with `init()` building UoW + use cases exactly like `TripHandlers.init()` (handlers/trips.go:37), using `uow.NewSQLUnitOfWork(h.DB)`, `clock.NewRealClock()`, `id.NewUUIDGenerator()`.
- Routes (all wrapped in `middleware.ResourcePermission(h.AuthSrv, "geofences", ...)`):
  - `GET /` list; `GET /new` form; `POST /new` create; `GET /{id}` view; `GET /{id}/edit` form; `POST /{id}/edit` update; `POST /{id}/delete` soft delete (is_active=0).
  - Datastar: `isDatastarRequest(r)` → `renderFragment(w, "geofence_list_table.html", ...)`, else `renderPage(w, r, "geofence_list.html", PageData{Title: "Geofences", ...})`; forms via `renderForm` — same as handlers/routes.go:37-70.
- Nav link in `internal/templates/layout.html` sidebar (after Routes link, line ~82) with the same `nav-item`/active pattern.
- Templates: `geofence_list.html`, `geofence_list_table.html`, `geofence_edit.html`, `geofence_view.html`.

**Drawing UI** (`geofence_edit.html` + `internal/static/js/leaflet.js` + `internal/static/js/geofence_draw.js`) — Avandab-native design, FlyFleet `AddGeofence.tsx` as reference only (verified at `/home/abhishek/Desktop/flyfleet/apps/dashboard-web/src/pages/admin/AddGeofence.tsx`: Leaflet `MapContainer`/`TileLayer`/`Circle`/`Polygon`, type toggle circle|polygon, `radius_m`, center lat/lng, coordinate array, name):

- Vendor Leaflet into `internal/static/js/` (served by `/static/*` → `cfg.StaticDir`, main.go:459-460; `internal/static/js/datastar.js` is the precedent).
- Keyless Google raster tiles: `https://mt1.google.com/vt/lyrs=m&x={x}&y={y}&z={z}&gl=IN`, `attribution: "Google"`, no API key. (OSM fallback: `https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png`. Honest caveat: keyless Google tiles are a ToS grey area — same pattern as FlyFleet production.)
- Form fields: name, kind (pickup/drop/depot/restricted/no_entry), shape (circle/polygon radio), radius_m slider (50–5000, default 500), route_name autocomplete (from `routes` source/destination), priority, is_active.
- Circle mode: click center → drag/radius input; Polygon mode: click-to-add vertices, double-click to close; store `polygon` as JSON `[[lat,lng],...]` (closed ring on save).
- POST form encodes `shape`, `center_lat`, `center_lng`, `radius_m`, `polygon` JSON; server validates (circle needs center+radius, polygon needs ≥3 points).
- Editing: `GET /{id}/edit` loads existing zone, JS re-renders Circle/Polygon on the map (mirrors AddGeofence.tsx load path incl. WKT-tolerant parse).

**API** (`internal/geofence/presentation/api/handlers/geofence_api.go`, registered like `tripAPIHandler.Register(r)` at main.go:357):

- `GET /api/v1/geofences` (list, ?kind=&active=), `POST /api/v1/geofences`, `GET /api/v1/geofences/{id}`, `PUT /api/v1/geofences/{id}`, `DELETE /api/v1/geofences/{id}` (soft).
- `GET /api/v1/vehicles/{id}/geofence-state` (engine_state + current zone).
- `GET /api/v1/trips/{id}/detentions`, `POST /api/v1/geofences/detentions/{id}/attach`, `POST /api/v1/geofences/detentions/{id}/waive`.
- `GET /api/v1/geofence-events?vehicle_id=&trip_id=&from=&to=`.
- All behind `middleware.RequireAPIAuth` group (main.go:341); mutating handlers also check `ResourcePermission`.

---

## 9) Config Table

Read via **plain SQL** (`database/sql`, `SELECT value FROM company_config WHERE tenant_id=? AND key=?`) in `internal/geofence/application/config.go` — **no sqlc regen** (sqlc.yaml schema is `db/migrations`; a new table would force regen of `db/generated/sqlite`; avoid by hand-writing queries).

Keys + defaults (fallback when row absent):

| key | default |
|---|---|
| `geofence.poll_interval_seconds` | 10 |
| `geofence.dwell_debounce_seconds` | 60 |
| `geofence.buffer_metres` | 20 |
| `geofence.hysteresis_metres` | 25 |
| `geofence.max_fix_age_seconds` | 30 |
| `geofence.auto_reach_pickup` | false (Phase C flips true) |
| `geofence.auto_start_transit` | false |
| `geofence.detention_free_minutes` | 30 |
| `geofence.detention_rate_per_hour` | 0 (0 = detention tracked but not billed) |
| `geofence.night_start_hour` | 22 |
| `geofence.night_end_hour` | 6 |
| `geofence.timezone` | Asia/Kolkata |
| `geofence.alert_dedupe_minutes` | 15 |

Admin edit surface: extend `settings.html` section (Settings handler) with a geofence config form; key-value CRUD via `company_config` plain SQL.

---

## 10) Go File List

**Create:**

| Path | Purpose |
|---|---|
| `db/migrations/00042_geofence_engine.sql` | §2 DDL (+ `company_config`) |
| `internal/geofence/domain/geo.go` | haversine, ray-cast PIP, edge distance |
| `internal/geofence/domain/geofence.go` | Geofence aggregate (id/tenant/name/kind/shape/geometry/route_name/priority/is_active; `Contains(lat,lng,phase)` with buffer/hysteresis) |
| `internal/geofence/domain/repository.go` | GeofenceRepository, EngineStateRepository, DetentionRepository, GeofenceEventRepository, ConfigRepository |
| `internal/geofence/application/dwell_engine.go` | 4-state machine, zone resolution, overlap rule, transition triggers |
| `internal/geofence/application/detention.go` | detention open/update/close, billable math |
| `internal/geofence/application/alerts.go` | `GeofenceAlertEvent` (JSON-tagged) + emission |
| `internal/geofence/application/config.go` | company_config plain-SQL reader + defaults |
| `internal/geofence/application/zone_crud.go` | create/update/soft-delete/list use cases (UoW) |
| `internal/geofence/infrastructure/persistence/sql/geofence_repository.go` | plain SQL implementations (database/sql + `repository.TxFromContext` pattern like trip_repository.go) |
| `internal/geofence/infrastructure/worker/dwell_worker.go` | poll loop, trip lookup, use-case invocation, outbox writes |
| `internal/geofence/presentation/api/handlers/geofence_api.go` | §8 API |
| `internal/geofence/presentation/web/handlers/geofence_web.go` | §8 Datastar handlers |
| `internal/geofence/presentation/web/viewmodels/geofence_viewmodel.go` | DTO → template |
| `internal/geofence/facade.go`, `facade_impl.go` | module facade (trip pattern) |
| `internal/templates/geofence_list.html` (+`_table.html`), `geofence_edit.html`, `geofence_view.html` | templates |
| `internal/static/js/leaflet.js`, `leaflet.css`, `geofence_draw.js` | drawing assets |
| `internal/invoice/application/attach_line_item.go` | admin attach/waive use case (audit-logged) |

**Modify:**

| Path | Change |
|---|---|
| `cmd/server/main.go` | register `/geofences` web routes + `/api/v1/geofences*`; start `dwellWorker.Run(ctx)` goroutine; (optionally) subscribe `eventBus` for `"TripCompletedEvent"` if bus-driven detention fallback is enabled |
| `internal/invoice/application/generate_invoice.go` | `InvoiceLineItemInput` + `LineItems` field; paid-invoice guard; line-item persistence; recompute totals |
| `internal/invoice/domain/aggregate/invoice_aggregate.go` | `LineItems []LineItem` on aggregate; `AddLineItem()` + recompute; rehydrate with lines |
| `internal/invoice/infrastructure/persistence/sql/invoice_repository.go` | `SaveLineItems`, `LoadLineItems` (plain SQL), Save/Find wiring |
| `internal/handlers/trips.go` | `CompleteTrip` (line 398): load closed detentions → pass `LineItems` → mark `attached` |
| `internal/handlers/app.go` | add `Geofences *GeofenceHandlers` field (pattern at line 44/81) |
| `internal/templates/layout.html` | sidebar nav link `/geofences` |
| `internal/templates/invoice_view.html` | render `invoice_line_items` table (line items exist only after 00042 + this spec) |
| `internal/vehicle/domain/aggregate/vehicle_aggregate.go` + converters + `internal/vehicle/application/create_vehicle.go`/`update_vehicle.go` + sqlc `db/query/vehicles.sql` | `TankCapacityLitres`, `FuelSensorFitted`, `MaintenanceDue` (regen sqlc here is fine — vehicles.sql already in queries; alternative: plain SQL in vehicle repo) |

**Untouched (explicit):** `internal/trip/domain/aggregate/trip_aggregate.go`, all `internal/trip/application/*.go` use cases (called, not modified), `internal/service/trip_service.go` + `bs.events` old bus, `internal/telemetry/sync.go` (ingestion stays), `internal/shared/outbox/*`, `internal/auth/casbin.go`.

---

## 11) Migration Plan

1. `goose up` applies `00042` (DDL + RBAC seeds). Casbin picks up `geofences:*` on next startup (or `authSvc.Reload()` after seeding in tests).
2. No sqlc regen for geofence/company_config/invoice_line_items — all new queries hand-written (`database/sql` + `repository.TxFromContext`).
3. Vehicle field additions: `db/query/vehicles.sql` + regen (existing generated file, safe) OR hand-written SQL in vehicle repo to avoid regen — pick one, keep consistent.
4. Rollback: `goose down` per DDL in §2.

---

## 12) Edge Cases

- **GPS jitter**: buffer 20m + hysteresis 25m + 60s debounce + `max_fix_age` 30s; entering/leaving pre-confirm reversions reset the debounce timer, never flip state.
- **Overlapping zones**: priority → radius/area → id; restricted/no_entry always evaluated first for bound vehicles.
- **Vehicle reassignment mid-dwell**: `engine_state.trip_id` vs snapshot `trip_id` mismatch → reset to `outside`, close open detentions (`closed`), no spurious transitions; assignment change carries no event today (aggregate `AssignVehicle` records none — verified), so snapshot-based detection is the only reliable signal.
- **Timezone-aware night window**: `company_config.geofence.timezone` default Asia/Kolkata; driver `users.timezone` (00029) override.
- **Deleted geofence with active dwell**: soft delete keeps `engine_state`; engine stops new entries, completes current dwell, emits `leaving` on confirmed exit, then clears binding.
- **Paid invoices**: never auto-adjusted (decision 1) — attach/waive admin endpoints only, `409` on paid, audit-logged.
- **No booking on trip**: detentions remain `closed`; admin attach later.
- **Route fallback miss**: pickup/drop zones with `route_name` that matches nothing are skipped for trip matching (never block transitions).
- **Stale fixes after trip end**: trip `completed`/`cancelled` → worker ignores further fixes for that trip_id, closes detentions.
- **Detention zero-config**: `detention_rate_per_hour = 0` → amounts 0, no line items until rate configured.
- **Relay payload is `map[string]any`** — consumers of `"TripCompletedEvent"` must read `payload["TripID"]` (string) — documented for the alerting spec.

---

## 13) Phased Rollout

- **Phase A — zones**: 00042, zone CRUD + Leaflet drawing UI + RBAC + company_config admin form. No engine.
- **Phase B — observation**: dwell worker + state machine + `geofence_events` + alerts (outbox `GeofenceAlertEvent`); `auto_reach_pickup`/`auto_start_transit` = false; detention rows created but not billed. Ship to staging, validate against real telemetry.
- **Phase C — trip transitions**: flip config flags; wire `ReachPickupUseCase`/`StartTransitUseCase`; add failure dashboards (transition rejections logged with aggregate guard errors).
- **Phase D — detention invoicing**: `invoice_line_items` + `GenerateInvoiceCommand.LineItems` + completion auto-attach + admin attach/waive; invoice view rendering. `detention_rate_per_hour` configured per company.

---

## 14) VERIFY Items

1. `grep -rn "geofence" internal/ --include="*.go"` → only new files (no prior implementation).
2. Latest migration is `00039_experiments.sql` (head); `00042` applies cleanly with `goose up` and rolls back.
3. Event names: relay dispatches struct names — verify worker/alerting consumers subscribe to `"TripStartedEvent"`, `"TripReachedPickupEvent"`, `"TripCompletedEvent"`, `"GeofenceAlertEvent"` exactly.
4. Trip transitions only via `internal/trip/application/reach_pickup.go` / `start_transit.go`; no other caller added; `deliver.go` callers unchanged.
5. `cmd/server/main.go:825` relay is the only bus the worker writes through (outbox_events); no import of `internal/service` baseService bus.
6. `GenerateInvoiceCommand` remains backward-compatible (LineItems nil → old behaviour; existing tests in `internal/invoice/` pass).
7. Casbin: after 00042, `Can(user, "geofences", "create")` true for admin/dispatcher, `read` for viewer.
8. Telemetry ingestion: `POST /api/v1/telemetry/snapshots` accepts payloads today (`telemetry/sync.go` `snapshotHandler`); worker reads `telemetry_snapshots` — confirm snapshot `vehicle_id` populated (nullable today; backfill by trip lookup).
9. `internal/static/js/leaflet.js` served at `/static/js/leaflet.js` (main.go:459 static file server); Google tile URL reachable without key.
10. `go build ./...` and `go test ./internal/geofence/... ./internal/invoice/...` green; geo functions unit-tested (ray-cast + haversine fixtures incl. degenerate polygons, boundary points, buffer/hysteresis thresholds).
11. Dwell state machine restart-resume: `engine_state` persisted; worker honors `last_fix_at` gap.
12. No sqlc regen needed except optionally `vehicles.sql`; `db/generated/sqlite` untouched otherwise.
