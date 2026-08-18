# TELEMATICS INGESTION LAYER — Implementation Spec v2

**Owner:** telematics ingestion (this spec)
**Project:** Avandab fleet system — `/home/abhishek/Desktop/temux/basic`
**Stack:** Go 1.26, chi v5.3.1, SQLite (modernc), goose v3.27.3, casbin v2.135.0, Datastar v1.0.2 + Tailwind web templates, outbox-relay event bus (`cmd/server/main.go:609-616`), typed IDs, UoW pattern.
**Module path:** `internal/telemetry`
**Consumers (4 other specs):** live-tracking UI spec, alerting spec, fuel spec, driver-payout/reporting spec
**Status:** updated for the GPS-resale decision (own-device first-class), third-party providers demoted to later adapters.

---

## 1. Architecture Overview

```
                        ┌──────────────────────────────────────────────────────────────┐
                        │                        OWN GPS DEVICES                        │
                        │  (sold/installed by Avandab — first-class, not fallback)     │
                        └───────┬──────────────────────────────┬───────────────────────┘
                                │ MQTT push                    │ REST/HTTP fallback
        topic: avandab/telemetry/devices/{imei}/gps             POST /api/v1/telemetry/devices/{imei}/gps
        QoS 1, username=IMEI, password=device secret            X-Device-Token header
                                │                              │
                                ▼                              ▼
        ┌───────────────────────────────────────────────────────────────────────────┐
        │  INGESTION FRONT DOORS  (internal/telemetry)                              │
        │   • mqtt_ingest.go   — real subscriber (replaces stub in                   │
        │                        internal/mqttservice/mqtt.go:31-52)                 │
        │   • http_ingest.go   — REST fallback + driver-app sync retrofit           │
        │   • webhooks.go      — POST /api/v1/telemetry/webhooks/{provider}/{token} │
        │                        (Razorpay-webhook pattern, per-provider token)     │
        └───────────────┬───────────────────────────────────────────────────────────┘
                        │ RawFrame (canonical, provider-neutral)
                        ▼
        ┌───────────────────────────────────────────────────────────────────────────┐
        │  CANONICAL PIPELINE (internal/telemetry/ingest.go) — owned by this spec   │
        │                                                                           │
        │  1. Validate + device lookup (telemetry_devices by IMEI)                  │
        │  2. Unknown IMEI ────────────────────────────► device_quarantine queue    │
        │  3. telemetry_raw_events  (append-only, partial-unique dedup, retention)  │
        │  4. telemetry_positions   (insert, out-of-order-guarded)                  │
        │  5. vehicle_latest_position (upsert, only newer device_time wins)         │
        │  6. telemetry_snapshots   (enriched: heading, ignition, engine_hours,     │
        │                            accuracy, driver_id — migration 00040)         │
        │  7. Guards: odometer rollback (audit-logged), fuel_level clamp            │
        │  8. outbox_events INSERT (same tx) → relay publishes to bus               │
        └───────────────┬───────────────────────────────────────────────────────────┘
                        │ PositionEvent / AlertEvent / SOSEvent
                        ▼
        ┌───────────────────────────────────────────────────────────────────────────┐
        │  OUTBOX-RELAY BUS  (verified: cmd/server/main.go:609-616,                 │
        │  internal/shared/outbox/relay.go — polls outbox_events every 5s,          │
        │  event_type = Go struct name via getEventTypeName,                        │
        │  dispatches to events.InMemoryBus)                                        │
        └───────┬───────────────────┬───────────────────┬───────────────────────────┘
                ▼                   ▼                   ▼
        live-tracking spec    alerting spec        fuel spec
        (consumes Position)   (consumes Alert,     (consumes Position
                              SOS; owns telemetry_  fuel_level/odometer;
                              alerts rebuild 00048) co-owns odometer guard)

        Third-party providers (LocoNav, WheelsEye): demoted — later adapters behind
        ProviderIngestor (HandleWebhook + Poll + VerifySignature). No build priority.
```

Verified wiring facts reused by this spec:

