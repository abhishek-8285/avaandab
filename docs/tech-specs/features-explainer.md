# Tech Spec — Public Feature Explainer Pages (`/features/<slug>`)

**App:** Avandab (MVTMS) transport platform · Go + chi
**Type:** Static marketing/help content, login-free
**Status:** Implemented

---

## 1. Summary
Add 16 public, SEO-friendly explainer pages at `/features/<slug>` so anonymous visitors who click a feature link in the public header/footer learn what the feature does instead of being redirected to `/login`. One shared, data-driven template renders all 16 from a Go content registry. Existing auth-gated app routes are untouched.

## 2. Context (verified file:line)
- Public routes group: `cmd/server/main.go:497`. Legal pages end at `:556`.
- `middleware.RequireAuth` gate: `cmd/server/main.go:560` — all feature routes live after this.
- Public nav links to those routes: `internal/templates/partials/public_header.html:27-46`.
- Public footers: `internal/templates/home.html` and `internal/templates/contact.html`.
- Standalone page pattern: `App.PolicyPage` at `internal/handlers/app.go:567` (looks up a named template, renders with `{Version: AppVersion}`).
- Templates parsed from disk via `ParseGlob` (`app.go:185`, `:190`); standalone pages are root templates (no `{{define}}`), e.g. `privacy.html`.

## 3. Decisions
- **16 slugs** (see §5). Excluded utilities: `/user/onboard`, `/files/{id}`, `/ops/dashboard`, `/agent-actions`, `/profile`, `/change-password`. `/trips/{id}/deliver-pod` described inside `trips`.
- **Sub-pages folded** into parent Capabilities (no extra URLs).
- **Shared template** `feature.html` + Go registry (no 16 duplicate HTML files).
- **Shared footer partial** `partials/footer.html` used by `home.html`, `contact.html`, and `feature.html` so links live in one place.

## 4. Architecture / Data Flow
```
GET /features/{slug}
  → chi route in PUBLIC group (before RequireAuth)
  → App.FeaturePage:
       slug = chi.URLParam(r,"slug")
       fc, ok = GetFeature(slug)            // features.go
       if !ok → http.NotFound(w,r)  (native 404, NOT /login)
       render template "feature.html" with {Version: AppVersion, Feature: fc}
  → browser: hero + 4 sections + related links; cookie-consent + loader.js active
```

## 5. The 16 Slugs
`dashboard, trips, routes, bookings, vehicles, drivers, customers, invoices, payments, reports, audit-logs, settings, users, company, kharcha, assistant`

## 6. Files
### 6.1 `internal/handlers/features.go` (NEW)
Holds `FeatureContent` struct, `featureRegistry` map (16 entries), and `GetFeature(slug)`.

### 6.2 `internal/templates/feature.html` (NEW, standalone root template)
Head mirrors `privacy.html`. Full marketing layout (Tailwind): `public_header.html` → gradient hero (breadcrumb, eyebrow badge, icon chip, h1, tagline, audience, CTAs, decorative product mockup) → "What it is" lead → Capabilities cards grid → Benefits grid → numbered Steps → "Who it's for" band → Use cases → FAQ accordion (`<details>`) → Related feature cards → green CTA band → `footer.html` partial → `cookie-consent` partial → `loader.js`. Tailwind rebuilt via `make css` / `npx @tailwindcss/cli` after any class changes.

### 6.3 `internal/handlers/app.go` (ADD `FeaturePage`)
Mirrors `PolicyPage`; looks up `feature.html`; 404 on unknown slug; sets CDN-friendly `Cache-Control`.

### 6.4 `cmd/server/main.go` (route + sitemap)
Route added in the public group (after `:556`, before `:559`):
```go
r.Get("/features/{slug}", app.FeaturePage)
r.Get("/features", func(w,r){ http.Redirect(w,r,"/",http.StatusSeeOther) })
```
Sitemap: 16 `<url>` entries added (`/features/<slug>`, `priority=0.3`).

### 6.5 `internal/templates/partials/footer.html` (NEW shared partial)
`.footer-top` + `.footer-bottom` with corrected `/features/<slug>` links, 4 new links (kharcha, users, company, assistant), and corrected social icons. Replaces the inline footer in `home.html` and `contact.html`.

### 6.6 Link repointing
- `public_header.html` nav (10 links) → `/features/<slug>`.
- `home.html` + `contact.html` footers → replaced by `{{template "footer.html" .}}`.
- `/login`, `/register`, `/contact-us`, `/contact-us#track-status-section` unchanged.

### 6.7 `internal/handlers/features_test.go` (NEW)
16 slugs → 200 + contains Title; unknown → 404; `/features` → 302 → `/`.

## 7. Icon Mapping (all confirmed present in the self-hosted subset)
dashboard→`grid_view`, trips→`commute`, routes→`route`, bookings→`description`, vehicles→`directions_bus`, drivers→`badge`, customers→`groups`, invoices→`receipt_long`, payments→`payments`, reports→`monitoring`, audit-logs→`history`, settings→`settings`, users→`manage_accounts`, company→`account_balance`, kharcha→`account_balance_wallet`, assistant→`support_agent`.

## 8. Content (final copy)
See `internal/handlers/features.go` for the authoritative copy of all 16 entries. `FeatureContent` carries: Slug, Title, Icon, Eyebrow, Tagline, Audience, Summary (meta description), Lead, WhatItIs, Capabilities (`[]FeatureCapability{Icon,Title,Text}`), Benefits (`[]FeatureBenefit{Icon,Text}`), Steps, UseCases, WhoFor, FAQ (`[]FAQItem{Question,Answer}`), Related. `FeaturePage` also passes `RelatedFeatures` (resolved `FeatureContent` per related slug) for the related-cards section. Icon ligatures must come from the verified self-hosted Material Symbols subset.

## 9. Verification Runbook
1. `go build ./cmd/server/...`
2. `go test ./internal/handlers/ -run 'TestAllTemplatesRenderCleanly|Feature' -count=1`
3. Serve :8080; `curl` checks: each `/features/<slug>` 200; unknown 404; sitemap contains 16 `/features/`; headless Chrome each slug → 200, no console errors, `window.Loader` defined.
4. Manual: open `/`, click header nav + footer items → land on `/features/<slug>`, never `/login`.

## 10. Deployment / Rollback
- Templates parsed from disk at startup; `deploy_avandab.sh` already ships `internal/templates` + `internal/static`. No DB/migration.
- Rollback: revert `features.go`, `feature.html`, `footer.html`, `app.go`, `main.go`, `public_header.html`, `home.html`, `contact.html`; delete `features_test.go`.
