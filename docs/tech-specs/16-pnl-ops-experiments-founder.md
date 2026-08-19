# PNL / Ops / Experiments / Founder Hardening — Implementation Spec v1

Status: ready
Depends-on: 00-migration-ownership-index.md (migration **00058** reserved), 01-telematics-ingestion.md, 03-fuel-audit-scorecard.md
Migration owner: db/migrations/00058_pnl_ops_experiments_founder.sql

This spec hardens four subsystems that currently work *partially* or *only in
memory*. It makes them persistent, multi-tenant, observable, and admin-driven.
Every change is copy-pasteable. Tenant is ALWAYS derived from `auth.ContextUser`,
never from a client body field.

---

## 0. Verified ground truth (file:line facts)

### 0.1 PNL (`internal/pnl/`)
- `internal/pnl/service.go:39` `Calculate()` derives margin for ONE trip only from
  `bookings.price`, `telemetry_snapshots` odometer, `vehicles.current_mileage`
  (efficiency), latest `fuel_prices` row, and approved `driver_expenses`. It
  **writes the result back onto the `trips` row** (`service.go:113`) and returns a
  `LivePnL`. There is **no aggregate/period API** and **no history table**.
- Errors are **swallowed** with `_ =`:
  - `internal/pnl/service.go:61-62` odometer lookup error ignored
  - `internal/pnl/service.go:79` fuel price lookup error ignored
  - `internal/pnl/service.go:93-100` toll + kharcha expense lookups error ignored
- `internal/pnl/handler.go:14` exposes only `GET /api/v1/trips/{id}/pnl`. On any
  error it returns `404` with the hardcoded body `{"error":"trip P&L unavailable"}`
  and also **swallows** the real error (`handler.go:16-18`). No UI page exists.
- Fuel-price source column is chosen by `fuel_type` (`service.go:69-76`);
  uncertainty 0.15 diesel/petrol, 0.25 cng/gas. When data is missing,
  `fuel_cost_status = "pending_verification"` and margin is NOT computed
  (`service.go:89-111`).
- Existing `trips` P&L columns (from `00033_live_trip_pnl.sql` / `00034_pnl_uncertainty.sql`,
  see `db/generated/sqlite/models.go:391,396`): `estimated_margin`,
  `fuel_consumed_liters`, `toll_costs`, `last_pnl_update`, `fuel_cost_low`,
  `fuel_cost_high`, `margin_low`, `margin_high`, `pnl_confidence`, `fuel_cost_status`.
- Config uncertainty is **hardcoded** in code (`service.go:73-76`); not externally tunable.

### 0.2 Operations health (`internal/operations/health/checker.go`)
- `checker.go:57-58`: `outbox_workers` and `notification_service` components are
  **hardcoded `Healthy: true`** with no probe. The DB ping (`:43`) is the only real check.
- `checker.go:32` `HealthHandler` aggregates components; `ReadinessHandler` (`:77`)
  only checks DB. No notion of "stale outbox" or "notification sink down".

### 0.3 Notifications (`internal/operations/notifications/service.go`)
- `service.go:38` `SendEmail` = **log stub only** (`log.Printf`), returns nil, never sends.
- `service.go:44-71` `SendInApp` persists to an **in-memory `map[string][]Notification`**
  (`inAppStore` at `:27`); lost on restart, capped at `maxInAppPerKey = 100` (`:30`).
- `service.go:73` `SendSMS`, `:77` `SendPush`, `:81` `SendWebhook` all `return fmt.Errorf(...)`
  ("not configured yet"). No real adapters.
- Interface is `ports.NotificationService` (`internal/shared/ports/notification.go:25`)
  with `NotificationMessage{ TenantID, UserID, Recipient, Subject, Body, Type, Metadata }`.
- The DB already has a `notifications` table (`00025_notifications_and_file_categories.sql`)
  with `id, user_id, title, message, channel, status, link, created_at, read_at` — but the
  in-memory service does NOT use it.

### 0.4 Errors (`internal/operations/errors/reporter.go`)
- `reporter.go:53-54`: `errors []ErrorReport` and `incidents []Incident` are **in-memory
  slices**. Lost on process restart. No dedup. Capped at 500/200 (`:84-85`).
- `reporter.go:70` `Report()` auto-creates an `Incident` for `CRITICAL`/`HIGH`
  (`:93-105`) and emails tech on `CRITICAL` (`:109-116`) — but the email goes to a
  hardcoded `tech-alerts@flyfleet.io` via the stub `SendEmail`, so it is never delivered.
- `reporter.go:121` `ListErrors()` and `:129` `ListIncidents()` read memory only.
- There is NO HTTP API to list errors/incidents anywhere. `reporter_test.go` exists.

### 0.5 Login audit (`internal/operations/audit/login_audit.go`)
- `login_audit.go:29` `history []LoginRecord` is **in-memory**; `:32` `knownDevices`
  map is in-memory. No persistence. `GetLoginHistory(userID)` (`:88`) scans memory.

### 0.6 Experiments (`internal/experiments/experiments.go`)
- `experiments.go:28` `Assign(rollout, force, tenantID, userID)` hashes
  `sha1(tenantID + ":" + userID)` (`:41`). For an **anonymous** visitor `userID == ""`,
  so every anonymous user hashes to the **same bucket** → all anon users get the same
  variant (collapse bug). There is **no salt**, **no sticky persistence**
  (assignment recomputed each call from the hash, deterministic but not stored),
  **no per-experiment config**, and only **one global experiment** `dashboard_v2`
   (`experiments.go:17`); `rollout` is a **caller-supplied parameter** to `Assign`,
   not a stored constant (no per-experiment config persists it).
- `Recorder.Record`/`RecordAsync` (`:66`,`:88`) write to `experiment_events` table
  (`00039_experiments.sql`) — that table EXISTS and persists. Assignment table does NOT.
- `newID` (`:105`) format `prefix-<unixnano>-<milli%100000>`.

### 0.7 Founder (`internal/founder/`)
- `service.go:14` `FounderService` holds a single `Notifier` interface (`:18-20`).
- `alerts/notifier.go:81` `TelegramBotNotifier.SendAlert` returns **nil (silent noop)**
  when `bot == nil || chatID == 0` (`:82-84`) — no warning, no count, no fallback.
- `cmd/server/main.go:883-884` `noopNotifier{}` is used when Telegram is unconfigured.
- `main.go:886` `founderConfigured()` requires both `FOUNDER_TELEGRAM_BOT_TOKEN` and
  `FOUNDER_TELEGRAM_CHAT_ID`. When unset, `runDailyDigest` (`:914`) still sends a
  **zero-valued** `digest.DailyDigestReport{}` (`:933`) — `visitors/signups/...` all 0.
