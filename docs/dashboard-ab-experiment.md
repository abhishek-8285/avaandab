# Dashboard A/B Experiment (dashboard_v2)

## What
Server-side A/B test comparing the legacy dashboard (Variant A, control) with
a redesigned operations dashboard (Variant B, treatment: KPI cards with delta
chips, Chart.js charts, alert feed, drill-down rows).

## Mechanics
- Framework: `internal/experiments/` — deterministic assignment via
  `sha1(tenant_id + ":" + user_id) % 100 < rollout` → Variant B.
- Config: `EXPERIMENT_ROLLOUT` (0-100, default 0 = control-only) and
  `EXPERIMENT_FORCE_VARIANT` (A|B, QA override) in `config.Experiment`.
- Per-request override: `?variant=a` / `?variant=b` query param (case-insensitive).
- Variant chosen in `DashboardHandlers.Index` (internal/handlers/dashboard.go)
  and passed to the template as `DashboardVariant`; template branches
  `{{if eq .DashboardVariant "B"}}` in `internal/templates/dashboard.html`.
- Event logging: `experiment_events` table (migration 00039), recorder on `App`
  (`App.Experiments`). Events: `dashboard_view` (server-side on page render,
  async with detached context — request ctx is cancelled by the time the
  goroutine runs) and `dashboard_click` (browser POST to `/dashboard/event`,
  keepalive fetch, CSRF-strict requires Origin header — browsers send it).
- Chart data flows to the browser via `window.__DASHBOARD_CHARTS__` rendered by
  the `json` template func (escapes `</` → `<\/` for script context).
  `internal/static/js/dashboard-charts.js` renders 3 charts (revenue 30d line,
  trips-by-status doughnut, bookings 30d bar) and hides canvases showing
  empty-state divs when there is no data.

## Data added
- sqlc queries: `GetRevenueByDay`, `CountBookingsByDay`, `GetOverdueTrips`,
  `GetIdleVehicles`; `DashboardData` extended with `RevenueSeries`,
  `RevenueByDay`, `BookingsByDay`, `StatusCounts`, `DeltaYesterday`,
  `OverdueTrips`, `IdleVehicles` (computed in the existing 3s cache + errgroup).
- Chart.js vendored at `internal/static/js/chart.umd.min.js` (v4.4.9, MIT,
  `CHARTJS-LICENSE.txt` alongside) — do NOT npm-install it; assets are served
  raw with `?v=` cache busting.

## sqlc gotcha (IMPORTANT)
Never wrap positional `?` params in SQL functions expecting them to be named:
sqlc v1.31.1 rewrites `sqlc.arg('x')` inside `LOWER(TRIM(...))` into `?2`/`?3`
placeholders and reorders args, silently corrupting queries. Use sqlite
`:name` syntax (`LOWER(TRIM(:source))`), which maps positionally and stably to
`Source`/`Destination` params across sqlc versions. The committed HEAD stamp
"sqlc v1.31.1" on routes.sql.go was hand-patched and does not reproduce with a
real v1.31.1 run — regenerate with `:name` form instead of trusting the stamp.

## Tests
- `internal/experiments/experiments_test.go`: rollout bounds, stability,
  tenant scoping, force override, rough balance.
- `internal/handlers/template_render_test.go`: dashboard.html renders under
  both variants (B case uses `DashboardVariant: "B"` + full ChartData/Stats).
- `test/dashboard_ab.test.js` (Playwright): registers a fresh user, asserts
  variant B elements/empty states, and the `?variant=a` legacy override.
  webServer in playwright.config.ts runs with `EXPERIMENT_ROLLOUT=100`.
  Registration is race-prone: emails must be unique per test (ms timestamp +
  random suffix) and tests run in parallel workers against a shared DB.

## Variant B design
Variant B uses the SAME design language as the control dashboard — plain
`bg-white rounded-lg shadow` cards with `text-stat` colored numbers
(blue/orange/green/red/purple/indigo/yellow/teal) and `text-subsection`
gray-800 section headers, `bg-gray-50` table heads. No kharcha-style colored
borders/icon chips, no experiment badge in the UI (assignment is invisible to
users). Additions on top of the control layout: a delta chip under Today's
Trips (▲/▼ vs yesterday), a 3-column charts row (revenue 30d line, trips-by-
status doughnut, bookings 30d bar), an alert feed row (overdue trips red /
idle vehicles orange / pending payments yellow counts), clickable trip rows →
`/trips/{id}`, idle vehicles → `/vehicles`, and empty states for charts and
alert lists.

## Material icons gotcha (IMPORTANT)
`internal/static/fonts/material-symbols-outlined.woff2` is a 96-glyph SUBSET
of Material Symbols. Icons NOT in the subset render as raw text (e.g.
`local_shipping`, `garage`, `directions_car`, `block`). Only use glyphs
verified present (route, warning, payments, person, check_circle, commute,
directions_bus, cancel, etc. — check with fontTools `getBestCmap()` before
adding new icons anywhere in the app). The dashboard variant B deliberately
uses no material icons for this reason.