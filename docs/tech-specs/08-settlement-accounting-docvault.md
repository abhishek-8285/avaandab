# Settlement, Accounting & Document Vault — Implementation Spec v1

Status: ready
Depends-on:
  - `db/migrations/00030` (driver_settlements schema, drivers.aadhaar/pan/bank_details, vehicles.rc_expiry/blocked) — already merged (head `00039`)
  - `db/migrations/00042` (canonical `company_config` table) — **CORRECTION: does NOT exist at head `00039`.** No `company_config` table is present anywhere in `db/migrations` (grep: 0 matches). The migrations below `INSERT INTO company_config`; those seeds will FAIL until the table exists. Either author `00042` first (per `00-migration-ownership-index.md`, spec 02 owns it) or make `00051` `CREATE TABLE IF NOT EXISTS company_config (...)` before seeding. See §0.10.
  - `db/migrations/00020` (outbox_events) — accounting relay trigger source — verified present
  - `internal/shared/outbox/relay.go` — two-way sync trigger — verified present (`maxAttempts=5`, exp backoff)
  - `internal/integration/accounting/client.go` — adapter to extend — verified single `stubClient`
Migration owner:
  - **Ownership (reconciled with index):** `00-migration-ownership-index.md` assigns `00050`/`00051`/`00052` to **spec 08** (this spec): 00051 settlement engine, 00050 accounting sync, 00052 document vault. Numbers below match the index.
  - `db/migrations/00051_driver_settlement_engine.sql`  (Feature 1)
  - `db/migrations/00050_accounting_sync.sql`            (Feature 2)
  - `db/migrations/00052_document_vault.sql`             (Feature 3)

---

## 0. Verified ground truth (file:line facts + grep proofs)

These facts were read directly from the repo at head `00039_experiments.sql`. Do not "fix"
these files blindly — the spec tells you WHAT to change and WHERE.

### 0.1 Driver settlement is in-memory only and never persists

`internal/service/driver_settlement_service.go`:
- `CreateSettlementForTrip` (lines 44–82): builds a `DriverSettlementRecord` and **returns
  it, but never INSERTs** into `driver_settlements`. Proof: no `INSERT`/`ExecContext` in the
  function body.
- `ProcessFinancialSettlement` (lines 86–148): uses **hardcoded** constants
  `defaultSettlementFare=1000.0`, `defaultSettlementAdvances=200.0`,
  `defaultSettlementDeductions=50.0` (lines 38–42). It picks `fare = bk.Price` only if a
  booking exists (101–105) but `advances`/`deductions` are still the hardcoded 200/50
  (107–114). It **never persists** a settlement row — only publishes an event
  `DriverPayoutSettled` (136–145).
- `DriverSettlementRecord` struct (13–26) has no TDS / commission / rate-model fields and
  no `settlement_lines` concept.

Grep proof:
```
$ grep -n "INSERT INTO driver_settlements" internal/service/*.go
(internal/service/driver_settlement_service.go: no match)
```

### 0.2 Kharcha approval writes to a settlement row that never exists (no-op)

`internal/service/kharcha_service.go`:
- `ApproveExpense` (lines 166–220) runs a transaction. Step 3 (lines 204–211) does:
  ```
  UPDATE driver_settlements
   SET advances_kharcha = advances_kharcha + ?,
       net_payout = MAX(0.0, net_payout - ?)
   WHERE trip_id = ? AND driver_id = ?
  ```
  Because Section 0.1 proves no settlement row is ever INSERTed, this `UPDATE` matches **0
  rows** and silently does nothing (`_, _ = tx.ExecContext(...)`, errors ignored).
- Status value `"settled"` referenced in `KharchaExpense.Status` comment (line 22) is **dead**:
  the schema `CHECK (status IN ('pending','processing','paid','disputed'))` in 00030 (line 54)
  does not even allow `settled`, and nothing ever sets it.

### 0.3 driver_settlements schema (00030) — what exists today

`db/migrations/00030_avandab_modules_and_rules.sql` (lines 46–61):
```
CREATE TABLE IF NOT EXISTS driver_settlements (
    id TEXT PRIMARY KEY,
    trip_id TEXT NOT NULL UNIQUE,
    driver_id TEXT NOT NULL,
    gross_fare REAL NOT NULL DEFAULT 0.0,
    advances_kharcha REAL NOT NULL DEFAULT 0.0,
    deductions REAL NOT NULL DEFAULT 0.0,
    net_payout REAL NOT NULL DEFAULT 0.0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','paid','disputed')),
    payment_ref TEXT,
    paid_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id)
);
```
No TDS columns, no commission, no `settlement_lines` table, `trip_id` is `UNIQUE` (one
settlement per trip — good, we keep that invariant).

### 0.4 Driver identity columns exist in DB but are orphaned in code paths

`db/migrations/00030` (lines 4–8) added to `drivers`:
```
blocked INTEGER NOT NULL DEFAULT 0;
blocked_reason TEXT;
aadhaar TEXT;
pan TEXT;
bank_details TEXT;
```
`internal/domain/driver/entity.go` (25–27) defines `Aadhaar *string`, `PAN *string`,
`BankDetails *string` — but verify the **repository** actually reads/writes them. As of this
spec they are effectively orphaned: no upsert path guarantees persistence on create/update.

### 0.5 Vehicle expiry/block columns — what already exists

- `db/migrations/00003_vehicles.sql`: `insurance_expiry DATE NOT NULL`, `fitness_expiry DATE
  NOT NULL`, `permit_expiry DATE NOT NULL` (base table, all NOT NULL).
- `db/migrations/00030` (lines 10–13): `blocked`, `blocked_reason`, `rc_expiry DATE`,
  `odometer REAL`.
- **Missing from DB:** `puc_expiry` (PUC/Pollution Under Control). `RCExpiry`,
  `InsuranceExpiry`, `Blocked` already exist — do NOT re-add them (the migration would fail
  on `ALTER TABLE ... ADD COLUMN`). Only `puc_expiry` is genuinely new.
- `internal/domain/vehicle/entity.go` (18–21) already models `InsuranceExpiry`,
  `FitnessExpiry`, `PermitExpiry`, `RCExpiry`; we add `PUCExpiry` to the entity.

### 0.6 Accounting client is a single stub; no real adapters; no idempotency

`internal/integration/accounting/client.go`:
- One `stubClient` (97–107). `NewClient` (102–107) always returns the stub regardless of
  provider.
- Interface `Client` (91–95): `ExportInvoice`, `SyncContacts`, `PushJournalEntry`.
- All three methods are fake: they log and return fake `SUCCESS` only when `cfg.Enabled`
  (109–145). No Tally/Zoho/QuickBooks adapters; no idempotency key.