- `service.go:119` `SendDailyDigest` formats `digest.FormatDailyDigest` (daily.go:21)
   which only has zero-valued scalar fields. `AlertEvent` **does** carry a
   `CompanyID` field (`alerts/event.go:34`), but no handler currently populates it —
   so `company_id` is never set on emitted alerts (verified: zero `CompanyID` writes
   in `internal/founder/service.go`).
- Categories exist (`alerts/event.go:17-25`): SYSTEM, REVENUE, CHURN_RISK, SECURITY,
  PRODUCT_USAGE, ACTIVATION, TRIAL_MONITORING, PAYMENT,   CUSTOMER_SUCCESS.

### 0.8 Verification Log (Principal Engineer QA pass)

Every claim below was read against live source on 2026-08-19. Verdicts: T = True,
F = False (corrected in body), I = Imprecise (corrected).

| # | Claim (spec §) | Verdict | Correction / Evidence | Sev | Effort |
|---|----------------|---------|------------------------|-----|--------|
| 1 | PNL `Calculate` writes `trips`, no aggregate/period API, no history, swallows errors (§0.1) | T | `service.go:113` UPDATE trips; handler only `GET /api/v1/trips/{id}/pnl` (`handler.go:14`); no `Aggregate`/`period` in `internal/pnl/`; `_ =` at `service.go:61-62,79,93,100` | – | – |
| 2 | PNL handler 404 hardcoded + swallows real err (§0.1) | T | `handler.go:17` `http.Error(w,...,404)` with fixed body, real err discarded | – | – |
| 3 | Health checker hardcodes `outbox_workers`+`notification_service` healthy (§0.2) | T | `checker.go:57-58` append `Healthy:true` no probe; only DB ping `:43` real | – | – |
| 4 | Email=log stub, InApp=in-memory, SMS/Push/Webhook=error (§0.3) | T | `service.go:38` log stub; `:27`/`maxInAppPerKey:30` map; `:73/:77/:81` `fmt.Errorf` | – | – |
| 5 | `notifications` table 00025 exists but unused (§0.3) | T | cols `id,user_id,title,message,channel,status,link,created_at,read_at`; service uses map only | – | – |
| 6 | Errors reporter in-memory, no dedup, lost on restart (§0.4) | T | `reporter.go:53-54` slices; caps `:84-85`; no fingerprint/dedup | – | – |
| 7 | Errors auto-incident CRITICAL/HIGH + emails via stub (never delivered) (§0.4) | T | `:93-105` incident; `:109-116` `SendEmail` to `tech-alerts@flyfleet.io` via stub | – | – |
| 8 | No HTTP API for errors/incidents (§0.4) | T | grep: no `/api` route in `errors/` or `main.go` | – | – |
| 9 | `login_audit` in-memory (§0.5) | T | `login_audit.go:29` `history` slice; `:32` `knownDevices` map; `GetLoginHistory:88` scans mem | – | – |
| 10 | Experiments `Assign` hashes `tenantID+userID`, anon `userID=""` collapses, no sticky, 1 experiment, no per-experiment config, no admin API (§0.6) | T | `experiments.go:41` `sha1(tenantID+":"+userID)`; `:17` only `dashboard_v2`; no assignment table; no HTTP handlers | – | – |
| 11 | `Recorder` persists to `experiment_events` (00039) (§0.6) | T | `experiments.go:66/:88` INSERT `experiment_events`; migration `00039` present | – | – |
| 12 | Founder Telegram-only, silent noop when unconfigured, daily digest zero-valued (§0.7) | T | only `TelegramBotNotifier` + `noopNotifier` (`main.go:881-883`); `notifier.go:82-84` `return nil`; `main.go:933` `DailyDigestReport{Date:...}` rest 0 | – | – |
| 13 | `AlertEvent` has no company field (§0.7) | **F** | `alerts/event.go:34` **has** `CompanyID string`; field exists but never populated. Body corrected. | Low | Low |
| 14 | Single hardcoded `rollout` in experiments (§0.6) | **I** | `rollout` is a **parameter** to `Assign`, not a stored constant. Body corrected. | Low | Low |
| 15 | Outbox migration `00053` (§5.3) | **F** | Only `00020_create_outbox_events.sql` exists; `00053` absent (head `00039`). Body corrected. | Low | Low |
| 16 | Migration `00058` free / no conflict | T | `ls db/migrations` → max `00039`; `00058` absent; no `00-index` file present | – | – |
| 17 | `ports.NotificationService` interface at `notification.go:25` (§0.3) | T | `internal/shared/ports/notification.go:25` confirms 5-method interface | – | – |

### 0.9 Decisions & Tradeoffs (explicit)

- **Notification channel adapters (SMTP / SMS / Push / Webhook)**
  - Decision: implement real adapters behind `ports.NotificationService`; SMTP via stdlib
    `net/smtp`, SMS/Push/Webhook as MOCK-gated no-ops until `SMS_ENABLED`/`PUSH_ENABLED`/`WEBHOOK_ENABLED`.
  - Tradeoff: stdlib `net/smtp` avoids a new dependency (`go-mail`) at cost of no pooled
    connection / retry; MOCK gate keeps prod safe when unconfigured.
  - Cost: ~3 new adapter files + wiring; low risk. Reuses existing env + `notification_log`.

- **Error dedup strategy**
  - Decision: `fingerprint = sha1(method+url+firstLine(message)+tenant_id)`; `INSERT … ON CONFLICT`
    path → `UPDATE occurrences=occurrences+1, last_seen=now`. New `error_reports` table (00058).
  - Tradeoff: fingerprint (not full stack) chosen so transient re-errors collapse into one
    incident row; risk of over-merging distinct errors with identical first lines — mitigated by
    including `tenant_id` + `url`.
  - Cost: schema + reporter rewrite; medium effort, high ops value.

- **Experiments sticky assignment (cookie vs DB)**
  - Decision: **DB** — new `experiment_assignments` table (00058); first call writes variant,
    subsequent calls read stored value. Anon uses `anon_token` cookie + `EXPERIMENT_SALT`.
  - Tradeoff: DB gives cross-device/server-restart stickiness and admin audit; cookie-only
    would break on clear/incognito and can't be queried. Cost: extra write per first assignment.
  - Cost: 1 table + `assignments.go`; medium effort.

