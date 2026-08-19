# LIVE MAP + SHARE LINKS + ETA + PREVENTIVE MAINTENANCE — Implementation Spec v2

**Project:** Avandab fleet system — `/home/abhishek/Desktop/temux/basic`
**Stack (verified):** Go 1.26, chi v5.3.1, SQLite (modernc), goose v3.27.3, Datastar v1.0.2 (vendored), Tailwind (vendored), Casbin RBAC, outbox→in-memory event bus.
**Status:** Implementation-ready (updated — map stack locked to the FlyFleet pattern; 00044 owns share/maintenance DDL).
**Audit date:** 2026-08-17

---

## 0. Verification Log (QA pass — 2026-08-19)

Migration renumber: share/maintenance `00043` → **`00044`**; `vehicles.maintenance_due` ref
`00041` → **`00042`** (geofence spec). Head `00039_experiments.sql`. All `main.go` route-group
line numbers corrected (public group ends ~714 `/contact-us`; protected group 729; API group
445-461; telemetry routes 448; Timeout 269; outbox 819-826; runDailyDigest 823). `dashboard.go`
is 99 lines, not 37.

### Verification Log table

| Claim | Verdict | Correction / Evidence (file:line) |
|---|---|---|
| No WebSocket hub / SSE exists | VERIFIED | grep `websocket\|Upgrader\|text/event-stream\|EventSource` → 0 in internal/ + cmd/ |
| `dashboard.go` is 37-line KPI renderer | WRONG | actual **99 lines** (`wc -l internal/handlers/dashboard.go`) |
| Route groups at 519/522, API 350-361, Timeout 255, outbox 609-616 | WRONG | `/contact-us` 714, protected 729, API 445-461, telemetry 448, Timeout 269, outbox 819-826, runDailyDigest 823 |
| `snapshotHandler` INSERT OR REPLACE @sync.go:72 | VERIFIED | `internal/telemetry/sync.go:72` |
| `assign_vehicle`/`assign_driver` compliance lines 66-88/66-82 | VERIFIED | `internal/trip/application/assign_vehicle.go:66`, `assign_driver.go:66` |
| `vehicles.maintenance_due` belongs to geofence 00041 | WRONG | corrected **00042** (geofence spec) |
| Share/maintenance owned by 00043 | WRONG | corrected **00044** |
| `files.uploadable_type` narrow CHECK | VERIFIED | `db/migrations/00001_initial.sql:67` |
| Leaflet/OSM/FlyFleet map refs | CANT-VERIFY | FlyFleet external repo not in this workspace; treat as design reference |

### Severity & Effort (major changes)

| Change | Severity | Effort |
|---|---|---|
| Migration renumber 00043 → 00044 | Low | S |
| SSE hub (`internal/realtime`) | High | M |
| Share links (PIN, sliding expiry, revoke) | High | M |
| Hybrid ETA calculator | Med | M |
| Preventive-maintenance worker + DTC | High | L |
| Maintenance dispatch blockers (2 paths + agent) | High | L |
| CSP on map pages | Low | S |

### Architectural Decisions (Decision / Tradeoff / Cost)

- **Real-time transport: SSE over WebSocket.** Decision: build SSE (`text/event-stream`) from
  scratch; no new deps. Tradeoff: one-directional (server→client) only — fine for live map.
  Cost: none; `EventSource` auto-reconnects. Multi-instance falls back to REST polling
  (`/api/v1/telemetry/live` is source of truth).
- **Single-process in-memory hub.** Decision: per-process fan-out, 64-buf drop + reconnect.
  Tradeoff: not shared across instances; acceptable for single-binary deploy. Cost: documented.
- **Share links bound to TRIP, token SHA-256, optional PIN, sliding expiry capped by
  `SHARE_LINK_MAX_TTL_HOURS`.** Decision: DB-leak safe, enumeration-resistant (uniform 404/410).
  Tradeoff: `/data` reads do not extend TTL (anti-thrash). Cost: crypto/rand + rate-limit.
- **Hybrid ETA (0.7 telemetry + 0.3 scheduled) + monotonic guard.** Decision: tolerate stale
  telemetry; guard prevents ETA jumping backward. Tradeoff: ±15 min window only (never exact).
  Cost: `audit_logs` writes on guard clamps.
- **Maintenance blocker in both assignment paths + agent.** Decision: non-fatal first release,
  then hard block; admin override column. Tradeoff: safer rollout. Cost: duplicated gate logic.
- **Cross-ref spec 17 (GPS provider strategy):** the live map consumes telemetry via the bus
  contract from ingestion spec 01; the GPS data SOURCE strategy is owned by `17-gps-telematics-provider-strategy.md`.

## 1. Architecture

### 1.1 Data flow

```
GPS device / mobile app
   │  POST /api/v1/telemetry/snapshots        (existing: internal/telemetry/sync.go snapshotHandler)
   ▼
telemetry_snapshots (00031) ──INSERT──▶ publish bus event "telemetry.snapshot"
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    ▼                       ▼                       ▼
        GET /api/v1/telemetry/live   SSE hub (new)            PM worker (new)
        (REST poll, JSON)           internal/realtime/        internal/maintenance/
                    │                       │                       │
                    ▼                       ▼                       ▼
              GET /tracking page      /api/v1/telemetry/stream   vehicles.maintenance_due
              (Leaflet + poll)       (EventSource, protected)   dtc_events, notifications
                    │                       │
                    └───────┬───────────────┘
                            ▼
                 GET /share/{token} (public, no auth, poll-only)
                 GET /share/{token}/data (JSON poll, rate-limited)
```

- **No WebSocket hub exists** (verified: `internal/handlers/dashboard.go` is a 99-line KPI renderer (not 37); grep for `websocket|Upgrader|text/event-stream|EventSource` across `internal/` and `cmd/` returns zero matches). Real-time push is designed from scratch as **SSE**: zero new dependencies, `EventSource` auto-reconnects, works through Datastar pages.
- **Single-process hub.** `internal/realtime` hub is in-memory, per-process. Fits the current single-binary deployment. Multi-instance deployments fall back to REST polling (the `/live` endpoint is the source of truth; SSE is an accelerator). Documented in edge cases (11.4). **Telemetry data source:** the hub consumes the in-memory `PositionEvent` published via the Dual-Write Fast-Path (Spec 01); REST `/api/v1/telemetry/live` remains the source of truth for recovery and multi-instance.
- **Ingestion hook:** the existing `snapshotHandler` (`internal/telemetry/sync.go:52-80`) publishes `events.Event{Type: "telemetry.snapshot", Payload: snapshot}` to the bus after a successful `INSERT`. When the ingestion spec (migration 00041) replaces this handler, it emits the same event type — hub API unchanged — and additionally publishes the in-memory `PositionEvent` via the Dual-Write Fast-Path (Spec 01); the hub consumes both.