### 0.7 Outbox relay exists and is the correct trigger mechanism

`internal/shared/outbox/relay.go`:
- `Relay` polls `outbox_events` (00020) for `published_at IS NULL` and dispatches to an
  `events.EventBus` (70–126). `maxAttempts=5` with exponential backoff (137–148).
- This is the canonical trigger for Feature 2 two-way sync: emit outbox events, let the relay
  fan them to the accounting adapter.

### 0.8 ComplianceService exists but is unwired

`internal/service/compliance_service.go`:
- `ComplianceService` (13–15) with `ValidateDriverCompliance` (27–75),
  `ValidateVehicleCompliance` (79–128), `EnforceDispatchCompliance` (131–153).
- It auto-sets `blocked=true` on expiry (62–63, 118–121) and audits. BUT: no trip
  assign/start handler calls `EnforceDispatchCompliance` today (grep proof below). It is
  **unwired**.

Grep proof (expected empty — confirms unwired):
```
$ grep -rn "EnforceDispatchCompliance\|ValidateDriverCompliance" internal/trip internal/handler internal/booking
(no matches)
```

### 0.9 Existing RBAC / permission seeding pattern

`db/migrations/00012_rbac.sql`, `00014_add_bookings_approve.sql`, `00015_add_audit_logs_permission.sql`,
`00027_add_driver_role.sql` show the RBAC `permissions`/`roles`/`role_permissions` seed style.
We follow it for new resources (`settlements`, `accounting_sync`, `documents`, `compliance`).

### 0.10 Verification Log (QA pass — head `00039`)

| # | Claim (from §0 / body) | Verdict | Correction / Evidence (file:line) |
|---|------------------------|---------|-----------------------------------|
| 1 | `CreateSettlementForTrip` never INSERTs; rates 1000/200/50 | VERIFIED | `driver_settlement_service.go:46` (no ExecContext/INSERT); consts `:39-41` = 1000/200/50 |
| 2 | `ApproveExpense` step-3 UPDATE is a no-op (0 rows) | VERIFIED | `kharcha_service.go:205` `_, _ = tx.ExecContext`; true because no settlement row exists today (claim 1) |
| 3 | `driver_settlements` schema (00030) | VERIFIED | `00030:46-61` matches quoted DDL; status CHECK `:54` = pending/processing/paid/disputed |
| 4 | `drivers.aadhaar/pan/bank_details` orphaned in code | VERIFIED | `00030:6-8` add cols; driver repo `grep aadhaar|pan|bank_details` = 0 hits → no persist path |
| 5 | 00003 insurance/fitness/permit NOT NULL; puc missing | VERIFIED | `00003:9-11` NOT NULL; `00030:12` adds `rc_expiry`; no `puc_expiry` anywhere |
| 6 | Accounting client single stub; no tally/zoho/qb; no idempotency | VERIFIED | `client.go:97-107` `stubClient` only; interface `:91-95`; all 3 methods fake `:109-145` |
| 7 | Outbox relay exists, maxAttempts=5, exp backoff | VERIFIED | `relay.go:16`, `:71` `published_at IS NULL`, `:147` `interval*2^attempts` |
| 8 | `ComplianceService` unwired | VERIFIED | grep `EnforceDispatchCompliance` only in `compliance_service.go` + `services_test.go`; no trip/handler call |
| 9 | `00042` `company_config` already merged; migrations only seed rows | **WRONG** | No `00042` in `db/migrations` (head `00039`); grep `company_config` = 0 across repo. Migrations 00050/00051 `INSERT INTO company_config` will FAIL. Fix: author 00042 or have 00051 CREATE the table. |
| 10 | Migration owner of 00050/00051/00052 = spec 08 (this spec), matching `00-migration-ownership-index.md` | VERIFIED | index assigns 00050 (accounting), 00051 (settlement), 00052 (doc vault) to spec 08. |
| 11 | `NewClient` returns stub "regardless of provider" | VERIFIED | `client.go:102-107` no provider switch; always `stubClient` |
| 12 | `DriverPayoutSettled` event published | VERIFIED | `driver_settlement_service.go:86` `ProcessFinancialSettlement` publishes at payload build |
| 13 | `ProcessFinancialSettlement` sets status `paid` + persists? | PARTIAL | Sets `Status:"paid"` in returned struct (`:86-148`) but still never INSERTs — the row is not persisted; "paid" is in-memory only. Spec §0.1 "never persists" holds. |

### 0.11 Severity & Effort (per correction)

| # | Change | Severity | Effort |
|---|--------|----------|--------|
| 9 | Create `company_config` (00042) or inline `CREATE TABLE IF NOT EXISTS` in 00051 before seeding | **Critical** (blocks all 3 migrations) | M (write a 10-line migration / guard) |
| 10 | Reconcile migration ownership with `00-migration-ownership-index.md` | Medium (doc/process conflict) | S (edit index) |
| 13 | Clarify §0.1: `ProcessFinancialSettlement` returns `paid` but does not persist | Low (spec wording) | S |
| 1–8,11,12 | No change — claims confirmed accurate | — | — |

### 0.12 Decisions, Tradeoffs & Cost

#### D1 — Accounting adapter strategy (Tally / Zoho / QuickBooks / mock)
- **Decision:** Ship the **`mock` adapter as the only working default**; real Tally/Zoho/QuickBooks adapters are stubbed HTTP clients gated behind `ACCOUNTING_ENABLED=true` + `ACCOUNTING_ADAPTER`. Keep the single `Client` interface; add `Config.Provider` factory.
- **Tradeoff:** Real adapters need per-vendor auth, rate limits, and idempotency-key handling that the mock does not. Building all three now multiplies surface area and certification effort (Zoho/Tally India-specific APIs, QuickBooks OAuth). Deferring real adapters keeps the relay + `accounting_sync_log` + `accounting_gl_rule` plumbing shippable and testable today.
- **Cost:** If a real adapter is required in v1, budget ~1–2 dev-days per vendor for auth + mapping + error/retry handling, plus a sandbox tenant. Risk: vendor API drift breaks reconcile; mitigated by the idempotency_key UNIQUE guard (§3.2) so replays are safe.