- MQTT stub: `internal/mqttservice/mqtt.go` lines 31–52 — subscribes `avandab/telemetry/drivers/+/gps`, logs only. `NewMQTTBroker` called at `cmd/server/main.go:327` with `MQTT_URL` (default `tcp://localhost:1883`). Paho MQTT client `github.com/eclipse/paho.mqtt.golang v1.5.1` already in `go.mod`.
- `internal/telemetry/sync.go` — `HandleTelemetrySync` echoes IDs without persisting (no-op); `snapshotHandler` writes `telemetry_snapshots` via raw SQL `INSERT OR REPLACE` (`db/migrations/00031_avandab_critical_fixes.sql` created that table).
- `internal/service/telemetry_service.go` — rule engine only logs (lines 69, 101 `s.log.Warn`) and publishes `GPSDeviationAlert`/`FuelTheftAlert` directly to the bus (not outbox). `telemetry_alerts` CHECK currently `alert_type IN ('gps_deviation','fuel_theft','temp_breach','speeding')` (migration 00030). Widening rebuild = migration 00048, owned by alerting spec — referenced only.
- Outbox: `internal/shared/outbox/outbox.go` `OutboxWriter.SaveEvents(ctx, aggregateID, aggregateType, events)` honors `repository.TxFromContext`; relay `internal/shared/outbox/relay.go`; table `outbox_events` (00020). Event type = struct name minus package (e.g. `PositionEvent`). **The outbox table is never written by any use case today — first write is a milestone.**
- Mobile driver app GPS: `mobile/src/services/telemetry.ts` (expo-location `watchPositionAsync`, 10 s / 20 m, offline SQLite via `mobile/src/services/storage.ts`), `mobile/src/services/mqtt.ts` (`publishLocation` → `avandab/telemetry/drivers/{driverId}/gps` with `{driver_id, latitude, longitude, timestamp}`; **no callers found in mobile/src — VERIFY AT IMPLEMENTATION**), `mobile/src/services/syncEngine.ts` (15 s POST `/api/v1/telemetry/sync`, expects `synced_ids`).
- No WebSocket hub exists anywhere (no `websocket`/`gorilla/websocket`/`Hub` usage in Go source; `gorilla/websocket v1.5.3` present only as an indirect go.mod dependency). Live-tracking UI spec builds its own push layer; this spec emits events on the bus only.
- RBAC: Casbin loaded from `role_permissions`/`user_roles` via `internal/auth/casbin.go` DBAdapter; permission names `resource:action`. Web pages use `middleware.ResourcePermission` (`internal/middleware/middleware.go:189`), API uses `middleware.RequirePermission` (`api_auth.go:50`) after `RequireAPIAuth`.
- Razorpay webhook pattern: `internal/payment/application/razorpay_webhook.go` — HMAC-SHA256 over raw body, hex-encoded signature, secret from env; mounted at `cmd/server/main.go:347` with `middleware.RateLimit(30)`, public (no session auth).
- UoW: `internal/repository/tx.go` (`WithTransaction`, `WithTxInContext`, `TxFromContext`); `internal/shared/ports/uow.go`; typed IDs in `internal/domain/types/ids.go`.
- Migrations: goose, embedded via `db/migrations.go` (`//go:embed migrations/*.sql` — new files auto-included). Latest = 00038. `sqlc.yaml` regenerates `db/generated/sqlite` from `db/query` — **this spec uses raw SQL only; no sqlc regen**.

---

## 2. Canonical Event Contracts

These structs are the cross-spec contract. Event type strings on `outbox_events.event_type` are the exact Go struct names (relay derives them via `getEventTypeName`, `internal/shared/outbox/outbox.go:67`). Consumers must match on these exact strings:

| event_type (exact) | struct | consumers |
|---|---|---|
| `PositionEvent` | PositionEvent | live-tracking UI, fuel spec, reporting |
| `AlertEvent` | AlertEvent | alerting spec |
| `SOSEvent` | SOSEvent | alerting spec, ops/notifications |

Outbox write contract (per event): `aggregate_id` = vehicle UUID (or IMEI when unassigned), `aggregate_type` = `"Vehicle"` (position) / `"Device"` (alert/SOS), `event_type` = struct name, `payload` = JSON of the struct below, written **in the same transaction** as the pipeline writes (`OutboxWriter.SaveEvents` honors `repository.TxFromContext`; alternatively a direct `INSERT INTO outbox_events` with the same tx — VERIFY AT IMPLEMENTATION which is used; the writer is the established pattern, see `internal/booking/infrastructure/persistence/sql/booking_repository.go:91`).

```go
// RawFrame is the provider-neutral input unit. Every front door (MQTT, HTTP,
// webhook, poll) must produce RawFrame; the canonical pipeline consumes only this.
type RawFrame struct {
    IMEI          string          `json:"imei"`                     // device identity everywhere in this spec
    DeviceTime    time.Time       `json:"device_time"`              // device clock (may lag network)
    Latitude      float64         `json:"latitude"`
    Longitude     float64         `json:"longitude"`
    Speed         float64         `json:"speed,omitempty"`
    Heading       float64         `json:"heading,omitempty"`        // degrees 0-360
    Ignition      *bool           `json:"ignition,omitempty"`
    EngineHours   *float64        `json:"engine_hours,omitempty"`
    Accuracy      *float64        `json:"accuracy,omitempty"`       // meters
    FuelLevel     *float64        `json:"fuel_level,omitempty"`     // percent 0-100, clamped
    Odometer      *float64        `json:"odometer,omitempty"`       // km, rollback-guarded
    DriverID      string          `json:"driver_id,omitempty"`
    TripID        string          `json:"trip_id,omitempty"`
    Provider      string          `json:"provider"`                 // "own" | "loconav" | "wheelseye"
    ProviderMsgID string          `json:"provider_msg_id,omitempty"`// dedup key (provider message id / device seq)
    RawPayload    json.RawMessage `json:"raw_payload,omitempty"`    // provider original, for audit
}

// PositionEvent — published once per accepted frame. Canonical; exact field names
// and JSON tags are contractual for all consuming specs.
type PositionEvent struct {
    EventID     string    `json:"event_id"`
    TenantID    string    `json:"tenant_id"`
    DeviceIMEI  string    `json:"device_imei"`
    VehicleID   string    `json:"vehicle_id"`
    DriverID    string    `json:"driver_id,omitempty"`
    TripID      string    `json:"trip_id,omitempty"`
    Latitude    float64   `json:"latitude"`
    Longitude   float64   `json:"longitude"`
    Speed       float64   `json:"speed"`
    Heading     float64   `json:"heading,omitempty"`
    Ignition    *bool     `json:"ignition,omitempty"`
    EngineHours *float64  `json:"engine_hours,omitempty"`
    Accuracy    *float64  `json:"accuracy,omitempty"`
    FuelLevel   *float64  `json:"fuel_level,omitempty"`
    Odometer    *float64  `json:"odometer,omitempty"`
    DeviceTime  time.Time `json:"device_time"`
    ReceivedAt  time.Time `json:"received_at"`
}

// AlertEvent — ingestion layer only relays device/provider-reported alerts
// (e.g. LocoNav DTC, fuel-theft kinds; own-device tamper). Rule evaluation
// (speeding, deviation, fuel-drop) belongs to the alerting spec (00048).
type AlertEvent struct {
    EventID    string    `json:"event_id"`
    TenantID   string    `json:"tenant_id"`
    DeviceIMEI string    `json:"device_imei"`
    VehicleID  string    `json:"vehicle_id"`
    DriverID   string    `json:"driver_id,omitempty"`
    TripID     string    `json:"trip_id,omitempty"`
    AlertType  string    `json:"alert_type"` // speeding, fuel_theft, dtc, tamper, power_cut ...
    Severity   string    `json:"severity"`   // warning | critical
    Latitude   float64   `json:"latitude"`
    Longitude  float64   `json:"longitude"`
    Details    string    `json:"details"`
    OccurredAt time.Time `json:"occurred_at"`
}

// SOSEvent — panic button / crash detection on own devices; provider adapters
// map their equivalents (VERIFY AT IMPLEMENTATION for LocoNav/WheelsEye).
type SOSEvent struct {
    EventID    string    `json:"event_id"`
    TenantID   string    `json:"tenant_id"`
    DeviceIMEI string    `json:"device_imei"`
    VehicleID  string    `json:"vehicle_id"`
    DriverID   string    `json:"driver_id,omitempty"`
    Latitude   float64   `json:"latitude"`
    Longitude  float64   `json:"longitude"`
    OccurredAt time.Time `json:"occurred_at"` // server time
    DeviceTime time.Time `json:"device_time"`
}
```

