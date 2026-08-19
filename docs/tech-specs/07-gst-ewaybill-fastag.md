# GST E-Invoicing, E-Way Bill & FASTag — Implementation Spec v1

Status: ready
Depends-on:
  - Spec 02 (company_config canonical table — 00042)
  - Spec 05 (alerting owns 00045/00046); EWB lifecycle 00047 + GST/RBAC 00048 owned HERE (spec 07)
Migration owner:
  - db/migrations/00047_*.sql  (e-way bill lifecycle: ALTER eway_bills + eway_bill_events) — OWNED HERE
  - db/migrations/00048_*.sql  (GST e-invoice: invoice line items, invoice_sequences, CGST/SGST/IGST cols, hsn_sac_master, company state code) — OWNED HERE
  - db/migrations/00049_*.sql  (FASTag tables: fastag_tags + fastag_transactions) — OWNED HERE
  - RBAC permission rows (integrations:gstn/einvoice/ewaybill/fastag) are seeded inside 00048 (GST e-invoice migration)

---

## 0. Verified ground truth

All facts below were read directly from the repo on the current HEAD. Line numbers are exact.

### 0.1 GSTN integration — stub, no IRN, no e-invoice
- `internal/integration/gstn/client.go:53-57` — `Client` interface exposes only
  `ValidateGSTIN`, `FetchGSTR1Summary`, `FetchGSTR3BSummary`. **No IRN generation,
  no e-invoice push.**
- `internal/integration/gstn/client.go:64-69` — `NewClient` returns a `stubClient`;
  every method early-returns `fmt.Errorf("gstn integration disabled")` when
  `cfg.Enabled == false` (lines 73, 90, 108). Real NIC calls do NOT exist.
- `internal/integration/gstn/client.go:9-14` — `Config{ Endpoint, APIKey, Enabled }`.
  **No `UseMock` flag, no NIC username/password/client-id placeholders.**

### 0.2 HTTP surface — no e-invoice routes
- `internal/integration/handler.go:46-50` — `/api/v1/integrations/gstn` only mounts
  `validate/{gstin}`, `gstr1-summary`, `gstr3b-summary`. **No `/einvoice/generate`
  or `/irn/generate` route.**
- `internal/integration/handler.go:41-45` — ewaybill routes: `generate`,
  `get/{ewbNumber}`, `cancel`. Uses permission `integrations:ewaybill`.

### 0.3 Config loading — no USE_MOCK
- `internal/integration/config.go:29-33` — `GSTN` config only loads
  `INTEGRATION_GSTN_ENDPOINT/API_KEY/ENABLED`. Same pattern for EWayBill (24-28)
  and FASTag (34-38). **No `*_USE_MOCK` env var anywhere.**

### 0.4 Invoice domain — flat tax, no line items, no tax split, no IRN
- `internal/invoice/application/generate_invoice.go:116-148` — `resolveBookingPricing`
  computes a **single flat tax**: `taxMinor = int64(math.Round(float64(subtotalMinor)
  * settings.GSTRate / 100.0))` (line 138), rounded to paisa. There is **no
  CGST/SGST/IGST split**, no HSN lookup, no line items.
- `internal/invoice/domain/aggregate/invoice_aggregate.go:30-49` — `InvoiceAggregate`
  carries `Subtotal, Tax, Discount, Total` (plus `PaymentStatus`, `DueDate`,
  `FinancialYear`, `CreditBalance`, `Remarks`, etc.) but **no `Cgst/Sgst/Igst`
  split, no `LineItems`, no `IRN`, no `IRNAckNo`, no `SignedQR`, no `EwbNumber`.**

### 0.5 PDF — hardcoded currency and company name
- `internal/pdf/invoice_pdf.go:52` — financial table header is **hardcoded `"Amount ($)"`**.
- `internal/handlers/invoices.go:167` — `pdfgen.GenerateInvoicePDF(invEntity,
  "Apex Transport Ltd")` **hardcodes the company name** instead of reading
  `CompanySettings.CompanyName`.

### 0.6 E-Way Bill — stub and NEVER persisted
- `internal/integration/ewaybill/client.go:50-54` — `Client` interface:
  `Generate, Get, Cancel`.
- `internal/integration/ewaybill/client.go:68-82` — `Generate` returns a fake
  `EwbNumber: "EWB"+uuid[:12]` and `QRCode: "data:image/png;base64,stubqrcode"`.
- `internal/integration/handler.go:64-77` — `GenerateEWayBill` calls the client and
  **encodes the response but never writes to `eway_bills`**. The table exists
  (`db/migrations/00031_avandab_critical_fixes.sql:19-32`) but is never inserted into.
- `db/migrations/00031_avandab_critical_fixes.sql:19-32` — `eway_bills` columns:
  `id, trip_id UNIQUE, ewb_number UNIQUE NOT NULL, irn UNIQUE, generation_date,
  valid_until, transporter_id, vehicle_number, status (active|cancelled|expired),
  raw_response, created_at`. FK `trip_id -> trips(id)`.
- `eway_bill_events` table **does NOT exist** (grep of `db/migrations` finds zero
   matches). It will be created by **00047 (Spec 07)**.
- `internal/domain/ewaybill/entity.go:17-29` — `EWayBill` domain entity has
  `IRN *string`, `TransporterID *string`, `VehicleNumber *string`; note `Status`
  is a plain `string` (not a pointer) and `ValidUntil` is `time.Time` (not
  `*string`). The struct also has `ID`, `TripID`, `GenerationDate`, `CreatedAt`.

### 0.7 FASTag — stub fake balance, no persistence, no reconcile
- `internal/integration/fastag/client.go:51-55` — `Client`: `GetBalance,
  DeductToll, ListTransactions`.
- `internal/integration/fastag/client.go:69-81` — `GetBalance` returns
  **hardcoded `Balance: 2475.50`** (line 77), `Currency: "INR"`.
- `internal/integration/fastag/client.go:100-123` — `ListTransactions` fabricates
  rows; **nothing is persisted** to any FASTag table (no `fastag_*` tables exist).
- `test/api_surface_test.go:130` — **pins the fake value**: `assert.Equal(t, 2475.50,
  res["balance"])`. Must be rewritten (see §7).
- `internal/integration/handler.go:51-55` — FASTag routes mount `balance`, `deduct`,
  `transactions` under permission `integrations:fastag`.

### 0.8 Company / Customer — missing state codes
- `internal/domain/company/entity.go:6-24` — `CompanySettings` has `GSTNumber *string`,
  `GSTRate float64`, `GSTEnabled bool`, but **no `StateCode`**.
- `internal/domain/customer/entity.go:18-34` — `Customer` has `GST *string` but
  **no parsed `StateCode`**; state must be derived from the GSTIN's first 2 chars.

### Verification Log

Cross-checked every §0 claim against the cited source on current HEAD
(head migration = `00039_experiments.sql`; migration numbers reconciled against
`docs/tech-specs/00-migration-ownership-index.md`).

