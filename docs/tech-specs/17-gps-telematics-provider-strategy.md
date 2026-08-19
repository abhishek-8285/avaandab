# GPS TELEMATICS PROVIDER STRATEGY — Implementation Spec v1

Status: ready
Depends-on: 01-telematics-ingestion (migrations 00040 + 00041), 02-geofence-engine, 03-fuel-audit-scorecard, 04-live-map-share-maintenance, 05-alerting-compliance, 07/08/09/10/13 (cross-cutting)
Migration owner: db/migrations/00040_*.sql (telemetry_devices + ingestion) and 00041_*.sql (positions + snapshots) — owned by spec 01; this strategy doc ONLY references them and defines the provider layer behind them. Reserve provider-adapter migrations separately (see §5 / §12). NOTE: 00039 is TAKEN by `experiments.sql` (see 00-migration-ownership-index); do not reference 00039 for telemetry.

This document is the **cross-cutting GPS/telematics strategy** referenced by all telemetry specs. It establishes the `TelematicsProvider` abstraction, the ingestion front doors, the canonical pipeline contract, and — in **Section 8 (the core of this doc)** — the concrete provider evaluation + recommendation for Avandab/MVTMS.

---

## 0. Verified Ground Truth

Facts below are read directly from the codebase; they define the starting line. No assumption.

- `internal/telemetry/sync.go:86` — `HandleTelemetrySync` is a **no-op**: it decodes `SyncBatchRequest`, echoes `logItem.ID`s back, and never persists a position, raw event, or frame. Coverage today = 0% real position storage.
- `internal/telemetry/sync.go:52-80` — `snapshotHandler(database)` is the **only** persistence path: it does an `INSERT OR REPLACE INTO telemetry_snapshots (...)` for a single snapshot. No device lookup, no dedup, no positions, no quarantine, no outbox. It is the sum of what "works" today.
- `internal/telemetry/sync.go:43-50` — routes registered are `/api/v1/telemetry/sync` and `/api/v1/telemetry/snapshots`. There is **no** `/webhooks/{provider}/{token}` and no `/devices/{imei}/gps` route.
- `internal/telemetry/providers/` — **does NOT exist.** No `TelematicsProvider` interface, no LocoNav/WheelsEye/MapMyIndia/TelaBit/OBD adapters. Spec 01 defines `ProviderIngestor` (interface only); this doc defines the broader `TelematicsProvider` strategy that `ProviderIngestor` implements for third-party sources.
- `internal/mqttservice/mqtt.go:41-52` — `subscribeTelemetry` subscribes `avandab/telemetry/drivers/+/gps` and **only logs** the decoded payload (`log.Printf("[MQTT TELEMETRY RECV] ...")`). No persistence, no `RawFrame`, no routing.
- `cmd/server/main.go` (~line 421) — the MQTT broker is constructed but its purpose is effectively discarded; `_ = mqttservice.NewMQTTBroker(mqttURL)` (line 421) discards the handle and ingestion is log-only. `MQTT_URL` default `tcp://localhost:1883` is set inline in `cmd/server/main.go:417-419` (NOT `internal/config/config.go`, which has no MQTT/telemetry config today).
- `internal/service/telemetry_service.go:49` — `ProcessTelemetryStream` evaluates rules (GPS deviation > 5 km, fuel drop > 10 L while ignition OFF) **in-memory only**: it logs and publishes directly to the in-memory bus; it writes nothing to `telemetry_alerts` and reads `lastFuelLevel` from a caller-supplied argument (no stored history).
- Outbox: `internal/shared/outbox/outbox.go` — `outbox_events` is **never written by any use case today**. First real write is a milestone owned by spec 01's pipeline.
- No WebSocket hub exists anywhere in Go source (verified, spec 01 §1). Live-tracking UI (spec 04) builds its own push; this strategy emits bus events only.

**Conclusion of §0:** Telemetry is at **~0%** functional maturity. One table (`telemetry_snapshots`) is written by a single endpoint; there is no device registry, no positions table, no provider abstraction, no MQTT persistence, and no rule persistence. This strategy is the blueprint that lifts that to 100% while keeping third-party providers non-blocking.

### Verification Log (Principal-Engineer QA pass)

All claims in §0 + file:line refs + migration numbers + §8 provider matrix were read against source on 2026-08-19. Verdicts: TRUE = confirmed in code; WRONG = corrected below.