---

## 3. Device Registry + Provisioning Model

Purpose: Avandab is itself a GPS vendor. Every sold device has a lifecycle record; inventory is trackable before assignment; unknown hardware is quarantined, never silently dropped.

### States

```
 register ──► inventory ──assign──► assigned ──activate──► active
   │              │                    │                    │
   │              │                    └──► retired ◄────────┘ (decommission / warranty swap)
   │              └────► retired
   └── unknown IMEI on wire ──► quarantine (device_quarantine queue; admin resolves:
                                 register-as-new / assign-existing / reject)
```

- `inventory` — registered, unsold/unassigned stock (bulk registration target).
- `assigned` — linked to `vehicle_id` (or customer), not yet activated.
- `active` — activated (`activated_at` set), accepts telemetry.
- `retired` — decommissioned; telemetry frames from retired devices are dropped to quarantine (reason `retired_device`).
- `quarantined` — device-level quarantine (e.g. suspected cloned IMEI); distinct from the open quarantine queue entries.

Rules: one device per vehicle (`UNIQUE ... WHERE vehicle_id IS NOT NULL`). Frames accepted only from `active` devices. `last_seen_at` updated on every accepted frame.

Bulk registration: POST `/api/v1/telemetry/devices/bulk` (JSON array, max 500 per call — VERIFY AT IMPLEMENTATION cap) for customers who buy our GPS; each entry: `imei`, `serial_number`, `firmware_version`, `sim_number`, `iccid`, `warranty_until`, optional `customer_id`. Duplicate IMEI within a batch → whole batch rejected (atomic via UoW) — VERIFY AT IMPLEMENTATION (batch-level vs per-row semantics).

---

## 4. Own-Device MQTT + HTTP Protocol Spec

### 4.1 MQTT (primary path — own hardware)

- Broker: external (no broker in `docker-compose.yml`/deploy scripts — VERIFY AT IMPLEMENTATION broker deployment; existing default `MQTT_URL=tcp://localhost:1883`).
- Subscribe (server): `avandab/telemetry/devices/{imei}/gps`, QoS 1, retained=false.
- Publish (device): same topic, QoS 1, retained=false. One JSON frame per message:

```json
{
  "imei": "866992051234567",
  "seq": 88421,
  "device_time": "2026-08-17T10:15:30Z",
  "latitude": 22.94146,
  "longitude": 73.30259,
  "speed": 12.5,
  "heading": 90.0,
  "ignition": true,
  "engine_hours": 1234.5,
  "accuracy": 3.2,
  "fuel_level": 78.4,
  "odometer": 45213.7,
  "sos": false
}
```

Field rules: `imei` must match the topic segment (mismatch → reject frame, audit-log). `seq` = monotonically increasing per device; serves as dedup key (`provider_msg_id` = `"mqtt:<seq>"`). `sos:true` → SOSEvent.

Auth: MQTT username = IMEI, password = per-device secret (provisioned at activation; stored as `device_secret_hash`). Server-side ACL on the broker must restrict each client to its own topic — **VERIFY AT IMPLEMENTATION** (broker ACL config; if the chosen broker cannot ACL, enforce by checking username==IMEI in the subscriber and dropping non-matching).

Transport: TLS/wss in production (`MQTT_URL` scheme). No TLS → refuse to start ingestion in production (VERIFY AT IMPLEMENTATION hard-fail vs warning; Razorpay secret precedent suggests hard-fail is acceptable).

Ack/retry: QoS 1 broker ack is the only ack. Device retries: on reconnect, device re-publishes unacked `seq` frames; server dedup (section 4.3) makes replays idempotent.

### 4.2 REST/HTTP fallback (own devices + driver app)

