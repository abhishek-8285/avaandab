# UI Completeness (Fragments, SSE, Exports, PWA, RBAC Wiring, Settlement/EWB/Map Pages) — Implementation Spec v1

Status: ready
Depends-on: spec `04-live-map-share-maintenance.md` (Live Map + SSE hub), spec `07` (Driver Settlement DDL), spec `08` (E-Way Bill DDL)
Migration owner: none new (this spec adds NO DB tables — reads existing tables owned by 04/07/08; see §3)

> Audience: a beginner Go + HTML dev (or an AI agent) can implement every section end-to-end
> without further clarification. Every snippet is copy-pasteable and references real files via
> `path:line`.

---

## 0. Verified ground truth (file:line facts + grep proofs)

All facts below were verified on this repo at `/home/abhishek/Desktop/temux/basic`.

### 0.1 SPA fragment branch is conditionally reachable (NOT dead code)
`internal/handlers/app.go:407-411` (inside `renderPage`):
```go
if s, ok := w.(interface{ IsSPARequest() bool }); ok && s.IsSPARequest() {
    w.Header().Set("X-Page-Title", data.Title)
    _, _ = w.Write([]byte(buf.String()))
    return
}
```
- `buf` is the *content only* (rendered from `contentTmpl` at `app.go:402-405`), not wrapped in `layout.html`.
- **CORRECTED (v1 was wrong):** an `http.ResponseWriter` in the chain DOES implement `IsSPARequest()`. `SPAMiddleware` (`internal/middleware/middleware.go:236`) is wired globally at `cmd/server/main.go:270` and, when `X-SPA-Request: true` and the path is not a download, wraps the writer in `SpaResponseWriter{isSPA:true}` (`middleware.go:240-245`). So the branch is **reachable** whenever a client sends that header. `grep -rn "IsSPARequest" internal/ --include=*.go` returns **two** hits — `app.go:407` and `middleware.go:231` — not the single hit claimed in v1.
- **Status:** the branch is currently *not exercised by the SPA client* (router.js never sends `X-SPA-Request`; see §0.10), but it is **not dead code**. Treat it as a live, header-gated fast path, not an unreachable branch.

### 0.2 `can()` RBAC func defined but never used in templates
`internal/handlers/app.go:110-129` defines `can` in `parseTemplates`' `FuncMap`:
```go
"can": func(user interface{}, resource string, action string) bool {
    ...
    return authSrv.Can(uid, resource, action)
},
```
`auth.Can` is live RBAC: `internal/auth/casbin.go:108,140`.
Grep proof `can(` is **never called** from any UI template:
```
grep -rn "can " internal/templates/ | grep -viE "cancel|cannot|scanc|cancelled"
```
→ matches only prose ("you can do") in `feature.html`/`assistant.html`/`cookie-consent.html`. The RBAC hook exists but the UI hardcodes roles instead.

### 0.3 Hardcoded role checks (replace with `can()`)
- `internal/templates/layout.html:125`: `{{if and .User (eq .User.Role "admin")}}` gating the **User Management** nav link.
- `internal/handlers/app.go:672`: `session.Role == "admin" || session.Role == "dispatcher" || session.Role == "accountant"` in `DownloadFile`.
- `grep -rn "eq .*Role" internal/templates/` → only `layout.html` + `user_edit.html` (the latter is a role `<select>`, not an auth gate).

### 0.4 Static dashboard, no SSE
- `internal/handlers/dashboard.go:18` `Index` calls `h.Services.Dashboard.GetDashboardData(...)` once and `renderPage` — no streaming endpoint.
- `internal/service/dashboard_service.go:45-274` `DashboardService` uses a `sync.RWMutex` + 3s TTL memory cache (`GetDashboardData`, `dashboard_service.go:55-66`). No goroutine, no `http.Flusher`, no channel.

### 0.5 Missing / unwired list-table fragments (more than the 7 in v1)
On disk only `booking_list_table.html` exists, but it is **never referenced**: `booking_list.html` renders an inline `<table>` + `pagination.html` and never `{{template "booking_list_table.html" .}}` (confirmed: `grep -rln "_list_table" internal/templates/` → 0 hits, because no template references any `*_list_table` partial). So even bookings have no fragment refresh.

The v1 list of 7 is correct as far as it goes, but it **undercounts**. Every one of the following `*_list.html` pages renders an inline `<table>` with no `*_list_table.html` partial and therefore cannot refresh incrementally — they all need a table partial:
`vehicle_list.html`, `customer_list.html`, `invoice_list.html`, `driver_list.html`, `user_list.html`, `route_list.html`, `payment_list.html` **plus** `trip_list.html` and `audit_logs_list.html` (both confirmed to contain `<table>` and to lack a `*_list_table.html`). That is **9** list pages needing a table partial (and `booking_list.html` needing to wire its existing one).

> The exact set is verified against the glob: only `booking_list_table.html` exists; the missing/needed partials are `vehicle_list_table.html`, `customer_list_table.html`, `invoice_list_table.html`, `driver_list_table.html`, `user_list_table.html`, `route_list_table.html`, `payment_list_table.html`, `trip_list_table.html`, `audit_logs_list_table.html`.

### 0.6 Orphan partials (4, defined but never `template`/`partials/` referenced)
```
grep -rln "partials/empty_state\|partials/field\|partials/page_header\|partials/stat_card" internal/  → 0 hits
```
Files: `internal/templates/partials/empty_state.html`, `field.html`, `page_header.html`, `stat_card.html`. These are reusable building blocks we **wire in** during this spec (§4).