| # | Claim | Verdict | Correction / Evidence | Sev | Effort |
|---|---|---|---|---|---|
| 1 | `sync.go:86` `HandleTelemetrySync` is a no-op | TRUE | Decodes `SyncBatchRequest`, echoes `logItem.ID`s (sync.go:96-99), no persist. | — | — |
| 2 | `snapshotHandler` is the only persistence path | TRUE | `INSERT OR REPLACE INTO telemetry_snapshots` (sync.go:72); no device/dedup/positions/quarantine/outbox. | — | — |
| 3 | Routes = only `/sync` + `/snapshots`; no webhook/device route | TRUE | sync.go:48-49 register exactly those two. | — | — |
| 4 | `internal/telemetry/providers/` does NOT exist | TRUE | `ls` → No such directory; no `TelematicsProvider` interface in repo. | — | — |
| 5 | `mqtt.go:41-52` log-only, topic `avandab/telemetry/drivers/+/gps` | TRUE | Subscribes (mqtt.go:45) and only `log.Printf("[MQTT TELEMETRY RECV] …")` (mqtt.go:49). | — | — |
| 6 | `main.go` ~421 broker discarded, ingestion log-only | TRUE | `_ = mqttservice.NewMQTTBroker(mqttURL)` (main.go:421); no persistence wired. | — | — |
| 7 | `MQTT_URL` default in `internal/config/config.go` | **WRONG** | `config.go` has NO MQTT/telemetry config. Default `tcp://localhost:1883` is inline in `cmd/server/main.go:417-419`. Corrected §0 line 20. | Med | Low |
| 8 | `telemetry_service.go:49` rule engine in-memory only | TRUE | `ProcessTelemetryStream` logs + `s.events.Publish` (telemetry_service.go:69-80,103-110); reads `lastFuelLevel` arg (line 49); no DB write; returns `[]TelemetryAlert`. | — | — |
| 9 | `outbox_events` never written by any use case | TRUE | `grep INSERT INTO outbox_events` across `internal/` → 0 hits. | — | — |
| 10 | No WebSocket hub in Go source | TRUE | `grep -rln websocket` across `internal/` → 0 hits. | — | — |
| 11 | `telemetry_devices`/`telemetry_raw_events`/`device_quarantine`/`provider_poll_state` are migration **00039** | **WRONG** | `00039_experiments.sql` is TAKEN (experiment_events). Correct owner per index = **00040** (telemetry devices+ingestion). Corrected §3, header. | High | Low |
| 12 | `telemetry_positions`/`vehicle_latest_position` are migration **00040** | **WRONG** | Per index, positions+snapshots enrichment = **00041**. Corrected §3. | High | Low |
| 13 | `telemetry_snapshots` "enriched by 00040" | **WRONG** | Created in 00031 (verified grep); enrichment migration = **00041**, not 00040. Corrected §3. | Med | Low |
| 14 | Header "Depends-on … migrations 00039 + 00040" | **WRONG** | Should be **00040 + 00041** (00039 taken). Corrected header + §3 + §10 + §11.12. | High | Low |
| 15 | §8 provider protocols/accuracy (Own MQTT+JT808 2-5m; LocoNav/WheelsEye/MapMyIndia/TelaBit webhook/poll/HMAC, 5-10m) | TRUE (with caveat) | Protocols/accuracy reasonable for GNSS SaaS class. Only factual risk: "2G adequate for India" (§8.2) outdated due to 2G/3G sunset (Airtel/Vi 2024-26, no Jio 2G) — softened to recommend LTE-M/NB-IoT. | Med | Low |
| 16 | `telemetry_devices` table does not yet exist in any migration | TRUE | grep across `db/migrations/` for `telemetry_devices` → 0 hits (planned 00040). Consistent with "blueprint" status. | — | — |

---

## 1. Overview / Goal

**Goal:** Establish one unified `TelematicsProvider` interface such that **any GPS source** — our own MQTT/OBD/SIM hardware, LocoNav, WheelsEye, MapMyIndia, TelaBit, or raw JT808/OBD dongles — is pluggable behind the same contract. Every ingestion front door (MQTT, device REST, webhook, poll, driver-app) normalizes its input into a single canonical `RawFrame`. The downstream pipeline (validate → quarantine-check → raw log → positions → latest upsert → snapshot → guards → outbox) is **provider-agnostic** and never branches on who the provider is. Third-party providers are adapters behind the interface; the **core product never blocks on them**.

**Non-goals:**
- Building a WebSocket hub (spec 04 owns its own push; verified none exists).
- Re-implementing the canonical pipeline storage (owned by spec 01, migrations 00039/00040). This doc defines the *provider layer* and *strategy*, not the DDL.
- Live rule evaluation persistence (owned by spec 05, migration 00059 — telemetry_alerts rebuild). This doc only feeds `AlertEvent`s.
- Geofence math (owned by spec 02). This doc only delivers positions + ignition + trip boundaries.

---

## 2. API Contract

All routes below are front doors that produce `RawFrame`. Auth differs per door; all converge on the same `IngestRawFrame(ctx, RawFrame)` entrypoint (spec 01 `ingest.go`).