### 1.2 SSE hub design (`internal/realtime/`)

New package, three files:

- `hub.go` — `Hub` struct:
  - `subs map[chan []byte]filter` guarded by `sync.RWMutex`; `filter func(e events.Event) bool` (default: accept all; tracking page passes `trip_id`/`vehicle_id` filters).
  - `Subscribe(ctx, filter) (<-chan []byte, unsubscribe func())` — buffered channel cap 64; slow consumers dropped (channel full → close, client reconnects).
  - `Publish(ctx, e events.Event)` — non-blocking fan-out to matching subscribers (called from the bus handler goroutine; never block ingestion).
  - `Run(ctx)` — heartbeat goroutine: writes `: keepalive\n\n` every `SSE_KEEPALIVE_SEC` (default 15) to all subscribers.
- `bus.go` — `AttachToBus(bus events.EventBus, h *Hub)`: subscribes to `PositionEvent` (published via the Dual-Write Fast-Path, Spec 01), `"trip.status_changed"`, `"maintenance.due"`, `"maintenance.cleared"` and forwards to the hub.
- `handler.go` — `StreamHandler(h *Hub)`:
  - Sets `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`.
  - Flushes immediately; writes `event: telemetry\ndata: {json}\n\n` per snapshot; exits on `r.Context().Done()` (client disconnect **or** the global 60 s `chiMiddleware.Timeout` — see below).
  - Optional `?trip_id=` / `?vehicle_id=` query filter.

**Position source (Dual-Write Fast-Path, Spec 01):** the hub's realtime telemetry feed is the
in-memory `PositionEvent` published directly by the ingestion pipeline (sub-second latency, not
the ~5 s outbox relay). REST `GET /api/v1/telemetry/live` stays the source of truth: on
reconnect and in multi-instance deploys, clients re-poll `/live` to catch events the local
process never saw.

**Timeout caveat (must implement):** `cmd/server/main.go:269` applies `chiMiddleware.Timeout(60s)` globally; it would kill SSE connections at 60 s. Fix: wrap with a path-skip middleware in `internal/middleware/middleware.go`:

```go
// SkipForPaths excludes paths from a middleware (SSE streams must outlive the 60s request timeout).
func SkipForPaths(m func(http.Handler) http.Handler, paths ...string) func(http.Handler) http.Handler
```

Applied in main.go: `r.Use(middleware.SkipForPaths(chiMiddleware.Timeout(60*time.Second), "/api/v1/telemetry/stream"))`. Alternatively rely on EventSource auto-reconnect, but the skip is trivial and cleaner.

### 1.3 Pages

- **`GET /tracking`** (protected web group, `cmd/server/main.go:729+`): server-rendered Datastar page (`internal/templates/tracking.html`). Leaflet map + markercluster; marker states running / stopped / no_signal / maintenance_due; popup: vehicle no., speed, fuel %, odometer, ETA window, trip number. Data via REST poll `GET /api/v1/telemetry/live` every `MAP_POLL_SEC` (default 10); SSE stream optional enhancement behind a toggle.
- **`GET /share/{token}`** (public, registered BEFORE the protected group at `main.go:729`, i.e. in the public block after `/contact-us` at `main.go:714`): no auth, rate-limited. Leaflet map, single vehicle marker, ETA window ±15 min, status. Polls `GET /share/{token}/data` every 30 s. No SSE on the public surface.
- **`GET /maintenance`** (protected): schedules, due list, DTC list, record entry forms (Datastar fragments).

### 1.4 RBAC

New resources `shares` and `maintenance` seeded in 00044 (permissions table + `role_permissions` for roles 1 admin, 2 dispatcher — **must re-seed explicitly**, the blanket `SELECT 1, id FROM permissions` in 00012 runs once at that migration, not on every insert).

---

## 2. Map stack config

**Stack locked** — mirrors FlyFleet (verified paths, with corrections where files moved):

| Purpose | FlyFleet reference (verified) | Avandab implementation |
|---|---|---|
| Base Google raster tiles (keyless) | `apps/dashboard-web/src/modules/dashboard/components/LiveOperationsMap.tsx:290` — `https://mt1.google.com/vt/lyrs=m&x={x}&y={y}&z={z}` | Leaflet `L.tileLayer`, same URL |
| Layer variants roadmap/satellite/hybrid/terrain = `lyrs` m/s/y/p | `apps/dashboard-web/src/pages/admin/DispatchCenter.tsx:24-27` | Layer control switching URL per style |
| India bias `&gl=IN` | `apps/dashboard-web/src/features/tracking/components/VehicleMap.tsx:26` (Maplibre GL with `mt1.google.com/vt/lyrs=m...&gl=IN`) | Append `&gl={MAP_GL}` (default `IN`) |
| OSM fallback tiles | `apps/dashboard-web/src/components/trips/RouteMap.tsx:143`, `.../LocationPickerMap.tsx:194` — `https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png` (note: these two files live in `src/components/trips/`, not `src/features/tracking/components/`) | Fallback `L.tileLayer`; switch via `tileerror` event per-provider (`MAP_TILE_PROVIDER=auto`) |
| Nominatim geocoding | `apps/dashboard-web/src/components/trips/AddressInput.tsx:104` (reverse), `:154` (forward search, `countrycodes=in`) | `fetch` from `NOMINATIM_URL`; used by share page address label + tracking search box |
| Marker clustering | `apps/dashboard-web/src/components/LiveMap.tsx` imports `leaflet.markercluster/dist/MarkerCluster.css` | `leaflet.markercluster` vendored (avandab is server-rendered Datastar, not React — vanilla Leaflet JS, not react-leaflet) |