| # | Claim | Verdict | Correction / Evidence |
|---|-------|---------|------------------------|
| 1 | GSTN `Client` interface has only ValidateGSTIN/FetchGSTR1/FetchGSTR3B, no IRN | VERIFIED | `gstn/client.go:53-57` |
| 2 | `NewClient` returns `stubClient`; methods error when `!Enabled` (73,90,108) | VERIFIED | `gstn/client.go:64-120` |
| 3 | `gstn.Config{Endpoint,APIKey,Enabled}`, no `UseMock` | VERIFIED | `gstn/client.go:9-14` |
| 4 | Handler mounts no e-invoice GSTN route | VERIFIED | `handler.go:46-50` |
| 5 | ewaybill routes use `integrations:ewaybill` | VERIFIED | `handler.go:41-45` |
| 6 | config.go loads only ENDPOINT/API_KEY/ENABLED, no USE_MOCK | VERIFIED | `config.go:24-38` |
| 7 | Flat single tax, no CGST/SGST/IGST split (line 138) | VERIFIED | `generate_invoice.go:138` (uses `math.Round(x*100)/100.0`, not bare `/100`) |
| 8 | InvoiceAggregate has no tax-split/IRN fields | VERIFIED | `invoice_aggregate.go:30-49` (also has PaymentStatus/DueDate/etc.) |
| 9 | PDF header hardcoded `"Amount ($)"` at line 52 | VERIFIED | `invoice_pdf.go:52` |
| 10 | Handler hardcodes `"Apex Transport Ltd"` at line 167 | VERIFIED | `handlers/invoices.go:167` |
| 11 | ewaybill `Client`: Generate/Get/Cancel only | VERIFIED | `ewaybill/client.go:50-54` |
| 12 | `Generate` returns `EWB`+uuid[:12] + stub QR | VERIFIED | `ewaybill/client.go:75,79` |
| 13 | `GenerateEWayBill` never writes `eway_bills` | VERIFIED | `handler.go:64-77`; grep `INSERT INTO eway_bills` = 0; grep `eway_bills` in `*.go` = 0 |
| 14 | `eway_bills` exists at 00031:19-32 with exact columns | VERIFIED | `00031_avandab_critical_fixes.sql:19-32` columns match |
| 15 | `eway_bill_events` does NOT exist | VERIFIED | grep `eway_bill_events` in `*.sql` = 0 |
| 16 | `EWayBill` entity field types | WRONG | `entity.go:17-29`: `Status` is `string` (not `*string`), `ValidUntil` is `time.Time` (not `*string`) — corrected above |
| 17 | FASTag `Client`: GetBalance/DeductToll/ListTransactions | VERIFIED | `fastag/client.go:51-55` |
| 18 | `GetBalance` hardcodes `2475.50` / `INR` (line 77) | VERIFIED | `fastag/client.go:77-78` |
| 19 | FASTag fabricates txns; no `fastag_*` table persists | VERIFIED | `fastag/client.go:100-123`; grep `fastag_*` in `*.go` = 0 |
| 20 | `test/api_surface_test.go:130` pins `2475.50` | VERIFIED | `test/api_surface_test.go:130` |
| 21 | FASTag routes use `integrations:fastag` | VERIFIED | `handler.go:51-55` |
| 22 | `CompanySettings` has GSTNumber/GSTRate/GSTEnabled, no StateCode | VERIFIED | `company/entity.go:6-24` |
| 23 | `Customer` has `GST *string`, no StateCode | VERIFIED | `customer/entity.go:18-34` |
| 24 | Migration numbers 00042/00047/00048/00049 consistent w/ index; head 00039 | VERIFIED | `00-migration-ownership-index.md:18-30`; glob max = `00039_experiments.sql` |

---

## 1. Overview / goal

Build three tax-and-toll compliance features behind real **adapter interfaces with a
config-flagged MOCK** so the whole system runs and is testable without any external
credentials:

1. **GST E-Invoicing** — compute per-line-item CGST/SGST (intra-state) or IGST
   (inter-state) via GSTIN state-code parsing + HSN/SAC rate lookup, persist line
   items, and generate an IRN via the NIC adapter (mock by default) returning
   `{irn, ack_no, ack_date, signed_qr}`.
2. **E-Way Bill lifecycle** — generate Part-A, attach Part-B (vehicle), update vehicle,
   extend, cancel, fetch by number / by trip, persist to `eway_bills` and emit
   `eway_bill_events`. Auto-generate on trip confirm when `goods_value > 50000`.
    **Schema changes (00047) are owned by Spec 07 (this spec) — we only add the EWB feature
   API/UI/business logic here and reference the columns 00047 adds.**
3. **FASTag** — persist tag wallet + toll transactions, reconcile provider pulls
   against trips, and auto-create `kharcha` (driver expense) rows for reconciled tolls.

### Non-goals
- No real NIC/GSTN/FASTag provider credentials or live network calls in this spec
  (MOCK is the default; real adapter is a thin extension).
- No changes to the 00047 e-way-bill schema migration (owned by Spec 07 / this spec).
- No GSTR-1/3B filing UI (existing stub endpoints remain).
- No e-way bill bulk/consolidated/vehicle-movement APIs beyond Part-A/B, extend, cancel.
- TDS is noted (§5.5) but the engine lives in 00051 (Spec 08 settlement).

### Severity & Effort

| Change | Severity | Effort | Rationale |
|--------|----------|--------|-----------|
| 00048 GST e-invoice schema (company state_code, CGST/SGST/IGST cols, line items, hsn_sac_master, sequences) | High | M | Blocks all GST e-invoice logic; data-model migration is risky to reorder later |
| Invoice flat-tax → line-item CGST/SGST/IGST split (§5.1–5.2) | Critical | M | Current invoices are non-compliant (single flat tax); wrong tax filings |
| IRN generation + NIC/MOCK adapter (§2.1, §2.2) | High | M | Legal requirement for B2B invoices > ₹?; deterministic MOCK keeps tests stable |
| PDF currency + company-name fix (§9, §0.5) | Low | S | Pure cosmetic/integration bug; no schema |
| 00049 FASTag tables + repo-backed MOCK + api_surface_test rewrite (§7.5) | High | M | Unblocks FASTag reconciliation; currently returns fake 2475.50 |
| EWB feature logic persisting to `eway_bills` + `eway_bill_events` (00047 columns) | High | L | Depends on this spec's 00047 merging first; coordination risk |
| EWB auto-generate on trip confirm (§5.4) + expiry/extend monitor (§5.5) | Med | M | Automation; safe behind `ewaybill_auto_generate` flag |
| FASTag reconcile + auto-kharcha (§5.6) | Med | M | Expense accuracy; gated by `fastag_auto_kharcha` |
| UI (line-item editor, tax split, IRN/QR, EWB card, FASTag dash) | Med | L | Dependent on all above APIs |

---

## 2. API contract

All routes mounted under `/api/v1/integrations` (existing `Handler.Register`,
`internal/integration/handler.go:39`). Permission strings follow the existing
`middleware.RequirePermission(authSrv, <resource>, <action>)` pattern.

