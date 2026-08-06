# Design Review: Booking Module — MVTMS

## Summary

The Booking module has a server-rendered web UI attached via `internal/handlers/bookings.go` with three templates (`booking_list.html`, `booking_view.html`, `booking_edit.html`) plus a shared `layout.html`. The REST API is handled separately by `internal/booking/presentation/api/handlers/booking_handler.go` with per-route Casbin RBAC. All screens are wired and reachable.

**Score: 27 / 50**

## Heuristics

| Lens | Score | Notes |
|------|-------|-------|
| First impression | 4 / 10 | Clear sidebar nav, but generic Tailwind gray/blue palette — no visual distinction. |
| Hierarchy | 5 / 10 | Status badges and table structure are good, but heading sizes and visual weight are flat. |
| Color voice | 4 / 10 | Relies entirely on Tailwind defaults (`gray-50`, `blue-600`, `red-600`, `green-600`). No brand hue or tinted neutrals. Looks like boilerplate SaaS. |
| Type voice | 5 / 10 | `font-sans` (system sans), no custom type scale. Body copy is readable at 14–16px but no deliberate hierarchy. |
| Interaction feel | 9 / 10 | Good focus rings, 44px touch targets, hover states on buttons and table rows. |

## Evidence

### Screens Attached & Wired (Verified)

| Screen | Route | Template | Status |
|--------|-------|----------|--------|
| Booking List | `GET /bookings` | `booking_list.html` | Attached |
| New Booking | `GET /bookings/new` | `booking_edit.html` | Attached |
| Create Booking | `POST /bookings/new` | — (redirect) | Attached |
| Booking View | `GET /bookings/{id}` | `booking_view.html` | Attached |
| Edit Booking | `GET /bookings/{id}/edit` | `booking_edit.html` | Attached |
| Update Booking | `POST /bookings/{id}/edit` | — (redirect) | Attached |
| Confirm | `POST /bookings/{id}/confirm` | — (redirect) | Attached |
| Cancel | `POST /bookings/{id}/cancel` | — (redirect) | Attached |
| **Complete** | — | — | **Missing in web UI** |
| Delete | `POST /bookings/{id}/delete` | — (redirect) | Attached |

### Findings

1. **BLOCKER — Missing Datastar fragment template**
   - `internal/handlers/bookings.go:80` calls `h.renderFragment(w, "booking_list_table.html", ...)` for AJAX/Datastar requests.
   - `internal/templates/booking_list_table.html` does not exist on disk.
   - Impact: any Datastar-powered table refresh (e.g., after create/cancel) will 500 with "template not found".

2. **BLOCKER — "Complete" action missing from web UI**
   - The `BookingHandlers` struct has `completeUC` wired (line 26) but `Routes()` does not register a `/complete` route.
   - The REST API has `POST /api/v1/bookings/{id}/complete` (with `bookings:update` permission) but the web UI has no way to complete a confirmed booking.
   - The `booking_view.html` template shows status badges for all states but no "Complete" button.

3. **CRITICAL — External CSS/JS dependencies in layout**
   - `layout.html:8` loads Tailwind via `cdn.tailwindcss.com` — this is a dev-only CDN build, not suitable for production (300KB+ runtime compilation, no PurgeCSS).
   - `layout.html:6` loads Google Material Icons from `fonts.googleapis.com`.
   - No offline/fallback for these resources.

4. **HIGH — Generic color palette**
   - Uses Tailwind defaults exclusively: `bg-gray-50` surfaces, `blue-600` buttons, `red-600`/`green-600` status badges.
   - No tinted neutrals, no brand hue — the UI would be indistinguishable from any other Tailwind CRUD app.

5. **MEDIUM — No dark mode**
   - Layout hardcodes `bg-gray-50`, `bg-white`, `text-gray-800` — no `dark:` variants or `prefers-color-scheme` support.

6. **LOW — No page description / SEO meta**
   - `layout.html` has no `<meta name="description">` or structured title beyond `{{.Title}} - MVTMS`.

## Recommendations

| Priority | Recommendation | Mode |
|----------|----------------|------|
| Blocker | Create `booking_list_table.html` fragment template with the same table structure as `booking_list.html` tbody | `surface` |
| Blocker | Add `POST /bookings/{id}/complete` route + "Complete" button on `booking_view.html` (shown for `confirmed` status) | `surface` |
| High | Replace Tailwind CDN with a built CSS pipeline (`tailwindcss` CLI or PostCSS) and add a `brand` color to `tailwind.config.js` | `redesign` |
| High | Define a color system with tinted neutrals and a semantic accent (see references/color.md) | `recolor` |
| Medium | Add `dark:` variants to layout and templates; add `theme` toggle | `responsive` |
| Low | Add SEO meta tags to `layout.html` | `voice` |