**Honest caveat (include in docs):** keyless Google raster tiles (`mt1.google.com/vt/...`) are not an officially documented API and violate the Google Maps Platform ToS — grey area. OSM fallback mitigates availability/ToS risk. Do not rely on Google tiles in any SLA; `MAP_TILE_PROVIDER=osm` is the compliant mode.

**Vendored assets** (no CDN — matches existing pattern `internal/static/js/datastar.js`, `internal/static/css/tailwind.css`):

- `internal/static/js/leaflet/leaflet.js` + `leaflet.css` (Leaflet 1.9.x)
- `internal/static/js/leaflet/leaflet.markercluster.js` + `MarkerCluster.css` + `MarkerCluster.Default.css`

Loaded only on map pages via a template block in `internal/templates/layout.html` (`{{if .Extra.MapAssets}}`).

**CSP** — currently NO CSP exists (`middleware.SecurityHeaders`, `internal/middleware/middleware.go:245-253` sets only nosniff/frame/xss/referrer). Add `middleware.ContentSecurityPolicy` (config-gated `CSP_ENABLED`, default false; opt-in per route to avoid breaking existing pages with inline handlers):

```
default-src 'self';
script-src 'self' 'unsafe-inline';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https://mt1.google.com https://tile.openstreetmap.org https://a.tile.openstreetmap.org https://b.tile.openstreetmap.org https://c.tile.openstreetmap.org;
connect-src 'self' https://nominatim.openstreetmap.org https://mt1.google.com;
font-src 'self';
frame-ancestors 'none'
```

Apply on `/tracking`, `/share/*`, `/maintenance` routes only.

**Env:**

| Var | Default | Meaning |
|---|---|---|
| `MAP_TILE_PROVIDER` | `auto` | `google` \| `osm` \| `auto` (google → OSM on tileerror) |
| `MAP_GOOGLE_STYLE` | `m` | m=roadmap, s=satellite, y=hybrid, p=terrain |
| `MAP_GL` | `IN` | Google tile country bias |
| `MAP_OSM_URL` | `https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png` | OSM fallback template |
| `NOMINATIM_URL` | `https://nominatim.openstreetmap.org` | Geocoding base |
| `MAP_POLL_SEC` | `10` | Tracking page REST poll interval |
| `CSP_ENABLED` | `false` | Emit CSP header on map/share/maintenance pages |

---

## 3. DDL — `db/migrations/00044_share_links_and_maintenance.sql`

Ownership: **00044 = share_links + maintenance_schedules + maintenance_records + dtc_events + company_settings maintenance defaults + admin-override columns + RBAC seeds**. `vehicles.maintenance_due` belongs to **00042 (geofence spec — reference only)**; this migration never adds it. Goose applies numerically, so 00042 always precedes 00044.

```sql
-- +goose Up

-- ── Share links (bind to TRIP, not vehicle) ──────────────────────────
CREATE TABLE share_links (
    id                 TEXT PRIMARY KEY,
    trip_id            TEXT NOT NULL,
    token_hash         TEXT NOT NULL UNIQUE,          -- SHA-256 of raw token
    pin_hash           TEXT,                          -- NULL = no PIN; SHA-256(pin||salt)
    pin_salt           TEXT,
    created_by         TEXT NOT NULL,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at         DATETIME NOT NULL,             -- sliding: refreshed on valid view, capped at max_ttl
    last_viewed_at     DATETIME,
    view_count         INTEGER NOT NULL DEFAULT 0,
    failed_pin_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until       DATETIME,                      -- 15 min lock after 5 failed PIN attempts
    revoked_at         DATETIME,                      -- NULL = active
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);
CREATE INDEX idx_share_links_trip   ON share_links(trip_id);
CREATE INDEX idx_share_links_expires ON share_links(expires_at);

-- ── Preventive maintenance ───────────────────────────────────────────
CREATE TABLE maintenance_schedules (
    id            TEXT PRIMARY KEY,
    vehicle_id    TEXT NOT NULL,
    service_type  TEXT NOT NULL CHECK (service_type IN
                  ('oil_change','brake','tyre','battery','engine','fitness','insurance','permit','general')),
    interval_km   REAL,                               -- due every N km
    interval_days INTEGER,                            -- or every N days
    last_done_km  REAL,
    last_done_at  DATETIME,
    due_km        REAL,                               -- absolute odometer threshold
    due_at        DATETIME,                           -- absolute date threshold
    active        INTEGER NOT NULL DEFAULT 1,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);
CREATE INDEX idx_maint_sched_vehicle ON maintenance_schedules(vehicle_id, active);

CREATE TABLE maintenance_records (
    id            TEXT PRIMARY KEY,
    vehicle_id    TEXT NOT NULL,
    schedule_id   TEXT,
    service_type  TEXT NOT NULL,
    performed_at  DATETIME NOT NULL,
    odometer_km   REAL,
    cost          REAL,
    vendor        TEXT,
    notes         TEXT,
    recorded_by   TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (schedule_id) REFERENCES maintenance_schedules(id)
);
CREATE INDEX idx_maint_rec_vehicle ON maintenance_records(vehicle_id, performed_at);

CREATE TABLE dtc_events (
    id          TEXT PRIMARY KEY,
    vehicle_id  TEXT NOT NULL,
    trip_id     TEXT,
    dtc_code    TEXT NOT NULL,
    severity    TEXT NOT NULL CHECK (severity IN ('info','warning','critical')),
    description TEXT,
    raw_payload TEXT,                                 -- JSON as received
    occurred_at DATETIME NOT NULL,
    resolved_at DATETIME,                             -- set by maintenance record or admin
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (trip_id) REFERENCES trips(id)
);
CREATE INDEX idx_dtc_vehicle ON dtc_events(vehicle_id, occurred_at);
-- DTC storm dedupe: one row per (vehicle, code, 1-minute bucket)
CREATE UNIQUE INDEX idx_dtc_dedupe ON dtc_events(vehicle_id, dtc_code, occurred_at);

-- ── Company defaults ─────────────────────────────────────────────────
ALTER TABLE company_settings ADD COLUMN maintenance_default_interval_km REAL;
ALTER TABLE company_settings ADD COLUMN maintenance_default_interval_days INTEGER;
ALTER TABLE company_settings ADD COLUMN maintenance_critical_dtcs TEXT;  -- comma-separated codes

-- ── Admin override (flag itself lives in 00042) ──────────────────────
ALTER TABLE vehicles ADD COLUMN maintenance_override_by TEXT;
ALTER TABLE vehicles ADD COLUMN maintenance_override_at DATETIME;
ALTER TABLE vehicles ADD COLUMN maintenance_override_reason TEXT;

-- ── RBAC seeds (re-seed explicitly; 00012's blanket insert does not re-run) ──
INSERT OR IGNORE INTO permissions (name, description) VALUES
('shares:create',  'Create trip share links'),
('shares:read',    'View share links'),
('shares:revoke',  'Revoke share links'),
('maintenance:read',   'Read maintenance data'),
('maintenance:create', 'Record maintenance work'),
('maintenance:update', 'Update schedules / override blocks');

INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions
WHERE name IN ('shares:create','shares:read','shares:revoke',
               'maintenance:read','maintenance:create','maintenance:update');

INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions
WHERE name IN ('shares:create','shares:read','shares:revoke','maintenance:read');

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN
  (SELECT id FROM permissions WHERE name IN ('shares:create','shares:read','shares:revoke',
   'maintenance:read','maintenance:create','maintenance:update'));
DELETE FROM permissions WHERE name IN ('shares:create','shares:read','shares:revoke',
   'maintenance:read','maintenance:create','maintenance:update');
DROP TABLE IF EXISTS dtc_events;
DROP TABLE IF EXISTS maintenance_records;
DROP TABLE IF EXISTS maintenance_schedules;
DROP TABLE IF EXISTS share_links;
ALTER TABLE company_settings DROP COLUMN maintenance_default_interval_km;
ALTER TABLE company_settings DROP COLUMN maintenance_default_interval_days;
ALTER TABLE company_settings DROP COLUMN maintenance_critical_dtcs;
ALTER TABLE vehicles DROP COLUMN maintenance_override_by;
ALTER TABLE vehicles DROP COLUMN maintenance_override_at;
ALTER TABLE vehicles DROP COLUMN maintenance_override_reason;
```