### 2.1 GST — Generate IRN
```
POST /api/v1/integrations/gstn/einvoice/irn
Permission: integrations:gstn
Request:
{
  "invoice_id": "uuid"            // existing invoice id (00048 adds line items + tax split)
}
Response 201:
{
  "invoice_id": "uuid",
  "irn": "4b7b ... 64-char hash",
  "ack_no": "122133213455",
  "ack_date": "2026-08-18 14:32:11",
  "signed_qr": "data:image/png;base64,iVBORw0KGgo...",
  "status": "ACTIVE"
}
Error: 404 if invoice not found; 409 if IRN already exists; 502 if adapter disabled/failed.
```
IRN is deterministic (hash of canonical invoice JSON — see §5.2) so the MOCK and real
adapter return the same value for the same input.

### 2.2 GST — Push e-invoice to GSTN
```
POST /api/v1/integrations/gstn/einvoice/push
Permission: integrations:gstn
Request:
{
  "invoice_id": "uuid",
  "irn": "4b7b ... "
}
Response 200:
{
  "invoice_id": "uuid",
  "irn": "4b7b ...",
  "ack_no": "122133213455",
  "ack_date": "2026-08-18 14:35:02",
  "signed_qr": "data:image/png;base64,...",
  "pushed": true
}
Error: 404 not found; 412 IRN missing (generate first); 502 adapter error.
```

### 2.3 E-Way Bill — Generate Part-A
```
POST /api/v1/integrations/ewaybill/generate-part-a
Permission: integrations:ewaybill
Request:
{
  "trip_id": "uuid",
  "doc_type": "INV",                 // INV | BIL | CHA | OTH
  "doc_no": "INV/2026/0001",
  "doc_date": "2026-08-18",
  "from_gstin": "27AABCU9603R1ZX",
  "to_gstin": "07AAACP0000M1Z9",
  "from_place": "Mumbai",
  "from_state_code": "27",
  "to_place": "New Delhi",
  "to_state_code": "07",
  "transporter_id": "27AABCU9603R1ZX",
  "goods_value": 185000.00,
  "distance": 1450
}
Response 201:
{
  "ewb_number": "31100...",
  "status": "PART_A_GENERATED",
  "valid_upto": "2026-08-21T14:32:11Z",
  "qr_code": "data:image/png;base64,...",
  "trip_id": "uuid",
  "irn": null
}
```
The response is persisted to `eway_bills` (columns added by 00047) and an
`eway_bill_events` row of type `PART_A_GENERATED` is written.

### 2.4 E-Way Bill — Attach Part-B (vehicle / update vehicle)
```
POST /api/v1/integrations/ewaybill/part-b
Permission: integrations:ewaybill
Request:
{
  "ewb_number": "31100...",
  "vehicle_number": "MH12AB1234",
  "transporter_id": "27AABCU9603R1ZX",
  "from_place": "Mumbai",
  "from_state_code": "27",
  "reason": "FIRST_TIME"            // FIRST_TIME | VEHICLE_BREAKDOWN | ...
}
Response 200:
{
  "ewb_number": "31100...",
  "status": "ACTIVE",
  "vehicle_number": "MH12AB1234",
  "valid_upto": "2026-08-21T14:32:11Z"
}
```

### 2.5 E-Way Bill — Extend / update validity
```
POST /api/v1/integrations/ewaybill/extend
Permission: integrations:ewaybill
Request:
{
  "ewb_number": "31100...",
  "from_place": "Nashik",
  "from_state_code": "27",
  "remaining_distance": 350,
  "transit_to_date": "2026-08-23",
  "reason": "TRANSIT"
}
Response 200:
{
  "ewb_number": "31100...",
  "status": "EXTENDED",
  "valid_upto": "2026-08-23T23:59:59Z",
  "extension_count": 1
}
```

### 2.6 E-Way Bill — Cancel
```
POST /api/v1/integrations/ewaybill/cancel
Permission: integrations:ewaybill
Request:
{
  "ewb_number": "31100...",
  "reason": "ORDER_CANCELLED",       // ORDER_CANCELLED | DATA_ENTRY_ERROR | ...
  "remark": "customer cancelled PO"
}
Response 200:
{
  "ewb_number": "31100...",
  "status": "CANCELLED",
  "cancelled_at": "2026-08-18T15:00:00Z"
}
```

### 2.7 E-Way Bill — Get by number
```
GET /api/v1/integrations/ewaybill/get/{ewbNumber}
Permission: integrations:ewaybill
Response 200:
{
  "ewb_number": "31100...",
  "trip_id": "uuid",
  "status": "ACTIVE",
  "irn": "4b7b...",                  // linked invoice IRN if present
  "generated_at": "2026-08-18T14:32:11Z",
  "valid_upto": "2026-08-21T14:32:11Z",
  "vehicle_number": "MH12AB1234",
  "qr_code": "data:image/png;base64,..."
}
Error: 404 if not found.
```

### 2.8 E-Way Bill — Get by trip
```
GET /api/v1/integrations/ewaybill/trip/{tripId}
Permission: integrations:ewaybill
Response 200: { "ewb_number": "31100...", "status": "ACTIVE", ... }  (latest active EWB)
Error: 404 if no EWB for trip.
```

### 2.9 FASTag — Balance
```
GET /api/v1/integrations/fastag/balance?vehicle_number=MH12AB1234&tag_id=TAG123
Permission: integrations:fastag
Response 200:
{
  "vehicle_number": "MH12AB1234",
  "tag_id": "TAG123",
  "balance": 2475.50,            // read from fastag_tags (was fake-stub 2475.50)
  "currency": "INR",
  "updated_at": "2026-08-18T14:00:00Z"
}
```

### 2.10 FASTag — Transactions
```
GET /api/v1/integrations/fastag/transactions?vehicle_number=MH12AB1234&limit=50
Permission: integrations:fastag
Response 200:
{ "transactions": [
  {
    "transaction_id": "uuid",
    "vehicle_number": "MH12AB1234",
    "tag_id": "TAG123",
    "plaza_id": "PLZ001",
    "plaza_name": "Toll Plaza 1",
    "amount": 85.00,
    "timestamp": "2026-08-18T10:00:00Z",
    "status": "SUCCESS",
    "reconciled": true,
    "trip_id": "uuid"
  }
]}
```

### 2.11 FASTag — Reconcile (pull from provider + match to trips + auto-kharcha)
```
POST /api/v1/integrations/fastag/reconcile
Permission: integrations:fastag
Request:
{
  "vehicle_number": "MH12AB1234",
  "from_date": "2026-08-10",
  "to_date": "2026-08-18"
}
Response 200:
{
  "pulled": 24,
  "matched": 20,
  "unmatched": 4,
  "kharcha_created": 20,
  "unmatched_txns": ["uuid","uuid","uuid","uuid"]
}
```

### 2.12 FASTag — Deduct (single toll, used by GPS toll plaza events)
```
POST /api/v1/integrations/fastag/deduct
Permission: integrations:fastag
Request:
{
  "vehicle_number": "MH12AB1234",
  "tag_id": "TAG123",
  "plaza_id": "PLZ001",
  "plaza_name": "Toll Plaza 1",
  "amount": 85.00,
  "trip_id": "uuid"
}
Response 200:
{
  "transaction_id": "uuid",
  "vehicle_number": "MH12AB1234",
  "amount": 85.00,
  "timestamp": "2026-08-18T10:00:00Z",
  "status": "SUCCESS",
  "kharcha_id": "uuid"             // auto-created driver_expense (toll) row
}
```

---

