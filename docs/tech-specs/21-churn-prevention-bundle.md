# Churn Prevention Bundle — Compliance, Driver, Shipper — Implementation Spec v1
Status: ready
Depends-on: 00-migration-ownership-index.md (migrations **00073** + **00074** + **00075** reserved), 07-gst-ewaybill-fastag.md, 13-mobile-app.md
Migration owner: db/migrations/00073_churn_compliance_shipper.sql (also 00074, 00075)

This spec closes the top 5 churn drivers: illegal docs (GSTN mock), driver drop-off (offline/vernacular/ePOD), enterprise invisibility (shipper portal). It makes three subsystems production-real while keeping mock-gated adapters for CI.

---
## 0. Verified ground truth (file:line facts)

### 0.1 GSTN/EWB/FASTag are mocks
- `internal/integration/gstn/client.go:66` `type stubClient struct{}` returns `Stub Legal Entity Pvt Ltd` `gstn/client.go:82 ValidateGSTIN` → `fmt.Errorf("gstn disabled")`, `client.go:99 FetchGSTR1` → `125000/42 invoices` hardcode
- `internal/integration/gstn/einvoice.go:117` `NewMockEInvoiceClient` SHA256 `mock_qr_` deterministic, not NIC signed
- `internal/handlers/invoices.go:672` `gstn.NewMockEInvoiceClient(Config{Enabled:true})` hardcodes mock, ignores `config.UseMock`
- `internal/integration/config.go:33` `UseMock: parseBoolDefault(os.Getenv("INTEGRATION_GSTN_USE_MOCK"),true)` default true → prod ships mock
- `internal/integration/ewaybill/client.go:73` `type stubClient` 7 methods → `EWB+uuid[:12]` `stubqrcode/mockqrcode` `now+24h`, zero `net/http` calls
- `internal/integration/fastag/client.go:19 UseMock bool` never set (`config.go:39` no env), `client.go:141` fallback `2475.50 INR`, `client.go:233` fabricates `PLZ001 85+5*i`, `client.go:250` static `{pulled:5 matched:5}` no NETC pull
- DB migrations exist: `00047_ewaybill_lifecycle.sql`, `00048_gst_einvoice.sql`, `00049_fastag.sql` → persistence ready, adapter not

### 0.2 Dispatch blockers unwired
- `internal/service/compliance_service.go:254` `EnforceDispatchCompliance()` checks `License/RC/PUC/Permit expiry < now` + `IsExempt` but `grep -rn EnforceDispatchCompliance handlers/trip =0` callers (only `service_test.go:42`)
- `internal/handlers/trips.go:328` `AssignDriver`, `trips.go:361` `AssignVehicle` no compliance gate → expired RC still assignable via API (UI `vehicle_view.html:172` banner only)

### 0.3 Driver offline/vernacular gaps
- `mobile/src/services/offlineQueue.ts:78` `enqueuePOD(tripId, {consignee_name,notes,photo_uri,lat,lng})` drops `consignee_phone` (captured `DeliveryVerificationScreen.tsx:115`) and `vehicle_id/trip_id/accuracy/speed`
- `mobile/src/services/storage.ts:44` only `queued_pods` + `queued_gps`, no `trips/routes/ewaybill` cache; `App.tsx:322` `saveTrips` but no `getOfflineTrip` fallback
- `mobile/src/services/syncEngine.ts:166` `break` on first flush failure → queue stuck
- `grep -r i18n mobile =0`, `mobile/src` no `locales/`, all `*.tsx` hardcoded English; `FUTURE_SCOPE Layer1` requires Hindi/Tamil/Telugu/Kannada/Marathi/Gujarati
- `mobile/src/components` no `ExpenseScreen.tsx`; web `templates/kharcha_dashboard.html:57` only web `fuel_litres` → driver cannot log advance on road

### 0.4 ePOD gaps
- `mobile/src/components/DeliveryVerificationScreen.tsx:43` single photo `compress 0.7`, `115` only `consignee_name/phone/notes`, no `Aadhaar masking`, `short/damage/refusal` (`FUTURE_SCOPE §1`), no signature pad
- `internal/handlers/kharcha.go:186` `ParseMultipartForm 10<<20` web-only, mobile phone field lost offline

### 0.5 Shipper portal missing
- `handlers/customers.go:1` admin CRUD only, `grep -rn "customer.*portal" =0`; `FUTURE_SCOPE Layer0` Shipper Portal (booking/invoice/tracking) has no `customer` role routes
- `handlers/bookings.go:50` `bookings:read` only dispatcher/admin, no customer-scoped `ListMyBookings`