Note: `files.uploadable_type` CHECK (`00001_initial.sql:67`) is narrow (`driver_license|vehicle_insurance|vehicle_permit|company_logo`) — maintenance attachments are **out of scope** for v1 (records store vendor/notes as text). Extending the CHECK requires a SQLite table rebuild; deferred to the compliance spec (00046).

---

## 4. Share-link flow

### 4.1 Creation (protected)

- `POST /trips/{id}/share` (`internal/handlers/trips.go` `Routes`, mounted with `middleware.ResourcePermission(h.AuthSrv, "shares", "create")`). Request: `{pin?: "1234", ttl_hours?: 24}` (ttl clamped to `SHARE_LINK_MAX_TTL_HOURS`).
- **Token:** `crypto/rand` 32 bytes → `base64.RawURLEncoding` (43 chars). Store only `SHA-256(token)` in `share_links.token_hash` (DB-leak safe). Return raw token once in the response + Datastar fragment showing `{BASE_URL}/share/{token}` + copy button.
- **PIN (optional):** 4–6 digits; store `SHA-256(pin || salt)` with per-link random 16-byte salt.
- **Expiry:** `expires_at = now + ttl_hours`; hard cap `created_at + SHARE_LINK_MAX_TTL_HOURS`.

### 4.2 Viewing (public, no auth)

- `GET /share/{token}` — `middleware.RateLimit(20)` per IP. 404/410 for unknown/expired/revoked tokens (do not distinguish — enumeration resistance). If `pin_hash` set and no valid PIN cookie → render PIN form. Else render `share_public.html` (Leaflet + marker + ETA window + status).
- `POST /share/{token}/verify` — `middleware.RateLimit(10)`. Checks `failed_pin_attempts`/`locked_until` (5 fails → lock 15 min). Success: set signed httpOnly cookie via existing `authStore` secret (`share_pin_{token}`), refresh `expires_at`. Failure: increment counter, 429-ish 403 with retry-after.
- `GET /share/{token}/data` — `middleware.RateLimit(30)`. JSON: `{trip_number, vehicle_label, status, lat, lng, speed, fuel_level, odometer, last_seen, eta_min, eta_max, eta_method, maintenance_due}`. **ETA only when trip in `started|reached_pickup|in_transit|delivered` and telemetry fresh (≤15 min)**; otherwise `eta_method: "scheduled"` with trip `arrival_time`, or omitted entirely for `draft|scheduled` trips.
- **Sliding expiry:** only page views + PIN verify refresh `expires_at = min(now + ttl_hours, created_at + SHARE_LINK_MAX_TTL_HOURS)` and bump `view_count`, `last_viewed_at`. **`/data` reads do NOT extend** (prevents indefinite keep-alive by pollers).
- **No-cache:** share pages use `middleware.NoCache` (existing, `middleware.go:209`) — prevents stale ETA caching.

### 4.3 Revoke / list (protected)

- `GET /shares` — list active/expired/revoked links for tenant (`shares:read`).
- `POST /shares/{id}/revoke` — sets `revoked_at` (`shares:revoke`); subsequent views → 410. Revoke also kills the signed PIN cookie path (page re-checks `revoked_at` first).

### 4.4 Route registration order (explicit requirement)

`/share/*` public routes are registered in the **public web group** (`cmd/server/main.go` public web group ends ~line 714, after `/contact-us`) — i.e. **before** the `RequireAuth` protected group at line 729. They must never appear inside the protected group. `GET /tracking`, `/shares`, `/maintenance` go inside the protected group. `GET /api/v1/telemetry/live` + `/stream` go inside the protected API group (`main.go:445-461`), registered via the existing `telemetry.RegisterTelemetryRoutes(r, database)` call at line 448.

---

## 5. ETA algorithm spec (`internal/eta/`)

### 5.1 Inputs

- Route: `distance`, `estimated_hours`, `reverse_distance` (from `routes`; `internal/domain/route/entity.go` `GetDistanceAndFare`).
- Trip: `status`, `started_at`, `departure_time`, `arrival_time` (`trips` table per 00007/00026).
- Telemetry: `telemetry_snapshots` rows for trip/vehicle — `timestamp, speed, odometer` (only fields available today; no heading/accuracy — enrichment is ingestion spec 00041, reference only).