- **Founder multi-channel**
  - Decision: `MultiChannelNotifier` fans out to every enabled channel in
    `founder_alert_channels`; disabled/misconfigured → `logger.Warn` + `noop_dropped_total++`
    (never silent nil). Populate `AlertEvent.CompanyID` (field already exists, `event.go:34`).
  - Tradeoff: replaces silent noop with observable drops; ensures founders see *something*.
    Cost: new `multichannel.go` + stats table; low/medium effort.

---

## 1. Overview / goal

Make four subsystems production-real:
1. **PNL**: add aggregate + period queries and a UI; persist a `pnl_snapshots`
   history table; stop swallowing errors (return them); expose config (uncertainty)
   via DB/env.
2. **Operations**: replace hardcoded health probes with real ones (stale outbox,
   notification sink ping); add real notification channel adapters (SMTP / SMS / Push /
   Webhook) behind the existing `ports.NotificationService` interface, persist InApp to
   `notifications` + a new `notification_log`; persist errors + incidents with
   fingerprint dedup; persist `login_audit`.
3. **Experiments**: fix the anonymous collapse (anon cookie + salt), add a sticky
   `experiment_assignments` table, add per-experiment config + lifecycle, and an admin API.
4. **Founder**: multi-channel dispatcher (Telegram / Email / Slack / SMS); a noop MUST
   warn + count dropped alerts (never silently swallow); populate `company_id`; wire real
   digest data from persisted tables.

**Non-goals**: GSTN/EWB integrations (spec 07), GPS provider adapters (spec 01), RAG/agent
RL (separate). We do NOT change `experiment_events` schema (keep append-only). We do NOT
create `company_config` (owned by 00042).

---

## 2. API contract

All routes require a valid session or Bearer token (mounted under the `/api/v1` group in
`cmd/server/main.go:443`). Tenant is taken from `auth.ContextUser`. RBAC resource names
used with `middleware.RequirePermission` (`internal/middleware/api_auth.go:50`).

### 2.1 PNL

#### GET /api/v1/pnl/trips  — list per-trip live P&L (existing single-trip upgraded)
Auth: `pnl` `read`.
Query: `?from=2026-01-01&to=2026-01-31&low_margin_only=true&limit=100&offset=0`
Tenant-scoped. Returns trip-level `LivePnL` rows (recomputed via `Calculate` or read
cached `trips` columns).

```json
{
  "count": 2,
  "period": { "from": "2026-01-01", "to": "2026-01-31" },
  "rows": [
    {
      "trip_id": "trip-1", "quoted_fare": 12000, "fuel_cost": 3800,
      "fuel_cost_low": 3230, "fuel_cost_high": 4370, "toll_cost": 600,
      "kharcha_approved": 400, "fuel_consumed_liters": 52.1,
      "estimated_margin": 7200, "margin_percentage": 60.0,
      "margin_low": 6890, "margin_high": 7510, "low_margin": false,
      "margin_available": true, "fuel_cost_status": "estimated",
      "confidence": "medium", "last_update": "2026-01-15T10:00:00Z"
    }
  ]
}
```

#### GET /api/v1/pnl/aggregate  — aggregate P&L for a period (NEW)
Auth: `pnl` `read`.
Query: `?from=2026-01-01&to=2026-01-31&granularity=day|week|month`
```json
{
  "period": { "from": "2026-01-01", "to": "2026-01-31" },
  "totals": {
    "quoted_fare": 240000, "fuel_cost": 76000, "toll_cost": 12000,
    "kharcha_approved": 8000, "estimated_margin": 144000,
    "margin_percentage": 60.0, "trips": 20, "low_margin_trips": 2,
    "fuel_cost_status": "estimated"
  },
  "series": [
    { "bucket": "2026-01-01", "quoted_fare": 12000, "estimated_margin": 7200,
      "fuel_cost": 3800, "toll_cost": 600, "kharcha_approved": 400, "trips": 1 }
  ],
  "confidence": "medium",
  "generated_at": "2026-01-31T23:59:00Z"
}
```

#### GET /api/v1/pnl/snapshots  — history of persisted aggregates (NEW)
Auth: `pnl` `read`. Query `?from=&to=&granularity=`
```json
{ "count": 3, "rows": [
  { "id": "snap-...", "tenant_id": "1", "granularity": "month",
    "period_start": "2026-01-01", "period_end": "2026-01-31",
    "quoted_fare": 240000, "fuel_cost": 76000, "toll_cost": 12000,
    "kharcha_approved": 8000, "estimated_margin": 144000,
    "margin_percentage": 60.0, "trips": 20, "low_margin_trips": 2,
    "fuel_cost_status": "estimated", "confidence": "medium",
    "created_at": "2026-02-01T00:05:00Z" }
] }
```

#### POST /api/v1/pnl/snapshots/rollup  — persist current aggregate as a snapshot (NEW)
Auth: `pnl` `write`. Body `{"granularity":"month","period_start":"2026-01-01","period_end":"2026-01-31"}`
→ `202` with the created snapshot row. Used by a nightly cron / outbox job.

#### GET /api/v1/pnl/config  /  PUT /api/v1/pnl/config  — uncertainty config (NEW)
Auth read: `pnl` `read`; write: `pnl` `write` (admin).
GET returns:
```json
{ "uncertainty": { "diesel": 0.15, "petrol": 0.15, "cng": 0.25, "gas": 0.25 },
  "source": "db" }
```
PUT body: `{"diesel":0.15,"petrol":0.15,"cng":0.25,"gas":0.25}`.

Error codes:
- `400` bad query/body
- `403` missing permission
- `404` trip not found (real error surfaced, NOT swallowed)
- `500` DB/compute error with `{"error": "...", "detail": "..."}`

### 2.2 Operations

#### GET /api/v1/health  (existing, upgraded probes)
Returns real component status. New components:
```json
{ "status": "DEGRADED", "timestamp": "2026-01-15T10:00:00Z",
  "components": [
    {"name":"database","healthy":true},
    {"name":"outbox_workers","healthy":false,"message":"oldest pending outbox event is 14m old (threshold 5m)"},
    {"name":"notification_service","healthy":false,"message":"SMTP ping failed: dial tcp 10.0.0.5:587 timeout"}
  ] }
```

#### GET /api/v1/errors  — list persisted errors (NEW)
Auth: `errors` `read`. Query `?severity=CRITICAL&from=&to=&limit=100&offset=0&fingerprint=`
```json
{ "count": 1, "rows": [
  { "id":"err-...","fingerprint":"a1b2c3","tenant_id":"1","user_id":"u1",
    "url":"/api/v1/x","method":"POST","status_code":500,
    "severity":"CRITICAL","message":"nil pointer","environment":"prod",
    "occurrences":12,"first_seen":"2026-01-15T09:00:00Z",
    "last_seen":"2026-01-15T10:00:00Z","created_at":"2026-01-15T09:00:00Z" } ] }
```