## 3. DB contract

### 3.1 Migration 00048 — GST e-invoice (OWNED HERE)

```sql
-- +goose Up
-- 00048 GST e-invoice: line items, sequences, CGST/SGST/IGST, HSN/SAC master, company state code

-- 1) Company state code (used for intra/inter-state tax determination)
ALTER TABLE company_settings ADD COLUMN state_code TEXT NOT NULL DEFAULT '27';

-- 2) HSN / SAC master
CREATE TABLE IF NOT EXISTS hsn_sac_master (
  code        TEXT PRIMARY KEY,
  description TEXT NOT NULL,
  type        TEXT NOT NULL CHECK (type IN ('HSN','SAC')),
  rate        REAL NOT NULL DEFAULT 0,   -- default GST rate %
  cess_rate   REAL NOT NULL DEFAULT 0,
  active      INTEGER NOT NULL DEFAULT 1,
  created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_hsn_sac_type ON hsn_sac_master(type, active);

-- Seed common codes (illustrative; expand as needed)
INSERT INTO hsn_sac_master (code, description, type, rate) VALUES
  ('996511','Goods transport by road','SAC',5.0),
  ('996512','Container freight transport','SAC',12.0),
  ('8708','Parts of motor vehicles','HSN',28.0),
  ('997113','Packaging services','SAC',18.0);

-- 3) Invoice CGST/SGST/IGST split columns
ALTER TABLE invoices ADD COLUMN cgst REAL NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN sgst REAL NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN igst REAL NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN irn TEXT;
ALTER TABLE invoices ADD COLUMN irn_ack_no TEXT;
ALTER TABLE invoices ADD COLUMN irn_ack_date TEXT;
ALTER TABLE invoices ADD COLUMN signed_qr TEXT;
ALTER TABLE invoices ADD COLUMN ewb_number TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_irn ON invoices(irn) WHERE irn IS NOT NULL;

-- 4) Invoice line items
CREATE TABLE IF NOT EXISTS invoice_line_items (
  id              TEXT PRIMARY KEY,
  invoice_id      TEXT NOT NULL,
  hsn_sac_code    TEXT,
  description     TEXT NOT NULL,
  unit            TEXT,
  quantity        REAL NOT NULL DEFAULT 1,
  rate            REAL NOT NULL DEFAULT 0,         -- taxable rate per unit
  taxable_value   REAL NOT NULL DEFAULT 0,
  cgst_rate       REAL NOT NULL DEFAULT 0,
  sgst_rate       REAL NOT NULL DEFAULT 0,
  igst_rate       REAL NOT NULL DEFAULT 0,
  cgst_amount     REAL NOT NULL DEFAULT 0,
  sgst_amount     REAL NOT NULL DEFAULT 0,
  igst_amount     REAL NOT NULL DEFAULT 0,
  total           REAL NOT NULL DEFAULT 0,         -- taxable + all taxes
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (invoice_id)   REFERENCES invoices(id) ON DELETE CASCADE,
  FOREIGN KEY (hsn_sac_code)  REFERENCES hsn_sac_master(code)
);
CREATE INDEX IF NOT EXISTS idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_line_items_hsn ON invoice_line_items(hsn_sac_code);

-- 5) Per-financial-year e-invoice sequence (NIC document number)
CREATE TABLE IF NOT EXISTS invoice_sequences (
  financial_year TEXT NOT NULL,
  tenant_id      TEXT NOT NULL,
  last_number    INTEGER NOT NULL DEFAULT 0,
  prefix         TEXT NOT NULL DEFAULT 'INV',
  PRIMARY KEY (financial_year, tenant_id)
);

-- 6) RBAC permission rows (resource=integrations, action=gstn already exists in handler;
--    ensure actions einvoice + ewaybill + fastag are registered)
INSERT INTO permissions (resource, action, description) VALUES
  ('integrations','einvoice','Generate/push GST e-invoice IRN'),
  ('integrations','ewaybill','Manage e-way bills'),
  ('integrations','fastag','Manage FASTag wallet & reconciliation')
  ON CONFLICT (resource, action) DO NOTHING;

-- 7) company_config feature-flag seed rows (canonical table is 00042; we only seed)
INSERT INTO company_config (key, value, description) VALUES
  ('gst_einvoice_enabled','false','GST e-invoice IRN generation feature flag'),
  ('ewaybill_auto_generate','true','Auto-create EWB on trip confirm when goods_value>50000'),
  ('fastag_auto_kharcha','true','Auto-create driver_expense(toll) on reconcile')
  ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS invoice_line_items;
DROP TABLE IF EXISTS invoice_sequences;
DROP TABLE IF EXISTS hsn_sac_master;
ALTER TABLE invoices DROP COLUMN IF EXISTS cgst;
ALTER TABLE invoices DROP COLUMN IF EXISTS sgst;
ALTER TABLE invoices DROP COLUMN IF EXISTS igst;
ALTER TABLE invoices DROP COLUMN IF EXISTS irn;
ALTER TABLE invoices DROP COLUMN IF EXISTS irn_ack_no;
ALTER TABLE invoices DROP COLUMN IF EXISTS irn_ack_date;
ALTER TABLE invoices DROP COLUMN IF EXISTS signed_qr;
ALTER TABLE invoices DROP COLUMN IF EXISTS ewb_number;
ALTER TABLE company_settings DROP COLUMN IF EXISTS state_code;
DELETE FROM permissions WHERE resource='integrations' AND action IN ('einvoice','ewaybill','fastag');
DELETE FROM company_config WHERE key IN ('gst_einvoice_enabled','ewaybill_auto_generate','fastag_auto_kharcha');
```

> Note: `company_config` is created by 00042 (Spec 02). Do NOT `CREATE TABLE
> company_config` here. `permissions` already exists (00012_rbac.sql) and
> `integrations:gstn`/`integrations:ewaybill`/`integrations:fastag` actions are
> already used by `Handler.Register`; the INSERT above is idempotent.

### 3.2 Migration 00049 — FASTag tables (OWNED HERE)