#### D2 — Document Vault storage (DB BLOB vs object store)
- **Decision:** Store files in an **external object store** (existing `Files` service, `internal/service/file_service.go`) and persist only `file_url` + metadata in `driver_documents`/`vehicle_documents` (as the spec already does). **Do NOT store file bytes as BLOB** in the DB.
- **Tradeoff:** BLOB keeps files transactionally co-located with metadata and simplifies backups of a single store, but bloats SQLite rows, kills `VACUUM` efficiency, and makes the DB the I/O bottleneck for large PDFs/scans. Object store gives cheap scale, CDN delivery, and per-object expiry/ACL, at the cost of a second consistency boundary (URL must be retained even if the object is deleted).
- **Cost:** Object store adds a storage dependency + presigned-URL/expiry logic in `DocumentService` (`§5.3.2`); negligible if reusing the existing `Files` service. BLOB alternative would require a migration column change + streaming reads and is rejected for this TMS.

---

## 1. Overview / goal

Build three production features that are currently stubbed/broken:

1. **Driver Settlement Engine** — make settlements real: persist a `driver_settlements`
   row per trip, compute gross fare via configurable rate calculators (per-km / fixed /
   % commission), apply TDS under Section 194C (1% with PAN, 2% without), deduct approved
   kharcha, pay out, and let the driver confirm or dispute.
2. **Accounting two-way sync** — push financial events to an external accounting system
   (Tally / Zoho / QuickBooks / mock) through a pluggable adapter, idempotently, triggered
   by the existing outbox relay, with mapping + GL-rule tables for reconciliation.
3. **Document Vault + Compliance** — store driver/vehicle documents (with expiry), persist
   Aadhaar/PAN/Bank, add PUC expiry, surface a compliance dashboard, and **hard-block
   dispatch** by wiring `ComplianceService` into trip assign/start.

### Non-goals
- No real GSTN/e-way bill generation (separate spec 07/05).
- No payroll/ITR filing; TDS here is *computed and recorded*, not remitted.
- No multi-currency.
- No changes to the `trip_id UNIQUE` invariant of `driver_settlements`.

---

## 2. API contract

All routes mounted under the existing auth middleware. Permission strings follow RBAC
(`settlements:read`, `settlements:write`, `settlements:approve`, `accounting:read`,
`accounting:sync`, `documents:upload`, `documents:read`, `compliance:read`).

### 2.1 Driver Settlement Engine

#### POST /api/settlements/generate
Generate (or regenerate) the settlement for a trip. Persists a `driver_settlements` row and
`settlement_lines`. Idempotent on `trip_id` (UNIQUE) — re-POST returns the existing row.
- Auth: `settlements:write`
- Request:
```json
{
  "trip_id": "trp_abc123",
  "force_recompute": false
}
```
- Response `200` / `201`:
```json
{
  "settlement_id": "stl_xyz",
  "trip_id": "trp_abc123",
  "driver_id": "drv_001",
  "gross_fare": 5000.00,
  "rate_model": "per_km",
  "rate_basis": {"km": 420, "rate_per_km": 11.90},
  "commission_amount": 250.00,
  "advances_kharcha": 600.00,
  "approved_deductions": 150.00,
  "tds_rate": 0.01,
  "tds_amount": 46.00,
  "net_payout": 3954.00,
  "status": "pending",
  "lines": [
    {"type":"gross_fare","label":"Trip fare (per_km 420km x 11.90)","amount":5000.00},
    {"type":"commission","label":"Platform commission 5%","amount":-250.00},
    {"type":"advances","label":"Advances & kharcha","amount":-600.00},
    {"type":"deduction","label":"Approved expense #exp_2","amount":-150.00},
    {"type":"tds","label":"TDS u/s 194C @1%","amount":-46.00}
  ]
}
```
- Errors: `404 trip_not_found`, `409 driver_not_assigned`, `422 booking_price_missing`
  (when rate_model=fixed/commission and no price/km source).

#### GET /api/settlements?driver_id=&status=&from=&to=
List settlements. Auth: `settlements:read`.
- Response `200`:
```json
{
  "items": [ { ...same object as generate response... } ],
  "total": 1,
  "page": 1,
  "page_size": 50
}
```

#### GET /api/settlements/{id}/deductions
Show the approved-kharcha breakdown that fed the deduction engine. Auth: `settlements:read`.
- Response `200`:
```json
{
  "settlement_id": "stl_xyz",
  "trip_id": "trp_abc123",
  "approved_expenses": [
    {"expense_id":"exp_2","category":"fuel","amount":150.00,"approved_at":"2026-08-10T09:00:00Z"}
  ],
  "total_deducted": 150.00,
  "tds": {"section":"194C","rate":0.01,"base":4600.00,"amount":46.00}
}
```

#### POST /api/settlements/{id}/mark-paid
Link a payment record (bank transfer / Razorpay payout) to the settlement. Auth:
`settlements:approve`.
- Request:
```json
{
  "payment_ref": "TXN_ICICI_98231",
  "paid_at": "2026-08-12T15:30:00Z",
  "mode": "bank_transfer"
}
```
- Response `200`: `{ "status":"paid", "payment_ref":"TXN_ICICI_98231", "paid_at":"..." }`
- Errors: `409 already_paid`, `422 missing_payment_ref`.

#### POST /api/settlements/{id}/confirm
Driver confirms receipt. Auth: `settlements:read` + own driver scope (driver role).
- Request: `{}`
- Response `200`: `{ "status":"paid", "confirmed_at":"..." }`

#### POST /api/settlements/{id}/dispute
Driver disputes the calculation. Auth: own driver scope.
- Request:
```json
{ "reason": "KM shown 420 but actual 460", "expected_net": 4100.00 }
```
- Response `200`: `{ "status":"disputed", "dispute_reason":"..." }`
- On dispute, status moves `pending|paid -> disputed`; finance must resolve (re-generate or
  manual adjustment) before it can be marked paid again.

### 2.2 Accounting two-way sync

#### GET /api/accounting/sync/status
Show adapter in use, last sync, pending log count. Auth: `accounting:read`.
- Response `200`:
```json
{
  "adapter": "mock",
  "enabled": false,
  "pending_events": 3,
  "last_synced_at": "2026-08-12T10:00:00Z"
}
```

#### POST /api/accounting/sync/trigger
Manually flush pending outbox → accounting. Auth: `accounting:sync`.
- Request: `{ "since_minutes": 60 }`
- Response `200`:
```json
{ "dispatched": 12, "failed": 0, "skipped_duplicates": 0 }
```

#### POST /api/accounting/contacts/sync
Push driver (vendor) + customer (contact) mappings. Auth: `accounting:sync`.
- Response `200`: `{ "synced": 40, "failed": 0 }`

#### GET /api/accounting/reconcile
Compare local `accounting_sync_log` vs acknowledged external IDs. Auth: `accounting:read`.
- Response `200`: `{ "total": 120, "acked": 118, "unacked": 2, "unacked_refs":["EXT-INV-9","EXT-JE-3"] }`

### 2.3 Document Vault + Compliance