#### GET /api/v1/errors/{id}  — error detail (NEW) → `404` if absent.
#### GET /api/v1/incidents  — list incidents (NEW) Auth `errors` `read`.
Query `?status=OPEN`.
```json
{ "count":1, "rows":[
  {"id":"inc-...","error_id":"err-...","status":"OPEN","severity":"CRITICAL",
   "created":"2026-01-15T09:00:00Z","assigned_to":"","resolved_at":null,"root_cause":""} ] }
```
#### POST /api/v1/incidents/{id}/resolve  — resolve incident (NEW) Auth `errors` `write`.
Body `{"root_cause":"bad migration"}` → `200`.

#### GET /api/v1/audit/logins  — login audit (NEW) Auth `audit` `read`.
Query `?user_id=&success=&from=&to=`.
```json
{ "count":1, "rows":[
  {"id":"la-...","user_id":"u1","user_email":"a@b.co","ip_address":"1.2.3.4",
   "user_agent":"Mozilla/...","success":true,"timestamp":"2026-01-15T10:00:00Z"} ] }
```

#### GET /api/v1/notifications/preferences  / PUT (NEW) Auth `notifications` `read`/`write`
```json
{ "channels": { "email": true, "sms": false, "push": false, "in_app": true, "webhook": false },
  "webhook_url": "", "email_address": "ops@flyfleet.io" }
```

### 2.3 Experiments

#### GET /api/v1/experiments  — list experiments + config (NEW) Auth `experiments` `read`
```json
{ "count":1, "rows":[
  {"key":"dashboard_v2","name":"Dashboard Redesign","status":"RUNNING",
   "rollout":50,"variants":["A","B"],"sticky":true,
   "audience":"all","salt":"<env EXPERIMENT_SALT>",
   "created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-15T00:00:00Z"} ] }
```
#### POST /api/v1/experiments  — create experiment (NEW) Auth `experiments` `write` (admin)
Body `{"key":"checkout_v2","name":"Checkout","rollout":10,"variants":["A","B"],"sticky":true,"audience":"all"}` → `201`.

#### PUT /api/v1/experiments/{key}  — update rollout/status/lifecycle (NEW) Auth `experiments` `write`
Body `{"rollout":75,"status":"RUNNING"}` or `{"status":"STOPPED"}` or `{"status":"ARCHIVED"}`.

#### GET /api/v1/experiments/{key}/assign?user_id=u1  — resolve variant (NEW)
Auth `experiments` `read`. Returns sticky assignment (writes `experiment_assignments` on first call):
```json
{ "experiment":"dashboard_v2","user_id":"u1","variant":"B","sticky":true,
  "assigned_at":"2026-01-15T10:00:00Z" }
```
Anonymous: pass `?anon_token=<cookie>` instead of `user_id`; server hashes with salt.

### 2.4 Founder

#### GET /api/v1/founder/channels  / PUT (NEW) Auth `founder` `read`/`write` (admin)
```json
{ "channels": {
    "telegram": {"enabled": true, "chat_id": "-100123"},
    "email":    {"enabled": true, "smtp_to": "founder@flyfleet.io"},
    "slack":    {"enabled": false, "webhook_url": ""},
    "sms":      {"enabled": false, "to": ""} },
  "digest_hour": 9, "noop_dropped_total": 0 }
```
#### GET /api/v1/founder/digest/preview  — preview digest with REAL data (NEW) Auth `founder` `read`
Returns `digest.DailyDigestReport` populated from `error_reports`/`incidents`/signups.

---

## 3. DB contract — migration 00058 (`db/migrations/00058_pnl_ops_experiments_founder.sql`)

`-- +goose Up` creates the following. Tenant column `tenant_id TEXT NOT NULL DEFAULT '1'`
on every table for scoping. All timestamps stored as `TEXT` ISO-8601 (matches existing
`experiment_events`).