```sql
-- +goose Up
-- 00049 FASTag wallet + transactions

CREATE TABLE IF NOT EXISTS fastag_tags (
  id            TEXT PRIMARY KEY,
  tenant_id     TEXT NOT NULL,
  vehicle_id    TEXT,
  tag_id        TEXT UNIQUE NOT NULL,
  vehicle_number TEXT,
  issuer        TEXT,
  tag_class     TEXT,
  balance       REAL NOT NULL DEFAULT 0,
  status        TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','BLOCKED','EXPIRED','CLOSED')),
  last_sync     DATETIME,
  created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);
CREATE INDEX IF NOT EXISTS idx_fastag_tags_vehicle ON fastag_tags(vehicle_number);
CREATE INDEX IF NOT EXISTS idx_fastag_tags_tenant ON fastag_tags(tenant_id);

CREATE TABLE IF NOT EXISTS fastag_transactions (
  id            TEXT PRIMARY KEY,
  tenant_id     TEXT NOT NULL,
  tag_id        TEXT NOT NULL,
  vehicle_id    TEXT,
  vehicle_number TEXT,
  trip_id       TEXT,
  plaza_id      TEXT,
  plaza_name    TEXT,
  amount        REAL NOT NULL DEFAULT 0,
  txn_timestamp DATETIME NOT NULL,
  status        TEXT NOT NULL DEFAULT 'SUCCESS' CHECK (status IN ('SUCCESS','FAILURE','PENDING')),
  source        TEXT NOT NULL DEFAULT 'PROVIDER' CHECK (source IN ('PROVIDER','MANUAL','GPS')),
  reconciled    INTEGER NOT NULL DEFAULT 0,
  kharcha_id    TEXT,
  created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (tag_id)  REFERENCES fastag_tags(tag_id),
  FOREIGN KEY (trip_id)  REFERENCES trips(id),
  FOREIGN KEY (kharcha_id) REFERENCES driver_expenses(id)
);
CREATE INDEX IF NOT EXISTS idx_fastag_txn_tag ON fastag_transactions(tag_id, txn_timestamp);
CREATE INDEX IF NOT EXISTS idx_fastag_txn_trip ON fastag_transactions(trip_id);
CREATE INDEX IF NOT EXISTS idx_fastag_txn_recon ON fastag_transactions(reconciled, tenant_id);

-- company_config seed (canonical table is 00042; seed only)
INSERT INTO company_config (key, value, description) VALUES
  ('fastag_merchant_id',''),
  ('fastag_provider','MOCK')
  ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS fastag_transactions;
DROP TABLE IF EXISTS fastag_tags;
DELETE FROM company_config WHERE key IN ('fastag_merchant_id','fastag_provider');
```

### 3.3 E-Way Bill lifecycle — owned by 00047 (Spec 07); DO NOT recreate

This spec (07) adds **no EWB schema**. Migration **00047** (Spec 07) is the
owner and performs:

```sql
-- 00047 (Spec 07) — what it ADDS to the existing eway_bills (00031) and NEW eway_bill_events
ALTER TABLE eway_bills ADD COLUMN part_a_json    TEXT;   -- full Part-A request payload
ALTER TABLE eway_bills ADD COLUMN part_b_json    TEXT;   -- vehicle / Part-B payload
ALTER TABLE eway_bills ADD COLUMN from_place     TEXT;
ALTER TABLE eway_bills ADD COLUMN from_state_code TEXT;
ALTER TABLE eway_bills ADD COLUMN to_place       TEXT;
ALTER TABLE eway_bills ADD COLUMN to_state_code   TEXT;
ALTER TABLE eway_bills ADD COLUMN goods_value    REAL;
ALTER TABLE eway_bills ADD COLUMN distance       INTEGER;
ALTER TABLE eway_bills ADD COLUMN doc_type       TEXT;
ALTER TABLE eway_bills ADD COLUMN doc_no         TEXT;
ALTER TABLE eway_bills ADD COLUMN doc_date       TEXT;
ALTER TABLE eway_bills ADD COLUMN transporter_doc_no TEXT;
ALTER TABLE eway_bills ADD COLUMN extension_count INTEGER DEFAULT 0;
ALTER TABLE eway_bills ADD COLUMN cancel_reason  TEXT;
ALTER TABLE eway_bills ADD COLUMN cancelled_at   DATETIME;
ALTER TABLE eway_bills ADD COLUMN qr_code        TEXT;
ALTER TABLE eway_bills ADD COLUMN ack_no         TEXT;
ALTER TABLE eway_bills ADD COLUMN gen_mode       TEXT DEFAULT 'MANUAL';
-- status CHECK extended to: active|part_a|cancelled|expired|extended

CREATE TABLE eway_bill_events (
  id          TEXT PRIMARY KEY,
  ewb_number  TEXT NOT NULL,
  trip_id     TEXT,
  event_type  TEXT NOT NULL,   -- PART_A_GENERATED|PART_B_ADDED|VEHICLE_UPDATED|EXTENDED|CANCELLED|EXPIRED
  payload     TEXT,
  created_by  TEXT,
  created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (ewb_number) REFERENCES eway_bills(ewb_number)
);
CREATE INDEX idx_ewb_events_ewb ON eway_bill_events(ewb_number);
```

**Spec 07 responsibility:** use the columns/table above (never define them), implement
the API/UI/business logic that writes them, and emit `eway_bill_events` on every
transition. If 00047 is not yet merged, gate EWB writes behind
`company_config('ewaybill_auto_generate')` and treat the schema as available at runtime.

### Company_config sequencing guard

`company_config` is created by Spec 02 @00042. THIS migration (00048/00049) seeds/uses it (§3.1 item 7, §3.2 seed rows). Two mitigations, choose one at implementation time: (a) build Spec 02 first (recommended), or (b) prepend `CREATE TABLE IF NOT EXISTS company_config (...)` guard with the canonical schema from Spec 02 @00042 so this migration never crashes `goose up` if run before 00042. Do not invent the schema here — reference Spec 02's canonical DDL.

---

## 4. UI

### 4.1 Invoice line-item editor + tax split + IRN/QR
- Page: `web/templates/invoices/edit_line_items.html` (new). Renders a table of
  `invoice_line_items` with columns: HSN/SAC (autocomplete from `hsn_sac_master`),
  description, unit, qty, rate, taxable value (computed), CGST%, SGST%, IGST%, amounts,
  line total.
- Tax split panel: shows `cgst/sgst/igst` totals; intra-state shows CGST+SGST,
  inter-state shows IGST only (computed by §5.1).
- IRN/QR block: after `POST /einvoice/irn`, show `irn`, `ack_no`, `ack_date` and render
  `signed_qr` as an `<img>` + downloadable PNG. Disable button if IRN exists.
- Partial: `web/templates/invoices/_tax_split.html`, `web/templates/invoices/_irn_qr.html`.
- JS asset: `web/static/js/invoice_line_items.js` (add row, recompute totals, call IRN API).

### 4.2 E-Way Bill page + trip_view card
- Page: `web/templates/ewaybill/index.html` (new) — list + generate Part-A form
  (pre-filled from trip + invoice `goods_value`).
- Trip view card: `web/templates/trips/_ewaybill_card.html` (new partial) embedded in
  `web/templates/trips/view.html`. Shows EWB number, status, valid-until countdown,
  QR, buttons: Attach Part-B, Extend, Cancel. Auto-generated EWB badge when
  `gen_mode='AUTO'`.
- JS asset: `web/static/js/ewaybill.js`.

### 4.3 FASTag dashboard + reconciliation view
- Dashboard: `web/templates/fastag/index.html` (new) — tag wallet table
  (`fastag_tags`), per-tag balance, low-balance warning.
- Transactions + reconcile: `web/templates/fastag/transactions.html` — table of
  `fastag_transactions` with `reconciled` flag and trip link; "Reconcile" button posts
  to `/fastag/reconcile` and shows matched/unmatched counts.
- JS asset: `web/static/js/fastag.js`.
- RBAC resources: pages gated by `integrations:einvoice`, `integrations:ewaybill`,
  `integrations:fastag`.

---

## 5. Business logic

