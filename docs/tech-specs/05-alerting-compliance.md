# ALERTING + COMPLIANCE + E-WAY BILL + AIS-140 READINESS — Implementation Spec v2

**Project:** Avandab fleet system — `/home/abhishek/Desktop/temux/basic`
**Stack (verified):** Go 1.26, chi v5, SQLite (modernc) via `database/sql`, goose embedded migrations (`db/migrations/`, embedded by `db/migrations.go`), Datastar v1.0.2 + Tailwind templates (`internal/templates/`), casbin RBAC (`internal/auth/casbin.go`), typed IDs (`internal/service/ids.go`, `internal/shared/id/`), UoW (`internal/shared/uow/`, `internal/shared/ports/uow.go`), outbox-relay event bus (`cmd/server/main.go:819-826`, `internal/shared/outbox/relay.go`), agent system (`internal/agent/`).
**Status:** Implementation-ready (updated — supersedes previous spec; incorporates decisions 1–6: migration ownership **00045–00046 (this spec)** with e-way bill 00047 + GST/RBAC 00048 owned by spec 07 and telemetry_alerts rebuild → **00059**, unified alerts pipeline, compliance gate both assignment paths + StartTrip, e-way bill lifecycle worker (consumes spec 07's 00047), SOS flow, AIS-140/VLT design contract, agent integration, telemetry_alerts rebuild).
**Scope:** Operational alerting pipeline, compliance dispatch gate, e-way bill lifecycle worker (consumes spec 07's 00047), SOS flow, AIS-140/VLT design contract, agent integration, RBAC seeds (folded into 00046), schema migrations **00045–00046 + telemetry_alerts rebuild 00059**.

---

## 0. Verified ground truth (re-checked against code)

| Claim | Verification |
|---|---|
| E-way bill client is a stub | `internal/integration/ewaybill/client.go:56-110` — `Generate/Get/Cancel` only; fake `EWB...` numbers, fake QR `data:image/png;base64,stubqrcode`, `ValidUpto = now+24h`. **No extension, no Part-B vehicle update.** |
| GSTN stub | `internal/integration/gstn/client.go:59-120` — fake GSTIN, fake GSTR-1/3B. |
| FASTag stub | `internal/integration/fastag/client.go:57-123` — fake balance 2475.50. |
| Accounting stub | `internal/integration/accounting/client.go:97-145` — fake `EXT-` IDs. |
| Integration HTTP shell is REAL | `internal/integration/handler.go` — routes `/api/v1/integrations/{ewaybill,gstn,fastag,accounting}/...`, permission-gated via `middleware.RequirePermission(h.authSrv, "integrations", "ewaybill")` etc. |
| MQTT stub | `internal/mqttservice/mqtt.go:31-52` — logs telemetry only. |
| gRPC stub | `internal/grpcservice/server.go` — `GetDriverLocation` returns hardcoded Pune coords (18.5204, 73.8567). |
| Notifications stub | `internal/operations/notifications/service.go:38-83` — `SendEmail` logs metadata only; `SendSMS/SendPush/SendWebhook` return "not configured yet" errors. In-memory map store. |
| Badge fed from audit logs | `internal/handlers/app.go:379-396` — `renderPage` pulls `a.Services.Audit.ListAuditLogs(...)` into `Notifications`/`UnreadCount`; rendered at `internal/templates/layout.html:164` (`#notif-badge`) and `:171`. |
| ComplianceService exists, UNWIRED | `internal/service/compliance_service.go` — `ValidateDriverCompliance`, `ValidateVehicleCompliance`, `EnforceDispatchCompliance`; only referenced by tests (`internal/service/services_test.go`, `test/master_integration_test.go`). Registered in `internal/service/service.go:54,90`. |
| Dead schema | `compliance_checks` (00030), `notifications` (00025), `eway_bills` (00031), `telemetry_alerts` (00030) — **zero** repository/query code references them (`internal/repository/` grep returns nothing). `TelemetryService` (`internal/service/telemetry_service.go:49-116`) generates alerts in-memory and publishes `GPSDeviationAlert`/`FuelTheftAlert` bus events but never persists. |
| files CHECK too narrow | `db/migrations/00001_initial.sql:67` — `CHECK (uploadable_type IN ('driver_license','vehicle_insurance','vehicle_permit','company_logo'))`. Subdir switch at `internal/service/file_service.go:88-98`. |
| Agent system | `internal/agent/orchestrator.go` (route → sub-agent → episode), `subagents.go` (booking/payments/kharcha/ops/support), `tools.go` (19 tools), `approval.go` (`Gate`/`GatedTool`/`MutatingTools()` at :62-64, admin identity swap at :80-84), `agent.go` (`normalizeArgs` :127-133), `handler.go` (`/assistant/chat`, `/api/agent/chat`), `approval_handler.go` (`/api/agent/actions`, approve/reject). RL: `internal/agent/rl/{service,store,reward}.go` — `SubmitAction`, `GetAction`, `UpdateActionDecision`, `SignalAction`, `RecordEpisode`, rewards `RewardActionExecuted=1.0 / ActionFailed=-1.5 / ActionRejected=-0.8`. |
| Branding | "FlyFleet Daily Report" at `internal/founder/digest/daily.go:22`; "FlyFleet account" at `internal/operations/audit/login_audit.go:70`. New alert templates must match — no "Avandab" in user-facing strings. |
| Assignment paths (BOTH) | **Web/API vertical-slice:** `internal/trip/application/assign_driver.go` (compliance check :66-82, license only), `assign_vehicle.go` (check :66-88, insurance/fitness/permit only), `start_trip.go`; wired in `internal/handlers/trips.go` (`AssignDriver` :308, `AssignVehicle` :325, `StartTrip` :342, Create auto-assign :164-178) and API handlers `internal/trip/presentation/api/handlers/trip_handler.go`. **Agent/legacy service path:** `internal/service/trip_service.go` `AssignDriver` :163, `AssignVehicle` :230, `StartTrip` :295 (agent tools `assign_driver`/`assign_vehicle` call these). |
| Event bus | `internal/events/bus.go` `InMemoryBus`; outbox relay polls `outbox_events` (00020) every 5s (`relay.go:50`), dead-letter after 5 attempts (`:142`). Outbox table is currently **never written** by any use case. Legacy services publish in-memory: `TripCreated`, `TripStarted` (trip_service.go:320), `TripCompleted`, `TripDelivered` (:419), `BookingCreated`/`BookingConfirmed` (booking_service.go:82,176), telemetry `GPSDeviationAlert`/`FuelTheftAlert`. `TripAssignedEvent` exists (`internal/domain/trip/events.go:28`) but is **never published**. |
| Migration head | HEAD = `00039_experiments.sql` (TAKEN by experiments — NOT free). `00040–00044` reserved by specs 01–04. **`00045–00046` owned by THIS spec; `00047` (e-way bill) + `00048` (GST) owned by spec 07; telemetry_alerts rebuild → `00059`.** |
| No geofence/SOS/AIS code | No `geofence*`, `SOS*`, or AIS-140 symbols anywhere in `internal/`, `db/`, `mobile/`. All three are **contracts defined here**, sourced from the ingestion spec. |
| Driver/vehicle expiry columns | `drivers`: `license_number`, `license_expiry` (00002). `vehicles`: `insurance_expiry`, `fitness_expiry`, `permit_expiry` (00003) + `rc_expiry` (00030). Vertical-slice aggregates: `internal/vehicle/domain/aggregate/vehicle_aggregate.go:44-46` (Insurance/Fitness/Permit — **no RC**), `internal/driver/domain/aggregate/driver_aggregate.go:30-31`. Legacy `domain.Vehicle` has `RCExpiry`. |
| RBAC seed pattern | `db/migrations/00012_rbac.sql` — `INSERT OR IGNORE INTO permissions (name, description) VALUES ...`, admin gets all via `SELECT 1, id FROM permissions`; dispatcher/accountant/viewer via `LIKE` patterns. Casbin adapter splits `resource:action` (`internal/auth/casbin.go:58-63`). |

### Verification Log (QA pass — 2026-08-19)

Migration ownership (final): alerts **00045**, compliance + files + RBAC seeds **00046** (THIS spec); e-way bill lifecycle **00047** (SPEC 07 — this spec's worker consumes it, does not own it); GST e-invoice RBAC **00048** (SPEC 07); FASTag **00049** (SPEC 07); telemetry_alerts rebuild **00059** (THIS spec, from reserved range). HEAD = `00039_experiments.sql` (TAKEN). Outbox cite `609-616`→`819-826`; `newFounderNotifier` 682-698→~820.

| Claim | Verdict | Correction / Evidence |
|---|---|---|
| E-way client is a stub (fake EWB/QR) | VERIFIED | `internal/integration/ewaybill/client.go:68-94` |
| ComplianceService exists, UNWIRED | VERIFIED | `internal/service/compliance_service.go:27,79,131` |
| Dead schema (compliance_checks/notifications/eway_bills/telemetry_alerts) | VERIFIED | no repo refs in `internal/repository/` |
| Two buses (service vs outbox-relay) | VERIFIED | `service.go:75`; `main.go:819` relay |
| No geofence/SOS/AIS code | VERIFIED | grep `geofence*\|SOS*\|AIS-140` → none in `internal/`/`db/`/`mobile/` |
| `TripAssignedEvent` never published to bus | PARTIAL | defined + `RecordEvent` at `trip_aggregate.go:110`; not dispatched to bus today |
| agent tools = 19 | VERIFIED | `internal/agent/tools.go` (19 `Name:` entries) |
| Migrations 00044–00048 owned by THIS spec | WRONG | corrected: THIS spec owns **00045–00046**; **00047** (e-way bill) + **00048** (GST RBAC) owned by **spec 07**; telemetry_alerts rebuild → **00059** |
| Latest 00038; 00039–00043 free | WRONG | HEAD `00039` TAKEN; `00040–00044` reserved by specs 01–04 |
| Outbox relay at `main.go:609-616` | WRONG | actual `main.go:819-826` |

### Severity & Effort (major changes)

| Change | Severity | Effort |
|---|---|---|
| Migration ownership: THIS spec = 00045–00046 (+ telemetry_alerts rebuild 00059); 00047 (e-way bill) + 00048 (GST) owned by spec 07 | Low | S |
| Unified alerts pipeline (dedup/storm/escalation) | High | L |
| Compliance gate both paths + StartTrip + PUC | High | L |
| E-way bill lifecycle worker | High | L |
| SOS flow + AIS-140 contract | Med | M |
| `telemetry_alerts` rebuild (13 types) | Med | M |

### Architectural Decisions (Decision / Tradeoff / Cost)

- **Two-store model (raw `telemetry_alerts` + canonical `alerts`).** Decision: raw log stays high-volume; `alerts` is deduped/cooldown/ackable. Tradeoff: dual-write on ingest. Cost: one rebuild (00059) + new table.
- **Channel adapters + stubs.** Decision: `in_app` + `telegram` REAL; email/whatsapp/sms log-only until provider VERIFIED. Tradeoff: no external side-effects pre-verification.
- **Compliance gate on both assignment paths + StartTrip + 7-day window + exemptions.** Decision: block by default; exemptions mitigate. Cost: gate in 2 vertical-slice + 1 legacy path.
- **E-way extension gated on geofence evidence.** Decision: only extend if within `EWAYBILL_EXTENSION_KM` of destination. Cost: depends on geofence_events bus contract (spec 01/02).
- **Cross-ref spec 17 (GPS provider strategy):** alert/telemetry event *types* consumed by this pipeline originate from ingestion spec 01, whose device-source strategy is governed by `17-gps-telematics-provider-strategy.md`.

---

## 1. Architecture — event → rule → alert → channel

```
producers                          pipeline (internal/alerts/)                     consumers
─────────                          ───────────────────────                          ─────────
telemetry_alerts (raw)   ──┐
telemetry bus events     ──┤
geofence_events (bus)    ──┼──▶ ingest.Event → rule match ─▶ dedup ─▶ storm batch ─▶ channel Provider fan-out
compliance failures      ──┤        │                                                    │
trip lifecycle (bus)     ──┤        ▼                                          ┌─────────┴──────────┐
SOSEvent (bus, ing.)     ──┘   alerts table (CANONICAL)                         in_app (REAL)  telegram (REAL)
                                  status/ack/resolve/escalation                  email/WA/SMS (log-only stubs)
                                       │
                                       └──▶ badge (layout.html) · agent get_open_alerts · RL rewards
```

- **Raw event store vs canonical store:** `telemetry_alerts` stays the raw, high-volume event log (rebuilt in 00059 with widened CHECK). The new `alerts` table is the canonical operational alert store — deduped, cooldown-limited, severity-ranked, ackable, resolvable, channel-tracked.
- **Rule engine:** `alert_rules` are per-source (telemetry, geofence, fuel, compliance, ewaybill, sos) threshold/type definitions. An alert fires when an incoming event matches a rule; `rule_overrides` lets admins tune threshold/severity/cooldown per tenant/vehicle without editing rules.
- **Routing:** rule → channels (in_app always for actionable alerts; telegram for critical+; email/WhatsApp/SMS only when configured and provider verified — see §18).
- **Storm batching:** multiple same-type alerts in a short window collapse into one alert with `occurrences` counter and a batched channel message (see §4).
- **Escalation:** unacked critical alerts escalate per `escalation_schedule` (JSON: steps with delay + channel + target role).
- **Outbox note:** the alert pipeline subscribes to the in-memory `events.EventBus` created in `internal/service/service.go:75` and to `outboxRelay`-dispatched events (`cmd/server/main.go:825`). For cross-restart durability of alert-relevant lifecycle events, emit `TripAssignedEvent`/`TripCancelledEvent` through an outbox write (see §7, §15) — the relay already exists.

---

## 2. Alert rules model + DDL 00045

**Migration ownership (coordinated with other specs):**

- `00045` = **THIS SPEC** — `alert_sources`, `alert_rules`, `rule_overrides`, `alerts`, `notifications_preferences`
- `00046` = **THIS SPEC** — `compliance_exemptions` + `files` uploadable_type CHECK rebuild + `compliance_checks` index + **this spec's RBAC seeds (§11) folded here**
- `00047` = **SPEC 07** (e-way bill lifecycle: `eway_bills` ALTER + `eway_bill_events`) — this spec's worker consumes the schema but does **NOT** own the migration
- `00048` = **SPEC 07** (GST e-invoice RBAC seeds) — NOT owned by this spec
- `00049` = **SPEC 07** (FASTag tables) — NOT owned by this spec
- `00059` = **THIS SPEC** — `telemetry_alerts` CHECK widening (rebuild) (§12), number taken from the reserved 00059–00061 range

**DDL 00045 — `db/migrations/00045_alerts_pipeline.sql`:**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS alert_sources (
    id          TEXT PRIMARY KEY,              -- e.g. 'telemetry', 'geofence', 'fuel', 'compliance', 'ewaybill', 'sos'
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alert_rules (
    id              TEXT PRIMARY KEY,
    source          TEXT NOT NULL REFERENCES alert_sources(name),
    alert_type      TEXT NOT NULL,             -- must be one of the 00059 CHECK values for telemetry-sourced rules
    name            TEXT NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'warning'
                    CHECK (severity IN ('info','warning','critical','blocker')),
    threshold       REAL,                      -- per-source threshold (km, litres, speed, count...)
    threshold_unit  TEXT,
    dedup_key_expr  TEXT NOT NULL DEFAULT 'source:alert_type:entity_id',
    cooldown_seconds INTEGER NOT NULL DEFAULT 300,
    storm_window_seconds INTEGER NOT NULL DEFAULT 60,
    storm_batch_min   INTEGER NOT NULL DEFAULT 3,   -- group if >=N same-type in window
    channel_routing TEXT NOT NULL DEFAULT 'in_app', -- JSON: {"in_app":true,"telegram":["critical"],"email":false,...}
    escalation_schedule TEXT,                  -- JSON array of {after_seconds, target_role, channel}
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rule_overrides (
    id          TEXT PRIMARY KEY,
    rule_id     TEXT NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    entity_id   TEXT,                          -- vehicle/driver/trip id; NULL = global override
    severity    TEXT,
    threshold   REAL,
    cooldown_seconds INTEGER,
    channels    TEXT,                          -- JSON routing override
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alerts (
    id              TEXT PRIMARY KEY,
    rule_id         TEXT REFERENCES alert_rules(id),
    source          TEXT NOT NULL,
    alert_type      TEXT NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'warning',
    status          TEXT NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','acknowledged','resolved','escalated','closed')),
    dedup_key       TEXT NOT NULL,
    entity_type     TEXT,                      -- 'trip'|'vehicle'|'driver'|'company'|'system'
    entity_id       TEXT,
    user_id         TEXT,                      -- routing target for in_app (admin/dispatcher)
    title           TEXT NOT NULL,
    message         TEXT NOT NULL,
    occurrences     INTEGER NOT NULL DEFAULT 1,
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_escalation_at DATETIME,
    escalation_step INTEGER NOT NULL DEFAULT 0,
    latitude        REAL,
    longitude       REAL,
    metadata        TEXT,                      -- JSON
    acked_by        TEXT,
    acked_at        DATETIME,
    resolved_by     TEXT,
    resolved_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_alerts_status_seen  ON alerts(status, last_seen_at);
CREATE INDEX IF NOT EXISTS idx_alerts_dedup        ON alerts(dedup_key, status);
CREATE INDEX IF NOT EXISTS idx_alerts_entity       ON alerts(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_alerts_user_status  ON alerts(user_id, status);

CREATE TABLE IF NOT EXISTS notifications_preferences (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel    TEXT NOT NULL CHECK (channel IN ('in_app','email','whatsapp','sms','telegram')),
    enabled    INTEGER NOT NULL DEFAULT 1,
    min_severity TEXT NOT NULL DEFAULT 'warning'
                 CHECK (min_severity IN ('info','warning','critical','blocker')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, channel)
);

-- Seed sources + default rules (idempotent)
INSERT OR IGNORE INTO alert_sources (id, name, description) VALUES
 ('src_telemetry','telemetry','Raw telemetry exception stream'),
 ('src_geofence','geofence','Geofence boundary/usage events'),
 ('src_fuel','fuel','Fuel refill/theft/siphon events'),
 ('src_compliance','compliance','Document expiry and dispatch blocks'),
 ('src_ewaybill','ewaybill','E-way bill lifecycle failures'),
 ('src_sos','sos','Emergency SOS events');

-- +goose Down
DROP TABLE IF EXISTS notifications_preferences;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS rule_overrides;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS alert_sources;
```

Default `alert_rules` rows (one per 00059 alert type, severity per type, cooldown 300s, storm window 60s/min 3, routing `{"in_app":true,"telegram":["critical","blocker"]}`) are seeded in the migration (preferred for determinism).

---

## 3. Channel adapters + badge fix

**Provider interface — `internal/alerts/channels/provider.go` (new):**

```go
type Message struct {
    AlertID  string
    Title    string
    Body     string
    Severity string
    UserID   string // in_app target
    Phone    string // sms/whatsapp
    Email    string
    Meta     map[string]any
}
type Provider interface {
    Name() string
    Send(ctx context.Context, msg Message) error // must be non-blocking-safe; internal retry w/ backoff
}
```

| Channel | Adapter | Status | Location |
|---|---|---|---|
| `in_app` | **REAL** — insert into `alerts` (user-scoped) + Datastar badge | implement | `internal/alerts/channels/in_app.go` (new) |
| `telegram` | **REAL** — reuse `gopkg.in/telebot.v3` + formatter infra | implement | `internal/alerts/channels/telegram.go` (new); reuse pattern from `internal/founder/alerts/notifier.go` (telebot bot, `telebot.ModeMarkdown`) |
| `email` | log-only stub (mirror `internal/operations/notifications/service.go:38-42`) | stub | `internal/alerts/channels/stubs.go` (new) |
| `whatsapp` | log-only stub; provider **VERIFY AT IMPLEMENTATION** (WhatsApp Business API vs Msg91 vs Twilio) — do not recommend without verification | stub | same |
| `sms` | log-only stub; provider **VERIFY AT IMPLEMENTATION** (Twilio / Msg91) | stub | same |

**Badge fix (exact files):**

1. `internal/handlers/app.go:379-396` — replace the audit-log feed in `renderPage` with a query against the canonical `alerts` table: `unreadCount = COUNT(alerts WHERE status IN ('open','acknowledged') AND user_id = session.UserID AND (severity >= user min_severity))`; `Notifications` becomes the 5 newest such alerts. Remove the `notif_read_at` cookie hack (:386-393).
2. `internal/templates/layout.html:164,171` — keep markup; data now comes from real alerts. `#notif-badge` appears when `HasUnread`.
3. New `internal/alerts/repository/sqlite/alerts_repo.go` — `UnreadCount(ctx, userID)`, `Recent(ctx, userID, limit)`, `Ack(ctx, alertID, userID)`, `Resolve(...)`, `MarkAllRead(ctx, userID)`.
4. Datastar: badge element re-renders via a `GET /alerts/unread` fragment endpoint (`internal/handlers/alerts.go`, new, mounted under the authed group in `cmd/server/main.go` alongside `/trips`), `data-on-load` refresh — pattern already used by `internal/handlers/trips.go:102-110` (`isDatastarRequest` → `renderFragment`).

---

## 4. Storm batching + dedup + escalation

Engine — `internal/alerts/pipeline/engine.go` (new), consumed by a subscriber wired in `cmd/server/main.go` next to the founder handlers (`:819-826`).

- **Dedup:** `dedup_key = rule.dedup_key_expr` evaluated per event (default `source:alert_type:entity_id`). On match with an existing `open`/`acknowledged` alert:
  - if within `cooldown_seconds` of `last_seen_at` → increment `occurrences`, update `last_seen_at`, **no new channel sends**;
  - else → re-send channels per routing (suppressed for storm-batched entries).
- **Storm batching:** per rule, count events of the same `dedup_key` in `storm_window_seconds`. When `>= storm_batch_min`, collapse: keep one alert row, set `occurrences`, and on channel send include "N occurrences since <time>". In-app lists a single row; telegram gets a single message. A background flusher (`internal/alerts/pipeline/flusher.go`) emits the batched channel message at window close.
- **Escalation:** `escalation_schedule` JSON per rule, e.g. `[{"after_seconds":900,"target_role":"admin","channel":"telegram"},{"after_seconds":3600,"target_role":"founder","channel":"telegram"}]`. A ticker (`internal/alerts/pipeline/escalator.go`, interval 60s) walks `alerts WHERE status='open' AND next_escalation_at <= now`, increments `escalation_step`, re-sends via the step channel to users holding `target_role` (casbin `RoleRequired` check, cf. `cmd/server/main.go:533`), and sets `next_escalation_at`. Ack/resolve cancels pending escalation.
- **Ack/resolve:** HTTP routes `POST /alerts/{id}/ack`, `POST /alerts/{id}/resolve` (permission `alerts:update`), surfaced as a small Datastar section in `internal/templates/layout.html` or a new `alerts_list.html`; both write `alerts` rows + audit log (`services.Audit.LogAction`, pattern `internal/service/audit_log_service.go:32`).
- **Ordering guarantee:** synchronous `InMemoryBus` dispatch means events process in publish order per subscription; the pipeline must be idempotent by `dedup_key` (enforce in repo with `INSERT ... WHERE NOT EXISTS` — a unique partial index `(dedup_key, status)` is not possible in SQLite for open-only).

---

## 5. Compliance gate

**Exact wiring (both assignment paths + StartTrip):**

1. **New `CheckDispatchCompliance(ctx, driverID, vehicleID) (ComplianceResult, error)`** on `internal/service/compliance_service.go` (extend existing type; keep `EnforceDispatchCompliance` as a thin wrapper for backward compat with `test/master_integration_test.go:102`). It verifies **5 documents**:
   - Driver: `drivers.license_expiry` (00002)
   - Vehicle: `vehicles.rc_expiry`, `fitness_expiry`, `insurance_expiry` (00003 + 00030)
   - **PUC**: new column — add `vehicles.puc_expiry DATE` in 00046 (`ALTER TABLE vehicles ADD COLUMN puc_expiry DATE;`) since no PUC field exists anywhere.
   - Plus doc-vault presence: `files.uploadable_type IN ('driver_license','vehicle_rc','vehicle_fitness','vehicle_insurance','vehicle_puc')` with `uploadable_id` matching the entity.
   - **7-day window:** any expiry `<= now + 7 days` (config `COMPLIANCE_EXPIRY_WINDOW_DAYS`) → `Valid=false`, reason "expires within 7 days" (this is the dispatch window; `compliance_checks` rows get `status='warning'` for window hits, `'expired'` for past).
   - **`compliance_exemptions`** (00046): if an active exemption row exists for (entity_type, entity_id, doc_type), skip that doc with audit trail (`compliance_exempted` action). Exemption creation is admin-only, permission `compliance:update`, and expiry-capped (see 00046 DDL below).
2. **Driver assignment** — `internal/trip/application/assign_driver.go:45` `checkDriverCompliance` → call the gate (driver side). Blocking error text: `"Dispatch blocked: <reason> (compliance)"`.
3. **Vehicle assignment** — `internal/trip/application/assign_vehicle.go:45` `checkVehicleCompliance` → call the gate (vehicle side).
4. **Legacy service path** — `internal/service/trip_service.go:163` (`AssignDriver`) and `:230` (`AssignVehicle`) → same gate (this is what agent tools `assign_driver`/`assign_vehicle` hit — `internal/agent/tools.go:434,460`).
5. **StartTrip re-validation** — `internal/trip/application/start_trip.go:41` and `internal/service/trip_service.go:295` → run full 5-doc gate on the trip's assigned driver+vehicle; block with same error style.
6. **UI surfacing** — `internal/handlers/trips.go:308-353` currently `http.Error` on failure; change to `renderPage`/`renderForm` with `FlashError` (pattern at :160) and, in `internal/templates/trip_view.html`, render a compliance banner (list of `compliance_checks` rows + missing docs) whenever the gate fails. Datastar fragment `GET /trips/{id}/compliance` re-renders the banner on doc upload.
7. **Checkpoint recording** — each gate run writes `compliance_checks` rows (entity_type ∈ driver/vehicle/cargo, check_type ∈ rc/fitness/insurance/puc/license, status ∈ valid/expired/blocked/warning — matches existing CHECK in 00030) and emits bus event `ComplianceBlocked` → alert pipeline (§1) creates a compliance alert.

**DDL 00046 — `db/migrations/00046_compliance_and_files.sql`:**

```sql
-- +goose Up
ALTER TABLE vehicles ADD COLUMN puc_expiry DATE;

CREATE TABLE IF NOT EXISTS compliance_exemptions (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('driver','vehicle')),
    entity_id   TEXT NOT NULL,
    doc_type    TEXT NOT NULL,   -- rc|fitness|insurance|puc|license
    reason      TEXT NOT NULL,
    exempt_until DATETIME NOT NULL,
    created_by  TEXT NOT NULL REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_compliance_exemptions_entity
    ON compliance_exemptions (entity_type, entity_id, doc_type);

CREATE INDEX IF NOT EXISTS idx_compliance_checks_entity
    ON compliance_checks (entity_type, entity_id, check_type, created_at);

-- files.uploadable_type CHECK rebuild (SQLite 12-step: new table, copy, drop, rename)
CREATE TABLE files_new (
    id TEXT PRIMARY KEY, filename TEXT NOT NULL, original_name TEXT NOT NULL,
    path TEXT NOT NULL, size INTEGER NOT NULL, mime_type TEXT NOT NULL,
    uploadable_type TEXT NOT NULL CHECK (uploadable_type IN
      ('driver_license','vehicle_insurance','vehicle_permit','company_logo',
       'vehicle_rc','vehicle_fitness','vehicle_puc')),
    uploadable_id TEXT, created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO files_new SELECT * FROM files;
DROP TABLE files;
ALTER TABLE files_new RENAME TO files;

-- +goose Down
ALTER TABLE vehicles DROP COLUMN puc_expiry;
DROP TABLE IF EXISTS compliance_exemptions;
DROP INDEX IF EXISTS idx_compliance_checks_entity;
-- (files rebuild is one-way; down leaves widened CHECK)
```

Code parity: extend the subdir switch in `internal/service/file_service.go:88-98` — `vehicle_rc`, `vehicle_fitness`, `vehicle_puc` → `vehicles/` (or `vehicles/rc|fitness|puc`).

---

## 6. Doc vault CHECK rebuild

Covered by 00046 above. Also update:

- `internal/domain/file` entity if it validates `uploadable_type` (grep `UploadableType` — only `file_service.go:127` assigns; `repository/sqlite` stores raw).
- Doc upload UI/endpoint: `internal/handlers` file upload handler + `internal/templates/driver_edit.html` / `vehicle_edit.html` (new sections to upload rc/fitness/puc), permission `files:create` (already seeded in 00012).
- Validation parity: `internal/service/file_service.go:47-53` allowed MIME list already accepts PDF/JPEG/PNG/WebP — no change needed.

---

## 7. E-way bill lifecycle worker

**Worker — `internal/ewaybill/worker.go` (new):** background loop (interval `EWAYBILL_WORKER_INTERVAL`, default 60s; goroutine started in `cmd/server/main.go` next to `runDailyDigest`, `:612-616`), driving rows of the `eway_bills` table (00031 schema + **00047 ALTER — owned by spec 07**).

**Lifecycle:**

```
trip created ──► generate (trip start, value>threshold) ──► Part-B vehicle update (on dispatch)
                                                                    │
                              extension (gated on geofence evidence) ▼
                              ──► cancel (trip cancelled)  ◄──────────┘
```

1. **Generate** — on `TripStarted` (subscribe to bus `TripStarted`; also poll for trips started while worker down): build `ewaybill.GenerateRequest` from trip + route (GSTINs from `company_settings`, transporter ID from settings), call provider `Generate`, persist `eway_bills` row (`status='active'`, `valid_until` from provider), set `trips.eway_bill_ref` (00030 column). Stub provider already returns EWB number + 24h validity.
2. **Part-B vehicle update on dispatch** — subscribe to a newly published `TripAssignedEvent` (currently never published — publish it from `internal/trip/application/assign_driver.go:61` / `assign_vehicle.go:61` / legacy `trip_service.go` assignments) and from the worker poll when `trips.vehicle_id` set but `eway_bills.vehicle_number` empty. **Requires a real-provider adapter method `UpdateVehiclePartB(ctx, ewbNumber, vehicleNumber) (EWayBill, error)` added to `internal/integration/ewaybill/client.go:50-54` interface — VERIFY provider support (NIC e-way bill Part-B API) at implementation; stub returns fake success.**
3. **Extension** — worker scans `eway_bills WHERE status='active' AND valid_until <= now + EXTENSION_LEAD_SECONDS (4h)` for trips in `in_transit`/`delivered`. **Gated:** only extend if geofence evidence exists — consume `geofence_events` from the bus (contract from ingestion/geofence spec: `GeofenceAlertEvent`/`geofence_events` rows with `{trip_id, vehicle_id, fence_id, lat, lng, at}`) and accept extension only when the latest vehicle position is within `EWAYBILL_EXTENSION_KM` (default 5 km) of trip destination (route endpoint from `internal/domain/route`). Provider method `Extend(ctx, ewbNumber) (EWayBill, error)` — stub returns fake `valid_until = now+24h`; **VERIFY provider support.** Without evidence: alert `ewaybill.extension_denied` (severity warning) via pipeline.
4. **Cancel** — on `TripCancelled` (bus) or worker poll of `trips.status='cancelled'` with `eway_bills.status='active'`: provider `Cancel(ewbNumber, reason)`, set `status='cancelled'`, `cancelled_at`, `cancellation_reason`.
5. **Audit trail** — every transition (generate/part_b/extend/cancel/failure) appends `eway_bill_events` (**00047, owned by spec 07**). No provider mutating call happens without a preceding event row (write-event-then-call, or compensate on failure).

**DDL 00047 (OWNED BY SPEC 07 — `07-gst-ewaybill-fastag.md` §3.3; reproduced here for reference; this spec does NOT redefine it):**

```sql
-- +goose Up
ALTER TABLE eway_bills ADD COLUMN part_b_updated_at DATETIME;
ALTER TABLE eway_bills ADD COLUMN extension_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE eway_bills ADD COLUMN extended_until DATETIME;
ALTER TABLE eway_bills ADD COLUMN last_extension_at DATETIME;
ALTER TABLE eway_bills ADD COLUMN geofence_proof TEXT;      -- JSON {trip_id, distance_km, fence_event_id, lat, lng}
ALTER TABLE eway_bills ADD COLUMN cancellation_reason TEXT;
ALTER TABLE eway_bills ADD COLUMN cancelled_at DATETIME;
ALTER TABLE eway_bills ADD COLUMN error_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE eway_bills ADD COLUMN last_error TEXT;

CREATE TABLE IF NOT EXISTS eway_bill_events (
    id          TEXT PRIMARY KEY,
    eway_bill_id TEXT NOT NULL REFERENCES eway_bills(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL CHECK (event_type IN
                ('generated','part_b_updated','extended','cancelled','extension_denied','provider_error','recovered')),
    details     TEXT,                       -- JSON: provider raw response, distances, reasons
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_eway_bill_events_bill ON eway_bill_events (eway_bill_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS eway_bill_events;
ALTER TABLE eway_bills DROP COLUMN part_b_updated_at;
ALTER TABLE eway_bills DROP COLUMN extension_count;
ALTER TABLE eway_bills DROP COLUMN extended_until;
ALTER TABLE eway_bills DROP COLUMN last_extension_at;
ALTER TABLE eway_bills DROP COLUMN geofence_proof;
ALTER TABLE eway_bills DROP COLUMN cancellation_reason;
ALTER TABLE eway_bills DROP COLUMN cancelled_at;
ALTER TABLE eway_bills DROP COLUMN error_count;
ALTER TABLE eway_bills DROP COLUMN last_error;
```

Reuse `internal/domain/ewaybill/entity.go` (`Validate` ≥12-char EWB, `IsActive`) in the worker; row-level trip link exists (`trip_id UNIQUE`, 00031). Raw provider responses go to `raw_response` (existing column).

---

## 8. SOS flow

- **Canonical event:** `SOSEvent{ID, TripID, VehicleID, DriverID, Lat, Lng, At, Trigger: driver_app|panic_button|telematics}` — **contract from ingestion spec** (no code exists today; `mobile/src` has no SOS yet). This spec subscribes to bus topic `"SOSEvent"` (struct-name event type via the relay).
- **Handling (`internal/alerts/pipeline/sos.go`):** immediately create canonical alert (`source='sos'`, severity `blocker`) and fan out:
  - `in_app` — all admins/dispatchers (users with `alerts:read`), immediately.
  - `telegram` — founder/ops channel via the existing notifier path (`internal/founder/alerts/notifier.go`, `newFounderNotifier` in `cmd/server/main.go:820`; reuse `FOUNDER_TELEGRAM_BOT_TOKEN`/`FOUNDER_TELEGRAM_CHAT_ID` or separate `ALERT_TELEGRAM_*`), immediately.
  - `email`/`whatsapp`/`sms` — only if configured per `notifications_preferences` + rule routing; adapters are log-only stubs (§3).
  - **Emergency escalation:** dial-out to `SOS_ESCALATION_PHONE` — **VERIFY provider (Twilio Voice / Msg91 / WhatsApp) at implementation; ship as log-only stub**.
- **Admin action UI:** `internal/handlers/alerts.go` + `internal/templates/alerts_list.html` (new): SOS row shows map coords (lat/lng), vehicle/driver, trip; buttons Ack / Resolve (`alerts:update`), each audited (`sos_ack` / `sos_resolve` audit actions).
- **State:** ack/resolve on the canonical `alerts` row; escalation schedule applies like any `blocker` (default: re-notify telegram every 10 min until acked — §4 escalator).

---

## 9. AIS 140 / VLT design contract

**Out of scope for implementation; interface contract only** — no code ships in this spec beyond the interface file.

`internal/vlt/contract.go` (new, design-only, doc-commented `// NOT IMPLEMENTED — AIS-140 readiness`):

```go
// VLTAdapter is the AIS-140 Vehicle Location Tracking adapter contract.
// No implementation is shipped in this spec (out of scope). Device vendor
// integrations (e.g. NI's AIS-140 aggregator, OEM telematics) must implement:
type VLTAdapter interface {
    // RegisterVehicle pushes vehicle + driver binding to the VLT aggregator.
    RegisterVehicle(ctx context.Context, v VehicleBinding) error
    // StreamLocation is the inbound path for AIS-140 1.0/2.0 location packets
    // (NMEA-0183 style ">RVR..." packets); returns the canonical location.
    StreamLocation(ctx context.Context, pkt LocationPacket) (Location, error)
    // Health reports aggregator connectivity.
    Health(ctx context.Context) (Health, error)
}

type VehicleBinding struct {
    VehicleID   string
    IMEI        string `json:"imei"`
    SIM         string `json:"sim"`
    DriverID    string
    RegisteredAt time.Time
}

type LocationPacket struct {
    IMEI       string
    Timestamp  time.Time
    Lat, Lng   float64
    SpeedKPH   float64
    Heading    int
    Ignition   bool
    Emergency  bool   // AIS-140 panic/emergency button bit
}

type Location struct {
    VehicleID   string
    Lat, Lng    float64
    SpeedKPH    float64
    Timestamp   time.Time
}

type Health struct {
    Connected bool
    LatencyMS int64
    LastPacketAt time.Time
}
```

Contract notes: `LocationPacket.Emergency` feeds the SOS pipeline (§8) once the ingestion spec lands; `StreamLocation` output maps to `telemetry_snapshots` (00031) and the rule engine. Marked `OUT_OF_SCOPE` in `docs/`; `FUTURE_SCOPE.md` unchanged.

---

## 10. Agent integration

**New tools (add to `internal/agent/tools.go` registry):**

```go
// read-only — no gate
{
  Name: "get_open_alerts",
  Description: "List open operational alerts (source, severity, entity, title). Optional severity filter.",
  Parameters: { "severity": optional string },
  Handler: func(...) — calls alerts repo ListOpen(ctx, severity, limit=20); returns JSON rows.
}

// mutating — approval-gated
{
  Name: "extend_ewaybill",
  Description: "Request e-way bill extension for a trip. Requires trip id. Subject to geofence evidence gate.",
  Parameters: { "trip_id": required string },
  Handler: func(...) — calls ewaybill worker/service ExtendForTrip(ctx, tripID).
}
```

- **Approval gate:** add `"extend_ewaybill"` to `MutatingTools()` in `internal/agent/approval.go:62-64`. Wiring is automatic: `cmd/server/main.go:425-434` gates every name in `MutatingTools()` when `AGENT_REQUIRE_APPROVAL=true`; the admin approves at `/agent-actions` (page) or `POST /api/agent/actions/{id}/approve|reject` (`internal/agent/approval_handler.go:25-26`). Approval executes under the admin's identity (`approval.go:80-84`), so the audit trail records the admin as actor.
- **Routing:** add `has("alert", "alerts", "eway", "eway bill", "e-way")` keywords to `orchestrator.go:127-149` `keywordRoute` → `"ops"` (alerts fit the ops sub-agent; add the two tools to the ops `pick(...)` list at `internal/agent/subagents.go:82`).
- **RL hooks:**
  - `get_open_alerts` — automatic `tool_ok`/`tool_error` rewards via the tracer (`orchestrator.go:70-72`, `reward.go:29-39`).
  - `extend_ewaybill` — action decision feeds rewards through the existing action lifecycle: `UpdateActionDecision` (`rl/service.go:92`) + `SignalAction` (`rl/service.go:56`) with `RewardActionExecuted=1.0`, `RewardActionFailed=-1.5`, `RewardActionRejected=-0.8` (`reward.go:22-24`). No new RL code needed; only register the tool + gate.
  - Optional (recommended): `ack_alert`/`resolve_alert` tools — read-adjacent but state-mutating; gate them too if added.

---

## 11. RBAC seeds (folded into 00046, this spec's subset)

`db/migrations/00046_compliance_and_files.sql` (RBAC seeds folded into this spec's 00046 migration — coordinated with other specs; geofences, fuel, scorecard, shares, maintenance resources are owned by their respective specs and contribute their own INSERT blocks to their own migrations; this spec only lists its own subset). Admin auto-grant pattern from `00012_rbac.sql:92-93`:

```sql
-- +goose Up
INSERT OR IGNORE INTO permissions (name, description) VALUES
 ('alerts:read', 'View operational alerts'),
 ('alerts:update', 'Ack/resolve operational alerts'),
 ('compliance:read', 'View compliance checks and exemptions'),
 ('compliance:update', 'Create compliance exemptions'),
 ('ewaybill:read', 'View e-way bills and events'),
 ('ewaybill:update', 'Trigger e-way bill lifecycle actions'),
 ('telemetry:read', 'Read telemetry alerts and snapshots');

-- Admin gets everything (existing pattern)
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name IN
 ('alerts:read','alerts:update','compliance:read','compliance:update',
  'ewaybill:read','ewaybill:update','telemetry:read');

-- Dispatcher (role 2): read + acknowledge
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name IN
 ('alerts:read','alerts:update','compliance:read','ewaybill:read','telemetry:read');

-- Viewer (role 4): read-only (matches 00012 viewer pattern)
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions WHERE name IN
 ('alerts:read','compliance:read','ewaybill:read','telemetry:read');

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN
 (SELECT id FROM permissions WHERE name IN
  ('alerts:read','alerts:update','compliance:read','compliance:update',
   'ewaybill:read','ewaybill:update','telemetry:read'));
DELETE FROM permissions WHERE name IN
 ('alerts:read','alerts:update','compliance:read','compliance:update',
  'ewaybill:read','ewaybill:update','telemetry:read');
```

Permission-gate new routes with `middleware.RequirePermission(authSrv, "alerts", "read")` etc. (pattern: `internal/integration/handler.go:42`).

---

## 12. telemetry_alerts rebuild — DDL 00059

SQLite CHECK can't be altered in place → rebuild (12-step). This spec OWNS this migration (**00059**, from the reserved range). New CHECK list — **exact**:

```sql
-- +goose Up
-- Rebuild telemetry_alerts with widened alert_type CHECK
CREATE TABLE telemetry_alerts_new (
    id TEXT PRIMARY KEY,
    trip_id TEXT,
    vehicle_id TEXT,
    driver_id TEXT,
    alert_type TEXT NOT NULL CHECK (alert_type IN
        ('night_driving','restricted_zone','unauthorized_movement','off_hours_use',
         'refill','theft_suspicion','abnormal_drain','siphon_confirmed','odometer_rollback',
         'speeding','temp_breach','gps_deviation','geofence_breach')),
    severity TEXT NOT NULL DEFAULT 'warning',
    details TEXT NOT NULL,
    latitude REAL,
    longitude REAL,
    resolved INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id)
);
INSERT INTO telemetry_alerts_new (id, trip_id, vehicle_id, driver_id, alert_type, severity, details, latitude, longitude, resolved, created_at)
    SELECT id, trip_id, vehicle_id, driver_id,
           CASE alert_type WHEN 'fuel_theft' THEN 'theft_suspicion' ELSE alert_type END,
           severity, details, latitude, longitude, resolved, created_at
    FROM telemetry_alerts
    WHERE alert_type IN ('gps_deviation','temp_breach','speeding')
       OR alert_type = 'fuel_theft';
DROP TABLE telemetry_alerts;
ALTER TABLE telemetry_alerts_new RENAME TO telemetry_alerts;
CREATE INDEX IF NOT EXISTS idx_telemetry_alerts_trip ON telemetry_alerts(trip_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_alerts_type ON telemetry_alerts(alert_type, created_at);

-- +goose Down
-- Reverse rebuild back to the 00030 CHECK set (fuel_theft, gps_deviation, temp_breach, speeding)
```

Notes: the 13-type list **replaces** the old 4-type CHECK (`fuel_theft` becomes `theft_suspicion`/`siphon_confirmed`; `gps_deviation`, `temp_breach`, `speeding` retained). The ingestion spec's producers must emit the new type names; `internal/service/telemetry_service.go:61,93` currently emits `gps_deviation`/`fuel_theft` — update `fuel_theft` → `theft_suspicion` and persist to the table (currently it never persists). `telemetry_alerts` remains the raw event store; canonical alerts live in `alerts` (00045).

---

## 13. Go file list (new/changed)

**New packages:**

```
internal/alerts/domain/alert.go                  — Alert, Rule, Source, ChannelState models
internal/alerts/repository/alerts_repo.go        — interface
internal/alerts/repository/sqlite/alerts_repo.go — impl (alerts, rules, overrides, prefs)
internal/alerts/channels/provider.go             — Provider interface + Message
internal/alerts/channels/in_app.go               — REAL in-app adapter
internal/alerts/channels/telegram.go             — REAL telegram adapter (reuse telebot)
internal/alerts/channels/stubs.go                — email/whatsapp/sms log-only stubs
internal/alerts/pipeline/engine.go               — event → rule → dedup → alert
internal/alerts/pipeline/flusher.go              — storm batch flush
internal/alerts/pipeline/escalator.go            — escalation ticker
internal/alerts/pipeline/sos.go                  — SOS fan-out
internal/alerts/service.go                       — facade used by handlers + agent
internal/handlers/alerts.go                      — web/API routes (list, ack, resolve, badge fragment)
internal/ewaybill/worker.go                      — lifecycle worker (generate/part-B/extend/cancel)
internal/ewaybill/service.go                     — ExtendForTrip used by agent tool
internal/vlt/contract.go                         — AIS-140 design-only contract
```

**Changed files (exact):**

```
cmd/server/main.go                        — wire pipeline subscriber, eway worker goroutine, alerts routes, badge; publish TripAssignedEvent hooks
internal/handlers/app.go:379-396          — badge from alerts table
internal/templates/layout.html:164,171    — badge data (no markup change required)
internal/handlers/trips.go:308-353        — compliance block → FlashError/banner; Create auto-assign
internal/templates/trip_view.html         — compliance banner + doc upload section
internal/trip/application/assign_driver.go:45,61   — gate + publish TripAssignedEvent
internal/trip/application/assign_vehicle.go:45,61  — gate + publish TripAssignedEvent
internal/trip/application/start_trip.go:41         — re-validate gate
internal/service/trip_service.go:163,230,295       — legacy-path gate + events
internal/service/compliance_service.go             — CheckDispatchCompliance + exemptions
internal/service/file_service.go:88-98             — new uploadable_type subdirs
internal/agent/tools.go                  — get_open_alerts, extend_ewaybill
internal/agent/approval.go:62-64         — MutatingTools += extend_ewaybill
internal/agent/subagents.go:82           — ops tools list
internal/agent/orchestrator.go:127-149   — route keywords
internal/integration/ewaybill/client.go:50-54 — interface += UpdateVehiclePartB, Extend (stubs)
internal/service/telemetry_service.go:61,93 — type names + persist raw alerts (00059)
internal/config/config.go                — new env knobs (§14)
.env.example                             — document new vars
docs/ — AIS-140 contract note + this spec
db/migrations/00045_alerts_pipeline.sql, 00046_compliance_and_files.sql (includes this spec's RBAC seeds),
db/migrations/00059_telemetry_alerts_rebuild.sql — e-way bill 00047 is owned by spec 07 (see 07-gst-ewaybill-fastag.md §3.3)
```

---

## 14. Config / env

| Var | Default | Used by |
|---|---|---|
| `COMPLIANCE_EXPIRY_WINDOW_DAYS` | `7` | gate (§5) |
| `EWAYBILL_WORKER_INTERVAL` | `60` (s) | worker poll |
| `EWAYBILL_EXTENSION_KM` | `5` | geofence extension gate |
| `EWAYBILL_EXTENSION_LEAD_SECONDS` | `14400` (4h) | extension lead |
| `EWAYBILL_MIN_INVOICE_VALUE` | `50000` (INR) | generate threshold (statutory) |
| `ALERT_STORM_WINDOW_SECONDS` / `ALERT_DEDUP_COOLDOWN_SECONDS` | `60` / `300` | pipeline defaults (rules override) |
| `ALERT_TELEGRAM_BOT_TOKEN` / `ALERT_TELEGRAM_CHAT_ID` | fallback to `FOUNDER_TELEGRAM_*` | telegram adapter |
| `SOS_ESCALATION_PHONE` | empty (stub) | SOS dial-out — **VERIFY provider** |
| `EMAIL_SMTP_*` / `SMS_*` / `WHATSAPP_*` | empty | log-only stubs; **VERIFY provider at implementation** |
| `INTEGRATION_EWAYBILL_*` | existing (`internal/integration/config.go:24-28`) | real adapter when provider lands |

All added to `internal/config/config.go` (pattern `getEnv`/`getEnvInt`, `:177-191`) and documented in `.env.example`.

---

## 15. Migration plan

| # | File | Owner | Contents |
|---|---|---|---|
| 00040–00044 | (other specs) | ingestion / geofence / fuel / map | reserved |
| 00045 | `00045_alerts_pipeline.sql` | THIS | alert_sources, alert_rules, rule_overrides, alerts, notifications_preferences + seeds |
| 00046 | `00046_compliance_and_files.sql` | THIS | `vehicles.puc_expiry`, compliance_exemptions, compliance_checks index, files CHECK rebuild, **this spec's RBAC seeds (§11)** |
| 00047 | `00047_ewaybill_lifecycle.sql` | **SPEC 07** | eway_bills ALTER (8 cols), eway_bill_events — owned by spec 07; this spec consumes it |
| 00059 | `00059_telemetry_alerts_rebuild.sql` | THIS | widened CHECK (13 types), data-preserving rebuild (from reserved range) |

Goose runs all `db/migrations/` via `db/migrations.go` embed — new files are picked up automatically (`cmd/server/main.go:138-153`). Order matters: 00046 permission names (this spec's RBAC seeds) must exist before code gates routes; 00059 before ingestion spec producers emit new types. Deploy: run `server` once with new binary (goose `Up` on boot).

---

## 16. Edge cases

1. **Dedup race:** two events for same `dedup_key` in one publish batch → repo upsert with `WHERE NOT EXISTS` on open status; `occurrences` incremented atomically.
2. **Cooldown vs storm:** storm collapse only counts events inside the storm window even when cooldown suppresses sends; `occurrences` reflects both.
3. **Channel failure:** in_app failure = DB error → pipeline retries with backoff (outbox relay pattern, `relay.go:137-148`); telegram failure → alert stays open, `next_escalation_at` shortened.
4. **Part-B ordering:** vehicle assigned *after* e-way generated → worker backfills `vehicle_number` + `part_b_updated_at`; vehicle assigned *before* generation → Part-B included in `GenerateRequest` (existing field `internal/integration/ewaybill/client.go:25`).
5. **Extension without evidence:** `extension_denied` alert; worker never calls provider without geofence proof row in `geofence_proof`.
6. **Trip cancelled mid-lifecycle:** cancel wins; `eway_bill_events` gets `cancelled`; provider failure → `provider_error` + retry with `error_count` cap (5) then manual queue.
7. **Provider outage:** `error_count`/`last_error` on eway_bills; worker backoff; alerts `ewaybill.provider_failure` (blocker for expired-in-flight).
8. **telemetry_alerts rebuild:** 00059 preserves rows (INSERT SELECT) — but `fuel_theft` rows would violate the new CHECK on re-insert; map old `fuel_theft` → `theft_suspicion` in the SELECT (as done in §12), or delete stale rows first (data is raw events; mapping preferred).
9. **files CHECK rebuild:** no FK references `files` (00001); drop/rename is safe. Keep `company_logo` + old types in the new CHECK.
10. **Exemption expiry:** gate treats `exempt_until < now` as no exemption; a ticker or lazy check writes `compliance_checks status='warning'` when an exemption lapses with docs still missing.
11. **Agent approval disabled** (`AGENT_REQUIRE_APPROVAL=false`): `extend_ewaybill` executes directly — acceptable since the worker gate (geofence evidence) still applies.
12. **RL store wiped** (`agent_rl.db`): approval queue empties; only learning resets — business data (eway_bills, alerts) untouched.
13. **Badge for multi-user:** `alerts.user_id` routing — unread count is per-session user (`app.go` fix); anonymous/marketing pages untouched.
14. **SOS spam:** dedup key `sos:<trip_id>` + cooldown 60s; repeated triggers bump `occurrences` and re-alert telegram every escalation step only.

---

## 17. Phased rollout

- **Phase 1 — Alerting core:** 00045; pipeline engine + dedup/cooldown; in_app (badge fix `app.go` + `layout.html`) + telegram adapters; alerts list page; storms. Ships without producers beyond telemetry bus events → wire `telemetry_service.go` persistence + type rename.
- **Phase 2 — Compliance:** 00046; `CheckDispatchCompliance`; wire both assignment paths + StartTrip; doc vault UI for rc/fitness/puc; block banner in `trip_view.html`; `compliance_exemptions` admin UI.
- **Phase 3 — E-way bill (00047, owned by spec 07):** worker generate → Part-B → extension (gated) → cancel; `eway_bill_events` audit; agent `extend_ewaybill` + `get_open_alerts` + gate + RL hooks; publish `TripAssignedEvent` from both assignment paths.
- **Phase 4 — SOS + RBAC (00046, this spec):** SOS subscriber + admin UI + escalation; `SOS_ESCALATION_PHONE` stub.
- **Phase 5 — Telemetry types + AIS-140 (00059):** ingestion spec producers align to the 13 types; `internal/vlt/contract.go` design-only doc.

Each phase lands behind env flags where useful (`ALERTS_ENABLED`, `EWAYBILL_WORKER_ENABLED`) so rollout is reversible; stubs guarantee no external side effects until providers are verified.

---

## 18. VERIFY items (at implementation)

1. **E-way Part-B provider support** — NIC e-way bill API vehicle-update endpoint; add `UpdateVehiclePartB` to `internal/integration/ewaybill/client.go` interface. Stub returns fake success today.
2. **E-way extension provider support** — `Extend` method; stub returns `now+24h`. Also confirm statutory extension limit (max extensions/validity) and encode as rule.
3. **Email provider** — SMTP vs transactional API (Postmark/SES/SendGrid). Currently log-only; do not recommend without verification.
4. **WhatsApp provider** — WhatsApp Business Cloud API vs Msg91 vs Twilio; country-specific template/consent rules for India fleet ops. Log-only until verified.
5. **SMS provider** — Twilio vs Msg91 (India DLT template registration required). Log-only until verified.
6. **SOS escalation phone provider** — Twilio Voice vs Msg91 click-to-call; verify emergency calling semantics + consent. Log-only until verified.
7. **Telegram channel reuse** — confirm `FOUNDER_TELEGRAM_*` bot token is shared or separate `ALERT_TELEGRAM_*` is provisioned; chat ID parse pattern in `cmd/server/main.go:683-685`.
8. **Geofence evidence source** — the `geofence_events` bus topic is a contract from the geofence/ingestion spec; confirm its payload field names (`trip_id`, `fence_id`, `lat`, `lng`) before wiring the extension gate.
9. **`TripAssignedEvent` publication** — currently never published (`internal/domain/trip/events.go:28` exists but no publisher); adding publishes from both assignment paths must not break existing subscribers (none subscribe today).
10. **`notifications` legacy table (00025)** — still dead after Phase 1; confirm no other feature (login audit, error reporter) depends on the in-memory `notifications.Service` before deprecating; keep `internal/operations/notifications` as-is for `opserrors.Reporter`/`loginAuditSvc` (they use the `ports.NotificationService` interface — the alert pipeline is separate).
11. **Viewer role scope** — 00046 viewer grants `alerts:read` etc.; confirm with product that viewers should see alerts (or restrict to dispatcher+).
12. **FlyFleet branding** — all user-facing alert copy (login alert `login_audit.go:70`, digest `daily.go:22`) uses FlyFleet; new alert templates must match (no "Avandab" in user-facing strings; Avandab stays the internal code name).