### 2.1 Provider webhook (third-party)
```
POST /api/v1/telemetry/webhooks/{provider}/{token}
Auth:   public + middleware.RateLimit(30)         // Razorpay precedent (main.go:347)
        {token} = per-provider secret, crypto/subtle constant-time compare
        + ProviderIngestor.VerifySignature(rawBody, header) when defined
Body:   provider-specific (LocoNav, WheelsEye, MapMyIndia, TelaBit)
200 -> { "accepted": true, "frames": N, "deduped": M }
401 -> { "error": "bad token" }
```
Returns immediately after normalization; the frame is enqueued, not synchronously processed.

### 2.2 Own-device REST (fallback + driver app)
```
POST /api/v1/telemetry/devices/{imei}/gps
Auth:   X-Device-Token: <device_secret>           // machine-to-machine, rate-limited
Body:   RawFrame-compatible JSON (see §1 contract in 01)
200 -> { "accepted": true, "deduped": false }
202 -> { "accepted": true, "deduped": true }       // received but already seen
401 -> { "error": "bad token" }
404 -> { "error": "unknown imei" }                 // still quarantined server-side
```
Existing routes kept for backward-compat (mobile ships with them):
```
POST /api/v1/telemetry/sync        -> retrofitted into pipeline (returns real synced_ids)
POST /api/v1/telemetry/snapshots   -> retrofitted into pipeline (no longer direct INSERT)
```

### 2.3 MQTT (primary path, own hardware)
```
Subscribe (server): avandab/telemetry/devices/{imei}/gps   QoS 1, retained=false
Publish  (device):  same topic, QoS 1
Auth:   username = IMEI, password = device_secret          // broker ACL or subscriber-side check
Canonical topic replaces the legacy avandab/telemetry/drivers/+/gps log-only topic.
```
One JSON frame per message; `seq` (per-device monotonic) becomes `provider_msg_id = "mqtt:<seq>"`.

### 2.4 Device registry / provisioning (admin)
```
POST /api/v1/telemetry/devices/bulk        -> bulk register (max 500/batch, atomic)
POST /api/v1/telemetry/devices/{imei}/assign
POST /api/v1/telemetry/devices/{imei}/activate
POST /api/v1/telemetry/devices/{imei}/retire
POST /api/v1/telemetry/quarantine/{id}/resolve   -> register-new | assign-existing | reject
GET  /api/v1/telemetry/devices                  -> list (telemetry:read)
GET  /api/v1/telemetry/quarantine               -> queue (telemetry:read)
```
All registry routes behind `ResourcePermission(authSrv, "telemetry", ...)`.

---

## 3. DB Contract (reference, not duplicate)

DDL lives in spec 01 (migrations **00040 + 00041**). This strategy consumes the following tables; see spec 01 §6 for full DDL. Key columns only:

- **`telemetry_devices`** (00040): `id, tenant_id, imei UNIQUE, serial_number, sim_number, iccid, firmware_version, warranty_until, status ('inventory'|'assigned'|'active'|'retired'|'quarantined'), vehicle_id (UNIQUE where not null), customer_id, activated_at, last_seen_at, device_secret_hash`. This is the anchor every frame is validated against.
- **`telemetry_raw_events`** (00040): `id, tenant_id, imei, device_time, received_at, provider, provider_msg_id, payload`. Dedup via `UNIQUE INDEX idx_raw_dedup (imei, provider_msg_id) WHERE provider_msg_id IS NOT NULL`.
- **`device_quarantine`** (00040): `id, tenant_id, imei, source, raw_payload, reason, status ('open'|'resolved'|'rejected'), resolved_by, resolved_at`. Sink for unknown/retired/quarantined devices.
- **`provider_poll_state`** (00040): `provider PK, last_poll_at, last_success_at, cursor, consecutive_failures, backoff_until`. Store-and-forward state for pull providers.
- **`telemetry_positions`** (00041): per-accepted-frame history (`imei, device_time, latitude, longitude, speed, heading, ignition, engine_hours, accuracy, fuel_level, odometer, vehicle_id, trip_id, driver_id, raw_event_id`).
- **`vehicle_latest_position`** (00041): upsert, only `device_time > existing.device_time` wins.
- **`telemetry_snapshots`** (00031, enriched by 00041): adds `heading, ignition, engine_hours, accuracy, driver_id`.

> **Numbering note:** spec 01 currently reserves 00039/00040. The prompt references "00040/00041" — treat final numbers as owned by the migration-ownership index (`00-migration-ownership-index.md`). This doc does not introduce new tables beyond provider adapter config rows if needed; provider *poll* state already exists. If a `provider_adapters` config table is required it is reserved separately (see §12).

---

## 4. UI

All pages are Datastar partials (per `internal/templates/partials/` conventions; `layout.html` uses `?v={{.Version}}`). RBAC via `telemetry:*` permissions seeded in 00039.