#### POST /api/documents/driver/{driver_id}
Upload a document (multipart form: `file`, `doc_type`, `expiry_date?`). Auth:
`documents:upload`.
- Response `201`:
```json
{
  "document_id":"doc_1",
  "driver_id":"drv_001",
  "doc_type":"aadhaar",
  "file_url":"/files/doc_1.pdf",
  "expiry_date":"2030-01-01",
  "status":"pending_review"
}
```
`doc_type` ∈ `aadhaar|pan|dl|bank_proof|medical|other` (driver) and
`rc|insurance|puc|fitness|permit|others` (vehicle).

#### POST /api/documents/vehicle/{vehicle_id}
Same contract as driver, doc_type vehicle-scoped.

#### GET /api/compliance/dashboard
Compliance overview with hard-block reasons. Auth: `compliance:read`.
- Response `200`:
```json
{
  "drivers": {"total":50,"blocked":2,"expiring_soon":3},
  "vehicles": {"total":30,"blocked":1,"expiring_soon":4},
  "blocked_drivers":[{"id":"drv_009","reason":"License expired 2026-07-01"}],
  "blocked_vehicles":[{"id":"veh_005","reason":"RC expired 2026-06-15"}],
  "documents_pending": 7
}
```

#### POST /api/compliance/verify/{entity_type}/{entity_id}/{document_id}
Mark a document verified (sets `status=verified`, may clear a block). Auth: `compliance:read`
(admin). Response `200` object.

---

## 3. DB contract (goose migrations)