### 0.6 Payout & settlement manual
- `internal/service/driver_settlement_service.go:600` `MarkPaid` manual `UPDATE status='paid', payment_ref='TXN_ICICI_98231'`, no `RazorpayX/ICICI/HDFC` API; `grep HPCL|IOCL|BPCL =0` fuel cards absent
- `service/driver_settlement_service.go:380` `TDS 194C` calc exists but `settlement_view.html:138` manual `payment_ref` text input

### 0.7 Verification log
Every claim read 2026-08-21. No spec numbers reused (00066-68 free per `00-migration-ownership-index.md:66-68`).

---
## 1. Overview / goal
Make three churn-critical flows production-real:

1. **Compliance real adapters** — GSTN NIC IRN + EWB Part A/B + FASTag NETC behind `UseMock` flag; mock stays for CI. Enforce hard dispatch blockers via `EnforceDispatchCompliance` on `AssignDriver/Vehicle`.
2. **Driver mobile offline-first + vernacular + ePOD + expense** — add missing `consignee_phone` + `trip/route` cache to `offlineQueue`, 6-language `i18n`, signature + short/damage/refusal, expense with receipt photo.
3. **Shipper portal** — new `customer` role, self-serve `GET /customer/bookings`, `GET /customer/invoices`, `GET /customer/tracking/:trip`.

**Non-goals:** Return load/STO/Vahak (spec 19), ESG/carbon, predictive ETA, Tally real sync (spec 20), full route optimization.

---
## 2. API contract
All under `r.Group(RequireAuth)` tenant from `auth.ContextUser`. RBAC via `middleware.ResourcePermission` or `RoleRequired`.

### 2.1 Compliance adapters (real)
No new public routes — behavior change. Existing `POST /invoices/{id}/generate-irn` now calls real NIC when `INTEGRATION_GSTN_USE_MOCK=false` and `GSTN_API_KEY` set; else mock. Same for `POST /ewaybill/generate` + `POST /fastag/reconcile`. Error `502` on NIC failure with `{"error":"gstn_unavailable","detail":"..."}`.

### 2.2 Dispatch blocker enforcement
- `POST /trips/{id}/assign-driver` Body `{"driver_id":"d1","override":false,"reason":""}` → `403 {"error":"dispatch_blocked","blocked_by":"license_expiry","expiry":"2026-08-19"}` if `EnforceDispatchCompliance` fails and `override=false`. `override=true` requires `users:update` + `reason` ≥10 chars → `200` + audit log.
- `POST /trips/{id}/assign-vehicle` same for RC/PUC/Permit.

### 2.3 Shipper portal (NEW)
- `GET /customer/bookings` Auth `bookings:read` (customer) Query `?q=&status=&limit=20&page=1` → `{"bookings":[...],"total":12}` scoped to `customer_id = user.customer_id`.
- `GET /customer/invoices` Auth `invoices:read` (customer) → same, scoped.
- `GET /customer/tracking/{trip_id}` Auth `trips:read` (customer) → `{"trip_number":"TRP-001","status":"in_transit","eta_min":"2026-08-21T10:00:00Z","eta_max":"...","vehicle_label":"MH01AB1234","last_seen":"..."}` (reuses `ShareData` logic but auth-gated).
- `POST /customer/feedback` Body `{"trip_id":"...","rating":5,"comment":""}` → `201`.

### 2.4 Driver mobile sync (existing endpoints hardened)
- `POST /trips/{id}/deliver-pod` now accepts `consignee_phone` (persisted), `quantity_short`, `damage_qty`, `refusal_reason`, `signature_dataurl` (base64). Offline queue flush `POST /offline/pods/flush` (NEW) Auth `trips:update` Body `{"pods":[{...}]}` → `207 Multi-Status` per pod result.

### 2.5 Expense (NEW mobile)
- `POST /kharcha/expense` Auth `kharcha:create` (driver) multipart `trip_id, type(fuel|toll|rto|tyre), amount, receipt_photo, notes, latitude, longitude` → `201 {"id":"exp_..."}`. Enqueued offline when `offline`.

Error codes: `400` validation, `403` blocked/permission, `404` not found, `409` duplicate invoice IRN, `502` NIC/NETC unavailable.

---
## 3. DB contract — migrations 00066, 00067, 00068