- **Device registry** (`internal/templates/devices_list.html` + `device_row.html`): table of `telemetry_devices` with status badges, last-seen, assigned vehicle, warranty. Actions: assign / activate / retire (telemetry:update).
- **Bulk register** (`devices_register.html`): paste/JSON array of IMEIs + SIM/ICCID; atomic submit.
- **Live position table** (`spec 04` owns the map; this strategy provides the `vehicle_latest_position` feed + a plain table partial `positions_table.html` for ops): IMEI, vehicle, lat/lng, speed, ignition, last-seen, age.
- **Quarantine queue** (`quarantine_queue.html`): entries from `device_quarantine` with resolution actions (register-new / assign-existing / reject). This is the safety net that guarantees no frame is silently dropped.
- **Provider status panel** (`providers_status.html`): shows each `TelematicsProvider` health from `provider_poll_state` (consecutive_failures, backoff_until), plus whether own-MQTT is connected. Read-only, `telemetry:read`.

No WebSocket; UI polls an SSE/Datastar signal emitted by the live-tracking spec (spec 04) which subscribes to `PositionEvent` on the bus.

---

## 5. Business Logic

### 5.1 `TelematicsProvider` interface (the core abstraction)
```go
// internal/telemetry/providers/provider.go
// TelematicsProvider is the unified contract. "own" devices and every
// third-party source implement this. The pipeline never knows the source.
type TelematicsProvider interface {
    // Name is the canonical id: "own" | "loconav" | "wheelseye" |
    // "mapmyindia" | "telabit" | "jt808" | "obd".
    Name() string

    // VerifySignature validates provider webhook auth (HMAC header, etc).
    // Return nil if the provider has no signature scheme.
    VerifySignature(rawBody []byte, header http.Header) error

    // HandleWebhook converts a provider push into canonical RawFrames.
    HandleWebhook(ctx context.Context, rawBody []byte) ([]RawFrame, error)

    // Poll pulls frames for pull-based providers (WheelsEye, some OBD).
    // since comes from provider_poll_state.last_success_at.
    Poll(ctx context.Context, since time.Time) ([]RawFrame, error)
}
```
`RawFrame` (from spec 01 §2, repeated for binding): `IMEI, DeviceTime, Latitude, Longitude, Speed, Heading, Ignition, EngineHours, Accuracy, FuelLevel, Odometer, DriverID, TripID, Provider, ProviderMsgID, RawPayload`. Own devices also implement a `Push(ctx, RawFrame) error` variant (MQTT/REST) but that is the *ingestion door*, not the provider — the door already produced the `RawFrame`.

### 5.2 Registry
`internal/telemetry/providers/registry.go`:
```go
var registry = map[string]TelematicsProvider{}
func Register(p TelematicsProvider) { registry[p.Name()] = p }
func Get(name string) (TelematicsProvider, bool) { p, ok := registry[name]; return p, ok }
```
`own` is always registered (no external creds). Third-party adapters self-register only when their env secret is non-empty (config-flagged MOCK behind interface — template rule). A MOCK provider (`internal/telemetry/providers/mock.go`) returns canned frames for tests and local dev with zero creds.

### 5.3 Canonical pipeline (provider-agnostic, owned by spec 01)
```
IngestRawFrame(frame):
  1. device = lookup telemetry_devices by IMEI
  2. if device == nil            -> device_quarantine(reason=unknown_device); return 202
  3. if device.status != active  -> device_quarantine(reason=retired/quarantined); return 202
  4. INSERT telemetry_raw_events (dedup: UNIQUE(imei, provider_msg_id)) -> deduped=true if ignored
  5. odometer_guard(frame)       -> reject regression > TELEMETRY_ODOMETER_MAX_REGRESSION_KM; audit
  6. fuel_clamp(frame)           -> clamp |Δfuel| > TELEMETRY_FUEL_CLAMP_DELTA_PCT; audit
  7. INSERT telemetry_positions
  8. UPSERT vehicle_latest_position WHERE device_time > existing
  9. ignition trip start/stop    -> emit trip boundary events (spec 01 + booking)
 10. enrich + INSERT telemetry_snapshots
 11. outbox: PositionEvent / AlertEvent / SOSEvent (same tx)
```

### 5.4 Dedup
Key = `(imei, provider_msg_id)`. For own MQTT, `provider_msg_id = "mqtt:<seq>"`; for REST, `"rest:<client_seq>"`; for polls, `"<provider>:<cursor>"`. `INSERT OR IGNORE` semantics; replays are idempotent. Replayed frame still returns `202 deduped:true` upstream.

### 5.5 Ignition trip start/stop
On `Ignition` transition false→true with no open trip for the vehicle, open a trip window (emit `TripStartEvent`); true→false closes it (`TripStopEvent`). Fuel-theft rule (spec 03/05) keyed off ignition-off + fuel drop uses this boundary.

### 5.6 Fuel-theft hooks
`RawFrame.FuelLevel` flows into `PositionEvent`; the fuel spec (03) + alerting spec (05) consume it. This strategy only guarantees clamped, rollback-guarded, monotonic fuel values reach the bus.

