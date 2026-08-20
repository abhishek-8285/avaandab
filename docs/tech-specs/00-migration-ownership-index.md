# Migration Ownership Index

Single source of truth for `db/migrations/` version numbers. Repo head is
`00039_experiments.sql` (TAKEN — never edit). Every new migration appends
`00040` and up. **This table is authoritative; spec §3 numbers MUST match it.**

## Rules
- ONE feature owns ONE migration number. Never reuse, never renumber an
  existing `db/migrations/0000x_*.sql`.
- `company_config` / `company_settings`: created ONCE. Spec 02 creates it at
  `00042`. Every other spec only seeds rows or adds columns — never a second
  `CREATE TABLE`.
- Every new migration has correct `-- +goose Up` / `-- +goose Down`.
- `tenant_id` is a free-form `TEXT` (no `tenants` table exists). Do NOT add
  `FOREIGN KEY (tenant_id) REFERENCES tenants(id)` — it will fail `goose up`.

## Canonical allocation (verified non-overlapping)

| # | Owner | Spec |
|---|-------|------|
| 00039 | experiments (TAKEN — do not touch) | existing |
| 00040 | telemetry devices / raw_events / provider_poll_state / quarantine | 01 |
| 00041 | telemetry_positions / vehicle_latest_position / snapshots enrichment | 01 |
| 00042 | geofence engine + **canonical `company_config` create** | 02 |
| 00043 | fuel audit + driver scorecard (seeds `company_config` only) | 03 |
| 00044 | live map + share links + maintenance | 04 |
| 00045 | alerting pipeline (alert_rules, alert_events, alert_routes, notification_prefs) | 05 |
| 00046 | compliance reporting + files | 05 |
| 00047 | e-way bill lifecycle (eway_bills, eway_bill_events) | 07 |
| 00048 | GST e-invoice (line items, invoice_sequences, CGST/SGST/IGST, hsn_sac_master, company state code) | 07 |
| 00049 | FASTag (tags + transactions) | 07 |
| 00050 | Accounting sync log + mapping + gl rules | 08 |
| 00051 | Driver settlement engine (INSERT fix + settlement_lines + TDS) | 08 |
| 00052 | Document vault (driver_documents, vehicle_documents + expiry cols) | 08 |
| 00053 | Event bus / outbox correction (status/attempts/last_error) | 09 |
| 00054 | Booking hardening (reverse_fare, status history, tenant FK) | 09 |
| 00055 | Trip POD hardening (pod_otp_hash, converter/aggregate fields) | 09 |
| 00056 | Auth hardening (api_tokens, sessions tenant, drivers.user_id, enc token) | 10 |
| 00057 | Payment Razorpay fields (order_id, payment_id, signature, event_id) | 11 |
| 00058 | PNL / ops / experiments / founder (pnl_snapshots, notification_log, error_reports, incidents, experiment_assignments, login_audit, founder channels) | 16 |
| 00059 | telemetry_alerts rebuild (widened CHECK, 13 types) | 05 |
| 00060 | experiments RBAC permissions & role assignments | 16 |
| 00061 | founder RBAC permissions & role assignments | 16 |
| 00062 | user theme preferences (users.theme_preference) | 12 |
| 00063 | RAG multi-tenant vectors + provider registry | 10 |
| 00064+ | future specs (predictive ETA, AIS-140/VLT, ERP) | reserved |

## Implementation rule
Pick the number from this table for your feature and update the spec's §3 to
match exactly. If two specs appear to need the same number, the FIRST spec in
the table wins and the other moves to the next free slot (update this index).
