# Feature Audit — status, flags, and what completes each feature

Living document. Every flaggable feature is listed with its E2E status, flag
key, commercial tier, and exactly what is needed to finish it. Update this
file when a feature changes status — it is the "everything works end-to-end"
checklist.

**How to add a feature (the plugin recipe):**
1. One line in the `Catalog` slice (`internal/features/features.go`): key,
   name, category, tier (`core`/`addon`), optional env flag + default.
2. Wrap its routes: `r.With(featureGate("your-key")).Route("/your", h.Routes)`.
3. Background work: `runLeadered("your_worker", func(ctx) { if !featureTick("your-key") { return }; ... })`.
4. Sidebar: wrap the nav item in `{{if and .Features (index .Features "your-key")}}`
   (+ `{{else}}` lock-item for add-ons).
5. Add a row here. Deleting a feature = remove its wiring block + catalog line.

**Toggle surface:** `/settings/features` (org admins, `settings:update`) or
`feature_flags` SQL per org. Route gates apply on the next request; workers
skip within one tick.

## Status legend
- **WORKING** — UI + logic + persistence, complete E2E
- **MOCKED (needs credentials)** — pipeline real; external adapter fabricates data until keys/contracts arrive
- **PARTIAL** — functional gap inside our code

## Operations

| Feature | Flag | Tier | Status | What completes it |
|---|---|---|---|---|
| GPS telemetry (ingest, devices, live map) | `telemetry` | Add-on | WORKING | Own MQTT/REST devices can register+stream today; safety-event producers = roadmap M2 |
| Geofences + detention billing | `geofences` | Add-on | WORKING | — |
| Fuel audit (KMPL, theft detection) | `fuel_audit` | Add-on | WORKING | — |
| e-POD (photo/signature/**OTP verified**) | — (trips) | Core | WORKING | OTP now issued at dispatch, verified at delivery, `pod_otp_verified` written; SMS push of OTP when SMS channel lands |
| Public trip share links | `share_links` | Core | WORKING | — |
| Customer portal | `customer_portal` | Core | WORKING | — |
| Maintenance scheduling | — (trips) | Core | WORKING | — |
| Live tracking UI (FlyFleet) | `telemetry` | Add-on | WORKING | Driver name on detail panel (M2); hardcoded Smart-Allocation mock to wire or remove (M2) |

## Commerce & Finance

| Feature | Flag | Tier | Status | What completes it |
|---|---|---|---|---|
| GST invoicing (CGST/SGST/IGST, HSN/SAC) | — (invoices) | Core | WORKING | — |
| GST e-invoice IRN | `gst_einvoice` | Add-on | MOCKED | GSP credentials (GSTN username/app-key flow); mock fabricates ACK/QR today |
| e-Way bills (lifecycle + monitor) | `ewaybill` | Core | WORKING (NIC mocked) | Auto-generate bug fixed (00090-era fix); NIC credentials for real EWB numbers |
| FASTag reconciliation | `fastag` | Add-on | MOCKED | NETC aggregator credentials; DB/reconcile logic real |
| Accounting sync (Tally/Zoho/QB) | `accounting_sync` | Add-on | MOCKED | Real adapter endpoints; consumer pipeline + idempotency real |
| Razorpay payments | `razorpay` | Core | WORKING | Production keys |
| Driver settlements + TDS 194C | `settlements` | Core | WORKING | — |

## Insights

| Feature | Flag | Tier | Status | What completes it |
|---|---|---|---|---|
| Trip P&L engine | `pnl` | Add-on | WORKING | Founder digest data sources (Spec 16 §5.8 populateDigest) |
| Driver safety scorecard | `scorecard` | Add-on | PARTIAL | 5 of 7 event feeds never produced (speeding/harsh/accel/idling/night) — **roadmap M2**, biggest safety gap |
| A/B experiments | `experiments` | Core | WORKING | — |

## Intelligence

| Feature | Flag | Tier | Status | What completes it |
|---|---|---|---|---|
| AI ops assistant | `agent` | Add-on | WORKING (keyless = keyword routing) | LLM API key for full reasoning; approval gate + RL loop real |
| RAG knowledge search | `rag` | Add-on | WORKING | Embedding API key for semantic quality (hash fallback otherwise) |
| Founder signals (Telegram) | `founder` | Core | WORKING | Digest metrics populate (M2); SMS/WhatsApp channels when credentials arrive |

## Platform

| Feature | Flag | Tier | Status | What completes it |
|---|---|---|---|---|
| Alerts pipeline (dedup/escalation) | — | Core | WORKING | Email/SMS/WhatsApp channel adapters (stubbed) |
| Compliance 5-doc gate | — | Core | WORKING | — |
| Feature flags (this system) | — | Core | WORKING | — |
| PWA | `pwa` | Core | WORKING | — |

## Roadmap (features we should have; not yet built)
1. **M2 — Safety producers** (unlocks Samsara-style safety): emit
   speeding/harsh-braking/harsh-accel/idling/night-driving events from
   telemetry frames; SOS event wiring; dashcam groundwork when hardware is
   chosen (schema + incident-footage links).
2. **M3 — India structural**: LR/consignment number + consignor party;
   Indian truck taxonomy (tipper/tanker/trailer/axle/tonnage); state picker;
   national vs state permits; MV tax.
3. **Notification channels**: SMTP email, SMS gateway, WhatsApp Business API.
4. **Mobile MQTT retrofit** (Spec 01 Phase 3): move phone live-tracking from
   15s HTTP sync to the device MQTT topic.
5. **Telemetry provider adapters** (LocoNav/WheelsEye/JT808) if third-party
   GPS resale becomes a channel; own devices already work.