### 0.7 Reports: pages only, no exports
`internal/handlers/reports.go:19-27` `Routes` mounts only:
`GET /`, `/revenue`, `/trips`, `/drivers`, `/vehicles`, `/customers`, `/pending-payments` — all `renderPage`. No CSV, no PDF.
`internal/pdf/invoice_pdf.go:14` `GenerateInvoicePDF` exists but is the **only** PDF export and is not linked from any report page.

### 0.8 Backend services exist but have NO UI
- `internal/service/driver_settlement_service.go:31` `DriverSettlementService` (`CreateSettlementForTrip`, payout math `net = fare - advances - deductions`) — no handler, no template.
- `internal/integration/ewaybill/client.go` `Config{Endpoint,APIKey,Enabled}` + `GenerateRequest`/`EWayBill` structs + `Cancellation` — **no** `handler.go` (confirmed: `internal/integration/ewaybill/` contains only `client.go`), no UI, no route.
- `internal/mqttservice/mqtt.go:18` `GPSTelemetryPayload{DriverID,Latitude,Longitude,Timestamp}` + `NewMQTTBroker` — stub; `subscribeTelemetry` logs only. No UI / no SSE wiring.

### 0.9 PWA absent
Grep `manifest.webmanifest|service-worker|sw.js|rel="manifest"` across `internal/static` and `internal/templates` → **0 hits**. No installable shell.

### 0.10 Router kills reactivity + never sets SPA header
`internal/static/js/router.js:149-178` intercept every `<form submit>` (except `/logout` and `data-on-submit` forms), `preventDefault`, and `loadPage(url, fetchOpts)` which at `router.js:84-88` does `document.body.innerHTML = doc.body.innerHTML` — this **nukes** any live Datastar signals / event listeners on the page (the `submit` interceptor is confirmed; the body-swap is at `router.js:87`). The router **never** sends `X-SPA-Request` (confirmed: `grep -rn "X-SPA-Request" internal/static/` → 0 hits). **CORRECTED (v1 was wrong):** the server branch at `app.go:407` does key off the `IsSPARequest()` interface method — but that method returns the value set by `SPAMiddleware` **only when the `X-SPA-Request` header is present** (`middleware.go:239`), so the branch is header-gated, not a pure interface check (see §0.1). Forms that *should* trigger a fragment refresh silently full-reload reactivity state.

---

### 0.11 Verification Log (Principal Engineer QA pass)

Claims checked against real files. Verdicts: ✅ TRUE / ❌ WRONG / ⚠️ PARTIAL.

| # | Claim (§) | Verdict | Correction / Evidence | Severity | Effort |
|---|---|---|---|---|---|
| 1 | §0.1 SPA branch dead/unreachable | ❌ WRONG | `SPAMiddleware` (`middleware.go:236`) wired at `main.go:270` wraps writer in `SpaResponseWriter{isSPA:true}` on `X-SPA-Request:true`. Branch reachable. `grep IsSPARequest` → 2 hits (`app.go:407`, `middleware.go:231`), not 1. | High | S (doc) |
| 2 | §0.2 `can()` never used in templates | ✅ TRUE | `grep -rn "can "` in templates → only prose ("you can do") in feature/assistant/cookie-consent. No `{{can ...}}` calls. | — | — |
| 3 | §0.3 hardcoded `eq .Role` | ✅ TRUE | `layout.html:125` `eq .User.Role "admin"`; `app.go:672` `isStaff` OR-list; `user_edit.html:39` is a role `<select>` (not a gate). | — | — |
| 4 | §0.4 static dashboard, no SSE | ✅ TRUE | `dashboard.go:18` single `GetDashboardData`+`renderPage`. `dashboard_service.go` has `sync.RWMutex` only (line 48); no `Flusher`/goroutine/channel. | — | — |
| 5 | §0.5 exactly 7 missing fragments | ⚠️ PARTIAL | 7 listed are missing, BUT `trip_list.html`+`audit_logs_list.html` also lack table partials (total 9), and `booking_list_table.html` exists yet is unreferenced (booking_list.html uses inline table). | Med | S (doc) |
| 6 | §0.6 4 orphan partials | ✅ TRUE | `grep -rln "stat_card\|empty_state\|page_header\|field.html"` → 0 references. | — | — |
| 7 | §0.7 no CSV/PDF export in reports | ✅ TRUE | `reports.go` mounts only `renderPage` routes; `grep encoding/csv` repo-wide → 0 (no CSV anywhere). `invoice_pdf.go:14` exists & is used by `invoices.go:167`, not reports. | — | — |
| 8 | §0.8 backend services lack UI | ✅ TRUE | `driver_settlement_service.go:46` `CreateSettlementForTrip` + NetPayout; `ewaybill/` has only `client.go`; `mqttservice/mqtt.go` stub logs only. | — | — |
| 9 | §0.9 PWA absent | ✅ TRUE | `grep manifest.webmanifest\|service-worker\|sw.js\|rel="manifest"` (static+templates) → 0 hits. | — | — |
| 10 | §0.10 router kills reactivity + branch "keys off interface not header" | ⚠️ PARTIAL | First clause TRUE (router never sends `X-SPA-Request`; body-swap at `router.js:87` nukes signals). Second clause WRONG: branch is header-gated via `SPAMiddleware` (`middleware.go:239`). | High | S (doc) |
| 11 | §3 migration `00031` (telemetry) exists | ✅ TRUE | `db/migrations/00031_avandab_critical_fixes.sql` present. | — | — |
| 12 | Migration `00044` exists (QA cross-check) | ❌ N/A | `00044` does **not** exist; latest is `00039_*`. Spec does not reference `00044`, so no spec impact, but the repo has no `00044` slot. | — | — |

