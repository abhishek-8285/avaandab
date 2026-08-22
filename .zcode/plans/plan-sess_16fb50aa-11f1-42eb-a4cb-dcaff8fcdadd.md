# Milestone: Per-org feature flags with add-on monetization + closed-source licensing + E2E completion

## 1. Feature registry — per-org, monetization-ready

Every feature is a catalog entry, always registered, **switchable per org (tenant) at runtime**. No Go .so runtime plugins (fragile); the deliverable is exactly your ask: add features cleanly, enable/disable any feature for any org from your side, and price add-ons.

- **`internal/features/` (new package):**
  - Catalog: `{Key, Name, Description, Category, Tier, DefaultEnabled, EnvFlag}` where **Tier = "core" | "addon"**. Core = every org gets it. Add-on = premium, off unless granted — this is your pricing surface. (Price display metadata — e.g. `DisplayPriceINR` — lives in the catalog for UI; actual billing/invoicing for add-ons stays external/manual this milestone.)
  - `Registry` (DB-backed, 60s per-org cache): `Enabled(ctx, key)` · `Set(ctx, key, bool, updatedBy)` · `Snapshot(ctx)`. Resolution: **org DB flag → env kill-switch → catalog tier default** (core=on, addon=off).
- **Migration `00089_feature_flags.sql`**: `feature_flags(tenant_id, feature, enabled, updated_by, updated_at, PK(tenant_id, feature))` — one row per org per granted/revoked feature + `features:update` permission (admin-only). Ownership-index row appended.
- **Gate middleware** `features.Gate(registry, key)` → clean 404 when off; applied per-request so toggles are live without restart.
- **Admin UI `/settings/features`**: catalog grouped by category with **Core / Add-on badges**, current on/off state for the acting admin's org, toggle action (`settings:update`-gated, audit-logged). A companion super-admin view is unnecessary this milestone — your team can flip any org by acting as that org's admin or via one SQL/API call, both documented.
- **Upsell surface in-app**: sidebar/nav shows disabled add-on features greyed with a lock icon and "Add-on — contact to enable" tooltip → users of a non-paying org see what they're missing (classic SaaS lever). Gated routes return a branded "feature not enabled for your organisation" page (not a bare 404) for add-ons, so the upsell is explicit.
- **Workers** (dwell, fuel engine, scorecard sweep, P&L snapshot, e-way bill worker, founder digest): flag check per leadered tick — flip off an org's telemetry add-on and its workers stop processing within one tick.
- **Modules converted to gated** (routes + workers): telemetry suite, FASTag, e-way bill, accounting sync, RAG, AI agent, fuel audit, scorecard, geofences, experiments, founder, P&L, share links, customer portal, PWA.
- **Adding a feature later** = one catalog line + its `RegisterRoutes(r, gate)` block + optional worker — documented in the audit doc as the recipe.

## 2. Closed-source licensing

- Root `LICENSE`: proprietary — © 2026 Avandab, all rights reserved, no use/distribution without written license.
- Root `NOTICE`: permissive-dep attributions + watch-items (paho MQTT EPL, mysql driver MPL — fine unmodified).
- Fix vendored licenses: real MIT text for markercluster (current file is a broken fetch placeholder), add datastar MIT, font OFL/Apache texts.
- `package.json` `"license": "ISC"` → `"UNLICENSED"`.
- **Delete `samsara-gps-wireframe.md` + `WIREFRAME_ROUTIFIC_vs_ONFLEET.md`** — verbatim competitor-site transcriptions; real copyright exposure for a commercial closed product.
- `git rm --cached .env` + `.gitignore` (history retains it — rotate real secrets after).

## 3. E2E gap fixes

- **e-POD OTP — implement for real**: 6-digit OTP at dispatch (hashed, persisted), surfaced to the operator in trip view until SMS is live; delivery verifies and finally writes `pod_otp_verified` (column has never been written; flag today is force-zeroed).
- **Agent reachable without API key**: mount on `AGENT_ENABLED` even keyless — keywordRoute fallback stops being dead code.
- **Fix e-way bill auto-generate bug**: `autogenerate.go:85` queries non-existent columns (`config_value`/`config_key` vs real `value`/`key`) — feature has never fired. Fix + test.
- Credential-gated mocks (FASTag NETC, GSTN IRP, NIC EWB, Tally/Zoho, SMS/email/push) stay mock by design — each documented with exactly what key unblocks it, and now per-org switchable the day credentials arrive.

## 4. `docs/FEATURE_AUDIT.md`

Living per-feature table: E2E status (WORKING / MOCKED-NEEDS-CREDENTIALS / PARTIAL) → what completes it → flag key → tier (core/addon). Includes the roadmap list (safety-event producers, LR/consignment, Indian truck taxonomy, dashcam awaiting hardware) so "everything E2E" stays an honest checklist.

## 5. Tests

Registry (tier defaults, env override, per-org DB flag, tenant isolation, cache refresh), gate middleware (on/off/disabled-page), admin toggle endpoint (permission + persistence + audit log), upsell page for disabled add-on, e-POD OTP (generate → verify → wrong-OTP rejected → column written), e-way bill auto-generate config read, template coverage for the features page.

## Verification

`go build` · `go vet` · `go test ./internal/...` · migration 00089 up/down on scratch DB · `npm run build:css` · `golangci-lint --new-from-rev=HEAD` · mobile jest stays green.