- `POST /api/v1/telemetry/devices/{imei}/gps` — same JSON body as MQTT. Auth: `X-Device-Token: <device_secret>`. Response `200 {"accepted": true, "deduped": bool}`; `202` for quarantined (accepted but queued); `401` bad token; `404` unknown IMEI (still quarantined server-side).
- Driver app (existing mobile): retrofit `mobile/src/services/syncEngine.ts` and `mobile/src/services/mqtt.ts` to the canonical contract (section 12, Phase 3). The driver app has no IMEI — it must present the registered device identity of the phone/handset (VERIFY AT IMPLEMENTATION: driver devices are registered in `telemetry_devices` with a synthetic IMEI, or `driver_id` carried in `driver_id` field with `imei` = device serial; decision needed with mobile spec).
- Existing `/api/v1/telemetry/sync` + `/api/v1/telemetry/snapshots` (verified no-op/persist paths in `internal/telemetry/sync.go`) are re-pointed into the same pipeline — they keep their URL (mobile ships with them) but their handlers now produce RawFrames. `HandleTelemetrySync` must finally persist and return real `synced_ids` (currently echoes them without storing).

### 4.3 Dedup + ordering

- Partial-unique dedup: `UNIQUE INDEX ON telemetry_raw_events(imei, dedup_key) WHERE dedup_key IS NOT NULL`. Pipeline uses `INSERT OR IGNORE`-style semantics so replays are no-ops.
- Out-of-order: raw events always appended with `received_at`; `telemetry_positions` insert is unconditional; `vehicle_latest_position` upsert guarded by `device_time > existing.device_time`; `telemetry_snapshots` `INSERT OR REPLACE` keeps the existing `id` scheme (`sync.go` currently generates `snap-<timestamp>` — replace with uuid; VERIFY AT IMPLEMENTATION migration of old snapshot ids).

### 4.4 Webhook ingestion (third-party + future own-device HTTP push)

`POST /api/v1/telemetry/webhooks/{provider}/{token}` — pattern of the Razorpay webhook (verified: public mount, `middleware.RateLimit(30)`, HMAC over raw body, secret from env). Here `{token}` is the per-provider secret from env, compared constant-time (`crypto/subtle`); provider-specific signature headers additionally verified via `ProviderIngestor.VerifySignature` when defined (e.g. LocoNav `User-Authentication` header). VERIFY AT IMPLEMENTATION: whether `{token}`-in-path or header-only auth is final for each provider.

---

## 5. ProviderIngestor Interface + Adapter Stubs

Demoted scope: adapters are stubs behind the interface; no build priority. Interface lives in `internal/telemetry/providers/ingestor.go`.

```go
// ProviderIngestor abstracts third-party telemetry providers.
// Implementations must be stateless w.r.t. the pipeline; polling state is
// persisted in provider_poll_state (migration 00039).
type ProviderIngestor interface {
    // Name is the canonical provider id used in routes and DB ("loconav", "wheelseye").
    Name() string
    // VerifySignature validates provider webhook auth. Return nil when the
    // provider has no signature scheme (then route-token auth alone applies).
    VerifySignature(rawBody []byte, header http.Header) error
    // HandleWebhook converts a provider webhook body into canonical RawFrames.
    // Provider-specific alert kinds surface as AlertEvents via RawFrame extension
    // (VERIFY AT IMPLEMENTATION: side-channel vs AlertFrame).
    HandleWebhook(ctx context.Context, rawBody []byte) ([]RawFrame, error)
    // Poll pulls frames for providers without webhooks (WheelsEye).
    // `since` comes from provider_poll_state.last_success_at.
    Poll(ctx context.Context, since time.Time) ([]RawFrame, error)
}
```

### 5.1 LocoNav (`internal/telemetry/providers/loconav.go`) — verified facts, no re-verification needed

- Webhooks documented; payloads include `latitude`, `longitude`, `speed`, `ignition`, `fuel`, alert kinds incl. DTC and fuel-theft; auth header `User-Authentication`.
- Stub: `VerifySignature` compares `User-Authentication` against `TELEMETRY_WEBHOOK_SECRET_LOCONAV` (VERIFY AT IMPLEMENTATION exact value semantics — raw secret vs hash). `HandleWebhook` maps to RawFrame; DTC/fuel-theft kinds → `AlertEvent`. Poll: not supported (webhook-only) → return `nil, nil`.

### 5.2 WheelsEye (`internal/telemetry/providers/wheelseye.go`) — verified facts

- Pull API: `GET https://api.wheelsEye.com/currentLoc?accessToken=<token>` returns:

```json
{"message":"Ok","success":true,"list":[{"vehicleNumber":"GJ11Z1234","latitude":22.94146,"longitude":73.30259,"speed":0,"createdDate":1618558042}]}
```

`createdDate` is epoch seconds. Richer variant includes `deviceNumber` (= IMEI). No public dev portal.

- Stub: `Poll` calls the endpoint (token from `WHEELSEYE_ACCESS_TOKEN`), maps `vehicleNumber` → vehicle → IMEI via `telemetry_devices`/`vehicles` (VERIFY AT IMPLEMENTATION mapping table when `deviceNumber` absent); `VerifySignature` returns nil (no webhooks; route-token only).

---

## 6. Full SQL DDL — Migrations 00039 and 00040

Conventions followed: `id TEXT PRIMARY KEY` (UUID), `tenant_id TEXT NOT NULL DEFAULT '1'` (00038 pattern), `DATETIME DEFAULT CURRENT_TIMESTAMP` (00030/00031 pattern), FK to existing tables (`vehicles`, `drivers`, `trips`, `customers`, `users`).