### 5.1 CGST/SGST vs IGST determination (GSTIN state-code parse)
```
func stateCodeFromGSTIN(gstin string) string {
    // first two chars are the state code, e.g. "27AABCU..." -> "27"
    if len(gstin) < 2 { return "" }
    return strings.ToUpper(gstin[:2])
}

func isIntraState(supplierGSTIN, recipientGSTIN string) bool {
    return stateCodeFromGSTIN(supplierGSTIN) == stateCodeFromGSTIN(recipientGSTIN)
}
```
- **Intra-state** (`company_settings.state_code == customer/recipient state code`):
  split taxable value into CGST and SGST, each at `rate/2`.
- **Inter-state**: full `rate` applied as IGST; CGST=SGST=0.
- Supplier GSTIN = `company_settings.gst_number`; recipient = `customer.gst`.
- Example: taxable 1000, rate 18, intra → CGST 90 + SGST 90 (total tax 180);
  inter → IGST 180.

### 5.2 HSN/SAC rate lookup
- On line-item save, look up `hsn_sac_master.rate` (and `cess_rate`) by `code`.
- If `code` is empty/default, fall back to `company_settings.gst_rate` (existing flat
  behaviour) so legacy invoices still compute.
- Tax amounts per line:
  ```
  cgst_amount = taxable * cgst_rate / 100      // intra
  sgst_amount = taxable * sgst_rate / 100      // intra
  igst_amount = taxable * igst_rate / 100      // inter
  line_total  = taxable + cgst_amount + sgst_amount + igst_amount (+ cess)
  ```
- Invoice-level `cgst/sgst/igst` = SUM of line amounts.

### 5.3 IRN generation (deterministic)
```
func computeIRN(inv InvoiceView) string {
    canonical := struct {
        SupplierGSTIN string
        RecipientGSTIN string
        InvoiceNo string
        InvoiceDate string
        TotalValue float64
        ItemHash string   // sha256 over sorted line items
    }{...}
    b, _ := json.Marshal(canonical)
    sum := sha256.Sum256(b)
    return hex.EncodeToString(sum[:])[:64]   // NIC IRN is 64 hex chars
}
```
- MOCK adapter returns this deterministic hash; real NIC adapter signs and returns the
  government IRN/ack — both must be deterministic for the same canonical input so tests
  (§7) are stable. `ack_no`, `ack_date`, `signed_qr` are produced by the adapter.

### 5.4 E-Way Bill auto-generate on trip confirm
- Trigger: trip status transitions to `confirmed` (subscriber on trip-confirmed event).
- Read linked invoice `goods_value` (`invoices.total` or a dedicated `goods_value`
  field from booking cargo). If `goods_value > 50000` AND
  `company_config('ewaybill_auto_generate')='true'`:
  - Build Part-A request from trip + invoice + `company_settings.state_code`.
  - Call EWB adapter `GeneratePartA`, persist to `eway_bills`
    (`gen_mode='AUTO'`, `status='part_a'`), write `eway_bill_events`.
- **Tenant safety:** derive `tenant_id` from `auth.ContextUser`, never from client body.

### 5.5 EWB expiry monitor + auto-cancel
- Background job (reuse existing cron/outbox): scan `eway_bills` where `status IN
  ('active','part_a')` and `valid_until < now()`.
  - If trip still in-transit → trigger Extend (adapter `Extend`) if within allowed
    extensions (`extension_count < 1` per NIC rule for most cases); on success update
    `valid_until`, increment `extension_count`, emit `EXTENDED`.
  - If trip completed/cancelled → Cancel (adapter `Cancel`) with reason, set
    `status='expired'`/`'cancelled'`, emit `EXPIRED`/`CANCELLED`.
- Emits an alert (Spec 05 pipeline) when expiry < 4h.

### 5.6 FASTag reconciliation (greedy match + auto-kharcha)
```
func Reconcile(ctx, vehicleNumber, from, to):
    providerTxns = adapter.ListTransactions(vehicleNumber, 0)  // full pull
    persisted    = repo.ListUnreconciled(vehicleNumber, from, to)
    trips        = repo.ActiveTripsForVehicle(vehicleNumber, from, to)

    for txn in persisted:                      // greedy: nearest trip by time window
        best = nil; bestΔ = ∞
        for trip in trips:
            if trip.covers(txn.txn_timestamp) and !trip.matched:
                Δ = |txn.txn_timestamp - trip.start|
                if Δ < bestΔ: best, bestΔ = trip, Δ
        if best != nil:
            txn.trip_id = best.ID
            txn.reconciled = true
            kharcha = createDriverExpense(trip.ID, type='toll', amount=txn.amount,
                                          description="FASTag "+txn.plaza_name)
            txn.kharcha_id = kharcha.ID
            repo.Save(txn)
    return counts{pulled, matched, unmatched, kharcha_created}
```
- Gated by `company_config('fastag_auto_kharcha')='true'` for auto `kharcha` creation.
- `DeductToll` (§2.12) also creates a `driver_expense(toll)` immediately when called
  from a GPS toll-plaza event and `trip_id` is supplied.
- **TDS note:** toll/kharcha are expense entries; TDS withholding on vendor payments is
   handled by the driver settlement engine (00051, Spec 08 settlement) — not here.

---

## 6. Config / env

All integrations use the existing `integration.Config` loader
(`internal/integration/config.go`). Add a `UseMock bool` to each sub-config and a
`*_USE_MOCK` env var. Real provider calls only fire when `Enabled=true` AND
`UseMock=false`.

| Var | Default | Purpose | Package reading |
|-----|---------|---------|-----------------|
| `INTEGRATION_GSTN_ENABLED` | `false` | Master on/off for GSTN adapter | `config.go` → `gstn.Config.Enabled` |
| `INTEGRATION_GSTN_USE_MOCK` | `true` | Use in-memory MOCK adapter (no creds) | `config.go` → `gstn.Config.UseMock` |
| `INTEGRATION_GSTN_ENDPOINT` | `https://api.gstn.org` | NIC/GSP endpoint | `gstn.Client` (real) |
| `INTEGRATION_GSTN_USERNAME` | `` | NIC username (placeholder) | `gstn.Config` |
| `INTEGRATION_GSTN_PASSWORD` | `` | NIC password (placeholder) | `gstn.Config` |
| `INTEGRATION_GSTN_CLIENT_ID` | `` | GSP client id (placeholder) | `gstn.Config` |
| `INTEGRATION_GSTN_CLIENT_SECRET` | `` | GSP client secret (placeholder) | `gstn.Config` |
| `INTEGRATION_EWAYBILL_ENABLED` | `false` | Master on/off EWB adapter | `config.go` → `ewaybill.Config.Enabled` |
| `INTEGRATION_EWAYBILL_USE_MOCK` | `true` | Use MOCK EWB adapter | `ewaybill.Config.UseMock` |
| `INTEGRATION_EWAYBILL_ENDPOINT` | `https://ewaybill.nic.in/api` | NIC EWB endpoint | `ewaybill.Client` (real) |
| `INTEGRATION_EWAYBILL_USERNAME/PASSWORD/APP_KEY` | `` | NIC EWB creds (placeholder) | `ewaybill.Config` |
| `INTEGRATION_FASTAG_ENABLED` | `false` | Master on/off FASTag adapter | `config.go` → `fastag.Config.Enabled` |
| `INTEGRATION_FASTAG_USE_MOCK` | `true` | Use MOCK FASTag adapter (returns seeded `fastag_tags` balance) | `fastag.Config.UseMock` |
| `INTEGRATION_FASTAG_ENDPOINT` | `https://api.fastag.org` | Aggregator endpoint | `fastag.Client` (real) |
| `INTEGRATION_FASTAG_MERCHANT_ID` | `` | Parked in `company_config('fastag_merchant_id')` + env | `fastag.Config.MerchantID` |
| `INTEGRATION_FASTAG_API_KEY` | `` | Aggregator API key (placeholder) | `fastag.Config.APIKey` |