> **QA correction (see §0.10 #9):** `company_config` does **not** exist at head `00039`
> (`grep company_config` across `db/migrations` = 0). The `INSERT INTO company_config`
> statements below will fail until the table is created. Before running 00050/00051/00052,
> either merge migration `00042` (canonical owner per `00-migration-ownership-index.md` spec 02)
> or prepend to `00051`:
> ```sql
> CREATE TABLE IF NOT EXISTS company_config (
>     key TEXT PRIMARY KEY,
>     value TEXT NOT NULL,
>     description TEXT
> );
> ```
> Keep the seeding as-is once the table exists.

### 3.1 Migration 00051 — Driver Settlement Engine

File: `db/migrations/00051_driver_settlement_engine.sql`

```sql
-- +goose Up
-- Feature 1: make driver settlements real + TDS + settlement lines.

-- Add TDS + commission columns to the existing driver_settlements table (00030).
ALTER TABLE driver_settlements ADD COLUMN commission_amount REAL NOT NULL DEFAULT 0.0;
ALTER TABLE driver_settlements ADD COLUMN tds_rate         REAL NOT NULL DEFAULT 0.0;
ALTER TABLE driver_settlements ADD COLUMN tds_amount       REAL NOT NULL DEFAULT 0.0;
ALTER TABLE driver_settlements ADD COLUMN rate_model       TEXT NOT NULL DEFAULT 'fixed';
ALTER TABLE driver_settlements ADD COLUMN rate_basis_json  TEXT;
ALTER TABLE driver_settlements ADD COLUMN confirmed_at     DATETIME;
ALTER TABLE driver_settlements ADD COLUMN disputed_at      DATETIME;
ALTER TABLE driver_settlements ADD COLUMN dispute_reason   TEXT;
-- widen status to allow 'settled'/'disputed' already allowed; add 'confirmed' optionally:
-- (status CHECK already allows pending,processing,paid,disputed — keep it.)

-- Per-line breakdown of a settlement (audit + driver statement).
CREATE TABLE IF NOT EXISTS settlement_lines (
    id             TEXT PRIMARY KEY,
    settlement_id  TEXT NOT NULL,
    trip_id        TEXT NOT NULL,
    line_type      TEXT NOT NULL CHECK (line_type IN ('gross_fare','commission','advances','deduction','tds','adjustment')),
    label          TEXT NOT NULL,
    amount         REAL NOT NULL,
    ref_id         TEXT,            -- expense_id / trip_id / ''
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (settlement_id) REFERENCES driver_settlements(id) ON DELETE CASCADE,
    FOREIGN KEY (trip_id) REFERENCES trips(id)
);
CREATE INDEX IF NOT EXISTS idx_settlement_lines_settlement ON settlement_lines(settlement_id);
CREATE INDEX IF NOT EXISTS idx_settlement_lines_trip ON settlement_lines(trip_id);

-- Rate model config (per company). Seed into company_config (00042 owns the table).
INSERT INTO company_config (key, value, description) VALUES
  ('settlement_rate_model', 'per_km', 'per_km | fixed | commission_pct'),
  ('settlement_rate_per_km', '11.90', 'Rupees per km when model=per_km'),
  ('settlement_fixed_fare', '5000.00', 'Flat fare when model=fixed'),
  ('settlement_commission_pct', '5.00', 'Percent of gross when model=commission_pct'),
  ('tds_section', '194C', 'TDS section applied to driver payouts'),
  ('tds_rate_with_pan', '1.00', 'Percent when driver PAN present'),
  ('tds_rate_without_pan', '2.00', 'Percent when driver PAN absent (206AA)');

-- +goose Down
DROP TABLE IF EXISTS settlement_lines;
ALTER TABLE driver_settlements DROP COLUMN commission_amount;
ALTER TABLE driver_settlements DROP COLUMN tds_rate;
ALTER TABLE driver_settlements DROP COLUMN tds_amount;
ALTER TABLE driver_settlements DROP COLUMN rate_model;
ALTER TABLE driver_settlements DROP COLUMN rate_basis_json;
ALTER TABLE driver_settlements DROP COLUMN confirmed_at;
ALTER TABLE driver_settlements DROP COLUMN disputed_at;
ALTER TABLE driver_settlements DROP COLUMN dispute_reason;
DELETE FROM company_config WHERE key IN
  ('settlement_rate_model','settlement_rate_per_km','settlement_fixed_fare',
   'settlement_commission_pct','tds_section','tds_rate_with_pan','tds_rate_without_pan');
```

> SQLite note: `ALTER TABLE ... DROP COLUMN` requires SQLite ≥ 3.35 (go-sqlite3 bundled
> version supports it). If your bundled SQLite is older, the DOWN uses a 12-step
> rebuild; see §11 OPEN ITEM. For UP, all `ADD COLUMN` are safe (new columns only).

### 3.2 Migration 00050 — Accounting two-way sync

File: `db/migrations/00050_accounting_sync.sql`

```sql
-- +goose Up
-- Feature 2: accounting two-way sync, idempotent, adapter-agnostic.

-- Idempotency-guarded sync log. UNIQUE(idempotency_key) makes replays safe.
CREATE TABLE IF NOT EXISTS accounting_sync_log (
    id               TEXT PRIMARY KEY,
    idempotency_key  TEXT NOT NULL UNIQUE,
    direction        TEXT NOT NULL CHECK (direction IN ('out','in')),
    entity_type      TEXT NOT NULL,            -- invoice|journal|contact|payout
    entity_id        TEXT NOT NULL,
    adapter          TEXT NOT NULL,            -- tally|zoho|quickbooks|mock
    payload_json     TEXT NOT NULL,
    external_id      TEXT,
    status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','acked','failed')),
    attempts         INTEGER NOT NULL DEFAULT 0,
    last_error       TEXT,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_accsync_idem ON accounting_sync_log(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_accsync_status ON accounting_sync_log(status);

-- Map internal entity -> external accounting ID (two-way reconcile).
CREATE TABLE IF NOT EXISTS accounting_mapping (
    id            TEXT PRIMARY KEY,
    entity_type   TEXT NOT NULL,               -- driver|customer|invoice|vehicle
    entity_id     TEXT NOT NULL,
    adapter       TEXT NOT NULL,
    external_id   TEXT NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (entity_type, entity_id, adapter)
);
CREATE INDEX IF NOT EXISTS idx_accmap_entity ON accounting_mapping(entity_type, entity_id);

-- GL posting rules: which account each event debits/credits.
CREATE TABLE IF NOT EXISTS accounting_gl_rule (
    id            TEXT PRIMARY KEY,
    event_type    TEXT NOT NULL,               -- DriverPayoutSettled|InvoiceExported|...
    debit_account TEXT NOT NULL,
    credit_account TEXT NOT NULL,
    description   TEXT,
    priority      INTEGER NOT NULL DEFAULT 0
);

-- Seed GL rules (driver payout: debit Driver Payable, credit Bank).
INSERT INTO accounting_gl_rule (id, event_type, debit_account, credit_account, description, priority) VALUES
  ('gl_payout',  'DriverPayoutSettled', 'Driver Payable',   'Bank - Current', 'Driver net payout', 10),
  ('gl_inv',     'InvoiceExported',     'Accounts Receivable','Service Revenue', 'Customer invoice', 10),
  ('gl_tds',     'TDSRemitted',         'TDS Payable',      'Driver Payable',  'TDS withheld',     20);

-- Adapter + enable flag in company_config (00042 owns table).
INSERT INTO company_config (key, value, description) VALUES
  ('accounting_adapter', 'mock',  'tally | zoho | quickbooks | mock'),
  ('accounting_enabled', 'false', 'master switch for real API calls'),
  ('accounting_endpoint','',      'base URL for chosen adapter'),
  ('accounting_api_key', '',      'API key/secret reference (vault-injected)');

-- +goose Down
DROP TABLE IF EXISTS accounting_gl_rule;
DROP TABLE IF EXISTS accounting_mapping;
DROP TABLE IF EXISTS accounting_sync_log;
DELETE FROM company_config WHERE key IN
  ('accounting_adapter','accounting_enabled','accounting_endpoint','accounting_api_key');
```

### 3.3 Migration 00052 — Document Vault + Compliance

File: `db/migrations/00052_document_vault.sql`

```sql
-- +goose Up
-- Feature 3: document vault + missing PUC expiry. Aadhaar/PAN/Bank already exist
-- on drivers (00030) and RC/Insurance/Blocked on vehicles (00003+00030); only PUC is new.

-- vehicles: add PUC expiry (genuinely missing from DB).
ALTER TABLE vehicles ADD COLUMN puc_expiry DATE;

-- Ensure drivers columns exist (idempotent guard for older dev DBs that skipped 00030).
-- NOTE: on a clean DB 00030 already added these; the checks below skip if present.
-- Use a tiny temp trigger-free guard via PRAGMA in the migration runner is overkill;
-- instead run ADD COLUMN only when absent (manual guard documented in §11).
-- (For spec correctness, assume 00030 applied; do NOT re-add aadhaar/pan/bank_details/blocked.)

-- Driver documents
CREATE TABLE IF NOT EXISTS driver_documents (
    id            TEXT PRIMARY KEY,
    driver_id     TEXT NOT NULL,
    doc_type      TEXT NOT NULL CHECK (doc_type IN ('aadhaar','pan','dl','bank_proof','medical','other')),
    file_url      TEXT NOT NULL,
    expiry_date   DATE,
    status        TEXT NOT NULL DEFAULT 'pending_review' CHECK (status IN ('pending_review','verified','rejected')),
    verified_by   TEXT,
    verified_at   DATETIME,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (driver_id) REFERENCES drivers(id)
);
CREATE INDEX IF NOT EXISTS idx_drvdocs_driver ON driver_documents(driver_id);

-- Vehicle documents
CREATE TABLE IF NOT EXISTS vehicle_documents (
    id            TEXT PRIMARY KEY,
    vehicle_id    TEXT NOT NULL,
    doc_type      TEXT NOT NULL CHECK (doc_type IN ('rc','insurance','puc','fitness','permit','others')),
    file_url      TEXT NOT NULL,
    expiry_date   DATE,
    status        TEXT NOT NULL DEFAULT 'pending_review' CHECK (status IN ('pending_review','verified','rejected')),
    verified_by   TEXT,
    verified_at   DATETIME,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);
CREATE INDEX IF NOT EXISTS idx_vehdocs_vehicle ON vehicle_documents(vehicle_id);

-- RBAC resources/permissions for the new surfaces.
INSERT INTO permissions (id, resource, action, description) VALUES
  ('perm_settlements_read','settlements','read','View driver settlements'),
  ('perm_settlements_write','settlements','write','Generate settlements'),
  ('perm_settlements_approve','settlements','approve','Mark paid / approve'),
  ('perm_accounting_read','accounting','read','View sync status'),
  ('perm_accounting_sync','accounting','sync','Trigger accounting sync'),
  ('perm_documents_upload','documents','upload','Upload driver/vehicle docs'),
  ('perm_documents_read','documents','read','View documents'),
  ('perm_compliance_read','compliance','read','View compliance dashboard');
-- Assign to admin role (role id assumed 'role_admin' per 00012 pattern).
INSERT INTO role_permissions (role_id, permission_id) SELECT 'role_admin', id FROM permissions
  WHERE resource IN ('settlements','accounting','documents','compliance');

-- +goose Down
DROP TABLE IF EXISTS vehicle_documents;
DROP TABLE IF EXISTS driver_documents;
ALTER TABLE vehicles DROP COLUMN puc_expiry;
DELETE FROM role_permissions WHERE permission_id IN
  (SELECT id FROM permissions WHERE resource IN ('settlements','accounting','documents','compliance'));
DELETE FROM permissions WHERE resource IN ('settlements','accounting','documents','compliance');
```

> If 00012 uses different role/permission column names, match that file's exact schema
> (grep `CREATE TABLE permissions` / `role_permissions`). The seed uses the documented pattern.

---

## 4. UI

Add pages under `internal/templates` (Go `html/template`) and JS under `internal/static`.
RBAC gated by permission resources above.

| Page | Template | JS asset | RBAC |
|------|----------|----------|------|
| Settlements list + generate | `templates/settlements/list.html` | `static/js/settlements.js` | `settlements:read` |
| Settlement detail (lines, mark-paid, confirm/dispute) | `templates/settlements/detail.html` | `static/js/settlements.js` | `settlements:*` |
| Accounting sync status + trigger | `templates/accounting/sync.html` | `static/js/accounting.js` | `accounting:read` |
| Document upload (driver/vehicle) | `templates/documents/upload.html` | `static/js/documents.js` | `documents:upload` |
| Compliance dashboard | `templates/compliance/dashboard.html` | `static/js/compliance.js` | `compliance:read` |

UI behaviours:
- **Settlements list**: table of gross/commission/advances/TDS/net/status; "Generate" button
  calls `POST /api/settlements/generate`. Detail page shows `lines` as a ledger.
- **Mark-paid**: modal with `payment_ref` + `paid_at`; disabled unless status `pending`/`disputed`.
- **Driver confirm/dispute**: only visible to driver-role session; dispute opens reason form.
- **Accounting**: shows adapter + enabled flag from `GET /api/accounting/sync/status`;
  "Trigger sync" posts to trigger endpoint; reconcile table lists `unacked_refs`.
- **Documents**: drag-drop upload → multipart `POST /api/documents/...`; show expiry badges
  (red if expired, amber if <30 days).
- **Compliance dashboard**: counts + blocked lists; "Verify" button per pending doc →
  `POST /api/compliance/verify/...`. A red **DISPATCH BLOCKED** banner appears when any
  assignee/vehicle is blocked.

Navigation: add menu entries gated by permission checks in the layout partial
`templates/partials/nav.html`.

---

## 5. Business logic

### 5.1 Driver Settlement Engine

#### 5.1.1 Rate calculators (gross fare)
Read config from `company_config` (seeded in 00051). Pseudo:

```
rate_model = config('settlement_rate_model')   // per_km | fixed | commission_pct
switch rate_model:
  case 'per_km':
      km   = trip.distance_km                   // from trips.odometer delta or trip.km
      rate = float(config('settlement_rate_per_km'))
      gross = km * rate
      basis = {km, rate_per_km: rate}
  case 'fixed':
      gross = float(config('settlement_fixed_fare'))
      basis = {fixed: gross}
  case 'commission_pct':
      fare  = booking.Price                    // must exist
      pct   = float(config('settlement_commission_pct')) / 100
      gross = fare * (1 - pct)                 // driver gets fare minus commission
      commission = fare * pct
      basis = {fare, commission_pct: pct}
```

#### 5.1.2 Deduction engine (from approved kharcha)
Sum all `driver_expenses` where `trip_id = ? AND driver_id = ? AND status='approved'`
(reuse `KharchaService.ListLedger` filtered, or a new `SumApprovedForTrip`). Each approved
expense becomes one `settlement_lines` row of type `deduction` (or aggregated as `advances`).

#### 5.1.3 TDS under Section 194C
```
pan_present = driver.PAN != nil && *driver.PAN != ""
rate = pan_present ? config('tds_rate_with_pan')   // 1.00
                   : config('tds_rate_without_pan') // 2.00  (Sec 206AA)
tds_base   = gross - commission - advances - approved_deductions   // pre-TDS payable
tds_amount = tds_base * (rate/100)
```
TDS rate stored on the settlement row (`tds_rate`, `tds_amount`) for audit.

#### 5.1.4 Net payable (final)
```
net_payout = gross
           - commission            (if commission_pct model, this is built into gross)
           - advances_kharcha
           - approved_deductions
           - tds_amount
net_payout = max(0, net_payout)
```

#### 5.1.5 Persistence (the fix)
New `GenerateSettlement(ctx, tripID, force)` method in `DriverSettlementService`:
1. Load trip; error `ErrTripNotFound` / `driver_not_assigned` if no driver.
2. If a settlement exists for `trip_id` and `!force`, return it (idempotent).
3. Compute gross/commission/deductions/tds/net (§5.1.1–5.1.4).
4. `INSERT` into `driver_settlements` (this is the missing write — replace the in-memory-only
   `CreateSettlementForTrip`). Use `trip_id` UNIQUE; on conflict return existing.
5. `INSERT` one `settlement_lines` row per component.
6. Emit outbox event `SettlementGenerated` (for accounting relay, §5.2).
7. **Fix `KharchaService.ApproveExpense`** so its step-3 `UPDATE driver_settlements` now
   matches a real row: after computing, also insert a `deduction` settlement_line and
   decrement `net_payout`/`increment advances_kharcha` on the existing settlement. The
   no-op becomes effective once rows exist.

State machine for `status`:
```
pending --mark-paid--> paid --confirm--> paid(confirmed_at set)
pending --dispute---> disputed --(finance resolve/recompute)--> pending|paid
paid    --dispute---> disputed
```
`mark-paid` writes `payment_ref`, `paid_at`, status=`paid` (idempotent: 409 if already paid
with different ref).

### 5.2 Accounting two-way sync

#### 5.2.1 Adapter refactor (`internal/integration/accounting/client.go`)
Replace the single `stubClient` with a factory:
```go
func NewClient(cfg Config) Client {
    switch strings.ToLower(cfg.Provider) {
    case "tally":      return &tallyClient{cfg: cfg}
    case "zoho":       return &zohoClient{cfg: cfg}
    case "quickbooks": return &quickbooksClient{cfg: cfg}
    default:           return &mockClient{cfg: cfg}   // safe default, no creds
    }
}
```
`Config` gains `Provider string`. Each real adapter implements `ExportInvoice`,
`SyncContacts`, `PushJournalEntry` with real HTTP calls **only when `cfg.Enabled`**; otherwise
return `ErrDisabled`. `mockClient` returns fake SUCCESS (current behaviour) — used in tests
and local dev. The interface is unchanged, so callers need no edits.

#### 5.2.2 Idempotent trigger via outbox relay
1. Financial services emit outbox events: `SettlementGenerated`, `DriverPayoutSettled`,
   `InvoiceExported` (already published by `driver_settlement_service.go:136`).
2. A new **accounting outbox consumer** (`internal/integration/accounting/consumer.go`)
   subscribes to the `events.EventBus` (wired in `main`/DI). On each event:
   - Build an `idempotency_key = event_type + ':' + aggregate_id` (e.g.
     `DriverPayoutSettled:stl_xyz`).
   - `INSERT INTO accounting_sync_log (idempotency_key, ...)` — UNIQUE constraint makes
     replays a no-op (catch `UNIQUE` violation → skip).
   - Map to adapter call (`PushJournalEntry` for payout using `accounting_gl_rule`).
   - On success update `status='sent'/'acked'`, `external_id`. On failure increment
     `attempts`, set `last_error`; the outbox relay retry (§0.7) replays later.
3. `accounting_mapping` stores returned `external_id` for two-way reconcile (`GET
   /api/accounting/reconcile`).

Two-way: inbound (e.g. payment acknowledgement from accounting) is a future hook — table
`direction='in'` supports it; v1 implements out-only with reconcile read-back.

### 5.3 Document Vault + Compliance

#### 5.3.1 Persistence of Aadhaar/PAN/Bank
Ensure `driver_repository.go` `Save`/`Update` writes `aadhaar`, `pan`, `bank_details`
(columns exist since 00030). Add fields to the driver create/update form + API. No new
migration column needed (they exist) — this is a **repository wiring** fix.

#### 5.3.2 Documents
`DocumentService` (new, in `internal/service/`) `UploadDriverDoc` / `UploadVehicleDoc`:
store file via existing `Files` service (`internal/service/file_service.go`), then `INSERT`
into `driver_documents`/`vehicle_documents`. `Verify` flips status + writes `verified_by`.

#### 5.3.3 PUC + expiry enforcement
Add `PUCExpiry time.Time` to `internal/domain/vehicle/entity.go` and load it in the vehicle
repository (column added in 00052). Extend `Vehicle.CanAssign()` (entity.go:67) and
`ComplianceService.ValidateVehicleCompliance` (compliance_service.go:79) to also check
`PUCExpiry` like the other expiry checks.

#### 5.3.4 Hard dispatch block (wire ComplianceService)
In the trip **assign** and **start** handlers (`internal/trip/...` or `internal/handlers/...`),
call `Services.Compliance.EnforceDispatchCompliance(ctx, &driverID, &vehicleID)` BEFORE
committing the assign/start. On error (`compliance hard-block: ...`) return `409
dispatch_blocked` and abort. This makes `ComplianceService` (currently unwired, §0.8) active.

---

## 6. Config / env

| Var | Default | Purpose | Package that reads |
|-----|---------|---------|--------------------|
| `ACCOUNTING_ADAPTER` | `mock` | `tally\|zoho\|quickbooks\|mock` | `internal/integration/accounting` (maps to `Config.Provider`) |
| `ACCOUNTING_ENABLED` | `false` | Master switch for real API | `accounting` client + consumer |
| `ACCOUNTING_ENDPOINT` | `""` | Base URL of adapter | `accounting` client |
| `ACCOUNTING_API_KEY` | `""` | API key/secret (inject from vault) | `accounting` client |
| `SETTLEMENT_RATE_MODEL` | `per_km` | Rate model selector | `DriverSettlementService` |
| `SETTLEMENT_RATE_PER_KM` | `11.90` | ₹/km | `DriverSettlementService` |
| `SETTLEMENT_FIXED_FARE` | `5000.00` | Flat fare | `DriverSettlementService` |
| `SETTLEMENT_COMMISSION_PCT` | `5.00` | % commission | `DriverSettlementService` |
| `TDS_RATE_WITH_PAN` | `1.00` | 194C rate w/ PAN | `DriverSettlementService` |
| `TDS_RATE_WITHOUT_PAN` | `2.00` | 206AA rate w/o PAN | `DriverSettlementService` |
| `OUTBOX_POLL_INTERVAL` | `5s` | Relay poll (existing) | `internal/shared/outbox` |

Env vars are **preferred at boot** and **fall through to `company_config` rows** (seeded in
§3) so tenants can override per-company without restart. Read precedence: env →
`company_config` → compiled default. All external calls fire only when `ACCOUNTING_ENABLED=true`
(or `company_config.accounting_enabled='true'`); otherwise `mock` adapter is used and no
network egress occurs. No external credentials are required to run tests.

---

## 7. Tests

Coverage gate: new packages ≥ 80%; `go test ./internal/service/... ./internal/integration/...`
must pass before merge.

### 7.1 Settlement math (unit)
`internal/service/driver_settlement_service_test.go`
- Case A (per_km): km=420, rate=11.90 → gross=4998.00; commission=0; advances=600;
  approved=150; PAN present → tds 1% on (4998-600-150)=4248 → 42.48; net=4248-42.48=4205.52.
- Case B (commission_pct 5%): fare=5000 → gross=4750, commission=250; net after advances
  600, deduct 150, tds 1% on 4000 =40 → 3950? verify 4750-600-150-40=3960.
- Case C (no PAN): same base, tds 2% → double TDS, lower net.
- Assert `net_payout >= 0` floor.

### 7.2 TDS boundary
- PAN empty/whitespace → use `tds_rate_without_pan` (2%). PAN non-empty → 1%. Table-driven.

### 7.3 Persistence + kharcha no-op fix
- `GenerateSettlement` then `SELECT` from `driver_settlements` → row exists (regression for
  §0.1). Re-POST same trip → returns same id (UNIQUE).
- Approve an expense for that trip → `advances_kharcha` increases and `net_payout` decreases
  (regression for §0.2 no-op). Also a `settlement_lines` deduction row appears.

### 7.4 Reconciliation idempotency
`internal/integration/accounting/consumer_test.go`
- Publish same event twice → `accounting_sync_log` has exactly 1 row for the idempotency_key
  (UNIQUE conflict handled). `reconcile` reports `unacked` correctly.
- `mock` adapter returns SUCCESS without network; `Enabled=false` returns `ErrDisabled`.

### 7.5 Document persist + expiry
`internal/service/document_service_test.go` + `vehicle` entity test
- Upload driver doc → `driver_documents` row; `Verify` flips status.
- Vehicle with `PUCExpiry` in past → `CanAssign()` returns compliance error; with future → ok.
- Aadhaar/PAN/Bank round-trip through driver repository save/load.

### Pass-before-merge checklist
- [ ] `goose up` 00050/00051/00052 applies clean; `goose down` reverses.
- [ ] No `INSERT INTO driver_settlements` missing (grep confirms writes exist).
- [ ] `EnforceDispatchCompliance` called in assign + start handlers (grep).
- [ ] `ACCOUNTING_ENABLED=false` + `mock` adapter → zero network calls in CI.

---

## 8. Future / GPS-provider

- **GPS-verified km payout**: tie `settlement_rate_model=per_km` to telematics-verified
  distance. Define a `TelematicsProvider` interface (per AGENTS.md convention); own MQTT/IMEI
  hardware is primary, LocoNav/WheelsEye/MapMyIndia/TelaBit/OBD are pluggable adapters. The
  settlement engine consumes `verified_km` from the telemetry aggregate instead of odometer
  delta, preventing km inflation disputes.
- **Telematics cost allocation**: allocate device/connectivity cost per trip as an extra
  `settlement_lines` line_type=`adjustment`, sourced from `telemetry_devices` (00040) usage.
- **TDS remittance + 26Q**: extend `accounting_gl_rule` with `TDSRemitted` and push to GSTN
  (separate spec).
- **Two-way accounting inbound**: consume payment-ack webhooks (`direction='in'`) to
  auto-`mark-paid` settlements.
- **AIS-140/VLT** compliance docs auto-uploaded from device provisioning.

---

## 9. Edge cases

1. **Trip with no booking + commission_pct model** → `422 booking_price_missing` (no fare
   source). Fall back to `fixed` only if configured.
2. **Re-generate after paid** → `force_recompute` allowed only when `status != 'paid'` unless
   admin override; otherwise 409.
3. **Approved expense after settlement generated** → deduction engine re-runs on next
   `GenerateSettlement(force=true)`; or `ApproveExpense` live-updates the open settlement.
4. **Driver PAN added after settlement** → TDS recomputed only on recompute; original kept for
   audit (immutable `settlement_lines`).
5. **Duplicate outbox event** → `accounting_sync_log` UNIQUE(idempotency_key) → skip, no
   double journal post.
6. **Accounting adapter down** → `last_error` + `attempts`; outbox relay retries with backoff
   (§0.7); status stays `pending`/`failed`, payout still recorded locally.
7. **Vehicle/driver blocked mid-trip** → dispatch block applies at assign/start only; already
   running trips unaffected (do not strand cargo).
8. **PUC/RC/Insurance missing (NULL)** → treated as NOT expired (no false block) but flagged
   `warning` on compliance dashboard; block only on concrete past date.
9. **Document without expiry** (e.g. PAN) → no expiry badge, only verification status.
10. **Negative net** → floored at 0; difference logged as `adjustment` for finance review.

---

## 10. Phased rollout (build order)

1. **00051** migration + `DriverSettlementService.GenerateSettlement` (persist) + unit tests
   (§7.1–7.3). Ship behind `settlements:*` RBAC.
2. Fix `KharchaService.ApproveExpense` no-op (§5.1.5) once rows exist.
3. **00052** migration (puc_expiry, doc tables, RBAC) + `DocumentService` + entity wiring +
   compliance dashboard + hard-block wiring (§5.3).
4. **00050** migration + accounting adapter refactor + outbox consumer + reconcile API (§5.2).
5. UI pages (§4) per feature, gated by permissions.
6. Integration tests + manual UAT with `mock` adapter (`ACCOUNTING_ENABLED=false`).

---

## 11. Open items / VERIFY

- **SQLite DROP COLUMN support**: confirm bundled `go-sqlite3`/SQLite ≥ 3.35 for the DOWN
  migrations; if older, provide table-rebuild DOWN scripts.
- **RBAC schema exact names**: confirm `permissions`/`role_permissions` column names in
  `00012_rbac.sql` and the admin role id (`role_admin`?) before seeding 00052.
- **`company_config` table**: confirm 00042 has merged and created the table before 00050/00051
  run (our seeds `INSERT` into it). If not merged, temporarily create the rows inline.
- **`trip.distance_km` source**: confirm where verified km lives (odometer delta vs trip
  column) for `per_km` model; may need a `trips.distance_km` column/derivation.
- **Driver.PAN format**: decide validation (10-char alphanumeric, uppercase) before TDS
  branch; whitespace-only treated as absent.
- **File storage backend**: confirm `Files` service upload path/url scheme used by
  `DocumentService`.
- **Outbox consumer wiring**: confirm DI location (`main.go`/wire) to subscribe the
  accounting consumer to `events.EventBus` and that relay already running.

---

## 12. File list

### Create
- `db/migrations/00051_driver_settlement_engine.sql`
- `db/migrations/00050_accounting_sync.sql`
- `db/migrations/00052_document_vault.sql`
- `internal/service/document_service.go`
- `internal/service/driver_settlement_service_test.go`
- `internal/integration/accounting/consumer.go`
- `internal/integration/accounting/tally_client.go`
- `internal/integration/accounting/zoho_client.go`
- `internal/integration/accounting/quickbooks_client.go`
- `internal/integration/accounting/mock_client.go`  (split from current `stubClient`)
- `internal/integration/accounting/consumer_test.go`
- `internal/templates/settlements/list.html`, `detail.html`
- `internal/templates/accounting/sync.html`
- `internal/templates/documents/upload.html`
- `internal/templates/compliance/dashboard.html`
- `internal/static/js/settlements.js`, `accounting.js`, `documents.js`, `compliance.js`
- HTTP handlers: `internal/handlers/settlement.go`, `internal/handlers/accounting.go`,
  `internal/handlers/documents.go`, `internal/handlers/compliance.go`

### Modify
- `internal/service/driver_settlement_service.go` — replace in-memory `CreateSettlementForTrip`/
  `ProcessFinancialSettlement` with persisting `GenerateSettlement`; add TDS + lines + status
  transitions (§5.1).
- `internal/service/kharcha_service.go` — `ApproveExpense` step-3 now effective + writes
  `settlement_lines` (§5.1.5).
- `internal/integration/accounting/client.go` — add `Provider` to `Config`, factory
  `NewClient`, keep interface (§5.2.1).
- `internal/domain/vehicle/entity.go` — add `PUCExpiry`; extend `CanAssign()` (§5.3.3).
- `internal/domain/driver/entity.go` — ensure Aadhaar/PAN/Bank round-trip (no-op if already).
- `internal/driver/infrastructure/persistence/sql/driver_repository.go` — persist
  aadhaar/pan/bank_details (§5.3.1).
- `internal/vehicle/infrastructure/persistence/sql/vehicle_repository.go` — load `puc_expiry`
  (§5.3.3).
- `internal/service/compliance_service.go` — add PUC check in `ValidateVehicleCompliance`
  (§5.3.3).
- Trip assign/start handlers — call `EnforceDispatchCompliance` (§5.3.4).
- `internal/templates/partials/nav.html` — menu entries gated by new permissions.
- `internal/config/config.go` — load new env vars (§6) with `company_config` fallthrough.