```sql
-- +goose Up

-- 1) PNL aggregate history ------------------------------------------------
CREATE TABLE IF NOT EXISTS pnl_snapshots (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL DEFAULT '1',
    granularity     TEXT NOT NULL,                -- day | week | month
    period_start    TEXT NOT NULL,
    period_end      TEXT NOT NULL,
    quoted_fare     REAL NOT NULL DEFAULT 0,
    fuel_cost       REAL NOT NULL DEFAULT 0,
    fuel_cost_low   REAL NOT NULL DEFAULT 0,
    fuel_cost_high  REAL NOT NULL DEFAULT 0,
    toll_cost       REAL NOT NULL DEFAULT 0,
    kharcha_approved REAL NOT NULL DEFAULT 0,
    estimated_margin REAL NOT NULL DEFAULT 0,
    margin_percentage REAL NOT NULL DEFAULT 0,
    trips           INTEGER NOT NULL DEFAULT 0,
    low_margin_trips INTEGER NOT NULL DEFAULT 0,
    fuel_cost_status TEXT NOT NULL DEFAULT 'estimated',
    confidence      TEXT NOT NULL DEFAULT 'medium',
    created_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pnl_snapshots_tenant_period
    ON pnl_snapshots (tenant_id, granularity, period_start);

-- PNL uncertainty config (per fuel type), tenant-scoped
CREATE TABLE IF NOT EXISTS pnl_config (
    tenant_id   TEXT NOT NULL DEFAULT '1',
    fuel_type   TEXT NOT NULL,                    -- diesel | petrol | cng | gas
    uncertainty REAL NOT NULL DEFAULT 0.15,
    PRIMARY KEY (tenant_id, fuel_type)
);

-- 2) Notification log (persisted deliveries) ------------------------------
CREATE TABLE IF NOT EXISTS notification_log (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    user_id     TEXT,
    channel     TEXT NOT NULL,                    -- EMAIL | IN_APP | SMS | PUSH | WEBHOOK
    recipient   TEXT NOT NULL,
    subject     TEXT NOT NULL,
    body        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'sent',     -- sent | failed | suppressed
    error       TEXT,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notification_log_tenant
    ON notification_log (tenant_id, created_at);

-- Per-user notification preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    tenant_id   TEXT NOT NULL DEFAULT '1',
    user_id     TEXT NOT NULL,
    channel_email INTEGER NOT NULL DEFAULT 1,
    channel_sms   INTEGER NOT NULL DEFAULT 0,
    channel_push  INTEGER NOT NULL DEFAULT 0,
    channel_in_app INTEGER NOT NULL DEFAULT 1,
    channel_webhook INTEGER NOT NULL DEFAULT 0,
    webhook_url TEXT NOT NULL DEFAULT '',
    email_address TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_id, user_id)
);

-- Notification channel config (global + per-tenant). Secrets live in env,
-- referenced by key; this table only stores enable flags + non-secret targets.
CREATE TABLE IF NOT EXISTS notification_channel_config (
    tenant_id     TEXT NOT NULL DEFAULT '1',
    channel       TEXT NOT NULL,                  -- EMAIL | SMS | PUSH | WEBHOOK
    enabled       INTEGER NOT NULL DEFAULT 0,
    target        TEXT NOT NULL DEFAULT '',       -- smtp_to / webhook_url / phone
    adapter       TEXT NOT NULL DEFAULT 'MOCK',   -- SMTP | TWILIO | FCM | MOCK
    PRIMARY KEY (tenant_id, channel)
);

-- 3) Errors + incidents (persisted, deduped) ------------------------------
CREATE TABLE IF NOT EXISTS error_reports (
    id          TEXT PRIMARY KEY,
    fingerprint TEXT NOT NULL,                    -- sha1(type+url+msg)
    tenant_id   TEXT NOT NULL DEFAULT '1',
    user_id     TEXT,
    url         TEXT NOT NULL DEFAULT '',
    method      TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    severity    TEXT NOT NULL DEFAULT 'MEDIUM',
    message     TEXT NOT NULL,
    stack_trace TEXT NOT NULL DEFAULT '',
    environment TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    occurrences INTEGER NOT NULL DEFAULT 1,
    first_seen  TEXT NOT NULL,
    last_seen   TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_error_reports_fingerprint
    ON error_reports (fingerprint);
CREATE INDEX IF NOT EXISTS idx_error_reports_tenant_sev
    ON error_reports (tenant_id, severity, last_seen);

CREATE TABLE IF NOT EXISTS incidents (
    id          TEXT PRIMARY KEY,
    error_id    TEXT NOT NULL,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    status      TEXT NOT NULL DEFAULT 'OPEN',     -- OPEN | ASSIGNED | RESOLVED
    severity    TEXT NOT NULL DEFAULT 'HIGH',
    assigned_to TEXT NOT NULL DEFAULT '',
    root_cause  TEXT NOT NULL DEFAULT '',
    created     TEXT NOT NULL,
    resolved_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents (status, tenant_id);

-- 4) Login audit (persisted) ----------------------------------------------
CREATE TABLE IF NOT EXISTS login_audit (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL DEFAULT '1',
    user_id     TEXT NOT NULL,
    user_email  TEXT NOT NULL DEFAULT '',
    ip_address  TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    success     INTEGER NOT NULL DEFAULT 1,
    timestamp   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_login_audit_user ON login_audit (tenant_id, user_id, timestamp);

-- 5) Experiments: assignments + config ------------------------------------
CREATE TABLE IF NOT EXISTS experiment_assignments (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT NOT NULL DEFAULT '1',
    experiment    TEXT NOT NULL,
    user_id       TEXT NOT NULL DEFAULT '',       -- '' for anonymous + anon_token
    anon_token    TEXT NOT NULL DEFAULT '',
    variant       TEXT NOT NULL,
    assigned_at   TEXT NOT NULL,
    UNIQUE (tenant_id, experiment, user_id, anon_token)
);
CREATE INDEX IF NOT EXISTS idx_exp_assign
    ON experiment_assignments (tenant_id, experiment, user_id, anon_token);

CREATE TABLE IF NOT EXISTS experiments_config (
    key          TEXT PRIMARY KEY,
    tenant_id    TEXT NOT NULL DEFAULT '1',
    name         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'DRAFT',  -- DRAFT | RUNNING | STOPPED | ARCHIVED
    rollout      INTEGER NOT NULL DEFAULT 0,     -- 0..100
    variants     TEXT NOT NULL DEFAULT '["A","B"]',
    sticky       INTEGER NOT NULL DEFAULT 1,
    audience     TEXT NOT NULL DEFAULT 'all',
    salt         TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

-- 6) Founder alert channels -----------------------------------------------
CREATE TABLE IF NOT EXISTS founder_alert_channels (
    channel      TEXT PRIMARY KEY,                -- telegram | email | slack | sms
    enabled      INTEGER NOT NULL DEFAULT 0,
    target       TEXT NOT NULL DEFAULT '',       -- chat_id / smtp_to / webhook / phone
    adapter      TEXT NOT NULL DEFAULT 'MOCK',   -- TELEGRAM | SMTP | SLACK | SMS | MOCK
    updated_at   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS founder_alert_stats (
    id          TEXT PRIMARY KEY,
    noop_dropped_total INTEGER NOT NULL DEFAULT 0,
    updated_at  TEXT NOT NULL
);

-- Seeds: default pnl_config (diesel/petrol 0.15, cng/gas 0.25)
INSERT OR IGNORE INTO pnl_config (tenant_id, fuel_type, uncertainty) VALUES
    ('1','diesel',0.15), ('1','petrol',0.15), ('1','cng',0.25), ('1','gas',0.25);

-- Seeds: enable Telegram founder channel by default OFF (configured via env at runtime)
INSERT OR IGNORE INTO founder_alert_channels (channel, enabled, target, adapter, updated_at)
    VALUES ('telegram',0,'','TELEGRAM', datetime('now'));

-- Seed RBAC permissions (roles 1=admin, 2=default per 00012_rbac.sql / 00001_initial.sql)
INSERT OR IGNORE INTO permissions (resource, action, description) VALUES
    ('pnl','read','View P&L and aggregates'),
    ('pnl','write','Configure P&L and roll up snapshots'),
    ('errors','read','View error reports and incidents'),
    ('errors','write','Resolve incidents'),
    ('audit','read','View login audit'),
    ('notifications','read','View notification preferences'),
    ('notifications','write','Edit notification preferences'),
    ('experiments','read','View experiments'),
    ('experiments','write','Create/update experiments'),
    ('founder','read','View founder channels/digest'),
    ('founder','write','Configure founder channels');
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
    SELECT 1, id FROM permissions WHERE resource IN ('pnl','errors','audit','notifications','experiments','founder');
-- tenant_id default '1' chosen to match existing experiment_events default (00039).

-- +goose Down
DROP TABLE IF EXISTS pnl_snapshots;
DROP TABLE IF EXISTS pnl_config;
DROP TABLE IF EXISTS notification_log;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notification_channel_config;
DROP TABLE IF EXISTS error_reports;
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS login_audit;
DROP TABLE IF EXISTS experiment_assignments;
DROP TABLE IF EXISTS experiments_config;
DROP TABLE IF EXISTS founder_alert_channels;
DROP TABLE IF EXISTS founder_alert_stats;
-- NOTE: do not DROP company_config (owned by 00042). We did not create it.
```