### `db/migrations/00039_telemetry_devices_and_ingestion.sql`

```sql
-- +goose Up

-- Device registry (GPS resale lifecycle)
CREATE TABLE telemetry_devices (
    id                 TEXT PRIMARY KEY,
    tenant_id          TEXT NOT NULL DEFAULT '1',
    imei               TEXT NOT NULL UNIQUE,
    serial_number      TEXT,
    firmware_version   TEXT,
    sim_number         TEXT,
    iccid              TEXT,
    warranty_until     DATE,
    status             TEXT NOT NULL DEFAULT 'inventory'
                       CHECK (status IN ('inventory','assigned','active','retired','quarantined')),
    vehicle_id         TEXT,
    customer_id        TEXT,
    activated_at       DATETIME,
    last_seen_at       DATETIME,
    device_secret_hash TEXT,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id)  REFERENCES vehicles(id),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);
CREATE UNIQUE INDEX idx_telemetry_devices_vehicle
    ON telemetry_devices(vehicle_id) WHERE vehicle_id IS NOT NULL;
CREATE INDEX idx_telemetry_devices_status ON telemetry_devices(status);
CREATE INDEX idx_telemetry_devices_tenant ON telemetry_devices(tenant_id, status);

-- Append-only raw frame log (dedup + retention + audit)
CREATE TABLE telemetry_raw_events (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT NOT NULL DEFAULT '1',
    imei          TEXT NOT NULL,
    device_time   DATETIME NOT NULL,
    received_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    provider      TEXT NOT NULL DEFAULT 'own',
    provider_msg_id TEXT,
    payload       TEXT NOT NULL,
    FOREIGN KEY (imei) REFERENCES telemetry_devices(imei)
);
CREATE UNIQUE INDEX idx_raw_dedup ON telemetry_raw_events(imei, provider_msg_id)
    WHERE provider_msg_id IS NOT NULL;
CREATE INDEX idx_raw_received ON telemetry_raw_events(received_at);
CREATE INDEX idx_raw_imei_time ON telemetry_raw_events(imei, device_time DESC);

-- Unknown-IMEI quarantine queue (admin resolution UI)
CREATE TABLE device_quarantine (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    imei        TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'own' CHECK (source IN ('own','loconav','wheelseye')),
    raw_payload TEXT NOT NULL,
    reason      TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved','rejected')),
    resolved_by TEXT,
    resolved_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (resolved_by) REFERENCES users(id)
);
CREATE INDEX idx_quarantine_status ON device_quarantine(status, created_at);

-- Provider polling state (store-and-forward during downtime)
CREATE TABLE provider_poll_state (
    provider            TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL DEFAULT '1',
    last_poll_at        DATETIME,
    last_success_at     DATETIME,
    cursor              TEXT,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    backoff_until       DATETIME
);

-- RBAC seeds (casbin loads role_permissions -> permissions, see 00012 pattern)
INSERT OR IGNORE INTO permissions (name, description) VALUES
('telemetry:read',   'View devices, positions, quarantine queue'),
('telemetry:write',  'Ingest telemetry (API-level, machine routes)'),
('telemetry:update', 'Provision devices: register, assign, activate, retire, resolve quarantine'),
('telemetry:delete', 'Delete devices / quarantine entries');
-- admin (role 1): everything
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name LIKE 'telemetry:%';
-- dispatcher (role 2): view + provision, no delete
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name IN ('telemetry:read','telemetry:update');

-- +goose Down
DROP TABLE IF EXISTS provider_poll_state;
DROP TABLE IF EXISTS device_quarantine;
DROP TABLE IF EXISTS telemetry_raw_events;
DROP TABLE IF EXISTS telemetry_devices;
DELETE FROM permissions WHERE name LIKE 'telemetry:%';
```

### `db/migrations/00040_telemetry_positions_and_snapshots.sql`

```sql
-- +goose Up

-- Position history (per accepted frame, after raw-log + guards)
CREATE TABLE telemetry_positions (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    imei        TEXT NOT NULL,
    device_time DATETIME NOT NULL,
    received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    latitude    REAL NOT NULL,
    longitude   REAL NOT NULL,
    speed       REAL,
    heading     REAL,
    ignition    INTEGER,
    engine_hours REAL,
    accuracy    REAL,
    fuel_level  REAL,
    odometer    REAL,
    driver_id   TEXT,
    trip_id     TEXT,
    vehicle_id  TEXT,
    provider    TEXT NOT NULL DEFAULT 'own',
    raw_event_id TEXT,
    FOREIGN KEY (raw_event_id) REFERENCES telemetry_raw_events(id),
    FOREIGN KEY (vehicle_id)   REFERENCES vehicles(id),
    FOREIGN KEY (trip_id)      REFERENCES trips(id),
    FOREIGN KEY (driver_id)    REFERENCES drivers(id)
);
CREATE INDEX idx_positions_imei_time  ON telemetry_positions(imei, device_time DESC);
CREATE INDEX idx_positions_vehicle_time ON telemetry_positions(vehicle_id, device_time DESC);
CREATE INDEX idx_positions_raw_event  ON telemetry_positions(raw_event_id);

-- Latest position per vehicle (upsert; only newer device_time wins)
CREATE TABLE vehicle_latest_position (
    vehicle_id  TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    imei        TEXT,
    device_time DATETIME NOT NULL,
    received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    latitude    REAL NOT NULL,
    longitude   REAL NOT NULL,
    speed       REAL,
    heading     REAL,
    ignition    INTEGER,
    engine_hours REAL,
    accuracy    REAL,
    fuel_level  REAL,
    odometer    REAL,
    driver_id   TEXT,
    trip_id     TEXT,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);

-- Snapshot enrichment (existing table from 00031; consumers rely on these fields)
ALTER TABLE telemetry_snapshots ADD COLUMN heading REAL;
ALTER TABLE telemetry_snapshots ADD COLUMN ignition INTEGER;
ALTER TABLE telemetry_snapshots ADD COLUMN engine_hours REAL;
ALTER TABLE telemetry_snapshots ADD COLUMN accuracy REAL;
ALTER TABLE telemetry_snapshots ADD COLUMN driver_id TEXT;

-- +goose Down
DROP TABLE IF EXISTS vehicle_latest_position;
DROP TABLE IF EXISTS telemetry_positions;
ALTER TABLE telemetry_snapshots DROP COLUMN heading;
ALTER TABLE telemetry_snapshots DROP COLUMN ignition;
ALTER TABLE telemetry_snapshots DROP COLUMN engine_hours;
ALTER TABLE telemetry_snapshots DROP COLUMN accuracy;
ALTER TABLE telemetry_snapshots DROP COLUMN driver_id;
```