**WRONG count: 2 (claims #1, #10) + 2 partials (#5, #10).** All other §0 claims verified TRUE.

---

## 1. Overview / goal

Make the UI **complete and live**: (a) add the 7 missing list-table fragments so every index page can refresh incrementally; (b) ship a real-time dashboard + live map over SSE; (c) add CSV + PDF export to every report; (d) add a PWA shell (manifest + service worker); (e) wire RBAC everywhere via the existing `can()` func (kill hardcoded `eq .Role`); (f) add UI pages for Driver Settlement, E-Way Bill, and Live Map that consume already-built backend services.

**Non-goals:** no new DB tables (§3); no new RBAC model (reuse Casbin); no new GPS hardware integration (MQTT stays a stub; map reads the existing telemetry table from spec 04); no auth changes.

---

## 2. API contract

All routes require an authenticated session (`auth.ContextUser`). Read routes reuse `middleware.ResourcePermission(h.AuthSrv, <resource>, "read")`.

### 2.1 Dashboard SSE — `GET /dashboard/stream`
- Auth: `reports`/`dashboard` `read` (reuse `middleware.ResourcePermission(h.AuthSrv, "dashboard", "read")`).
- Response: `text/event-stream; charset=utf-8`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`.
- Body: SSE frames every `DASHBOARD_SSE_INTERVAL_SEC` (default 5). Frame format (Datastar-compatible `@signals` merge):
  ```
  event: datastar-merge-signals
  data: {"dashboard":{"todaysTrips":12,"activeTrips":3,"availableVehicles":8,"availableDrivers":5,"monthlyRevenue":240000,"statusCounts":{"scheduled":2,...}}}

  ```
  When `DASHBOARD_SSE_ENABLED=false` → respond `200` with a single frame echoing the current snapshot then close (graceful no-op so the client still renders).
- Client uses `EventSource("/dashboard/stream")`; on `error` it silently retries (browser default).

### 2.2 Live Map SSE — `GET /map/stream`
- Auth: `trips` `read`.
- Response: `text/event-stream`.
- Body: one frame per vehicle snapshot pulled from the telemetry table (spec 04, `telemetry_snapshots`):
  ```
  event: datastar-merge-signals
  data: {"vehicles":[{"id":"veh-1","lat":19.07,"lng":72.87,"driver":"Ravi","trip":"TRP-9","eta":"14:30","status":"in_transit"}]}

  ```
- Reuse the SSE hub from spec `04-live-map-share-maintenance.md` (`/api/v1/telemetry/stream`) if present; otherwise implement a thin hub in `internal/handlers/map.go` (§5.1).

### 2.3 Reports export — CSV
- `GET /reports/revenue.csv`, `/reports/trips.csv`, `/reports/drivers.csv`, `/reports/vehicles.csv`, `/reports/customers.csv`, `/reports/pending-payments.csv`
- Auth: `reports` `read`.
- Query params pass through (`?q=&status=&from=&to=`). `EXPORT_MAX_ROWS` caps rows (default 50000); if exceeded, `Link` header carries `rel="next"` cursor.
- Response: `Content-Type: text/csv; charset=utf-8`, `Content-Disposition: attachment; filename="<name>_<YYYYMMDD>.csv"`, `X-Export-Rows: <n>`.
- Body: UTF-8 CSV with BOM (`\xEF\xBB\xBF`) so Excel renders Unicode. Columns per report (§5.2).

### 2.4 Reports export — PDF
- `GET /reports/pending-payments.pdf` (table of outstanding invoices) and `GET /reports/revenue.pdf` (summary + monthly table).
- Auth: `reports` `read`.
- Response: `Content-Type: application/pdf`, `Content-Disposition: attachment; filename="..."`.
- Reuse `internal/pdf/invoice_pdf.go:14` helper pattern; new helper `GenerateReportPDF(rows, cols, title, companyName)` in a new file `internal/pdf/report_pdf.go` (§5.3).

### 2.5 Settlement page
- `GET /settlements` → page (resource `settlements` `read`). `POST /settlements` → create (resource `settlements` `create`) calling `DriverSettlementService.CreateSettlementForTrip` (`driver_settlement_service.go:43`).

### 2.6 E-Way Bill page (card inside trip view)
- `GET /trips/{id}/ewaybill` → card fragment (resource `ewaybill` `read`) backed by `internal/integration/ewaybill/client.go`.
- `POST /trips/{id}/ewaybill` → generate (resource `ewaybill` `create`); fires `EWayBillClient.Generate` only when `EWayBillConfig.Enabled` (adapter pattern per `_TEMPLATE.md` conventions). Offline/mock returns a stub `EwbNumber: "MOCK-<uuid>"`.

### 2.7 Live Map page
- `GET /map` → full page (resource `trips` `read`) rendering `internal/templates/map.html` + Leaflet (CDN) + `EventSource("/map/stream")`.

### 2.8 PWA assets
- `GET /manifest.webmanifest` → `Content-Type: application/manifest+json`.
- `GET /sw.js` → `Content-Type: text/javascript`, `Service-Worker-Allowed: /`, `Cache-Control: no-cache`.
- `GET /static/icons/icon-192.png`, `/icon-512.png` → static files (add via `filepath` from `internal/static`).

---

## 3. DB contract

**No new migration.** This spec reads existing tables only:
- Live map reads `telemetry_snapshots` (owned by spec `04`, migration `00031`).
- Settlement reads `driver_settlements` (owned by spec `07`) via `DriverSettlementService`.
- E-Way Bill reads `eway_bills` (owned by spec `08`) via `EWayBillClient`.
Reserve **no** new `00XXX` slot in `docs/tech-specs/00-migration-ownership-index.md` (nothing to add). If a column is missing, open a VERIFY item (§11) rather than adding DDL here.

---

## 4. UI

### 4.1 New list-table fragments (replace inline `<table>` in each `*_list.html`)
Create 7 partials that mirror `internal/templates/booking_list_table.html` exactly in markup (`th-cell`/`td-cell` classes, `data-numeric`, `statusBadge`). Each takes the same data map the page passes (`.Vehicles`, `.Drivers`, `.Customers`, `.Invoices`, `.Users`, `.Routes`, `.Payments`).

| Create file | Data var | Iter item fields (examples) |
|---|---|---|
| `internal/templates/vehicle_list_table.html` | `.Vehicles` | `.RegistrationNumber`, `.Model`, `.Status` |
| `internal/templates/customer_list_table.html` | `.Customers` | `.Name`, `.GSTIN`, `.Contact` |
| `internal/templates/invoice_list_table.html` | `.Invoices` | `.InvoiceNumber`, `.Total`, `.Status` |
| `internal/templates/driver_list_table.html` | `.Drivers` | `.Name`, `.LicenseNumber`, `.Status` |
| `internal/templates/user_list_table.html` | `.Users` | `.Name`, `.Email`, `.Role.Name` |
| `internal/templates/route_list_table.html` | `.Routes` | `.Source`, `.Destination`, `.DistanceKm` |
| `internal/templates/payment_list_table.html` | `.Payments` | `.InvoiceNumber`, `.Amount`, `.Method` |

Then edit each `*_list.html` to wrap the table in a Datastar refresh target and include the partial:
```html
<div id="list-table" data-signals="...">
  {{template "vehicle_list_table.html" .}}
</div>
```
And add a refresh trigger (Datastar `@get` or a `setInterval` SSE). Keep the fallback: `renderFragment` (`app.go:516-523`) already strips `_table.html` → `.html` on miss, so a missing partial degrades to the full page (this is the bug we fix by *creating* the partials).

### 4.2 Real-time dashboard
- Edit `internal/templates/dashboard.html`: wrap stat cards in `id="dash-signals"` and render from `data-signals` so `/dashboard/stream` updates them. Add at bottom:
  ```html
  <script>
    if (window.EventSource) {
      const es = new EventSource('/dashboard/stream');
      es.onmessage = (e) => { /* datastar auto-merges @signals */ };
    }
  </script>
  ```
- Add `GET /dashboard/stream` handler (§5.1).

### 4.3 Live Map (Leaflet, spec 04)
Create `internal/templates/map.html` (full page, title "Live Map"). Include Leaflet from CDN with `integrity` omitted (vendored fallback acceptable):
```html
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<div id="map" class="h-[70vh] w-full rounded-lg"></div>
<script src="/static/js/map.js"></script>
```
Create `internal/static/js/map.js` that inits the map, opens `EventSource('/map/stream')`, and drops a `L.marker` per vehicle, updating position on each frame. Gate the script behind `{{if can .User "trips" "read"}}` in the page.

### 4.4 Reports export buttons
Edit each `report_*.html` (and `reports_index.html`) to add a toolbar linking the export endpoints, gated by `can`:
```html
{{if can .User "reports" "read"}}
<a href="/reports/trips.csv?{{query_string}}" class="btn">Export CSV</a>
<a href="/reports/trips.pdf" class="btn">Export PDF</a>
{{end}}
```

### 4.5 Settlement page
Create `internal/templates/settlement_list.html` + `settlement_view.html` (mirror `invoice_list.html`/`invoice_view.html`). Show columns from `DriverSettlementRecord` (`driver_settlement_service.go:18-30`): `GrossFare`, `AdvancesKharcha`, `Deductions`, `NetPayout`, `Status`. Use the orphan partial `stat_card.html` for the payout summary (wire it in — kills one orphan).

### 4.6 E-Way Bill card in trip view
Edit `internal/templates/trip_view.html`: add a card block:
```html
{{if can .User "ewaybill" "read"}}
<div id="ewaybill-card" class="card">{{template "ewaybill_card.html" .EWayBill}}</div>
{{end}}
```
Create `internal/templates/partials/ewaybill_card.html` rendering `.EwbNumber`, `.Status`, `.ValidUpto`, `.QRCode` (as `<img>` if present).

### 4.7 PWA
Create `internal/static/manifest.webmanifest`:
```json
{
  "name": "Avandab Fleet",
  "short_name": "Avandab",
  "start_url": "/dashboard",
  "display": "standalone",
  "background_color": "#0b1020",
  "theme_color": "#4f46e5",
  "icons": [
    {"src": "/static/icons/icon-192.png", "sizes": "192x192", "type": "image/png"},
    {"src": "/static/icons/icon-512.png", "sizes": "512x512", "type": "image/png"}
  ]
}
```
Edit `internal/templates/layout.html` `<head>` (after `layout.html:17`) to add:
```html
<link rel="manifest" href="/manifest.webmanifest">
<meta name="theme-color" content="#4f46e5">
```
Create `internal/static/js/sw.js` (cache-first for `/static`, network-first for pages) and register it from `layout.html` (after `layout.html:52`):
```html
<script>
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(()=>{});
</script>
```
Both gated by `PWA_ENABLED` in `config.go` (§6) — when disabled, the `<link rel="manifest">` and registration script are omitted (use a `{{if .PWAEnabled}}` field added to `PageData`; see §5.4).

### 4.8 RBAC wiring (replace hardcoded roles)
- Edit `internal/templates/layout.html:125`:
  ```html
  {{if can .User "users" "read"}}
  ```
  (keep the inner `<a href="/users">` block unchanged).
- Edit `internal/handlers/app.go:672` `DownloadFile` to use `h.AuthSrv.Can(session.UserID, "files", "read")` instead of the `isStaff` role OR-list — or simpler, keep staff behavior but drive it from `can`. Document the change.
- Add nav links for the new pages, each gated by `can`:
  ```html
  {{if can .User "settlements" "read"}}<a href="/settlements" ...>Settlements</a>{{end}}
  {{if can .User "trips" "read"}}<a href="/map" ...>Live Map</a>{{end}}
  ```
- Wire the 4 orphan partials into real pages so they are no longer dead:
  - `stat_card.html` → settlement summary + dashboard (§4.5).
  - `page_header.html` → every `*_list.html` top (replaces ad-hoc `<h1>` + breadcrumb).
  - `field.html` → `driver_edit.html` / `vehicle_edit.html` form rows.
  - `empty_state.html` → every `*_list_table.html` `{{else}}` branch when range is empty.

---

## 5. Business logic

### 5.1 SSE loop (dashboard + map)
New file `internal/handlers/realtime.go`:
```go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"transport-app/internal/shared"
)