> `permissions` / `role_permissions` tables are assumed to exist (from `00012_rbac.sql`).
> If column `description` is absent on `permissions`, drop it from the INSERT.

### Company_config sequencing guard

`company_config` is created by Spec 02 @00042. THIS migration (00058) seeds/uses it (seeds `pnl_config` instead — see §3; 00058 only references `company_config`, e.g. §6 config mirroring, and never creates it). Two mitigations, choose one at implementation time: (a) build Spec 02 first (recommended), or (b) prepend `CREATE TABLE IF NOT EXISTS company_config (...)` guard with the canonical schema from Spec 02 @00042 so this migration never crashes `goose up` if run before 00042. Do not invent the schema here — reference Spec 02's canonical DDL.

---

## 4. UI

All UI mounts under the protected web app (server-rendered templates + htmx partials,
matching existing admin pages). RBAC resources above gate each page.

- `/ops/pnl` — P&L dashboard: date-range picker, granularity toggle (day/week/month),
  aggregate KPI cards (quoted fare, fuel cost, toll, margin, margin %), a low-margin
  trips table, and a "history" tab reading `/api/v1/pnl/snapshots`. Template:
  `web/templates/pnl_dashboard.html`; partial `web/templates/partials/pnl_series.htmx`.
- `/ops/errors` — error list + detail drawer; resolve button on incidents.
  `web/templates/errors.html`.
- `/ops/audit/logins` — login audit table. `web/templates/login_audit.html`.
- `/ops/health` — live health panel polling `/api/v1/health` every 15s, showing
  component chips (green/red). `web/templates/health.html`.
- `/experiments` — experiment list, create modal, rollout slider, lifecycle controls,
  sticky toggle. `web/templates/experiments.html`.
- `/settings/notifications` — channel preference form (email/sms/push/in_app/webhook).
  `web/templates/notification_prefs.html`.
- `/founder/channels` — founder multi-channel config form + digest preview button.
  `web/templates/founder_channels.html`.
- JS assets: `web/assets/js/pnl.js`, `web/assets/js/experiments.js`,
  `web/assets/js/health.js`, `web/assets/js/founder.js`.

Wire routes in `cmd/server/main.go` protected group next to `pnl.RegisterRoutes`
(`main.go:449`). Register experiment/founder/errors handlers similarly.

---

## 5. Business logic

### 5.1 PNL aggregate + history (`internal/pnl/aggregate.go` NEW)
- `Aggregate(ctx, from, to, granularity)` computes period totals by scanning `trips`
  (cached columns from `Calculate`) OR recomputing. Use cached `trips.estimated_margin`
  etc. when `last_pnl_update` is within period; otherwise recompute via `Calculate`.
- Bucketing: `day`→`substr(period_start,1,10)`, `week`→`date(period_start,'weekday 1')`,
  `month`→`substr(period_start,1,7)`.
- Rollup: `SaveSnapshot(ctx, granularity, periodStart, periodEnd)` inserts into
  `pnl_snapshots`. Called by `POST /api/v1/pnl/snapshots/rollup` and the nightly cron.
- Config: read uncertainty from `pnl_config` (fallback to code defaults). Add
  `loadUncertainty(ctx, tenantID) map[string]float64` reading the table; if empty, use
  `service.go:73-76` constants.

### 5.2 Stop swallowing errors
Refactor `service.go:61,79,93,100` to return wrapped errors instead of `_ =`:
```go
if err := s.db.QueryRowContext(ctx, ...).Scan(&startOdometer, &latestOdometer); err != nil {
    return LivePnL{}, fmt.Errorf("pnl: odometer lookup failed for trip %s: %w", tripID, err)
}
```
`handler.go:16` must return `500` (or `404` only on `sql.ErrNoRows`) and encode the
real error: `json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})`.

### 5.3 Health probes (real)
In `checker.go`, replace `:57-58` hardcoded `true` with:
- `outbox_workers`: `SELECT MIN(created_at) FROM outbox_events WHERE status='pending'`
   (table from `00020_outbox_events.sql`; `00053` does not exist — migration head is
   `00039`). If oldest pending > `OUTBOX_STALE_THRESHOLD` (env,
  default 5m) → `Healthy:false`, message with age.
- `notification_service`: ping the configured channel sink:
  - EMAIL → SMTP `c.Target` `net.DialTimeout` 2s;
  - WEBHOOK → `http.Head` target with 2s timeout;
  - MOCK → healthy.
  On any failure → `Healthy:false`. Overall `status` = `DEGRADED` if any non-DB
  component unhealthy (DB down → `DOWN`).

### 5.4 Notification adapters (`internal/operations/notifications/adapters/`)
Implement behind `ports.NotificationService`:
- `SendEmail` → SMTP adapter (`adapter:"SMTP"`): read `notification_channel_config`
  where `channel='EMAIL'`, send via `net/smtp` using `SMTP_HOST/PORT/USER/PASS` env.
  MOCK → keep `log.Printf` stub. Always insert a `notification_log` row (status sent/failed).
- `SendSMS` → Twilio-style adapter (MOCK returns `fmt.Errorf` until `SMS_ENABLED`).
- `SendPush` → FCM adapter (MOCK until `PUSH_ENABLED`).
- `SendWebhook` → `http.Post` to configured `webhook_url` (MOCK until `WEBHOOK_ENABLED`).
- `SendInApp` → **persist** to existing `notifications` table (id,user_id,title,message,
  channel='in_app',status='unread') AND append `notification_log`. Replace in-memory map
  with DB writes. Keep a small LRU cache for hot reads but source of truth = DB.
- A `Dispatch(ctx, msg)` helper reads `notification_preferences` for the user and only
  sends enabled channels.

### 5.5 Error persistence + dedup (`internal/operations/errors/reporter.go` rewrite)
- `fingerprint = sha1(method + url + firstLine(message))`. On `Report`, `SELECT id FROM
  error_reports WHERE fingerprint=? AND tenant_id=?`. If found → `UPDATE SET
  occurrences=occurrences+1, last_seen=now`. Else `INSERT` and, if CRITICAL/HIGH,
  auto-`INSERT` into `incidents`. Still notify tech (now via real Email adapter).