### 5.2 Steps

1. **Freshness gate.** Latest snapshot `ts`. If `now − ts > ETA_STALE_MIN` (15) → skip telemetry path; return **scheduled fallback**.
2. **Rolling average speed.** `v_avg = mean(speed)` over snapshots in `now − ETA_WINDOW_MIN` (30) with `speed > 0`, minimum 3 samples. Fewer samples → scheduled fallback.
3. **Remaining distance** (odometer-delta preferred):
   - Odometer available: `remaining = max(0, route_dist − (odom_now − odom_start))`, where `odom_start` = first snapshot odometer after `started_at` (or `vehicles.odometer` at start).
   - No odometer: `remaining = route_dist × (1 − min(1, elapsed_since_start / estimated_hours))` (time-proportional).
4. **Components:**
   - `eta_telemetry = remaining / v_avg`
   - `eta_scheduled = estimated_hours × (remaining / route_dist)` (already time-proportional when odometer missing — acceptable)
5. **Hybrid:** `eta_hybrid = 0.7 × eta_telemetry + 0.3 × eta_scheduled` when both valid; else whichever is valid.
6. **Arrival:** `arrival_at = now + eta_hybrid`. Reported to callers as `eta_min = arrival_at − 15min`, `eta_max = arrival_at + 15min` (**±15 min window** — share pages never promise exactness).
7. **Monotonic guard:** per-trip in-memory `last_arrival_at`. New value must not move **backwards more than 5 min** vs previous: `arrival_at = min(arrival_at_new, last_arrival_at − 5min)` (ETA may improve, but slowly). On clamp → write `audit_logs` row (`action='eta_guard'`, table_name='trips', record_id=trip_id, new_values={clamped_from, clamped_to, reason}).
8. **Audit:** `audit_logs` rows only for guard clamps and staleness fallback switches (`action='eta_fallback'`) — per-calculation logging is too noisy. No new table.

### 5.3 Service shape

`internal/eta/service.go` — `EtaService{db *sql.DB}` with `Calculate(ctx, tripID) (EtaResult, error)`. Pure read path (no UoW needed); used by live endpoint, share data endpoint, vehicle popups. Unit tests with synthetic snapshot series (`internal/eta/service_test.go`).

---

## 6. Preventive-maintenance worker (`internal/maintenance/`)

### 6.1 Worker loop

`internal/maintenance/worker.go` — `Worker{db, bus, logger}`, started from `cmd/server/main.go` next to `runDailyDigest` (line 823 pattern): `go maintenanceWorker.Run(ctx)`. Ticker `PM_CHECK_INTERVAL_MIN` (default 15), gated by `PM_ENABLED` (default `true`; if `vehicles.maintenance_due` column absent — 00042 not yet applied — log warn and skip flag updates).

Per tick, per active schedule (`maintenance_schedules.active=1`):

- **Odometer due:** latest odometer (max `odometer` in `telemetry_snapshots` for the vehicle, else `vehicles.odometer`) `>= due_km` (or `last_done_km + interval_km`).
- **Date due:** `now >= due_at` (or `last_done_at + interval_days`).
- On due → `UPDATE vehicles SET maintenance_due = 1 WHERE id = ?` (column from 00042), insert `notifications` row (table exists, 00025), publish `events.Event{Type: "maintenance.due"}` (feeds SSE + founder alerts).

### 6.2 DTC intake

- Subscribe to bus event **`"alert.dtc"`** (payload: `{vehicle_id, trip_id, dtc_code, severity, description, occurred_at}`) — emitted by the ingestion pipeline (00041 reference) and, defensively, by a startup catch-up scan of `telemetry_alerts` (00030) rows matching a `dtc_` alert_type. The existing founder `AlertEvent` (`internal/founder/alerts/event.go:28`, `Metadata map[string]interface{}`) is the transport when ingestion routes through founder — worker also subscribes to `"AlertEvent"` and extracts `Metadata["dtc_codes"]` if present.
- Insert into `dtc_events` (dedup via unique index `(vehicle_id, dtc_code, occurred_at)`); if `severity == 'critical'` or code ∈ `company_settings.maintenance_critical_dtcs` → `maintenance_due = 1` + publish `maintenance.due`.
- **Resolution:** recording a `maintenance_record` for the vehicle triggers recompute: if no active schedule due and no unresolved critical DTC → `maintenance_due = 0`, publish `maintenance.cleared`.

### 6.3 Dispatch blockers — exact files (both assignment paths, verified)

Block condition: `vehicles.maintenance_due = 1` (or unresolved critical DTC) **and** no admin override. Error message: `vehicle X is blocked for maintenance (due: <service_type>, since <date>); override requires maintenance:update`.

**Path A — web UI + REST API use cases (shared):**

1. `internal/trip/application/assign_vehicle.go` — `checkVehicleCompliance` (lines 66-88): add maintenance check after the existing status/expiry checks. Needs `v.MaintenanceDue` on the vehicle aggregate (field added when 00042 lands — `internal/vehicle/domain/aggregate/vehicle_aggregate.go`) plus unresolved-critical-DTC lookup via new `MaintenanceRepository` (see below). Override: new `OverrideMaintenance bool` + `OverrideReason string` on `AssignVehicleCommand`; when set, `logAudit(txCtx, "assign_vehicle_override", ...)`.
2. `internal/trip/application/assign_driver.go` — `checkDriverCompliance` (lines 66-82): add same maintenance check **when the trip already carries a vehicle** (reassignment case; `t.VehicleID` from `repo.Find`). Driver-assign stays blocked for a blocked vehicle; final authority remains the vehicle-assign step. Same override fields on `AssignDriverCommand`.
3. Handlers wiring: `internal/handlers/trips.go:72-73` (web) and `internal/trip/presentation/api/handlers/trip_handler.go:74` (REST) pass `override` form/JSON field through to the commands. RBAC: override accepted only when actor holds `maintenance:update` (checked via `middleware.ResourcePermission` on a distinct route variant or in-use-case via `authSrv` — use-case has no actor; enforce in handlers).

**Path B — service layer (agent tools, tests):**