```sql
-- +goose Up
-- 00073_churn_compliance_shipper.sql
CREATE TABLE IF NOT EXISTS customer_users (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(customer_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_customer_users_user ON customer_users(user_id);
CREATE INDEX IF NOT EXISTS idx_customer_users_customer ON customer_users(customer_id);

-- Enforce dispatch blocker audit (override log)
CREATE TABLE IF NOT EXISTS dispatch_overrides (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '1',
    trip_id TEXT NOT NULL,
    vehicle_id TEXT,
    driver_id TEXT,
    blocked_by TEXT NOT NULL,
    reason TEXT NOT NULL,
    overridden_by TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
-- shipper feedback
CREATE TABLE IF NOT EXISTS trip_feedback (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '1',
    trip_id TEXT NOT NULL REFERENCES trips(id),
    customer_id TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
-- POD hardening columns (if not exists from 00055)
ALTER TABLE trips ADD COLUMN pod_signature_data TEXT;
ALTER TABLE trips ADD COLUMN pod_consignee_phone TEXT;
ALTER TABLE trips ADD COLUMN pod_quantity_short REAL DEFAULT 0;
ALTER TABLE trips ADD COLUMN pod_damage_qty REAL DEFAULT 0;
ALTER TABLE trips ADD COLUMN pod_refusal_reason TEXT;

INSERT OR IGNORE INTO permissions (resource, action, description) VALUES
    ('customer_portal','read','Shipper view own bookings/invoices/tracking'),
    ('customer_portal','write','Shipper feedback');
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.name='customer' AND p.resource='customer_portal';

-- +goose Down
DROP TABLE IF EXISTS trip_feedback;
DROP TABLE IF EXISTS dispatch_overrides;
DROP TABLE IF EXISTS customer_users;
ALTER TABLE trips DROP COLUMN pod_signature_data;
ALTER TABLE trips DROP COLUMN pod_consignee_phone;
ALTER TABLE trips DROP COLUMN pod_quantity_short;
ALTER TABLE trips DROP COLUMN pod_damage_qty;
ALTER TABLE trips DROP COLUMN pod_refusal_reason;
```

```sql
-- +goose Up
-- 00074_churn_driver_offline.sql
-- offline: no DB change (mobile SQLite). Server side: expense table already exists via kharcha.
-- Add expense type check widening + offline sync log
CREATE TABLE IF NOT EXISTS offline_sync_log (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '1',
    user_id TEXT NOT NULL,
    kind TEXT NOT NULL, -- pod | expense | gps
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
-- +goose Down
DROP TABLE IF EXISTS offline_sync_log;
```

```sql
-- +goose Up
-- 00075_churn_expense_vernacular.sql
-- No DB — mobile bundle + server kharcha already has receipt. This migration seeds i18n key table.
CREATE TABLE IF NOT EXISTS i18n_keys (
    key TEXT PRIMARY KEY,
    en TEXT NOT NULL,
    hi TEXT NOT NULL DEFAULT '',
    ta TEXT NOT NULL DEFAULT '',
    te TEXT NOT NULL DEFAULT '',
    kn TEXT NOT NULL DEFAULT '',
    mr TEXT NOT NULL DEFAULT '',
    gu TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO i18n_keys (key,en) VALUES ('trip.accept','Accept'),('trip.reject','Reject');
-- +goose Down
DROP TABLE IF EXISTS i18n_keys;
```

Company_config sequencing: do not touch `company_config` (owned 00042).

---
## 4. UI
- `templates/layout.html:98` add `{{if can .User "customer_portal" "read"}} <a href="/customer/bookings">My Shipments</a> {{end}}`
- `templates/customer_bookings.html` NEW, `customer_invoices.html` NEW, `customer_tracking.html` NEW — reuse `booking_list_table.html` partial scoped.
- `templates/vehicle_view.html:172` banner stays, `trips_view.html` add `override_reason` modal when `dispatch_blocked` (JS fetch 403 → show textarea)
- `templates/trip_feedback.html` NEW star rating.
- Mobile: `mobile/src/locales/{hi,ta,te,kn,mr,gu}.json` (6 files) + `i18n.ts` `react-i18next`; `DeliveryVerificationScreen.tsx` add `SignaturePad` ( `react-native-signature-canvas` ), `short/damage/refusal` inputs, `consignee_phone` persist; `ExpenseScreen.tsx` NEW (type picker, amount, receipt `expo-image-picker`, notes, gps); `storage.ts` add `offline_expenses` table; `offlineQueue.ts` add `consignee_phone, vehicle_id, trip_id, accuracy` to `QueuedPOD` + new `QueuedExpense` + `pendingExpenses()` + `enqueueExpense()`; `syncEngine.ts` fix `break` → `continue` on partial failure + flush pods/expenses/gps in batch.
- JS: `static/js/customer_portal.js` polling `GET /customer/tracking/{id}` every 30s.