- `ListErrors`/ `ListIncidents` query DB (tenant-scoped), not memory.
- On startup, load is gone (persisted). No cap.

### 5.6 Login audit persistence (`internal/operations/audit/login_audit.go` rewrite)
- `RecordLogin` → `INSERT` into `login_audit`. Keep `knownDevices` as an in-memory cache
  seeded from DB for the notify-on-new-device check, but source of truth = DB.
- `GetLoginHistory` → query DB.

### 5.7 Experiments sticky + anon fix (`internal/experiments/`)
- `Assign` change:
  - If `userID == ""` (anonymous): use `anonToken` + `EXPERIMENT_SALT` env, hash
    `sha1(salt + ":" + anonToken + ":" + experiment)`. Different anon users → different
    buckets. Collapse fixed.
  - Sticky: first assignment → `INSERT` into `experiment_assignments`
    (user_id OR anon_token). Subsequent calls `SELECT variant` and return stored value
    (ignore recompute). Force still overrides.
  - Rollout + variant set now come from `experiments_config` (per-experiment), not a
    single global constant. `DashboardExperiment` stays for back-compat but reads config.
- `experiments_config` lifecycle: DRAFT→RUNNING→STOPPED→ARCHIVED. STOPPED = 0 rollout,
  everyone gets A. ARCHIVED = not assignable.

### 5.8 Founder multi-channel (`internal/founder/alerts/multichannel.go` NEW)
- `MultiChannelNotifier` implements `founder.Notifier`. Holds adapters:
  Telegram (existing), Email (SMTP), Slack (webhook), SMS (Twilio). Reads
  `founder_alert_channels`. Per alert, fan-out to every `enabled` channel.
- **Noop rule**: if NO channel is enabled (or all adapters fail to build), the notifier
  MUST `logger.Warn("founder alert dropped: no channel enabled", "category", e.Category)`
  and `UPDATE founder_alert_stats SET noop_dropped_total = noop_dropped_total + 1`.
  It must NEVER return nil silently (replace `notifier.go:82-84` behavior).
- Populate `company_id`: add `CompanyID string` to `alerts.AlertEvent`
  (`event.go`), set it from the event payload where available (customer.activated etc.
  already has `company` name; map name→id via `companies` table or carry id in payload).
  For churn alerts, `companyID` already passed to `EvaluateCustomerHealth`.
- Digest: `populateDigest(ctx, db)` reads `error_reports` (critical count), `incidents`
  (open count), signups/activations from `users`/`companies` (tenant-scoped) and fills
  `digest.DailyDigestReport` for real instead of zero values (`main.go:933`).

---

## 6. Config / env