4. `internal/service/trip_service.go` — `AssignVehicle` (line ~225) and `AssignDriver` (line 163): after existing `CanAcceptTrip`/conflict checks, call `s.store.IsMaintenanceBlocked(ctx, vehicleID)` (new `SQLRepository` method, `internal/repository/sqlite/vehicles.go`) and return a typed error.
5. `internal/agent/tools.go` — `list_available_vehicles` (line ~389): filter out `maintenance_due` vehicles (add to the query); `assign_vehicle`/`assign_driver` (lines 416-452) surface the service-layer error verbatim. `GetAvailableVehicles` query: `db/query/vehicles.sql:52` — add `AND maintenance_due = 0` (guarded: column from 00042; query regenerated after both migrations exist).

**Override UX:** vehicle page (`vehicle_edit.html`/`vehicle_view.html`) — admin sets `maintenance_override_by/at/reason` (00044 columns) with audit; override only lifts the *block*, flag stays set until work recorded.

### 6.4 Repository wiring

- `internal/shared/ports/uow.go` — add `Maintenance() any` to `RepositoryProvider`; implement in `internal/shared/uow/uow.go` `repositoryProvider` (new `internal/maintenance/infrastructure/sql/maintenance_repository.go`).
- sqlc: `db/query/maintenance.sql` (schedules/records/dtc CRUD + `IsMaintenanceBlocked`), `db/query/share_links.sql`; regenerate with `sqlc generate` (sqlc.yaml exists; output `db/generated/sqlite`).

---

## 7. API routes (all under existing auth groups)

### Public (before protected group, `main.go: ~714` block)

| Method | Path | Handler | Auth / limits |
|---|---|---|---|
| GET | `/share/{token}` | `ShareHandlers.View` | none; `RateLimit(20)` |
| POST | `/share/{token}/verify` | `ShareHandlers.VerifyPIN` | none; `RateLimit(10)` |
| GET | `/share/{token}/data` | `ShareHandlers.Data` | none; `RateLimit(30)`; `NoCache` |

### Protected web group (`main.go:729+`)

| Method | Path | Handler | RBAC |
|---|---|---|---|
| GET | `/tracking` | `TrackingHandlers.Page` | `tracking` page (any authenticated) |
| GET | `/shares` | `ShareHandlers.List` | `shares:read` |
| POST | `/trips/{id}/share` | `TripHandlers.CreateShare` | `shares:create` |
| POST | `/shares/{id}/revoke` | `ShareHandlers.Revoke` | `shares:revoke` |
| GET | `/maintenance` | `MaintenanceHandlers.Index` | `maintenance:read` |
| GET/POST | `/maintenance/schedules` | `MaintenanceHandlers.Schedules` | read / `maintenance:create` |
| POST | `/maintenance/records` | `MaintenanceHandlers.Record` | `maintenance:create` |
| GET | `/maintenance/dtc` | `MaintenanceHandlers.DtcList` | `maintenance:read` |

### Protected API group (`main.go:445-461`, via `telemetry.RegisterTelemetryRoutes` at :448)

| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/api/v1/telemetry/live` | `telemetry.LiveHandler` | JSON: `[{trip_id, vehicle_id, lat, lng, speed, fuel_level, odometer, status: running\|stopped\|no_signal\|maintenance_due, eta_min, eta_max, ts}]`; optional `?trip_id=` |
| GET | `/api/v1/telemetry/stream` | `telemetry.StreamHandler` (SSE) | `RequireAPIAuth`; skipped from 60 s Timeout; `?trip_id=` filter |

Marker state rules: `maintenance_due` overrides others when flag set; else `no_signal` if `now − ts > TELEMETRY_STALE_MIN` (15); else `running` if `speed > 0` (or `stopped`).

---

## 8. Go file list

### Create

| File | Purpose |
|---|---|
| `internal/realtime/hub.go` | SSE hub (subscribe/publish/heartbeat) |
| `internal/realtime/hub_test.go` | Fan-out, slow-consumer drop, filter tests |
| `internal/realtime/bus.go` | `AttachToBus` — forwards `PositionEvent` (Dual-Write Fast-Path) / `trip.status_changed` / `maintenance.due` / `maintenance.cleared` to hub |
| `internal/realtime/handler.go` | `StreamHandler` (SSE) |
| `internal/telemetry/live.go` | `LiveHandler` (REST JSON) + snapshot→bus publish helper |
| `internal/eta/service.go` | Hybrid ETA calculator (5.2) |
| `internal/eta/service_test.go` | Synthetic snapshot series tests |
| `internal/maintenance/worker.go` | PM loop + DTC intake + flag recompute |
| `internal/maintenance/domain/` | Schedule/Record/DtcEvent entities + repository interface |
| `internal/maintenance/infrastructure/sql/maintenance_repository.go` | sqlite impl (sqlc-backed) |
| `internal/handlers/tracking.go` | `/tracking` page handler |
| `internal/handlers/share.go` | ShareHandlers (View/VerifyPIN/Data/List/Revoke) |
| `internal/handlers/maintenance.go` | MaintenanceHandlers (Index/Schedules/Record/DtcList) |
| `internal/templates/tracking.html` | Leaflet map page (Datastar poll) |
| `internal/templates/share_public.html` | Public share page (+ `_pin_form.html` fragment) |
| `internal/templates/maintenance_index.html` | Schedules/due/DTC list + record forms |
| `db/migrations/00044_share_links_and_maintenance.sql` | Section 3 DDL |
| `db/query/share_links.sql`, `db/query/maintenance.sql` | sqlc queries |

### Modify

| File | Change |
|---|---|
| `cmd/server/main.go` | Public `/share/*` routes before line 729; `/tracking`, `/shares`, `/maintenance` in protected group (:729+); `telemetry.RegisterTelemetryRoutes` gains live/stream (existing call :448); hub init + `AttachToBus` + `go hub.Run(ctx)` near outbox relay (:819-826); `go maintenanceWorker.Run(ctx)`; `SkipForPaths` wrapper on `chiMiddleware.Timeout` (:269); new handler groups on `app` |
| `internal/middleware/middleware.go` | `SkipForPaths` + `ContentSecurityPolicy` (section 2) |
| `internal/config/config.go` | Section 9 vars |
| `internal/telemetry/sync.go` | `snapshotHandler` publishes `telemetry.snapshot` to bus after INSERT (bus passed in via `RegisterTelemetryRoutes`) |
| `internal/handlers/app.go` | New `Tracking`, `Share`, `Maintenance` handler groups + `NewApp` wiring |
| `internal/handlers/trips.go` | `CreateShare` handler + route (:63-80), override pass-through on AssignDriver/AssignVehicle (:72-73, :308) |
| `internal/trip/application/assign_vehicle.go` | Maintenance block + override (6.3) |
| `internal/trip/application/assign_driver.go` | Maintenance block when trip has vehicle + override |
| `internal/trip/presentation/api/handlers/trip_handler.go` | Override field on AssignDriver/AssignVehicle (:157-166) |
| `internal/service/trip_service.go` | `IsMaintenanceBlocked` in AssignDriver/AssignVehicle (service layer) |
| `internal/repository/sqlite/vehicles.go` | `IsMaintenanceBlocked`, `SetMaintenanceDue`, override accessors |
| `internal/repository/sqlite/trips.go` | Share-link / ETA helper reads (or use sqlc directly) |
| `internal/shared/ports/uow.go` + `internal/shared/uow/uow.go` | `Maintenance() any` in provider |
| `internal/agent/tools.go` | `list_available_vehicles` filter (:389-412); errors surface (:416-452) |
| `internal/vehicle/domain/aggregate/vehicle_aggregate.go` | `MaintenanceDue bool` + override fields (populated after 00042 lands) |
| `internal/templates/layout.html` | Conditional Leaflet asset block |
| `internal/templates/trip_view.html` | "Share link" button + copy widget (near :20-28 assignment forms) |
| `internal/templates/vehicle_view.html` | Maintenance due badge + override form |
| `internal/static/js/leaflet/*`, `internal/static/css/leaflet/*` | Vendored Leaflet + markercluster (new files under existing static dirs) |

---

## 9. Config / env (add to `internal/config/config.go`)

| Var | Default | Used by |
|---|---|---|
| `MAP_TILE_PROVIDER` | `auto` | tracking/share templates |
| `MAP_GOOGLE_STYLE` | `m` | template |
| `MAP_GL` | `IN` | template |
| `MAP_OSM_URL` | `https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png` | template |
| `NOMINATIM_URL` | `https://nominatim.openstreetmap.org` | geocode endpoint |
| `MAP_POLL_SEC` | `10` | tracking page |
| `CSP_ENABLED` | `false` | middleware |
| `SHARE_LINK_TTL_HOURS` | `24` | share creation |
| `SHARE_LINK_MAX_TTL_HOURS` | `168` | hard cap |
| `SHARE_LINK_MAX_ACTIVE` | `20` | per-trip cap (create fails with 409) |
| `ETA_STALE_MIN` | `15` | freshness gate |
| `ETA_WINDOW_MIN` | `30` | rolling speed window |
| `ETA_GUARD_MAX_REGRESS_MIN` | `5` | monotonic clamp |
| `SSE_KEEPALIVE_SEC` | `15` | hub heartbeat |
| `TELEMETRY_STALE_MIN` | `15` | marker no_signal state |
| `PM_ENABLED` | `true` | worker gate |
| `PM_CHECK_INTERVAL_MIN` | `15` | worker tick |
| `PM_CRITICAL_DTCS` | `P0A0F,P1602` | env fallback when company setting empty |

---

## 10. Migration plan

1. **00042 (geofence spec, reference only)** adds `vehicles.maintenance_due`. Goose ordering guarantees it precedes 00044. Until it lands, worker skips flag writes (`PM_ENABLED` tolerant), and vehicle aggregate simply lacks the field.
2. **00044 (this spec)** — section 3 DDL. Applied automatically at boot (`main.go:150` `provider.Up`); no manual step.
3. **sqlc regenerate** after 00044 + 00042: `sqlc generate` → new `db/generated/sqlite/{share_links,maintenance}.sql.go` + modified `vehicles.sql.go`. Do this only after both migrations exist on disk (sqlc reads schema from `db/migrations`).
4. **Vendored assets** — Leaflet + markercluster committed under `internal/static/` (no CDN).
5. **RBAC** — seeded inside 00044 (roles 1, 2). Existing sessions unaffected; Casbin reloads policies from DB per request (`auth.NewCasbinAuthorizationService(database)`).
6. Rollout order per phase 12 — each phase shippable independently.

---

## 11. Edge cases

1. **Stale telemetry.** No snapshot in 15 min → marker `no_signal`; ETA falls back to scheduled (`estimated_hours`-based); share page shows "estimate" label; `eta_method` field in `/data` payload. Wrong-route drivers: speed-based ETA will drift — clamp remaining distance at `route_dist` and rely on the 0.3 scheduled weight + monotonic guard; log `eta_fallback` audit row when telemetry disagrees with scheduled by > 50%.
2. **Token abuse.** Unknown/expired/revoked → identical 404/410 (no existence oracle). PIN: 5 fails → 15 min lock (`locked_until`); per-IP `RateLimit(10)` on verify. Token stored hashed (SHA-256) — DB leak yields nothing usable. `SHARE_LINK_MAX_ACTIVE` cap per trip prevents link sprawl. Revoke kills instantly (checked before PIN cookie).
3. **Sliding expiry thrashers.** A client polling `/share/{token}/data` every 30 s keeps the link alive indefinitely — cap at `created_at + SHARE_LINK_MAX_TTL_HOURS` (hard stop). Only page + verify views refresh; `/data` reads do not extend.
4. **Fleet scale.** Single-process hub: fan-out is O(subscribers); 64-buffer drop + EventSource reconnect prevents slow-client backpressure. SQLite: `/live` query uses `idx_telemetry_snapshots_trip` (trip_id, timestamp) with a `WHERE timestamp > now−15min` window and `LIMIT` per trip (latest row per vehicle via window function). Poll page at `MAP_POLL_SEC=10`; markercluster keeps DOM nodes bounded. Multi-instance: SSE only sees local process events — document polling as the consistent path; hub optional.
5. **Maintenance flag races.** Worker tick vs assignment transaction: assignment reads flag in the same UoW tx as the trip update; worker UPDATE is a single statement — no torn state. Override recorded atomically in 00044 columns; race between override and block → block wins unless override exists at read time.
6. **DTC storm.** Dedupe: one `dtc_events` row per (vehicle, code, occurred_at bucket 1 min) via the partial unique index (00044).
7. **No odometer.** ETA uses time-proportional remaining; speed component still valid; document `eta_method: "telemetry_time_prop"`.
8. **`files.uploadable_type` narrow CHECK** — maintenance docs out of v1 scope (see section 3 note).
9. **SSE through `chiMiddleware.Compress(5)`** — chi's compress writer implements `http.Flusher`; keep `X-Accel-Buffering: no` and flush per event. Verified acceptable; if proxied (nginx/cloudflare), buffering disabled by that header.

---

## 12. Phased rollout

- **Phase 1 — Map + polling (no new infra):** `GET /tracking`, `GET /api/v1/telemetry/live`, Leaflet + markercluster, marker states. Templates + handlers + vendored assets. Works with zero server changes beyond routes.
- **Phase 2 — SSE:** `internal/realtime` hub + `/stream` + `SkipForPaths` timeout fix + consume in-memory `PositionEvent` (Spec 01 Dual-Write Fast-Path). Tracking page opts in; polling remains fallback.
- **Phase 3 — Share links:** 00044 share_links DDL + public routes + PIN + sliding expiry + revoke + `/share/*` templates. ETA wired from Phase 4 algorithm; until then share shows scheduled window.
- **Phase 4 — ETA:** `internal/eta` hybrid calculator + monotonic guard + audit; wire into `/live`, `/share/{token}/data`, popups.
- **Phase 5 — Preventive maintenance:** 00044 maintenance tables + worker + DTC intake + flags; blockers **wired but non-fatal** (log + warn) for one release; then hard block in both assignment paths + agent tools + override UI. Depends on 00042 (`maintenance_due` column).
- **Phase 6 — Hardening:** CSP on map pages, load tests on `/live` at 500+ vehicles, share-link abuse drills.

---

## 13. VERIFY items

**Avandab paths (all verified during this audit):**

- [ ] `internal/handlers/dashboard.go` — 37-line KPI page, no WS/SSE anywhere (grep `websocket|text/event-stream|EventSource` → zero).
- [ ] `cmd/server/main.go` — public web group ends ~line 714 (`/contact-us`); protected web group starts line 729; protected API group lines 445-461 with `telemetry.RegisterTelemetryRoutes(r, database)` at 448; `chiMiddleware.Timeout(60s)` at 269; outbox relay + bus at 819-826; `runDailyDigest` goroutine pattern at 823.
- [ ] `internal/telemetry/sync.go` — `snapshotHandler` insert at line 72; payload struct lines 31-41 (no heading/ignition/engine_hours/accuracy).
- [ ] `db/migrations/00031_avandab_critical_fixes.sql` — `telemetry_snapshots` cols (id, trip_id, vehicle_id, timestamp, lat, lng, speed, fuel_level, odometer) + `idx_telemetry_snapshots_trip`.
- [ ] `db/migrations/00005_routes.sql` + 00031 + 00038 — routes cols incl. `distance`, `estimated_hours`, `reverse_distance`, `reverse_standard_fare`, `is_active`.
- [ ] `db/migrations/00001_initial.sql:67` — `files.uploadable_type` narrow CHECK.
- [ ] `internal/trip/application/assign_vehicle.go` (`checkVehicleCompliance` :66-88) and `assign_driver.go` (`checkDriverCompliance` :66-82) — the two UoW assignment paths (web `internal/handlers/trips.go:72-73`; API `internal/trip/presentation/api/handlers/trip_handler.go:74`).
- [ ] `internal/service/trip_service.go` — service-layer `AssignDriver` :163 / `AssignVehicle` ~:225 (agent path via `internal/agent/tools.go:416,434`).
- [ ] `internal/shared/ports/uow.go` `RepositoryProvider` — add `Maintenance()`; impl in `internal/shared/uow/uow.go`.
- [ ] `db/query/vehicles.sql:52` `GetAvailableVehicles` — add maintenance filter after 00042.
- [ ] `internal/middleware/middleware.go:245` `SecurityHeaders` — no CSP today; `RateLimit` in `ratelimit.go` (per-IP, 1-min window, 16 shards).
- [ ] `internal/config/config.go` — plain env loader; add section 9 vars.
- [ ] `internal/templates/layout.html:15-16` — vendored datastar.js + tailwind.css (no CDN precedent).
- [ ] `db/migrations/00012_rbac.sql` — permission seed pattern; role_permissions (1=admin all, 2=dispatcher); re-seed needed in 00044.
- [ ] `internal/founder/alerts/event.go:28` — `AlertEvent` with `Metadata` map (DTC transport).

**FlyFleet references (verified; note corrected paths):**

- [ ] `apps/dashboard-web/src/modules/dashboard/components/LiveOperationsMap.tsx:290` — keyless Google raster default.
- [ ] `apps/dashboard-web/src/pages/admin/DispatchCenter.tsx:24-27` — lyrs m/s/y/p variants.
- [ ] `apps/dashboard-web/src/features/tracking/components/VehicleMap.tsx:26` — `&gl=IN`.
- [ ] `apps/dashboard-web/src/components/trips/RouteMap.tsx:143` and `.../LocationPickerMap.tsx:194` — OSM fallback (files moved from `features/tracking/components/` → `components/trips/`).
- [ ] `apps/dashboard-web/src/components/trips/AddressInput.tsx:104,154` — Nominatim reverse/forward.
- [ ] `apps/dashboard-web/src/components/LiveMap.tsx` — leaflet.markercluster CSS imports.

**Post-implementation:**

- [ ] `goose` applies 00044 clean on a fresh DB and on the existing `transport.db` (with 00031-00038 applied).
- [ ] `sqlc generate` output compiles; `go build ./...` + `go vet ./...` clean.
- [ ] Public `/share/{token}` reachable while logged out; `/tracking` 302s to `/login` when logged out.
- [ ] SSE stream survives > 60 s (Timeout skip verified with curl `--max-time 90`).
- [ ] ETA monotonic guard clamps and writes `audit_logs` on synthetic regressing snapshots.
- [ ] Assignment blocked for `maintenance_due` vehicle in web UI, REST API, and agent; override with `maintenance:update` succeeds and is audited.
- [ ] Marker states + cluster render with 100+ simulated vehicles (poll + SSE).