---
## 5. Business logic
**Compliance adapter factory** `internal/integration/gstn/factory.go:NewClient(cfg)` returns `realHttpClient` if `!cfg.UseMock && cfg.APIKey!=""` else `stubClient`. Real client does `POST https://api.gstn.gov.in/einvoice/generate` with `X-API-Key`, maps `InvoiceView → NIC JSON` per `spec 07 §3.1`, parses `IRN + SignedQR`. Same pattern for `ewaybill/factory.go` (`POST https://api.ewaybillgst.gov.in/...`) and `fastag/client.go` (`GET https://netc.npci.org.in/...` with `HMAC`).

**Hard blocker** pseudo:
```go
func EnforceWithOverride(ctx context.Context, tripID, driverID, vehicleID string, override bool, reason string, userRole string) error {
  if err:= compliance.EnforceDispatchCompliance(ctx, driverID, vehicleID); err!=nil {
    if !override { return &BlockedError{err} }
    if userRole!="admin" && !can(user,"users","update") { return ErrForbidden }
    if len(reason)<10 { return ErrReason }
    _ = repo.InsertDispatchOverride(ctx, tripID, blockedBy, reason, userID) // dispatch_overrides
    _ = audit.Log(ctx, "dispatch_override", reason)
    return nil
  }
  return nil
}
```
Called at start of `TripHandlers.AssignDriver` `trips.go:328` and `AssignVehicle` `trips.go:361` before `assignDriverUC.Execute`.

**Offline flush** `handlers/offline.go:HandleFlush` loops `pods` → `for _,p:=range pods { if err:= kharcha.DeliverWithPOD(...); err!=nil { results[i]=status:failed } else { results[i]=ok } }` → never `break`, returns `207` with per-item status.

**Shipper scoping** `ListMyBookings` → `SELECT * FROM bookings WHERE tenant_id=? AND customer_id IN (SELECT customer_id FROM customer_users WHERE user_id=?)`.

---
## 6. Config / env

| Var | Default | Purpose | Package |
|-----|---------|---------|---------|
| `INTEGRATION_GSTN_USE_MOCK` | `true` | false → real NIC IRN | `integration/gstn` |
| `GSTN_API_KEY` / `GSTN_API_SECRET` | `""` | NIC HMAC | `gstn` |
| `INTEGRATION_EWB_USE_MOCK` | `true` | false → real EWB | `ewaybill` |
| `EWB_API_KEY` | `""` | EWB NIC | `ewaybill` |
| `INTEGRATION_FASTAG_USE_MOCK` | `true` | false → NETC | `fastag` (fix: add env load `config.go:39`) |
| `FASTAG_API_KEY` | `""` | NETC | `fastag` |
| `CUSTOMER_PORTAL_ENABLED` | `true` | gate portal routes | `handlers` |
| `OFFLINE_FLUSH_BATCH` | `20` | pods per flush | `offline` |

---
## 7. Tests
Coverage gate new code ≥80%.

- `internal/integration/gstn/factory_test.go`: `UseMock=true` → stub IRN `mock_qr_` prefix; `UseMock=false + APIKey` → real client type (httptest server) returns `NIC_IRN_`; `ApiKey=""` fallback to mock.
- `handlers/trips_dispatch_block_test.go`: expired `vehicles.maintenance_due='2020-01-01'` → `POST /trips/{id}/assign-vehicle` without override → `403 blocked_by`; with `override=true, reason="Urgent ..."` + admin → `200` and `dispatch_overrides` row exists.
- `handlers/customer_portal_test.go`: seed `customer_users(user-1,cust-1)` + `bookings` for `cust-1`/`cust-2`; `GET /customer/bookings` as `user-1` → only `cust-1` bookings; `user-2` (other customer) cannot see `cust-1` (tenant + customer scoping).
- `mobile/__tests__/offlineQueue.test.ts` extend: `enqueuePOD` with `consignee_phone` persists and `pendingPODs` returns it; `syncEngine.test.ts` `break` → `continue` (2 pods, 1 fails, 2nd still flushed).
- `mobile/__tests__/DeliveryVerification.test.tsx` phone fallback `OfflineQueue.enqueuePOD` retains phone.
- `internal/handlers/template_render_test.go` loop includes 3 new customer templates → `renderPage` 200.