// streamDashboards writes SSE frames until the client disconnects or ctx is done.
func (h *DashboardHandlers) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	interval := h.Config.DashboardSSEInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if !h.Config.DashboardSSEEnabled {
		// Graceful no-op: emit one snapshot then close.
		h.writeDashFrame(w, flusher)
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// Prime frame immediately.
	h.writeDashFrame(w, flusher)
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			h.writeDashFrame(w, flusher)
		}
	}
}

func (h *DashboardHandlers) writeDashFrame(w http.ResponseWriter, f http.Flusher) {
	data, err := h.Services.Dashboard.GetDashboardData(r.Context())
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: datastar-merge-signals\ndata: {\"dashboard\":%s}\n\n",
		dashJSON(data))
	f.Flush()
}
```
> `r_ctx(r)` = `r.Context()`; `dashJSON` = `json.Marshal` of a small struct (todaysTrips, activeTrips, availableVehicles, availableDrivers, monthlyRevenue, statusCounts). Reuse `shared.TenantIDFromContext` for tenant scoping. Map SSE mirrors this with `vehicles` payload built from `telemetry_snapshots` (spec 04). Respect `EXPORT_MAX_ROWS`/per-tenant limits by filtering the vehicle set server-side.

### 5.2 CSV helper
New file `internal/handlers/export.go`:
```go
package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
)