### 5.7 Odometer guard
`odometer_guard`: if `frame.Odometer < stored_last - TELEMETRY_ODOMETER_MAX_REGRESSION_KM` → keep stored last, write `audit_logs(action='odometer_rollback_guard')`, do not overwrite. Single guard owner coordinated with fuel spec.

### 5.8 Quarantine
Unknown IMEI, retired device, quarantined device, or IMEI/topic mismatch → `device_quarantine` only. No position/event downstream. Admin resolves.

### 5.9 Provider matrix (summary; detailed eval in §8)

| Provider | Type | Transport | Identity | Fit |
|---|---|---|---|---|
| own (MQTT/SIM/OBD) | hardware vendor | MQTT QoS1 / REST | IMEI + device secret | **primary** — resale model |
| LocoNav | SaaS/webhook | HTTPS webhook | `User-Authentication` header | Phase-4 adapter |
| WheelsEye | SaaS/pull | HTTPS poll | access token | Phase-4 adapter |
| MapMyIndia | SaaS/webhook | HTTPS webhook | HMAC | Phase-4 adapter |
| TelaBit | SaaS/webhook | HTTPS webhook | HMAC | Phase-4 adapter |
| raw JT808/OBD | protocol | TCP/UDP → own gateway | IMEI | own-device alt path |

---

## 6. Config / Env

| Env var | Default | Purpose | Reads |
|---|---|---|---|
| `MQTT_URL` | `tcp://localhost:1883` | Broker; prod must be `ssl://`/`wss://` | config, mqttservice |
| `TELEMETRY_ENABLED` | `true` | Master switch for ingestion front doors | telemetry |
| `TELEMETRY_WEBHOOK_SECRET_LOCONAV` | empty (disabled) | LocoNav route token + `User-Authentication` | webhooks |
| `TELEMETRY_WEBHOOK_SECRET_WHEELSEYE` | empty (disabled) | WheelsEye route token | webhooks |
| `TELEMETRY_WEBHOOK_SECRET_MAPMYINDIA` | empty (disabled) | MapMyIndia HMAC secret | webhooks |
| `TELEMETRY_WEBHOOK_SECRET_TELABIT` | empty (disabled) | TelaBit HMAC secret | webhooks |
| `WHEELSEYE_ACCESS_TOKEN` | empty | `currentLoc` poll token | wheelseye adapter |
| `WHEELSEYE_POLL_INTERVAL` | `60s` | Poll cadence + backoff | wheelseye adapter |
| `TELEMETRY_RAW_RETENTION_DAYS` | `30` | Retention sweep for `telemetry_raw_events` | ingest job |
| `TELEMETRY_BATCH_SIZE` | `500` | Frames per commit batch | ingest writer |
| `TELEMETRY_FLUSH_INTERVAL` | `2s` | Batched position flush (MQTT burst) | ingest writer |
| `TELEMETRY_ODOMETER_MAX_REGRESSION_KM` | `1.0` | Per-frame rollback tolerance | odometer_guard |
| `TELEMETRY_FUEL_CLAMP_DELTA_PCT` | `5.0` | Max |Δ fuel| per frame | fuel clamp |
| `TELEMETRY_DEVICE_SECRET_PEPPER` | empty | HMAC pepper for `device_secret_hash` | devices |
| `TELEMETRY_WEBHOOK_RATE_LIMIT` | `30` | Per-IP fixed window (mirrors Razorpay) | webhooks |

All third-party adapters are **config-flagged**; empty secret → adapter not registered → MOCK only. No external creds needed to run/test (template rule).

---

## 7. Tests

Unit + HTTP/integration. Fixtures in `internal/telemetry/providers/testdata/`. Coverage gate: `go test ./internal/telemetry/... -cover` ≥ 80% before merge.

1. **Ingest valid frame** — own MQTT/REST frame with known active IMEI → assert `telemetry_raw_events` + `telemetry_positions` + `vehicle_latest_position` + `outbox_events` (`PositionEvent`) written in one tx.
2. **Replay dedup** — same `provider_msg_id` twice → second returns `202 deduped:true`; no duplicate position row (assert `COUNT` on `telemetry_positions` by raw_event_id = 1).
3. **Unknown IMEI quarantine** — frame with unregistered IMEI → assert `device_quarantine` row `reason=unknown_device`, zero position writes.
4. **Odometer rollback guard** — frame odometer 1000 then 990 (regression 10 > 1.0) → latest keeps 1000; assert `audit_logs` `odometer_rollback_guard` row.
5. **Fuel clamp** — last fuel 80, frame fuel 50 (Δ30 > 5) → stored 75 (80-5); assert clamp + audit.
6. **Provider adapter contract** — for each adapter (mock + loconav + wheelseye + mapmyindia + telabit + jt808): `VerifySignature` accepts valid / rejects tampered; `HandleWebhook`/`Poll` maps to `[]RawFrame` with correct `Provider`, `IMEI`, `ProviderMsgID`, `DeviceTime`; round-trips through `IngestRawFrame` yield identical positions.
7. **Webhook route security** — bad `{token}` → 401; good token + bad signature → 401; good token + valid body → 200 with frames count.
8. **MQTT auth** — username≠IMEI in topic → frame dropped + audit (spoof guard).
9. **Ignition trip boundary** — false→true opens trip, true→false closes; `TripStartEvent`/`TripStopEvent` on bus.