Pass-before-merge: `go build ./...`, `go vet ./...`, `go test ./internal/...`, `goose up/down` on 00066-68 reversible.

---
## 8. Future / GPS-provider
- After mock toggle, integrate real `TelematicsProvider` for NETC FASTag auto-recharge alert (spec 20 fuel cards).
- STO/load board (spec 19) reuses `customer_users` pattern for `sto_users`.
- ESG `pnl_daily` already has `fuel_cost`; future add `co2 = fuel_liters * 2.68` (diesel) per trip.

---
## 9. Edge cases
- NIC 502 → return `502 gstn_unavailable` + `Retry-After 60`, keep mock IRN not generated; UI shows `Generate IRN failed, retry` not `pending`.
- Driver offline 3 days → `queued_pods` `created_at` >7d auto-expire `DELETE WHERE created_at < datetime('now','-7 days')`.
- Empty `reason` override → `400 reason is required (≥10 chars)`.
- Customer with 2 `customer_id` → `customer_users` `user_id` maps to both → `IN (SELECT ...)` returns both customers' bookings (multi-customer shipper).
- Anonymous shipper portal → `RequireAuth` 302 to `/login`.
- Vernacular missing key → fallback `en`.

---
## 10. Phased rollout
1. Migrations 00066-68 + seed permissions (reversible).
2. Compliance factory + mock flag fix + blocker enforcement (`trips.go:328,361`).
3. Offline queue fix (`consignee_phone` + `continue` + `offline_expenses`) + `ExpenseScreen` + `i18n` 6 locales.
4. Shipper portal handlers + templates + `customer_users` wiring.
5. ePOD signature + short/damage/refusal columns.
6. Full `go test` + `goose down/up` + manual E2E (mock→real toggle).

---
## 11. Open items / VERIFY
- `GSTN_API_KEY` / `EWB_API_KEY` / `FASTAG_API_KEY` real creds and base URLs: VERIFY at implementation (spec says `VERIFY AT IMPLEMENTATION` — check NIC docs for HMAC algo).
- `Parivahan/Sarathi` DL/RC real API creds: VERIFY — fallback to local expiry check if creds absent (keep stub).
- `Trip POD` `pod_otp_hash` column exists via `00055` placeholder — VERIFY column name `pod_otp_hash` vs `pod_signature_data` mismatch; choose one.
- `customer` role `role_id`: VERIFY `roles` table id for `customer` (seed suggests `5=driver`, need `6=customer` or reuse `viewer`); check `00064_org_admin`.
- `INTEGRATION_FASTAG_USE_MOCK` env not loaded `config.go:39` — VERIFY to add `getEnvBool` line.
- `company_config` `company state code` required for CGST/SGST split: VERIFY `company_settings.state_code` exists via `00048`.

---
## 12. File list
CREATE:
- `db/migrations/00073_churn_compliance_shipper.sql`
- `db/migrations/00074_churn_driver_offline.sql`
- `db/migrations/00075_churn_expense_vernacular.sql`
- `internal/integration/gstn/factory.go` (real vs mock)
- `internal/integration/ewaybill/factory.go`
- `internal/integration/fastag/factory.go` + fix `internal/config/config.go:302` load `FASTAG_USE_MOCK`
- `internal/service/compliance_service.go` (add `BlockedBy` string return)
- `internal/handlers/offline.go` (flush endpoint)
- `internal/handlers/customer_portal.go` (ListMyBookings/Invoices/Tracking/Feedback)
- `templates/customer_bookings.html`, `customer_invoices.html`, `customer_tracking.html`, `trip_feedback.html`
- `mobile/src/locales/{hi,ta,te,kn,mr,gu}.json`, `mobile/src/i18n.ts`
- `mobile/src/components/ExpenseScreen.tsx`
- `mobile/src/services/storage.ts` (add `offline_expenses`)
MODIFY:
- `internal/handlers/trips.go:328,361` (call `EnforceWithOverride`)
- `internal/handlers/invoices.go:672` use factory not `NewMockEInvoiceClient`
- `internal/integration/config.go:33,39` correct `UseMock` defaults
- `internal/templates/layout.html:98` add `customer_portal` nav
- `mobile/src/services/offlineQueue.ts:78` add `consignee_phone` + `QueuedExpense`
- `mobile/src/services/syncEngine.ts:166` `break` → `continue`
- `mobile/src/components/DeliveryVerificationScreen.tsx:115` add signature + short/damage/refusal
- `docs/tech-specs/00-migration-ownership-index.md` reserve 00066-68