Adapter wiring in `config.go`:
```go
GSTN: gstn.Config{
    Endpoint: os.Getenv("INTEGRATION_GSTN_ENDPOINT"),
    APIKey:   os.Getenv("INTEGRATION_GSTN_API_KEY"),
    Enabled:  parseBool(os.Getenv("INTEGRATION_GSTN_ENABLED")),
    UseMock:  parseBoolDefaultTrue(os.Getenv("INTEGRATION_GSTN_USE_MOCK")),
    Username: os.Getenv("INTEGRATION_GSTN_USERNAME"),
    Password: os.Getenv("INTEGRATION_GSTN_PASSWORD"),
},
```
`NewClient` selects adapter:
```go
func NewClient(cfg Config) Client {
    if cfg.UseMock || !cfg.Enabled {
        return &mockClient{cfg: cfg}   // deterministic fake, no network
    }
    return &nicClient{cfg: cfg}        // real HTTP calls (only when Enabled && !UseMock)
}
```
Same pattern for `ewaybill` and `fastag`. The MOCK must satisfy the full interface
(IRN generate, Part-A/B, extend, reconcile) so code/tests need zero credentials.

### Integration Decision & Tradeoffs (GSTN / EWB / FASTag)

**Decision: adapter interface (`Client`) with a config-flagged in-memory MOCK as the
default; real NIC/GSP/aggregator adapter is a thin extension behind
`Enabled && !UseMock`.** This mirrors the existing `stubClient` pattern already in
`gstn/client.go`, `ewaybill/client.go`, `fastag/client.go`.

- **Adapter vs direct:** Adapter. Keeps HTTP/SDK/vendor specifics out of the
  application core; lets unit/integration tests run with no credentials and lets us
  swap vendors (ClearTax, Zoho, Karza, LocoNav) without touching call sites.
- **Mock strategy:** Deterministic in-memory MOCK. IRN is a SHA-256 of canonical
  invoice JSON (§5.3) so MOCK and real adapter return identical values for identical
  input — required for stable tests and for the `api_surface_test` rewrite (§7.5).
  MOCK for FASTag must read from seeded `fastag_tags` (not the hardcoded `2475.50`).
- **Vendor lock-in:** Real NIC EWB/IRN and GSP APIs are India-specific and not
  portable; isolate behind the interface and parameterize via `Config.Provider` /
  `Config.Endpoint` so a different GSP can be plugged in. FASTag aggregators vary
  (NHAI, banks, Park+, Paytm) — same interface, different `Client` impl.
- **Compliance risk (High):** GST e-invoice IRN and e-way bills are statutory.
  Shipping MOCK as default is fine for dev/test but production MUST gate real adapter
  behind `Enabled=true && UseMock=false` with validated credentials; a MOCK IRN in
  production would be a legal/financial compliance violation. Add startup guard that
  refuses to serve e-invoice/EWB endpoints when `Enabled=false` in production.
- **Cost:** MOCK = near-zero infra cost; real adapters imply GSP/aggregator
  subscription fees (per-API-call or monthly), credential management, and NIC rate
  limits. Budget a GSP retainer for IRN/EWB; FASTag aggregator per-transaction fee.

---

## 7. Tests

### 7.1 Tax math (intra / inter state)
- `internal/invoice/application/generate_invoice_test.go`:
  - Intra-state (`company state 27`, `customer 27...`): taxable 1000, rate 18 →
    CGST 90, SGST 90, IGST 0, total 1180.
  - Inter-state (`company 27`, `customer 07...`): taxable 1000, rate 18 →
    IGST 180, CGST 0, SGST 0, total 1180.
  - Round to paisa (2dp) with `math.Round(x*100)/100`; assert sum equality to 0.01.

### 7.2 IRN deterministic
- `internal/integration/gstn/client_test.go`: same canonical invoice → identical IRN
  across two `GenerateIRN` calls (MOCK). Different line items / total → different IRN.

### 7.3 EWB persist + link
- `internal/integration/ewaybill/handler_test.go` (or app test): `GeneratePartA`
  returns 201, `eway_bills` row exists with `trip_id`, `status='part_a'`, and an
  `eway_bill_events` row of `PART_A_GENERATED` exists. `GetByTrip` returns it.
- `Part-B` sets `vehicle_number`, `status='active'`. `Extend` increments
  `extension_count`. `Cancel` sets `status='cancelled'` + `cancelled_at` + event.

### 7.4 FASTag reconcile
- Seed `fastag_tags` (balance 1000) + 3 `fastag_transactions` overlapping one trip's
  time window. Call `Reconcile` → `matched=3`, each `trip_id` set, `reconciled=1`,
  three `driver_expense(toll)` rows created with `kharcha_id` linked.
- `GetBalance` now reads from `fastag_tags` (NOT the hardcoded 2475.50).

### 7.5 Rewrite `test/api_surface_test.go` fake pin
Replace the hardcoded assertion at line 130:
```go
// OLD (remove):
//   assert.Equal(t, 2475.50, res["balance"])
// NEW: balance now comes from fastag_tags; seed a tag with known balance.
cfg := integration.Config{
    EWayBill: ewaybill.Config{Enabled: true, UseMock: true},
    FASTag:   fastag.Config{Enabled: true, UseMock: true, MerchantID: "MOCK"},
}
// after seeding a fastag_tags row with balance 1234.50:
assert.Equal(t, 1234.50, res["balance"])
```
This requires `fastag.Config` to gain `UseMock`/`MerchantID` and a repository-backed
MOCK (see §3.2, §6). Coverage gate: `go test ./...` must pass before merge;
`api_surface_test` must not pin a magic constant.

---

## 8. Future / GPS-provider

- **GSTN ITC from telemetry:** feed FASTag + fuel telemetry into GSTR-2A/2B ITC
  matching; auto-flag mismatches. Define `TelematicsProvider` interface (GPS/VLT
  primary, LocoNav/WheelsEye/MapMyIndia pluggable) — same adapter pattern as here.
- **GPS-driven auto-EWB extension:** when GPS shows vehicle still en-route near
  `valid_upto`, trigger `Extend` automatically (extends §5.5 with live position).
- **Provider adapter pattern:** each integration already behind `Client` interface with
  `mockClient`/`nicClient` (or aggregator) selection in `NewClient`. Add a
  `Provider` field to `Config` to support multiple GSPs/aggregators
  (ClearTax, Zoho, Karza) behind the same interface.

---

## 9. Edge cases

- **No GSTIN on customer:** skip IRN; invoice still issued with `irn IS NULL`; UI hides
  QR. Tax computed using company `gst_rate` fallback (no split possible → treat as
  inter-state IGST or exempt per config).