Note: `telemetry_alerts` CHECK-widening rebuild is migration **00048**, owned by the alerting spec — this spec only references it (current CHECK: `alert_type IN ('gps_deviation','fuel_theft','temp_breach','speeding')`, migration 00030; 00048 will widen to include `dtc`, `tamper`, `sos` etc.).

---

## 7. Go File List

### Create

| Path | Purpose |
|---|---|
| `internal/telemetry/contracts.go` | `RawFrame`, `PositionEvent`, `AlertEvent`, `SOSEvent`, event-type constants, alert-kind enum |
| `internal/telemetry/ingest.go` | Canonical pipeline: validate → quarantine-check → raw insert (dedup) → position insert → latest upsert → snapshot enrich → guards → outbox write → bus events |
| `internal/telemetry/devices.go` | Device registry service: register / bulk-register / assign / activate / retire / quarantine, secret provisioning |
| `internal/telemetry/device_store.go` | Raw-SQL persistence for `telemetry_devices`, `device_quarantine`, `provider_poll_state` (no sqlc regen) |
| `internal/telemetry/mqtt_ingest.go` | Real MQTT subscriber: topic `avandab/telemetry/devices/+/gps`, auth (username=IMEI), frame→RawFrame |
| `internal/telemetry/http_ingest.go` | REST fallback handler `POST /api/v1/telemetry/devices/{imei}/gps` + retrofitted `/sync`, `/snapshots` handlers |
| `internal/telemetry/webhooks.go` | `POST /api/v1/telemetry/webhooks/{provider}/{token}`, per-provider secret from env, constant-time compare, rate-limited |
| `internal/telemetry/quarantine.go` | Quarantine queue logic + admin resolution (register-new / assign-existing / reject) |
| `internal/telemetry/odometer_guard.go` | Odometer rollback guard + audit-log write; fuel_level clamp (coordinates with fuel spec) |
| `internal/telemetry/providers/ingestor.go` | `ProviderIngestor` interface |
| `internal/telemetry/providers/loconav.go` | LocoNav adapter stub (webhook mapping + `User-Authentication` check) |
| `internal/telemetry/providers/wheelseye.go` | WheelsEye adapter stub (`currentLoc` poll, epoch-seconds mapping, backoff via `provider_poll_state`) |
| `internal/telemetry/handlers/devices_handler.go` | Web UI handlers: device list, bulk register, quarantine queue (Datastar pages, `ResourcePermission` `telemetry:*`) |
| `internal/templates/devices_list.html` | Datastar device list page |
| `internal/templates/devices_register.html` | Bulk registration page |
| `internal/templates/quarantine_queue.html` | Unknown-IMEI resolution UI |
| `internal/templates/partials/device_row.html` | Row partial (Datastar `data-on-click`/`data-signals` style per `internal/templates/partials/`) |
| `db/migrations/00039_telemetry_devices_and_ingestion.sql` | DDL above (auto-embedded by `db/migrations.go`) |
| `db/migrations/00040_telemetry_positions_and_snapshots.sql` | DDL above |

### Modify

| Path | Change |
|---|---|
| `internal/mqttservice/mqtt.go` | Replace stub `subscribeTelemetry` (lines 41–52): subscribe `avandab/telemetry/devices/+/gps`, hand frames to `telemetry/mqtt_ingest.go`; keep `PublishTripUpdate`. Keep `GPSTelemetryPayload` for compat — VERIFY AT IMPLEMENTATION whether mobile still publishes the old driver topic (migration bridge) |
| `internal/telemetry/sync.go` | `HandleTelemetrySync` persists via pipeline and returns real `synced_ids`; `snapshotHandler` routes into pipeline instead of `INSERT OR REPLACE` |
| `cmd/server/main.go` | Wire MQTT subscriber callback (line ~327), mount webhook route next to Razorpay (line ~347, `RateLimit`), mount device/quarantine web routes, register `telemetry:*` handlers in `App` |
| `internal/config/config.go` | Add `Telemetry` config struct (env below) |
| `internal/handlers/app.go` | Add `TelemetryDevices *TelemetryDeviceHandlers` handler group + wiring |
| `mobile/src/services/mqtt.ts` | `publishLocation` → canonical payload (`imei`-based topic `avandab/telemetry/devices/{imei}/gps`, add `seq`/`device_time`); **no callers today — VERIFY AT IMPLEMENTATION which screen should invoke it** |
| `mobile/src/services/syncEngine.ts` | Canonical batch body + dedup ids (`provider_msg_id` = offline log id) |
| `mobile/src/services/telemetry.ts` | Emit canonical frames (add `seq`, `accuracy`, heading from `loc.coords`) |
| `mobile/src/constants/network.ts` | Optional: device token/imei env plumbing — VERIFY AT IMPLEMENTATION |