// writeCSV writes UTF-8 CSV with BOM and attachment headers.
func writeCSV(w http.ResponseWriter, filename string, header []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		"attachment; filename=\""+filename+"\"")
	w.Header().Set("X-Export-Rows", strconv.Itoa(len(rows)))
	w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM
	cw := csv.NewWriter(w)
	_ = cw.Write(header)
	_ = cw.WriteAll(rows)
	cw.Flush()
}
```
Column sets (examples):
- trips.csv: `TripNumber, Customer, Route, Status, Pickup, Delivery, Driver, Vehicle`.
- revenue.csv: `Month, Revenue, Bookings`.
- drivers.csv: `Name, LicenseNumber, Status, Phone`.
- vehicles.csv: `Registration, Model, Status, LastTrip`.
- customers.csv: `Name, GSTIN, Contact, Outstanding`.
- pending-payments.csv: `InvoiceNumber, Customer, Total, Paid, Outstanding, DueDate`.
Apply `EXPORT_MAX_ROWS` cap: `if len(rows) > cfg.ExportMaxRows { rows = rows[:cfg.ExportMaxRows] }` and set `Link` header for the next cursor when truncated.

### 5.3 PDF helper
New file `internal/pdf/report_pdf.go` using `github.com/go-pdf/fpdf` (already imported in `invoice_pdf.go:8`):
```go
func GenerateReportPDF(title, companyName string, header []string, rows [][]string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, companyName, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "B", 9)
	for _, h := range header {
		pdf.CellFormat(180/float64(len(header)), 8, h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 9)
	for _, row := range rows {
		for _, c := range row {
			pdf.CellFormat(180/float64(len(header)), 7, truncate(c, 40), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil { return nil, err }
	return buf.Bytes(), nil
}
```

### 5.4 Router fix (`internal/static/js/router.js`)
The `submit` interceptor at `router.js:149-178` must **not** swallow forms that target a fragment or carry `data-spa="false"`. Two changes:

1. Skip forms that should keep native behavior / Datastar reactivity:
```js
document.addEventListener('submit', function(e) {
    const form = e.target.closest('form');
    if (!form) return;
    if (form.getAttribute('action') === '/logout') return;
    if (form.hasAttribute('data-on-submit')) return;
    // NEW: let Datastar / HTMX / explicit SPA-opt-out forms behave natively.
    if (form.hasAttribute('data-spa') && form.getAttribute('data-spa') === 'false') return;
    if (form.hasAttribute('data-datastar-ignore')) return;
    ...
```
2. After `loadPage` swaps `document.body.innerHTML` (`router.js:87`), **re-bootstrap** Datastar so signals re-attach:
```js
// NEW: re-init Datastar after SPA swap
if (window.Datastar && window.Datastar.reinitialize) {
    window.Datastar.reinitialize(document.body);
}
```
If `datastar.js` exposes no `reinitialize`, instead dispatch `document.dispatchEvent(new Event('datastar:load'))` (the vendored Datastar `v1.0.2` listens for `DOMContentLoaded`/a load signal — verify in `internal/static/js/datastar.js`). Fallback: drop the body-swap for pages that contain `@signals` and instead navigate natively (`window.location.href = url`).
3. The server branch (`app.go:407`) is **reachable** when a client sends `X-SPA-Request: true` (handled by the wired `SPAMiddleware`, see §0.1) — it is not dead code. The SPA router currently does not send that header, so the branch is dormant but live. We do **not** add `X-SPA-Request` from the router in this spec; instead we re-bootstrap Datastar after the body swap (item 2 above). Optionally delete `app.go:407-411` to remove confusion (VERIFY §11) — but note deletion would break any future client that sets the header.

### 5.5 `can()` usage in templates
`can` is already registered (`app.go:110`). Use it as `{{if can .User "<resource>" "<action>"}}`. `.User` is `*auth.SessionData` (passed as `PageData.User` in `buildTemplateData`, `app.go:367-380`). The func resolves the userID and calls `authSrv.Can` (`casbin.go:140`). No code change needed in `app.go` — only template edits (§4.8).

---

## 6. Config / env

Add to `internal/config/config.go` (struct `Config`, `config.go:14`) and `Load()` (`config.go:78+`):

| Var | Default | Purpose | Read by |
|---|---|---|---|
| `DASHBOARD_SSE_ENABLED` | `true` | Toggle `/dashboard/stream` loop | `DashboardHandlers.Stream` (§5.1) |
| `DASHBOARD_SSE_INTERVAL_SEC` | `5` | Frame interval seconds | `DashboardHandlers.Stream` (§5.1) |
| `PWA_ENABLED` | `true` | Emit manifest `<link>` + SW registration | `renderPage` / `layout.html` |
| `EXPORT_MAX_ROWS` | `50000` | Max CSV rows before truncation + cursor | `writeCSV` (§5.2) |

Add struct fields:
```go
type Config struct {
    // ... existing ...
    DashboardSSEEnabled   bool          `env:"DASHBOARD_SSE_ENABLED"`
    DashboardSSEInterval  time.Duration `env:"DASHBOARD_SSE_INTERVAL_SEC"` // parse as seconds
    PWAEnabled            bool          `env:"PWA_ENABLED"`
    ExportMaxRows         int           `env:"EXPORT_MAX_ROWS"`
}
```
In `Load()`: `cfg.DashboardSSEInterval = time.Duration(atoiEnv("DASHBOARD_SSE_INTERVAL_SEC", 5)) * time.Second`. Add `PWAEnabled bool` to `PageData` (`app.go:278-289`) and set it from `a.Config.PWAEnabled` in `buildTemplateData` (`app.go:362-381`) so `layout.html` can `{{if .PWAEnabled}}`.

E-Way Bill / MQTT stay config-flagged behind existing `ewaybill.Config.Enabled` (`client.go:23`) and MQTT broker URL — no new env for those.

---

## 7. Tests

Add to `internal/handlers/`:

1. **Template renders all pages** — extend `internal/handlers/template_render_test.go` to loop over every `.html` in `internal/templates/` (excluding partials) and assert `renderPage` returns 200 with non-empty body. Already partly covered (`template_render_test.go:180-200`); add the 7 new `*_list_table.html` + `map.html` + `settlement_list.html` + `settlement_view.html` + `ewaybill_card.html`.
2. **Missing fragments resolve** — test `renderFragment(w, "vehicle_list_table.html", data)` returns 200 (proves §4.1 created the file; before the fix this 500s).
3. **CSV content-type** — `httptest` GET `/reports/trips.csv`; assert `Content-Type` contains `text/csv`, body starts with BOM (`\xEF\xBB\xBF`), and `X-Export-Rows` header present.
4. **SSE first frame** — `httptest` GET `/dashboard/stream` with a short `context.WithTimeout`; read first line, assert it starts with `event: datastar-merge-signals` and contains `{"dashboard":`.
5. **RBAC `can()` booleans** — unit test the template func: build `App` with a fake `authSrv` (`Can(userID,res,act) bool`), render a snippet `{{if can .User "users" "read"}}YES{{end}}` with `User` allowed/denied; assert output. Also assert `layout.html` no longer contains `eq .User.Role "admin"` after §4.8.
6. **PWA** — GET `/manifest.webmanifest` → `application/manifest+json`; GET `/sw.js` → `text/javascript`.
7. **Coverage gate**: `go test ./internal/handlers/... ./internal/pdf/...` must pass; `go build ./...` clean. Add to `AGENTS.md` pass-before-merge checklist.

---

## 8. Future / GPS-provider

- **Live map from telemetry**: spec 04 already defines `telemetry_snapshots` (00031) + SSE hub `/api/v1/telemetry/stream`. This spec's `/map/stream` should subscribe to that hub (in-memory pub/sub) rather than poll the DB. When `mqttservice` (`mqtt.go:18`) graduates from stub, the same `GPSTelemetryPayload` feeds the hub — no UI change.
- **GPS ETA on dashboard**: once positions exist, compute ETA client-side (Haversine vs trip route polyline from `route` entity) and push via `/dashboard/stream` as `eta` per active trip. Third-party adapters (LocoNav/WheelsEye) plug behind the `TelematicsProvider` interface per `_TEMPLATE.md` conventions; own MQTT is primary.
- **Offline export**: when `EXPORT_MAX_ROWS` truncates, expose a queued background job that emails a signed link (future spec).

---

## 9. Edge cases

- SSE client disconnect → `r.Context().Done()` closes the goroutine (§5.1); no leaked timers.
- `DashboardService` cache (3s TTL) + SSE both read the same source; SSE just re-reads on interval — acceptable; if load is high, have SSE read the cached snapshot via a `GetCachedDashboardData` method.
- CSV with `>` 50k rows → truncated, `Link` next-cursor set, flash message on the page.
- `PWA_ENABLED=false` → no manifest link, no SW register (progressive enhancement).
- `can()` with `nil` user (public page) → returns `false` (`app.go:111-113`); nav links correctly hidden.
- Missing `telemetry_snapshots` rows → `/map/stream` sends empty `vehicles:[]`; map shows "no units".
- `EWayBillConfig.Enabled=false` → generate returns `MOCK-<uuid>` so the UI is fully exercisable without NIC creds.

---

## 10. Phased rollout (build order)

1. **Phase A — fragments + RBAC (no backend risk):** create 7 `*_list_table.html` (§4.1), wire `page_header`/`field`/`empty_state`/`stat_card` orphans (§4.8), replace `eq .Role` with `can` in `layout.html` + `app.go:672`. Tests §7.1, §7.2, §7.5.
2. **Phase B — exports:** `export.go` + `report_pdf.go`, add routes in `reports.go`, add buttons to report pages. Tests §7.3.
3. **Phase C — realtime:** `realtime.go` SSE + `dashboard.html` wiring + `map.go`/`map.html`/`map.js` + config vars. Tests §7.4.
4. **Phase D — new pages:** Settlement + E-Way Bill cards (consume existing services). 
5. **Phase E — PWA:** manifest + sw.js + layout edits + config. Tests §7.6.
6. **Phase F — router fix:** `router.js` reactivity re-bootstrap (§5.4). Ship last (JS behavior change).

---

## 11. Open items / VERIFY

- [ ] Confirm vendored Datastar `internal/static/js/datastar.js` exposes a re-init hook (`reinitialize` or a `datastar:load` event). If not, Phase F falls back to native navigation for `@signals` pages. **VERIFY before coding §5.4.**
- [ ] Decide: keep or delete `app.go:407-411` SPA branch. **CORRECTED:** the branch is *not* dead — `SPAMiddleware` (`middleware.go:236`, wired `main.go:270`) sets `IsSPARequest()==true` when `X-SPA-Request: true` is sent. It is dormant only because the SPA client never sends that header. Options: (a) leave as a live header-gated fast path for future native-SPA clients; or (b) delete if we commit to full `body.innerHTML` swaps forever. For this spec, leave it (low risk) and do not add the header from `router.js` (we re-bootstrap Datastar instead, §5.4).
- [ ] Confirm RBAC resource names for new pages: `settlements`, `ewaybill`, `dashboard` must exist in the Casbin model/policy, else `can` always returns false → pages invisible. **VERIFY policy file before §4.8 nav links.**
- [ ] `EXPORT_MAX_ROWS` cursor scheme: confirm `from`/`to` query params are honored by the report services; if not, Phase B truncates silently.
- [ ] Leaflet CDN vs vendored: confirm CSP/`Content-Security-Policy` allows `unpkg.com`; else vendor `leaflet.js`/`leaflet.css` into `internal/static/vendor`.
- [ ] `PageData` gets `PWAEnabled` field — confirm `buildTemplateData` propagates it to all pages (incl. `renderAuthPage`).

---

## 12. Engineering Decisions, Tradeoffs & Cost (PE QA pass)

These decisions resolve the open architecture questions from the verification pass. They are binding for implementation.

### 12.1 Real-time transport: SSE vs polling
- **Decision:** Use **SSE** (`text/event-stream`) for `/dashboard/stream` and `/map/stream`, exactly as §2.1/§2.2 specify. Do **not** add a client polling loop.
- **Tradeoff:** SSE is unidirectional server→client, auto-reconnects via `EventSource`, and integrates with Datastar `@signals`. Cost: one long-lived goroutine + `http.Flusher` per connected client; must disable proxy buffering (`X-Accel-Buffering: no`) or frames stall.
- **When to switch:** If concurrent ops sessions exceed a few hundred, stop polling `GetDashboardData` per tick and subscribe the hub to the spec-04 telemetry pub/sub (already designed). Polling (client `setInterval` GET) is the fallback only if SSE is blocked by an edge proxy we cannot configure.
- **Cost:** M (new `realtime.go`, map hub, config vars). Risk: low.

### 12.2 Export strategy: CSV + PDF
- **Decision:** Server-side **CSV** via `encoding/csv` (currently absent repo-wide — must be imported new) with UTF-8 BOM, and **PDF** via `github.com/go-pdf/fpdf` reusing the `GenerateInvoicePDF` pattern into a shared `GenerateReportPDF` (`internal/pdf/report_pdf.go`).
- **Tradeoff:** CSV is cheap and streamable; render it server-side with `EXPORT_MAX_ROWS` cap + `Link` next-cursor. PDF is CPU-heavy per page — only render **summary / bounded tables** (revenue.pdf summary+monthly; pending-payments.pdf outstanding list), never the full 50k-row CSV equivalent.
- **Scope correction:** v1 claimed "no CSV anywhere" — TRUE; there is no `encoding/csv` in the repo, so CSV is net-new, not a fix. PDF exists only for invoices and is not wired to reports (§0.7 confirmed).
- **Cost:** CSV L (handlers+routes+helpers+col maps), PDF M (helper+2 routes). Risk: low.

### 12.3 PWA scope
- **Decision:** Ship a **progressive-enhancement** PWA: `manifest.webmanifest` + a cache-first service worker for `/static`, network-first for HTML pages, gated by `PWA_ENABLED`.
- **Tradeoff:** SW caching can serve stale data pages. Mitigation: network-first for `text/html`, cache-first only for `/static/*`; never cache authenticated API/JSON. Do **not** build offline-first data access — fleet data is sensitive and must stay live.
- **Scope:** Shell + static asset caching only. No offline booking/editing.
- **Cost:** S (2 static assets + 2 `layout.html` edits + config field). Risk: low.

---

## 13. File list (exact paths)

### Create
- `internal/templates/vehicle_list_table.html`
- `internal/templates/customer_list_table.html`
- `internal/templates/invoice_list_table.html`
- `internal/templates/driver_list_table.html`
- `internal/templates/user_list_table.html`
- `internal/templates/route_list_table.html`
- `internal/templates/payment_list_table.html`
- `internal/templates/trip_list_table.html`        (NEW — added by PE QA; `trip_list.html` has a table, no fragment)
- `internal/templates/audit_logs_list_table.html`  (NEW — added by PE QA; `audit_logs_list.html` has a table, no fragment)
- `internal/templates/map.html`
- `internal/templates/settlement_list.html`
- `internal/templates/settlement_view.html`
- `internal/templates/partials/ewaybill_card.html`
- `internal/handlers/realtime.go`            (SSE: `/dashboard/stream`, `/map/stream`)
- `internal/handlers/export.go`              (`writeCSV` + CSV/PDF route handlers)
- `internal/handlers/settlement.go`         (Settlement page handlers)
- `internal/handlers/ewaybill.go`           (EWB card handlers; wraps `integration/ewaybill`)
- `internal/handlers/map.go`                (Live Map page + stream hub subscribe)
- `internal/pdf/report_pdf.go`              (`GenerateReportPDF`)
- `internal/static/js/map.js`               (Leaflet + EventSource)
- `internal/static/js/sw.js`                (service worker)
- `internal/static/manifest.webmanifest`    (PWA manifest)
- `internal/static/icons/icon-192.png`, `internal/static/icons/icon-512.png` (placeholders)

### Modify
- `internal/templates/layout.html`           (§4.7 manifest link + SW; §4.8 `can` for User Mgmt + new nav links; `{{if .PWAEnabled}}`)
- `internal/templates/dashboard.html`        (§4.2 SSE signal targets + EventSource)
- `internal/templates/report_revenue.html`   (§4.4 export buttons)
- `internal/templates/report_trips.html`     (§4.4 export buttons + refresh)
- `internal/templates/report_drivers.html`   (§4.4)
- `internal/templates/report_vehicles.html`  (§4.4)
- `internal/templates/report_customers.html` (§4.4)
- `internal/templates/report_pending_payments.html` (§4.4 CSV + PDF)
- `internal/templates/reports_index.html`    (§4.4 buttons)
- `internal/templates/vehicle_list.html`     (§4.1 wrap table in partial)
- `internal/templates/booking_list.html`     (PE QA: wire existing `booking_list_table.html` partial — it is currently unreferenced)
- `internal/templates/customer_list.html`    (§4.1)
- `internal/templates/invoice_list.html`     (§4.1)
- `internal/templates/driver_list.html`      (§4.1)
- `internal/templates/user_list.html`        (§4.1)
- `internal/templates/route_list.html`       (§4.1)
- `internal/templates/payment_list.html`     (§4.1)
- `internal/templates/trip_view.html`        (§4.6 EWB card)
- `internal/templates/driver_edit.html`      (§4.8 wire `field.html`)
- `internal/templates/vehicle_edit.html`     (§4.8 wire `field.html`)
- `internal/templates/partials/page_header.html` (use in list pages)
- `internal/templates/partials/empty_state.html`  (use in `*_list_table.html` else)
- `internal/templates/partials/stat_card.html`   (use in settlement + dashboard)
- `internal/handlers/app.go`                 (§4.8 `DownloadFile` role→`can`; §6 `PageData.PWAEnabled`; optional delete §11 dead branch 407-411)
- `internal/handlers/reports.go`             (§2.3/§2.4 mount export routes)
- `internal/handlers/dashboard.go`           (§2.1 mount `Stream`)
- `internal/config/config.go`                (§6 new env vars + fields)
- `internal/static/js/router.js`             (§5.4 reactivity re-bootstrap + skip opts)
- `cmd/server/main.go`                       (§2 mount `/map`, `/settlements`, `/dashboard/stream`, `/reports/*.csv|.pdf`, `/manifest.webmanifest`, `/sw.js`; `735`/`770` area)
- `internal/handlers/template_render_test.go`(§7.1/§7.2 new pages)
- `docs/tech-specs/00-migration-ownership-index.md` (no new slot — note "UI completeness spec 12: reads 04/07/08")

### No-op / reference only
- `internal/service/driver_settlement_service.go` (consume, do not edit)
- `internal/integration/ewaybill/client.go` (consume, do not edit)
- `internal/mqttservice/mqtt.go` (consume telemetry shape, do not edit)
- `docs/tech-specs/04-live-map-share-maintenance.md` (reference for map SSE hub)