- **Intra vs inter mismatch:** if `company_settings.state_code` empty, default to
  supplier state parsed from `gst_number`; log warning.
- **IRN already exists:** `POST /einvoice/irn` returns 409; re-push allowed (§2.2).
- **EWB goods_value ≤ 50000:** auto-generate skipped; manual only.
- **EWB extend limit:** NIC allows limited extensions; block beyond `extension_count`
  cap, return 422 with reason.
- **EWB cancel after Part-B / movement:** allowed with valid reason; set expiry.
- **FASTag transaction without matching trip:** left `reconciled=0`, `trip_id=NULL`,
  surfaced in "unmatched" list; manual assign API (future).
- **FASTag negative/zero balance:** `GetBalance` warns; `DeductToll` may fail at
  provider — handle `status='FAILURE'`, no kharcha created.
- **Multi-tenant isolation:** every query filters by `tenant_id` from `auth.ContextUser`;
  never trust client `tenant_id`.
- **Currency:** INR only; remove hardcoded `$` in `invoice_pdf.go:52`.

---

## 10. Phased rollout (build order)

1. **00048 migration** + extend `CompanySettings`/`InvoiceAggregate` with
   `StateCode`, `Cgst/Sgst/Igst`, line items, IRN fields. Unit tests for tax split.
2. **GSTN adapter**: add `UseMock`, `GenerateIRN`/`PushEInvoice`, MOCK impl;
   wire `config.go`. IRN API (§2.1, §2.2) + PDF currency/company-name fix.
3. **00049 migration** + `fastag_tags`/`fastag_transactions` repos; FASTag MOCK backed
   by seeded tags; rewrite `api_surface_test`.
4. **EWB feature logic** (no schema here): Part-A/B/extend/cancel/get-by-trip handlers
   persisting to `eway_bills` + `eway_bill_events` (columns from 00047). Coordinate
   with Spec 05 for 00047 merge.
5. **Business automation**: EWB auto-generate on trip confirm; expiry monitor +
   auto-cancel/extend; FASTag reconcile + auto-kharcha.
6. **UI**: line-item editor, tax split, IRN/QR; EWB trip_view card; FASTag dashboard.
7. **Tests + coverage gate**; manual smoke with `USE_MOCK=true`.

---

## 11. Open items / VERIFY

- **00047 status:** confirm THIS SPEC (07) has merged the `eway_bills` ALTER + `eway_bill_events`
  before EWB persistence code ships; otherwise EWB writes fail at runtime.
- **`company_config` vs `company_settings`:** confirm which table holds feature flags
  (00042 says `company_config`). This spec seeds both-style rows idempotently; verify
  the table name actually used by the config reader.
- **`goods_value` source:** confirm whether invoices carry cargo value or trips/bookings
  do; EWB auto-generate depends on it.
- **EWB extension cap:** confirm NIC business rule (typically 1 extension for road
  transport) and encode as config.
- **PDF currency:** confirm INR symbol/format expected by finance; replace `$`.
- **IRN canonical schema:** confirm exact fields NIC hashes (supplier/recipient GSTIN,
  invoice no/date, total, item hash) — adjust §5.3 if government spec differs.
- **RBAC seed ownership:** 00048 (Spec 07) seeds the `integrations:*` RBAC rows for
   GST/einvoice/ewaybill/fastag; 00046 (Spec 05) seeds alerting/authorization RBAC.
   Coordinate to avoid duplicate inserts (use `ON CONFLICT DO NOTHING`).

---

## 12. File list

### Create
- `db/migrations/00048_gst_einvoice.sql` — line items, sequences, CGST/SGST/IGST, hsn_sac_master, company state code, RBAC + company_config seeds.
- `db/migrations/00049_fastag.sql` — fastag_tags + fastag_transactions + seeds.
- `internal/integration/gstn/einvoice.go` — `EInvoiceClient` interface, `GenerateIRN`/`PushEInvoice`, request/response types, MOCK + NIC adapters.
- `internal/integration/ewaybill/lifecycle.go` — Part-A/B/extend/cancel/get-by-trip on top of existing `Client`; persistence + events helpers.
- `internal/integration/fastag/repo.go` — repository-backed MOCK reading `fastag_tags`/`fastag_transactions`.
- `internal/invoice/domain/aggregate/invoice_line_item.go` — line-item value object + tax-split helpers.
- `internal/invoice/application/generate_einvoice.go` — IRN generation use case + tax computation.
- `internal/fastag/application/reconcile.go` — reconciliation + auto-kharcha use case.
- `internal/ewaybill/application/autogenerate.go` — trip-confirm subscriber + expiry monitor.
- `internal/invoice/domain/aggregate/invoice_aggregate_ext.go` — extend `InvoiceAggregate` with Cgst/Sgst/Igst/IRN fields (or edit `invoice_aggregate.go`).
- `web/templates/invoices/edit_line_items.html`, `_tax_split.html`, `_irn_qr.html`.
- `web/templates/ewaybill/index.html`, `web/templates/trips/_ewaybill_card.html`.
- `web/templates/fastag/index.html`, `web/templates/fastag/transactions.html`.
- `web/static/js/invoice_line_items.js`, `web/static/js/ewaybill.js`, `web/static/js/fastag.js`.
- `internal/integration/gstn/client_test.go`, `internal/invoice/application/generate_invoice_test.go`, `internal/integration/ewaybill/lifecycle_test.go`, `internal/fastag/application/reconcile_test.go`.

### Modify
- `internal/integration/config.go` — add `UseMock`, `Username/Password/ClientID/ClientSecret/MerchantID` to each sub-`Config`; add `parseBoolDefaultTrue`.
- `internal/integration/gstn/client.go` — keep `Client`; add `EInvoiceClient` or extend `Client` with IRN methods + adapter selection.
- `internal/integration/ewaybill/client.go` — extend `Client` with `GeneratePartA`, `AttachPartB`, `Extend`, `GetByNumber`, `GetByTrip`; MOCK satisfies all.
- `internal/integration/fastag/client.go` — extend `Client` with `Reconcile`; MOCK reads seeded tags.
- `internal/integration/handler.go` — add GST e-invoice routes (§2.1, §2.2), EWB Part-A/B/extend/get-by-trip (§2.3–2.8), FASTag reconcile (§2.11); update permission names.
- `internal/invoice/application/generate_invoice.go` — replace flat tax (line 138) with line-item + CGST/SGST/IGST logic (§5.1, §5.2).
- `internal/invoice/domain/aggregate/invoice_aggregate.go` — add `Cgst, Sgst, Igst, IRN, IRNAckNo, IRNAckDate, SignedQR, EwbNumber` fields + rehydrate/update.
- `internal/domain/company/entity.go` — add `StateCode string`.
- `internal/domain/customer/entity.go` — add `StateCode` (parsed from GST) helper.
- `internal/pdf/invoice_pdf.go` — replace `"Amount ($)"` (line 52) with INR; accept currency from `CompanySettings`.
- `internal/handlers/invoices.go` — replace hardcoded `"Apex Transport Ltd"` (line 167) with `CompanySettings.CompanyName`.
- `test/api_surface_test.go` — rewrite the `2475.50` pin (line 130) to read from seeded `fastag_tags` (§7.5).