| Var | Default | Purpose | Package |
|-----|---------|---------|---------|
| `PNL_UNCERTAINTY_DIESEL` | `0.15` | diesel margin uncertainty | `pnl` (overrides `pnl_config`) |
| `PNL_UNCERTAINTY_CNG` | `0.25` | cng margin uncertainty | `pnl` |
| `OUTBOX_STALE_THRESHOLD` | `5m` | outbox worker health age | `health` |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` | – | real email sending | `notifications` |
| `SMS_ENABLED` / `TWILIO_*` | `false` | SMS adapter | `notifications` |
| `PUSH_ENABLED` / `FCM_*` | `false` | Push adapter | `notifications` |
| `WEBHOOK_ENABLED` | `false` | Webhook adapter | `notifications` |
| `EXPERIMENT_SALT` | `random-per-deploy` | anon assignment salt | `experiments` (set stable in env!) |
| `FOUNDER_TELEGRAM_BOT_TOKEN` / `FOUNDER_TELEGRAM_CHAT_ID` | – | Telegram channel | `founder` |
| `FOUNDER_SMTP_TO` / `FOUNDER_SLACK_WEBHOOK` / `FOUNDER_SMS_TO` | – | other founder channels | `founder` |
| `FOUNDER_DIGEST_HOUR` | `9` | digest schedule (UTC) | `founder` |

`company_config` (00042) may later mirror `SMTP_*` flags; we do NOT create it here.

---

## 7. Tests

Use `testing` + `net/http/httptest`. Place under each package; integration cases under
`test/`. Coverage gate: new code ≥ 80% (run `go test ./internal/pnl/... ./internal/operations/... ./internal/experiments/... ./internal/founder/...`).

### 7.1 PNL aggregate
`internal/pnl/aggregate_test.go`: seed `trips` with known margins, assert
`Aggregate` totals = sum; assert `SaveSnapshot` inserts `pnl_snapshots`; assert
`Calculate` **returns** error when odometer query fails (no swallow).

### 7.2 Health probe
`internal/operations/health/checker_test.go`: inject a DB with a stale `outbox_events`
row older than threshold → assert `outbox_workers.Healthy==false` and `status=="DEGRADED"`.
Mock unreachable SMTP target → `notification_service.Healthy==false`.

### 7.3 Notification SMTP
`internal/operations/notifications/adapters/smtp_test.go`: with `adapter=SMTP` and a
`httptest` SMTP sink (or `SMTP_ENABLED=false` MOCK branch) assert a `notification_log`
row is written with `status='sent'`/`'failed'`; assert `SendInApp` persists to
`notifications` table and is queryable after restart (new service instance).

### 7.4 Error dedup
`internal/operations/errors/reporter_test.go`: call `Report` twice with same
method+url+msg → assert single `error_reports` row with `occurrences==2`; different msg
→ two rows; CRITICAL → `incidents` row created.

### 7.5 Experiments anon collapse + sticky
`internal/experiments/experiments_test.go`:
- Two anonymous users with DIFFERENT `anonToken` + same `EXPERIMENT_SALT` must NOT
  always get the same variant across many tokens (statistically different; assert at
  least 2 distinct variants over 50 tokens at rollout 50).
- `userID=" "` (empty, no token) is rejected/forced to A.
- Sticky: assign `u1` → B, flip `experiments_config.rollout` to 0, re-assign `u1` →
  still B (from `experiment_assignments`).

### 7.6 Founder noop warn + count
`internal/founder/alerts/multichannel_test.go`: with `founder_alert_channels` all
disabled → `SendAlert` must `Warn` (capture via `slog` test handler) and increment
`noop_dropped_total`; must NOT return nil-ignored. With Telegram enabled + bad token →
still logs warn for the failed channel.

**Pass-before-merge checklist**
- [ ] `go build ./...` clean
- [ ] `go test ./internal/pnl/... ./internal/operations/... ./internal/experiments/... ./internal/founder/...` green
- [ ] `goose up` / `goose down` on 00058 reversible with no error
- [ ] `grep -rn "_ = s.db" internal/pnl/` returns nothing (no swallow)
- [ ] existing tests `pnl/service_test.go`, `errors/reporter_test.go` still pass

---

## 8. Future / GPS-provider

- **Telemetry-driven PNL actuals**: once `01-telematics-ingestion` positions land,
  replace odometer MIN/MAX with a verified `distance_km` from GPS path; fuel actuals from
  `fuel_prices` history per region. Adds `pnl_snapshots.actual_margin` column later.
- **Geo-alerts**: `health` + `founder` can subscribe to geofence events (spec 02) to
  alert founders on vehicle-off-route as a `SYSTEM` alert through the multi-channel fan-out.
- **AIS-140 / VLT**: when VLT hardware (spec 01 adapter) arrives, P&L fuel estimate
  tightens using onboard fuel sensor telemetry → lower `uncertainty`.
- **Scaling**: `error_reports`/`notification_log` grow fast → partition by month, add
  TTL purge job; `pnl_snapshots` is naturally small (one row per period).

---

## 9. Edge cases

- Anonymous user with empty `userID` AND no `anon_token` → assign Variant A, do not crash.
- `Calculate` called with unknown `tripID` → return `sql.ErrNoRows`; handler → `404`.
- `fuel_prices` empty / telemetry missing → margin `MarginAvailable=false`, aggregate
  still returns totals with `fuel_cost_status:"pending_verification"`.
- Outbox table missing (pre-00053) → health probe logs warn, reports healthy=false with
  message "outbox_events table unavailable".
- Notification adapter panic → recovered, `notification_log.status='failed'`, error stored.
- `founder_alert_channels` empty (fresh DB) → treated as all-disabled → noop warn+count.
- Multi-tenant: every list query filters `tenant_id` from `auth.ContextUser`; a user
  cannot read another tenant's errors/pnl/audit.
- `experiments_config` missing for a key → fall back to `rollout=0` (Variant A).
- Restart loses in-memory caches but NOT data (all now persisted).

---

## 10. Phased rollout (build order)

1. **Migration 00058** + seed RBAC (no app code yet; reversible).
2. **Errors + incidents** persistence + `/api/v1/errors` + dedup (highest value, unblocks ops).
3. **Login audit** persistence + `/api/v1/audit/logins`.
4. **Notification adapters** (SMTP + InApp DB) + `notification_log` + `/api/v1/notifications/preferences`.
5. **Health probes** real (outbox + notification ping).
6. **PNL aggregate/snapshots** + UI + config + error-surface fix.
7. **Experiments** anon fix + sticky + config + admin API + UI.
8. **Founder** multi-channel + noop warn/count + digest wiring + UI.
9. Run full test suite + `goose down`/`up` verification.

---

## 11. Open items / VERIFY

- **`company_config` columns** unknown (owned by 00042, not yet created). We avoided
  touching it; confirm `notification_channel_config` is sufficient or that 00042 adds the
  flag columns we need. MUST verify before coding if any channel flag must live there.
- **`permissions` table schema**: confirm `description` column exists (assumed from
  `00012_rbac.sql`); if not, drop it from seed.
- **`companies` table / company_id mapping**: founder alerts need `company_id`; confirm
  payloads from `customer.activated` events carry an id (currently only `company_name`).
  Decide: add id to event payload or lookup by name.
- **`outbox_events` status values**: confirm `status='pending'` is the stalled marker
  (00053 may rename). Verify before health probe query.
- **`EXPERIMENT_SALT` stability**: if left defaulted per-deploy, sticky assignments break
  on restart. MUST be set in env for production.
- **SMTP library choice**: stdlib `net/smtp` vs `go-mail`. Pick stdlib to avoid new dep
  unless team prefers `go-mail`.
- **Digest data sources**: signups/visitors currently zero-valued with no table; confirm
  we derive from `users`/`companies` created_at or add an analytics event. Minimum: fill
  critical_errors/open_incidents from new persisted tables.

---

## 12. File list

CREATE:
- `db/migrations/00058_pnl_ops_experiments_founder.sql`
- `internal/pnl/aggregate.go` (Aggregate, SaveSnapshot, loadUncertainty)
- `internal/pnl/aggregate_test.go`
- `internal/pnl/handler.go` (extend RegisterRoutes: add /pnl/trips, /pnl/aggregate,
  /pnl/snapshots, /pnl/snapshots/rollup, /pnl/config) — modify existing
- `internal/operations/health/probes.go` (outbox + notification probes)
- `internal/operations/notifications/adapters/smtp.go`
- `internal/operations/notifications/adapters/sms.go`
- `internal/operations/notifications/adapters/push.go`
- `internal/operations/notifications/adapters/webhook.go`
- `internal/operations/notifications/adapters/*_test.go`
- `internal/operations/errors/reporter.go` (rewrite to persist + dedup; keep interface)
- `internal/operations/audit/login_audit.go` (rewrite to persist)
- `internal/experiments/assignments.go` (sticky + anon + config-backed Assign)
- `internal/experiments/experiments.go` (refactor Assign to use config + assignments)
- `internal/experiments/admin.go` (admin API handlers + DB for experiments_config)
- `internal/founder/alerts/multichannel.go` (MultiChannelNotifier)
- `internal/founder/alerts/event.go` (add CompanyID field)
- `internal/founder/handler.go` (founder channels + digest preview API)
- `web/templates/pnl_dashboard.html`, `web/templates/errors.html`,
  `web/templates/login_audit.html`, `web/templates/health.html`,
  `web/templates/experiments.html`, `web/templates/notification_prefs.html`,
  `web/templates/founder_channels.html`
- `web/assets/js/pnl.js`, `web/assets/js/experiments.js`, `web/assets/js/health.js`,
  `web/assets/js/founder.js`

MODIFY:
- `internal/pnl/service.go` (stop swallowing errors; read uncertainty from config)
- `internal/operations/health/checker.go` (use real probes instead of `:57-58`)
- `internal/operations/notifications/service.go` (SendInApp→DB; real adapters)
- `internal/founder/service.go` (pass companyID; wire multichannel notifier)
- `internal/founder/digest/daily.go` (populate real data)
- `cmd/server/main.go` (wire new handlers; newFounderNotifier→MultiChannelNotifier;
  runDailyDigest uses populated report)
- `internal/pnl/handler.go` (error surfacing fix)