No changes to: `internal/events/`, `internal/shared/outbox/`, `internal/auth/`, `internal/repository/`, `sqlc.yaml`, `db/query/` (raw SQL only).

---

## 8. Config / Env Table

All via env vars in `internal/config/config.go` (`TelemetryConfig`). No sqlc regen, no DB config tables.

| Env var | Default | Purpose |
|---|---|---|
| `MQTT_URL` (exists, main.go:323) | `tcp://localhost:1883` | Broker; production must be `ssl://`/`wss://` |
| `TELEMETRY_WEBHOOK_SECRET_LOCONAV` | empty (disabled) | LocoNav route token + `User-Authentication` check |
| `TELEMETRY_WEBHOOK_SECRET_WHEELSEYE` | empty (disabled) | WheelsEye route token |
| `WHEELSEYE_ACCESS_TOKEN` | empty | `currentLoc` poll token |
| `WHEELSEYE_POLL_INTERVAL` | `60s` | Poll cadence; backoff via `provider_poll_state` |
| `TELEMETRY_RAW_RETENTION_DAYS` | `30` | Retention sweep for `telemetry_raw_events` (scheduled job) |
| `TELEMETRY_BATCH_SIZE` | `500` | Frames per commit batch (SQLite write planning) |
| `TELEMETRY_FLUSH_INTERVAL` | `2s` | Batched position flush (MQTT burst absorption) |
| `TELEMETRY_ODOMETER_MAX_REGRESSION_KM` | `1.0` | Per-frame rollback tolerance before guard/audit |
| `TELEMETRY_FUEL_CLAMP_DELTA_PCT` | `5.0` | Max |Δ fuel| per frame; clamp + audit (coordinated with fuel spec) |
| `TELEMETRY_DEVICE_SECRET_PEPPER` | empty | HMAC pepper for `device_secret_hash` — VERIFY AT IMPLEMENTATION (bcrypt vs HMAC for low-power devices) |
| `TELEMETRY_WEBHOOK_RATE_LIMIT` | `30` | Per-IP fixed window (mirrors Razorpay `RateLimit(30)`) |

---

## 9. Migration Plan

1. **00039** — `telemetry_devices`, `telemetry_raw_events`, `device_quarantine`, `provider_poll_state`, RBAC seeds `telemetry:*`. (this spec)
2. **00040** — `telemetry_positions`, `vehicle_latest_position`, `telemetry_snapshots` enrichment columns. (this spec)
3. **00048** — `telemetry_alerts` CHECK widening rebuild. (alerting spec — referenced only)
4. No sqlc regen: pipeline uses raw SQL via `database/sql`; new tables are NOT added to `db/query` (decision 6). VERIFY AT IMPLEMENTATION: `sqlc diff`/CI check stays green with schema ahead of queries.
5. Goose embed picks up new files automatically (`db/migrations.go` embeds `migrations/*.sql`); standard `goose up` ordering applies (00039 → 00040 → … → 00048).

---

## 10. Security

- **Webhook route** (`/api/v1/telemetry/webhooks/{provider}/{token}`): public + `RateLimit` (Razorpay precedent, main.go:347); `{token}` per-provider env secret, `crypto/subtle` constant-time compare; body-signature verification delegated to `ProviderIngestor.VerifySignature` (LocoNav `User-Authentication`).
- **Device REST** (`/api/v1/telemetry/devices/{imei}/gps`): `X-Device-Token` (device secret, stored hashed); no session auth (machine-to-machine), rate-limited.
- **MQTT**: username=IMEI + password=device secret; topic must match authenticated IMEI (ACL or subscriber-side check — VERIFY AT IMPLEMENTATION broker ACL); TLS mandatory in production.
- **Driver-app routes** (`/sync`, `/snapshots`): stay behind `RequireAPIAuth` (verified group at main.go:350–361) — mobile already sends Bearer token (verified `syncEngine.ts`).
- **Admin UI**: `ResourcePermission(authSrv, "telemetry", ...)` per page/action; casbin loaded from seeded `telemetry:*` permissions; `users:create`-style granularity per 00012 pattern.
- **Secrets**: never logged; raw payloads stored verbatim in `telemetry_raw_events.payload`/`device_quarantine.raw_payload` (audit), but MQTT passwords/device secrets never included in event payloads.
- **Quarantine**: unknown IMEI frames never reach positions/latest/events; only `device_quarantine` + raw log.

---

## 11. Edge Cases