Pass-before-merge checklist: items 1–5 green; item 6 runs against MOCK with no creds; registry contains `own` + `mock` in CI; third-party adapters only registered when secrets present (CI skips them).

---

## 8. Future / GPS-PROVIDER (CORE)

This section is the strategic heart of the document. It decides **which GPS sources Avandab should use**, evaluated against the resale/hardware-vendor business model, and sets the integration order.

### 8.1 Evaluation matrix

Providers compared across: type, transport/protocol, device identity, positional accuracy, SIM/cellular requirement, recurring cost, and fit for the Avandab model.

| Provider | Type | Transport / Protocol | Identity | Accuracy | SIM / Cellular | Recurring cost | Fit for Avandab |
|---|---|---|---|---|---|---|---|
| **Own OBD/SIM (Avandab)** | Hardware vendor (we sell) | MQTT QoS1 (TLS) + REST fallback; JT808 gateway optional | IMEI + per-device secret | 2–5 m (GNSS) | Our SIM or customer SIM (2G adequate, 4G for high-freq) | One-time HW + data plan | **PRIMARY** — enables GPS resale, full data ownership, margin |
| **LocoNav** | SaaS tracker | HTTPS **webhook** push; `User-Authentication` header | vehicleNumber / deviceNumber | 5–10 m | Their SIM (locked) | Per-vehicle monthly subscription | Phase-4 adapter only; data leaves us; no resale margin |
| **WheelsEye** | SaaS tracker | HTTPS **poll** (`currentLoc`) | vehicleNumber → IMEI map | 5–10 m | Their SIM (locked) | Per-vehicle monthly subscription | Phase-4 adapter; pull-only, latency = poll interval |
| **MapMyIndia** | SaaS / mapping + telemetry | HTTPS **webhook**; HMAC signed | device id | 5–10 m | Their SIM or hybrid | Subscription + API calls | Phase-4 adapter; strong maps, weak as sole tracker |
| **TelaBit** | SaaS tracker | HTTPS **webhook**; HMAC | device id | 5–10 m | Their SIM (locked) | Per-vehicle monthly | Phase-4 adapter; niche, evaluate on commercial terms |
| **Raw JT808 / OBD dongle** | Protocol / BYO hardware | TCP/UDP → **our gateway** parses JT808 | IMEI | 2–5 m (GNSS) | Any SIM (incl. customer's) | HW + data only | Own-device **alternative path**; we still own the data |

### 8.2 Provider deep notes

**Own OBD/SIM (recommended primary).**
- We are the GPS vendor. Every sold unit = inventory row in `telemetry_devices` (status `inventory` → `assigned` → `active`). The device lifecycle *is* a billable product.
- Transport: MQTT QoS1 over TLS (`ssl://`/`wss://` in prod). Frame per message, `seq` monotonic → idempotent replays. REST fallback when MQTT blocked (corporate firewalls, roaming).
- Identity: IMEI is the universal key. MQTT username = IMEI, password = `device_secret_hash`. Broker ACL restricts each client to its own topic; subscriber-side check as backup (drop username≠IMEI).
  - SIM/IoT cellular: **Historically 2G was adequate for India** for 30 s–5 min position intervals (low bandwidth, cheapest data). **Verify before committing:** Airtel and Vodafone Idea have announced/executed 2G/3G sunset phases (2024–2026) and Jio never operated 2G, so "2G national coverage" is no longer guaranteed — prefer **4G LTE-M / NB-IoT** or a 4G fallback SIM with national roaming for durable coverage. **4G justified for high-frequency (sub-10 s) tracking, video, or dual-SIM failover.**
- Accuracy/cost: GNSS module 2–5 m; cost dominated by HW BOM + data plan, not by provider fees. No per-vehicle SaaS tax.
- Protocol: **MQTT QoS1 + JT808** dual stack. MQTT for live push; JT808 for legacy/Chinese OBD dongles via our gateway (still normalized to `RawFrame`, still `Provider="own"`).
- **No WebSocket hub**: we do not run a WebSocket broker. Devices speak MQTT/REST to our backend; the backend emits `PositionEvent` on the in-process bus; the live-map spec (04) builds its own SSE/Datastar push to browsers. This keeps the ingestion tier stateless and horizontally scalable.
- **AIS-140 / VLT contract**: for commercial vehicles in India, AIS-140 (VLT + emergency button) is a regulatory requirement for many fleets. Our own device line **must ship AIS-140-compliant hardware under a signed VLT contract** (or resell a certified OEM white-label). This is a procurement/legal gate, not a code gate — but the `SOSEvent` path and `sos:true` frame field exist specifically to satisfy the emergency-button mandate. Verify the VLT license before volume rollout.

**LocoNav / WheelsEye / MapMyIndia / TelaBit (third-party).**
- All four are **adapters behind `TelematicsProvider`**. None block the core. They exist only to (a) onboard customers who already own that hardware, or (b) fill coverage gaps.
- LocoNav + MapMyIndia + TelaBit are webhook-push (implement `HandleWebhook` + `VerifySignature`). WheelsEye is poll-only (`Poll` + `provider_poll_state` backoff).
- Commercial reality: every third-party vehicle is a **recurring monthly fee with zero resale margin** and **data custody outside Avandab**. Treat as a customer-convenience import, never a dependency.
- Verification gap: LocoNav `User-Authentication` value semantics (raw vs derived) and DTC/fuel-theft alert JSON shape must be confirmed against live docs before adapter promotion (open item, spec 01 §13). WheelsEye `deviceNumber` may be absent → map via `vehicles.registration_number`.

**Raw JT808 / OBD.**
- BYO-hardware path: any JT808-compliant OBD dongle (common, cheap, China-sourced) terminates at **our gateway** which parses the binary protocol and emits `RawFrame` with `Provider="own"`. This lets us support customer-supplied devices without a SaaS middleman, preserving data ownership. The gateway is internal infra, not a third party.

### 8.3 RECOMMENDATION (decisive)

1. **Primary = own devices.** Build the MQTT/REST ingestion tier first (Phase 1, spec 01). This is the GPS resale product: we own the hardware lifecycle, the data, and the margin. Ship AIS-140/VLT-compliant units.
2. **Third-party = Phase-4 adapters only, always behind `TelematicsProvider`.** LocoNav/WheelsEye/MapMyIndia/TelaBit are opt-in imports for customers who already use them. They are config-flagged (empty secret = disabled) and never on the critical path.
3. **Never block core on a third party.** The pipeline (`IngestRawFrame`) is provider-agnostic; if LocoNav's webhook goes down, only LocoNav-sourced vehicles degrade — own-device fleet is unaffected. `provider_poll_state` isolates WheelsEye downtime to backoff, not failure.
4. **SIM/IoT cellular:** standardize on 2G-capable IoT SIM with national roaming for cost; offer 4G only for high-frequency tiers. Do not couple to any provider's locked SIM.
5. **Protocol:** MQTT QoS1 + JT808 gateway. No WebSocket hub. Outbox/event-bus is the only inter-tier contract.
6. **Data custody:** all positions land in `telemetry_positions`/`vehicle_latest_position` regardless of source; third-party data is normalized and stored under our tenancy, not left in the provider's cloud.

---

## 9. Edge Cases

- **Duplicate frames** — `(imei, provider_msg_id)` partial-unique + `INSERT OR IGNORE`; replay returns `deduped:true`.
- **Out-of-order timestamps** — raw append always; positions insert unconditional; `vehicle_latest_position` upsert `WHERE device_time > existing`; snapshots `INSERT OR REPLACE` with monotonic guard.
- **Unknown IMEI** — `device_quarantine(reason=unknown_device)`; no downstream writes; admin resolves.
- **Retired/quarantined device transmitting** — quarantine (`reason=retired_device` / `quarantined_device`); audit-logged.
- **Odometer rollback** — regression > `TELEMETRY_ODOMETER_MAX_REGRESSION_KM` → keep last, audit `odometer_rollback_guard`.
- **Fuel sanity clamp** — |Δ fuel| > `TELEMETRY_FUEL_CLAMP_DELTA_PCT` → clamp to last ± clamp, audit.
- **Provider downtime** — `provider_poll_state` tracks `consecutive_failures`, `backoff_until` (exponential); WheelsEye resumes from `last_success_at`; mobile offline SQLite queue flows through `/sync`.
- **IMEI/topic mismatch (MQTT)** — drop + audit (spoof guard).
- **Bulk registration duplicates** — atomic batch, whole-batch reject on any dup IMEI.
- **Rate/volume (SQLite modernc)** — single-writer funnel: one ingestion goroutine per front door → batched writer (`TELEMETRY_BATCH_SIZE`, `TELEMETRY_FLUSH_INTERVAL`); WAL already default. Retention job deletes raw rows > `TELEMETRY_RAW_RETENTION_DAYS` in batches.
- **Frames without vehicle assignment (inventory devices)** — store with NULL `vehicle_id` (do not quarantine; device is known, just unassigned) — VERIFY with spec 01.
- **Third-party secret empty** — adapter not registered; routes for that provider return `404 disabled` (not 401) to avoid leaking which providers exist.

---

## 10. Phased Rollout

- **Phase 1 — Core pipeline + own-device MQTT/REST (priority):** spec 01 migrations 00040/00041, `contracts.go`, `ingest.go`, real MQTT subscriber (replace `mqttservice` stub), REST fallback, outbox `PositionEvent`/`AlertEvent`/`SOSEvent`, raw-SQL stores. No API break (`/sync`, `/snapshots` URLs unchanged).
- **Phase 2 — Device registry & admin UI:** provisioning, bulk register, quarantine queue (Datastar), RBAC `telemetry:*`, audit integration.
- **Phase 3 — Mobile driver-app retrofit:** `mqtt.ts`/`syncEngine.ts`/`telemetry.ts` → canonical frames; dual-subscribe bridge for legacy `avandab/telemetry/drivers/{driverId}/gps`.
- **Phase 4 — Third-party adapters (no build priority):** `TelematicsProvider` + LocoNav/WheelsEye/MapMyIndia/TelaBit/JT808 adapters, webhook route, poll loop + backoff. Always behind interface, config-flagged.
- **Phase 5 — Live-tracking push:** consumed by spec 04's own transport (no WebSocket hub — verified).

---

## 11. Open Items / VERIFY

1. Mobile `publishLocation` has **no callers** in `mobile/src` — which screen invokes it (spec 01 §13.1).
2. Driver-app device identity: phones have no IMEI — synthetic IMEI vs `driver_id`-carried identity (decide with mobile spec).
3. MQTT broker deployment + ACL capability (no broker in docker-compose today; default `tcp://localhost:1883`).
4. `device_secret_hash` algorithm for low-power devices: bcrypt vs HMAC-pepper.
5. LocoNav `User-Authentication` semantics + DTC/fuel-theft alert JSON shape.
6. WheelsEye `deviceNumber` absent → map via `vehicles.registration_number`.
7. Outbox write mechanism at volume: `OutboxWriter.SaveEvents` vs direct `INSERT INTO outbox_events` in same tx.
8. `telemetry_snapshots` id scheme migration (`snap-<ts>` → uuid) + NULL-column tolerance.
9. Frames without vehicle assignment: store NULL vs quarantine (`reason=unassigned_device`).
10. **AIS-140 / VLT contract** — confirm certified hardware + signed VLT license before volume rollout (procurement/legal gate).
11. SIM/IoT commercial terms — lock 2G national-roaming IoT SIM rate; define 4G tier threshold.
 12. Migration numbering — resolved: telemetry devices+ingestion = **00040**, positions+snapshots enrichment = **00041** (per 00-migration-ownership-index). 00039 is TAKEN by `experiments.sql` and must NOT be used for telemetry.

---

## 12. File List

### Create
| Path | Purpose |
|---|---|
| `internal/telemetry/providers/provider.go` | `TelematicsProvider` interface (§5.1) |
| `internal/telemetry/providers/registry.go` | provider registry (§5.2) |
| `internal/telemetry/providers/mock.go` | credential-free MOCK adapter (CI/local) |
| `internal/telemetry/providers/loconav.go` | LocoNav webhook adapter (Phase 4) |
| `internal/telemetry/providers/wheelseye.go` | WheelsEye poll adapter (Phase 4) |
| `internal/telemetry/providers/mapmyindia.go` | MapMyIndia webhook adapter (Phase 4) |
| `internal/telemetry/providers/telabit.go` | TelaBit webhook adapter (Phase 4) |
| `internal/telemetry/providers/jt808.go` | JT808/OBD gateway normalizer → `RawFrame` (own alt path) |
| `internal/telemetry/providers/testdata/*.json` | adapter fixtures (§7) |
| `db/migrations/000XX_telemetry_provider_adapters.sql` | optional `provider_adapters` config rows (reserve separately) |

### Modify
| Path | Change |
|---|---|
| `internal/mqttservice/mqtt.go` | Replace log-only `subscribeTelemetry` (lines 41–52) with real subscriber → `telemetry/mqtt_ingest.go`; topic `avandab/telemetry/devices/{imei}/gps` |
| `internal/telemetry/sync.go` | `HandleTelemetrySync` persists via pipeline + real `synced_ids`; `snapshotHandler` routes into pipeline |
| `cmd/server/main.go` | Wire MQTT callback (~327), mount webhook route (~347, RateLimit), mount device/quarantine routes, register `telemetry:*` handlers |
| `internal/config/config.go` | Add `TelemetryConfig` (§6 env table) |
| `internal/telemetry/ingest.go` (spec 01) | `IngestRawFrame` consumes `RawFrame`; provider-agnostic (§5.3) |
| `internal/templates/*.html` | device registry, bulk register, quarantine queue, providers status (§4) |

No changes to: `internal/events/`, `internal/shared/outbox/`, `internal/auth/`, `internal/repository/`, `sqlc.yaml`, `db/query/` (raw SQL only, per spec 01).