- **Duplicate frames** (MQTT QoS1 replays, mobile retries, provider redelivery): partial-unique `(imei, provider_msg_id)` index + `INSERT OR IGNORE`; response carries `deduped: true`.
- **Out-of-order timestamps**: raw append always; positions insert unconditional; `vehicle_latest_position` upsert `WHERE device_time > old`; snapshots `INSERT OR REPLACE` with monotonic guard (VERIFY AT IMPLEMENTATION: snapshot id scheme change from timestamp-based `generateSnapshotID` to uuid).
- **Unknown IMEI**: quarantine queue entry (`reason: unknown_device`), no downstream writes; admin resolution UI: register new device / assign existing / reject.
- **Retired/quarantined device still transmitting**: frame → quarantine (`reason: retired_device` / `quarantined_device`), audit-logged.
- **Odometer rollback**: if `odometer < last - TELEMETRY_ODOMETER_MAX_REGRESSION_KM` → reject new value, keep last, write `audit_logs` entry (action `odometer_rollback_guard`, via existing `AuditLogService.LogAction` pattern — verified `internal/service/audit_log_service.go:32`); behavior coordinated with fuel spec (single guard owner).
- **Fuel sanity clamp**: |Δ fuel_level| > clamp → clamp to last value ± clamp, audit; fuel spec consumes `PositionEvent.FuelLevel` and must expect clamped values (cross-spec contract note).
- **Provider downtime / store-and-forward**: `provider_poll_state` tracks `consecutive_failures`, `backoff_until` (exponential backoff); WheelsEye `Poll` resumes from `last_success_at`; mobile offline SQLite queue already exists (`mobile/src/services/storage.ts`, `offline_gps_logs`) and flows through `/sync` once connectivity returns.
- **Rate/volume planning (SQLite modernc)**: single-writer discipline — one ingestion goroutine per front door funneling into a batched writer (`TELEMETRY_BATCH_SIZE`, `TELEMETRY_FLUSH_INTERVAL`); WAL already default in `DATABASE_URL` (`_journal_mode=WAL`, verified config.go:99); index `received_at`/`imei, device_time` for retention sweep; retention job deletes raw rows older than `TELEMETRY_RAW_RETENTION_DAYS` in batches (VERIFY AT IMPLEMENTATION: batch delete size + schedule).
- **IMEI/topic mismatch on MQTT**: drop + audit-log (spoof guard).
- **Bulk registration duplicates**: atomic batch, whole-batch reject on any dup IMEI (VERIFY AT IMPLEMENTATION batch semantics).
- **Snapshot enrichment backfill**: existing `telemetry_snapshots` rows get NULL new columns — consumers must tolerate NULL (`COALESCE`); VERIFY AT IMPLEMENTATION whether alerting/fuel specs require backfill.

---

## 12. Phased Rollout

- **Phase 1 — Core pipeline + own-device MQTT/HTTP (this spec, priority):** migrations 00039/00040, `contracts.go`, `ingest.go`, real MQTT subscriber, REST fallback, outbox events (`PositionEvent`/`AlertEvent`/`SOSEvent`), raw-SQL stores. Replaces the stub + no-op sync with no API break (`/sync`, `/snapshots` URLs unchanged).
- **Phase 2 — Device registry & admin UI:** provisioning endpoints, bulk registration, quarantine queue pages (Datastar), RBAC `telemetry:*`, audit integration.
- **Phase 3 — Mobile driver-app retrofit:** `mqtt.ts`/`syncEngine.ts`/`telemetry.ts` → canonical frames; topic migration bridge for old `avandab/telemetry/drivers/{driverId}/gps` (VERIFY AT IMPLEMENTATION: dual-subscribe window length).
- **Phase 4 — Third-party adapters (no build priority):** `ProviderIngestor` + LocoNav/WheelsEye stubs, webhook route, poll loop + backoff.
- **Phase 5 — Live-tracking push:** consumed by the live-tracking spec's own transport (no WebSocket hub exists — verified; each consumer spec builds its own push).

---

## 13. Open Items — VERIFY AT IMPLEMENTATION

1. Mobile `publishLocation` has **no callers** in `mobile/src` — which screen/effect must invoke it for the driver-app GPS to actually stream (currently only offline SQLite + `/sync` are exercised).
2. Driver-app device identity: driver phones have no IMEI — synthetic IMEI registration vs `driver_id`-carried identity in `RawFrame.IMEI`; decide with mobile spec.
3. MQTT broker deployment + ACL capability (no broker in docker-compose/deploy scripts today; default `tcp://localhost:1883`).
4. `device_secret_hash` algorithm for low-power devices: bcrypt vs HMAC-pepper.
5. LocoNav `User-Authentication` value semantics (raw secret vs derived); webhook JSON shape for DTC/fuel-theft alert kinds (mapping to `AlertEvent`).
6. WheelsEye vehicle→IMEI mapping when `deviceNumber` absent (join via `vehicles.registration_number`/`vehicle_number`).
7. Outbox write mechanism for high-volume ingestion: `OutboxWriter.SaveEvents` vs direct `INSERT INTO outbox_events` in the same tx (writer is the established pattern; volume may favor direct insert + explicit `published_at`).
8. `telemetry_snapshots` id scheme migration (timestamp-based `snap-<ts>` ids → uuid) and NULL-column tolerance in consumers.
9. Snapshot/position writes for frames without a vehicle assignment (inventory devices) — drop vs store with NULL `vehicle_id` vs quarantine (`reason: unassigned_device`).
10. Datastar version pin: `internal/static/js/datastar.js` exists and `layout.html` uses `?v={{.Version}}` (default 20260812); the v1.0.2 claim came from context — confirm exact version before relying on v1.0.2-only APIs.
11. `TELEMETRY_WEBHOOK_RATE_LIMIT` window semantics (fixed 1-minute window per `ratelimit.go`) sufficient for provider bursts (LocoNav retry storms).
12. Production TLS hard-fail policy for `MQTT_URL` scheme.
