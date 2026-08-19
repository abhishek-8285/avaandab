# Event Bus, Multi-Tenancy, Booking & Trip-POD Hardening — Implementation Spec v1

Status: ready
Depends-on:
  - `docs/tech-specs/00-migration-ownership-index.md` (reserve 00053/00054/00055)
  - `db/migrations/00020_create_outbox_events.sql` (outbox base table)
  - `db/migrations/00032_kharcha_and_epod.sql` (pod columns live here)
  - `db/migrations/00042_*.sql` (`company_config`) — **CORRECTION:** `00042` is *reserved* by spec 02 (geofence engine) but is **NOT present** in `db/migrations/` at head `00039_experiments.sql`. Do **not** assume the table or a `tenant_id` FK exists. Scope booking tenancy by a plain `tenant_id TEXT` column; defer any FK to `company_config` until `00042` actually merges.
Migration owner:
  - `db/migrations/00053_eventbus_outbox.sql`   (event bus + outbox correction)
  - `db/migrations/00054_booking_hardening.sql` (reverse_fare, status history, tenant FK)
  - `db/migrations/00055_trip_pod.sql`          (pod_otp_hash, converter/aggregate fields, conflict)

---

## 0. Verified ground truth (file:line proofs)

**0.1 Event bus is duplicated and disconnected**
- `internal/service/service.go:75` — `baseService{... events: events.NewInMemoryBus()}` → creates **bus #2** for automation handlers.
- `cmd/server/main.go:819` — `eventBus := events.NewInMemoryBus()` → **bus #1** for outbox relay + founder.
- `cmd/server/main.go:821` — `founderSvc.RegisterEventHandlers(eventBus)` (bus #1).
- `cmd/server/main.go:825-826` — `outboxRelay := outbox.NewRelay(database, eventBus, logger); go outboxRelay.Run(ctx)` (bus #1).
- `internal/service/service.go:103` — `s.Founder.RegisterEventHandlers(bs.events)` (bus #2).
- `internal/service/service.go:111-164` — `initEventHandlers()` subscribes on `s.Bookings.events` (bus #2).
- **Consequence:** events published from `service` layer (bus #2) never reach the outbox relay or founder (bus #1), and outbox-sourced events never reach automation handlers (bus #2). They are two independent buses.

**0.2 Event-type name drift is SECONDARY — NOT the root cause**
- `internal/service/booking_service.go:177` — in-process publish uses `Type: "BookingConfirmed"`.
- `internal/service/service.go:115` — subscriber listens for `"BookingConfirmed"`.
- **These MATCH.** In-process automation (auto trip-creation from a confirmed booking) therefore fires correctly *today*; the name is not what breaks it.
- `internal/booking/domain/aggregate/booking_aggregate.go:195` — `BookingConfirmedEvent` is the Go type; it is written to the outbox by `internal/booking/infrastructure/persistence/sql/booking_repository.go:91` (`SaveEvents(..., b.Events())`). `getEventTypeName` (`outbox.go:67`) stores the Go type name `"BookingConfirmedEvent"` (not `"BookingConfirmed"`).
- `internal/shared/outbox/relay.go:112` — relay republishes `events.Event{Type: e.eventType, ...}` verbatim → `"BookingConfirmedEvent"` onto **bus #1**.
- The only subscriber (`service.go:115`) and the founder handlers both register for `"BookingConfirmed"` on **bus #2** / **bus #1** respectively — so the outbox-sourced name drifts from every subscriber AND it lands on the disconnected bus #1.
- **Conclusion:** the name drift is a real **latent** bug, but it is masked by — and secondary to — the duplicated-bus defect (0.1). Fixing only the name would NOT repair event flow, because the relay (bus #1) and the automation subscribers (bus #2) are different buses. A canonical catalog (`events.go` + `EventTypeOf`, see 5.1) is still warranted to kill the drift, but the *headline* root cause is the duplicated bus.

**0.3 Errors swallowed in bus**
- `internal/events/bus.go:43-52` — `Publish` loops handlers calling `h(ctx, e)` and discards the returned error with `_ = h(ctx, e)`. A failing handler (e.g. trip auto-creation) is invisible.
- Handlers that call `s.Invoices.GenerateInvoiceFromTrip` / `s.Settlements.CreateSettlementForTrip` return values are themselves discarded with `_ =` (`service.go:142`, `service.go:161`).

**0.4 Unsubscribe bug (index-based removal)**
- `internal/events/bus.go:56-73` — `Subscribe` captures `idx := len(b.subs[eventType]) - 1` and the returned closure removes by **index** (`b.subs[eventType] = append(subs[:idx], subs[idx+1:]...)`). If another handler for the same type is removed first, every later stored index is stale and the wrong handler is removed. Must remove by **handler identity** (slice filter), not index.

**0.5 No event catalog**
- `internal/events/events.go` — **does not exist** (verified: `File not found`). There is no single source of event-type constants; callers hand-type strings (`"BookingConfirmed"`, `"TripCompleted"`, `"TripDelivered"`, `"TripStarted"`...), guaranteeing drift.

**0.6 Booking: Confirmed editable/deletable; draft dead; reverse fare ignored**
- `internal/booking/domain/aggregate/booking_aggregate.go:139-170` — `Update` allows edits whenever status is NOT `cancelled`/`completed`, i.e. **Confirmed can be edited**. Business rule (`docs/business-rules/booking.md:42-43`) says only `Pending` is editable → Confirmed must be **immutable**.
- `internal/booking/domain/aggregate/booking_aggregate.go:85-102` — `Confirm` transitions `Pending→Confirmed`. No draft concept exists in the aggregate (only `pending/draft` constants; `NewBookingAggregate` always starts at `pending`). So "draft dead" = the `BookingDraft` state is never used and the create flow never enters draft; `Confirm` accepts only pending, which is correct, but the **deletion** path is wrong:
- `internal/domain/booking/entity.go:63-67` — `CanDelete()` permits `pending OR confirmed`. Business rule (`docs/business-rules/booking.md:48`) says only `Pending`/`Draft` deletable → **Confirmed must NOT be deletable**.
- `internal/service/booking_service.go:56-59` — `CreateBooking` sets `req.Price = route.StandardFare` when price ≤ 0 and **never consults reverse fare**.
- `internal/agent/tools.go:105-117` — agent `get_quote` DOES consult `r.ReverseStandardFare` when `rev` is true (reversed direction). → **parity gap**: service-created bookings and agent quotes compute fare differently.

**0.7 Trip POD: full e-POD dropped; legacy entity lacks reached_pickup; conflicting guards**
- `internal/service/trip_service.go:387-430` — `DeliverTripWithPOD(ctx, id, podURL string)` only stores `delivered.PODURL` (single URL). The richer `DeliverWithPODRequest` (consignee name/phone, signature URL, notes, OTP) is **not persisted** (see `trip_service.go:446-472` which only forwards `podURL` to the legacy method).
- `internal/trip/infrastructure/persistence/sql/converters/trip_converter.go:11-31` — `SQLTripModel` has **no POD columns** (`pod_photo_url`, `pod_signature_url`, `pod_consignee_name`, `pod_consignee_phone`, `pod_otp_verified`, `pod_captured_at`, `pod_lat`, `pod_lng`, `pod_notes`) even though `db/migrations/00032` added them to the `trips` table. They are silently lost on read/write.
- `internal/domain/trip/entity.go:32-41` — legacy `TripStatus` const block **omits `reached_pickup`** (DDD aggregate has it: `trip_aggregate.go:18`).
- **Conflicting guards:** `internal/domain/trip/entity.go:69-74` — `CanDeliver()` allows `TripStarted || TripInTransit`; `internal/trip/domain/aggregate/trip_aggregate.go:181-195` — `Deliver()` requires `TripInTransit` only, and `ReachPickup` requires `TripStarted` then `StartTransit` requires `reached_pickup`. Two incompatible state machines. Canonical = DDD aggregate.
- `cmd/server/main.go:797` — `r.Post("/trips/{id}/deliver-pod", app.Kharcha.DeliverWithPOD)` → the POD entrypoint is **mis-homed** on the Kharcha service (`internal/handlers/kharcha.go`). It must be re-homed to the Trip handlers/service.
- `db/migrations/00032:17` — `pod_otp_verified INTEGER` exists but there is **no OTP issuance or hash** anywhere; `OTPVerified` is a client flag only (`trip_service.go:457-459`).
- `test/kharcha_epod_test.go:254-272` — POD tests assert **only `Status == delivered`**; consignee/signature/OTP are never persisted or asserted.

**0.8 Hardcoded tenant `"1"`**
- Grep `TenantID: "1"` returns **26** matches across `internal/handlers/{drivers,vehicles,trips,invoices,payments}.go` (4+4+13+3+2). Another match is `internal/middleware/api_auth.go:112`, giving 27 in code; 65 repo-wide including `test/`. Full handler list in section 2.4. Correct derivation exists: `shared.TenantIDFromContext(r.Context())` (`internal/shared/tenant.go:29`), already used in `internal/handlers/bookings.go` and `dashboard.go`.

---

## Verification Log

QA pass — every claim re-verified against source. **Root cause of the disconnected event flow is the duplicated bus (0.1), NOT the event-name mismatch (0.2).**

| # | Claim (spec §) | Verdict | Correction / Evidence | Severity | Effort |
|---|---------------|---------|----------------------|----------|--------|
| 1 | Name mismatch is THE bug (0.2, §1, §5.1, §9.411) | **WRONG** (root-cause) | In-process publish `booking_service.go:177` `"BookingConfirmed"` == subscriber `service.go:115` `"BookingConfirmed"` → MATCH; auto trip-creation works today. Outbox stores `"BookingConfirmedEvent"` (aggregate type via `outbox.go:67`), relay→bus#1, name drifts but is *masked* by bus disconnect. Real cause = duplicated bus. | High (scope) | Low |
| 2 | Event bus duplicated/disconnected (0.1) | CONFIRMED | `main.go:819` bus#1 (relay+founder); `service.go:75` bus#2 (subscribers). Two independent buses. | Critical | M |
| 3 | Errors swallowed in `Publish` (0.3) | CONFIRMED | `bus.go:43-52` `_ = h(ctx,e)`; `service.go:142` & `:161` `_ =` (line is **161**, not 160). | High | S |
| 4 | Unsubscribe by index (0.4) | CONFIRMED | `bus.go:56-73` `idx := len(...) -1` closure. | Medium | S |
| 5 | No event catalog (0.5) | CONFIRMED | `internal/events/` has only `bus.go` (+`bus_test.go`); `events.go` absent. | — | — |
| 6 | Confirmed editable/deletable; reverse fare ignored (0.6) | CONFIRMED | `booking_aggregate.go:150-152` blocks only cancelled/completed; `entity.go:63-67` `CanDelete` allows pending‖confirmed; `booking_service.go:56-59` uses `route.StandardFare`; `agent/tools.go:109` uses `ReverseStandardFare`. | High | M |
| 7 | Trip POD dropped; no converter cols; conflicting guards (0.7) | CONFIRMED | `trip_service.go:387-472` persists only PODURL; `converter:11-31` no POD cols; `entity.go:32-41` omits `reached_pickup`; `entity.go:69-74` `CanDeliver` (started‖in_transit) vs `trip_aggregate.go:181-195` `Deliver` (in_transit only). | High | M |
| 8 | Hardcoded `TenantID:"1"` exists (0.8) | CONFIRMED | present in handlers; correct `TenantIDFromContext` exists (`tenant.go:29`). | High | M |
| 9 | `TenantID:"1"` count = 37 in 5 handler files (0.8) | **WRONG** (count) | actual **26** in those files (4+4+13+3+2); 27 incl `middleware/api_auth.go:112`; 65 repo-wide. | Low (doc) | — |
| 10 | `00042 company_config` already taken; FK to it (Depends-on, §3.2) | **WRONG** | `00042` is *reserved* by spec 02 but **absent** from `db/migrations/` (head `00039`); no `company_config` table/FK exists. Scope by `tenant_id TEXT`; defer FK. | High | S |

### Decisions, Tradeoffs & Cost

**A. Single-bus injection strategy**
- *Decision:* create one `events.NewInMemoryBus()` in `cmd/server/main.go`, inject into `service.NewServices(store, cfg, log, eventBus)` (add a `bus` param), `outbox.NewRelay(database, eventBus, logger)`, and `founderSvc.RegisterEventHandlers(eventBus)`; delete the local `events.NewInMemoryBus()` at `service.go:75`.
- *Tradeoff:* one synchronous bus means a slow or panicking handler blocks the caller goroutine; mitigated by returning errors from `Publish` (5.2) and a per-handler `recover` (§9). All subscribers now share fate — but that is the intended coupling (founder was previously silently dead).
- *Cost:* Low–Medium. Touches the `NewServices` signature (all construction sites) and removes the per-service bus. Risk: founder notifications begin firing for real — verify no double-handling with the in-process path.

**B. Multi-tenant context propagation**
- *Decision:* replace every hardcoded `TenantID:"1"` in handlers (26 sites) with `shared.TenantIDFromContext(r.Context())`; keep the `"1"` fallback in `tenant.go` for unauthenticated/test paths. Add a CI grep gate (`grep -rn 'TenantID: "1"' internal/handlers` → empty).
- *Tradeoff:* handlers must run under auth middleware that sets the tenant; any path bypassing it silently falls back to `"1"`. Queries must include `tenant_id` filters or isolation is cosmetic.
- *Cost:* Medium (26 handler edits + middleware + tests). Risk: cross-tenant reads if a query forgets the filter — covered by `test/tenant_test.go`.

**C. Booking immutability**
- *Decision:* `Confirmed` is immutable (no edit/delete); only `Pending` is editable/deletable; `Cancelled`/`Completed` locked. Keep `Confirmed` cancellable per BR line 31.
- *Tradeoff:* breaks legacy behavior that allowed editing/deleting confirmed bookings; any admin UI doing so must be disabled (§4). Cancel remains allowed (status change, not edit/delete).
- *Cost:* Medium (tighten `booking_aggregate.go:139` `Update` guard + `entity.go:63` `CanDelete` + service checks + `booking_status_history`). Risk: existing confirmed bookings edited manually in prod — document as a breaking change; no data backfill required.

---

## 1. Overview / goal

Unify the two disconnected in-memory event buses into a single injected `EventBus`, introduce a canonical event-type catalog (which also eliminates the *latent* outbox Go-type-name drift from 0.2), fix the unsubscribe bug, surface handler errors, and implement a real dead-letter path in the outbox. Then harden three domains:

1. **Multi-tenancy** — replace every hardcoded `TenantID: "1"` in handlers with `shared.TenantIDFromContext`.
2. **Booking** — make `Confirmed` immutable (no edit/delete), fix the delete guard, add existence + reverse-fare parity through a shared `PricingService`, and record a status-history table.
3. **Trip POD** — persist the full e-POD (consignee, signature, OTP-verified, geo, notes), add OTP issuance + verify, unify on the DDD aggregate, enforce time-window conflict checks, and re-home the `/deliver-pod` route to the Trip service.

**Non-goals:** replacing SQLite with Postgres, building a real broker (Kafka/NATS), GSTN/EWB integration, RBAC role changes (handled by spec 08 auth hardening 00056). Outbox remains SQLite-backed; dead-letter is a column/state, not a separate infra.

---

## 2. API contract

### 2.1 Event bus (internal change, no new HTTP route)
All publishers/handlers use constants from `internal/events/events.go`. Example published envelope:

```json
{ "type": "booking.confirmed", "payload": { "booking_id": "bk_...", "tenant_id": "1", "occurred_at": "2026-08-19T12:00:00Z" } }
```

### 2.2 Booking endpoints (existing routes unchanged; behavior changes)
- `POST /bookings` — create. Request already validated. **New:** when `price<=0`, compute via `PricingService.Quote(routeID, reverse=false)` instead of raw `route.StandardFare`. Returns `201` with booking JSON.
- `POST /bookings/{id}/confirm` — `bookings:approve`. On success publishes `booking.confirmed`.
- `POST /bookings/{id}/edit` — `bookings:update`. **403** if status is `confirmed`/`cancelled`/`completed` (immutable). Legacy allowed confirmed edits.
- `POST /bookings/{id}/delete` — `bookings:delete`. **403** if status is not `pending` (was: allowed confirmed).
- `GET /bookings/{id}/history` — `bookings:read`. New endpoint returning `booking_status_history` rows (section 3, 00054).

```json
// GET /bookings/{id}/history → 200
[ { "status_from": "pending", "status_to": "confirmed", "changed_by": "user_123", "changed_at": "2026-08-19T12:01:00Z", "reason": "auto" } ]
```

Error codes: `400 invalid_state`, `403 forbidden`, `404 not_found`, `422 validation_error`.

### 2.3 Trip POD endpoints
- `POST /trips/{id}/deliver-pod` — **re-homed** from Kharcha to Trip handlers (`app.Trips.DeliverWithPOD`). `trips:update`.
  Request JSON:
  ```json
  {
    "pod_photo_url": "https://.../pod.jpg",
    "signature_url": "https://.../sig.png",
    "consignee_name": "Mahesh Kumar",
    "consignee_phone": "9876543210",
    "notes": "Left at reception",
    "otp": "482913"
  }
  ```
  - If `otp` present: server verifies against `pod_otp_hash`; on mismatch → `401 otp_mismatch`. On success sets `pod_otp_verified=1` and records `pod_verified_at`.
  - At least one of `pod_photo_url` / `signature_url` required → else `422 pod_evidence_required`.
  - Status must be `in_transit` (canonical aggregate) → else `409 invalid_state` (`cannot deliver: trip must be in_transit`).
  - Response `200`: `{ "trip_number": "TR-0001", "status": "delivered" }`.
- `POST /trips/{id}/issue-otp` — new. `trips:update`. Issues a 6-digit OTP, stores `pod_otp_hash` (bcrypt), sets `pod_otp_issued_at`. Response `200`: `{ "issued": true }`. (OTP delivered out-of-band; never returned in body.)
- `GET /trips/{id}` — already exists; response now includes full POD block:
  ```json
  {
    "id": "trip_...", "status": "delivered",
    "pod": {
      "photo_url": "https://.../pod.jpg", "signature_url": "https://.../sig.png",
      "consignee_name": "Mahesh Kumar", "consignee_phone": "9876543210",
      "otp_verified": true, "captured_at": "2026-08-19T18:00:00Z",
      "lat": 28.61, "lng": 77.20, "notes": "Left at reception"
    }
  }
  ```

### 2.4 Tenant resolution (no new routes)
Every handler that currently passes `TenantID: "1"` must instead compute:
```go
tenantID := string(shared.TenantIDFromContext(r.Context()))
```
Affected handlers/lines (replace the literal):
- `internal/handlers/drivers.go:59,119,151,167,216,257,266`
- `internal/handlers/vehicles.go:58,125,154,169,216,255,264`
- `internal/handlers/trips.go:88,153,169,176,189,220,299,316,333,347,361,375,389,403,411,419,439`
- `internal/handlers/invoices.go:56,91,139`
- `internal/handlers/payments.go:54,136,177`

---

## 3. DB contract (goose, Up + Down)

### 3.1 `db/migrations/00053_eventbus_outbox.sql`
Extends `outbox_events` (created in 00020) with retry/dead-letter columns. No table drop.

```sql
-- +goose Up
ALTER TABLE outbox_events ADD COLUMN status TEXT NOT NULL DEFAULT 'pending'
  CHECK (status IN ('pending','published','failed','dead'));
ALTER TABLE outbox_events ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE outbox_events ADD COLUMN last_error TEXT;
ALTER TABLE outbox_events ADD COLUMN dead_lettered_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox_events(status);

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_status;
ALTER TABLE outbox_events DROP COLUMN dead_lettered_at;
ALTER TABLE outbox_events DROP COLUMN last_error;
ALTER TABLE outbox_events DROP COLUMN attempts;
ALTER TABLE outbox_events DROP COLUMN status;
```

### 3.2 `db/migrations/00054_booking_hardening.sql`
Adds reverse-fare parity column, tenant scoping, and status history. (Route already has `reverse_standard_fare` per `internal/agent/tools.go:110`; if not present on `routes`, also add it here — VERIFY in section 11.) **No FK to `company_config`:** `00042` is not present at head `00039`, so scope tenancy with a plain `tenant_id TEXT` column and revisit the FK after `00042` merges.

```sql
-- +goose Up
-- 1) Booking immutability + tenant scoping + reverse-fare parity tracking
ALTER TABLE bookings ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '1';
ALTER TABLE bookings ADD COLUMN fare_source TEXT NOT NULL DEFAULT 'standard'
  CHECK (fare_source IN ('standard','reverse','manual'));
CREATE INDEX IF NOT EXISTS idx_bookings_tenant ON bookings(tenant_id);
-- NOTE: FK to company_config(tenant_id) DEFERRED — 00042 not merged at head 00039.
-- Add the FK only in a later migration once company_config(tenant_id) exists.

-- 2) Status history
CREATE TABLE booking_status_history (
  id            TEXT PRIMARY KEY,
  booking_id    TEXT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  tenant_id     TEXT NOT NULL DEFAULT '1',
  status_from   TEXT,
  status_to     TEXT NOT NULL,
  changed_by    TEXT,
  reason        TEXT,
  created_at    DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bsh_booking ON booking_status_history(booking_id);

-- +goose Down
DROP INDEX IF EXISTS idx_bsh_booking;
DROP TABLE booking_status_history;
DROP INDEX IF EXISTS idx_bookings_tenant;
ALTER TABLE bookings DROP COLUMN fare_source;
ALTER TABLE bookings DROP COLUMN tenant_id;
```

### 3.3 `db/migrations/00055_trip_pod.sql`
Adds OTP hash + issue/verify timestamps to the POD columns already present (00032: `pod_photo_url`, `pod_signature_url`, `pod_consignee_name`, `pod_consignee_phone`, `pod_otp_verified`, `pod_captured_at`, `pod_lat`, `pod_lng`, `pod_notes`).

```sql
-- +goose Up
ALTER TABLE trips ADD COLUMN pod_otp_hash TEXT;
ALTER TABLE trips ADD COLUMN pod_otp_issued_at DATETIME;
ALTER TABLE trips ADD COLUMN pod_verified_at DATETIME;
ALTER TABLE trips ADD COLUMN pod_issued_by TEXT;
-- tenant scoping already exists on trips via tenant_id (00042/earlier); add index if missing
CREATE INDEX IF NOT EXISTS idx_trips_pod_otp ON trips(pod_otp_verified);

-- +goose Down
DROP INDEX IF EXISTS idx_trips_pod_otp;
ALTER TABLE trips DROP COLUMN pod_verified_at;
ALTER TABLE trips DROP COLUMN pod_otp_issued_at;
ALTER TABLE trips DROP COLUMN pod_otp_hash;
```

---

## 4. UI

- `web/templates/trip_view.html` (or equivalent) — render the new `pod` block (photo thumbnail, signature thumbnail, consignee name/phone, OTP-verified badge, captured-at, geo link).
- `web/templates/trip_edit.html` — add **"Issue OTP"** button calling `POST /trips/{id}/issue-otp`.
- Driver mobile POD form (existing `/trips/{id}/deliver-pod` view) — add fields: `pod_photo_url`, `signature_url`, `consignee_name`, `consignee_phone`, `notes`, `otp`. Client-side validation: require photo OR signature; if OTP was issued, require `otp` field.
- `web/templates/booking_view.html` — render a **Status History** section from `GET /bookings/{id}/history`.
- `web/templates/booking_edit.html` — disable all editable fields when status is `confirmed`/`cancelled`/`completed` (server also rejects, section 5).
- RBAC resources unchanged (`bookings:*`, `trips:*`). No new RBAC rows required by this spec.

---

## 5. Business logic + state machines

### 5.1 Single injected bus + catalog
Create `internal/events/events.go`:
```go
package events

// Canonical event-type catalog. Use these constants for BOTH Publish and
// Subscribe and for outbox persistence (never Go type names / free strings).
const (
    BookingCreated   = "booking.created"
    BookingConfirmed = "booking.confirmed"
    BookingCancelled = "booking.cancelled"
    BookingCompleted = "booking.completed"
    BookingUpdated   = "booking.updated"

    TripCreated     = "trip.created"
    TripScheduled   = "trip.scheduled"
    TripAssigned    = "trip.assigned"
    TripStarted     = "trip.started"
    TripReachedPickup = "trip.reached_pickup"
    TripInTransit   = "trip.in_transit"
    TripDelivered   = "trip.delivered"
    TripCompleted   = "trip.completed"
    TripCancelled   = "trip.cancelled"
)

// EventTypeOf maps a domain event value to its canonical string, so the
// outbox writer stores "booking.confirmed" instead of "BookingConfirmedEvent".
var EventTypeOf = map[any]string{
    bookingevents.BookingConfirmedEvent{}: BookingConfirmed,
    // ... one entry per event type ...
}
```
- **Wiring (single bus):** in `cmd/server/main.go`, construct `eventBus := events.NewInMemoryBus()` once; inject it into `service.NewServices(store, cfg, log, eventBus)` (add `bus` param to `NewServices`) AND into `outbox.NewRelay(database, eventBus, logger)` AND `founderSvc.RegisterEventHandlers(eventBus)`. Delete the separate `events.NewInMemoryBus()` at `service.go:75`. Now founder + automation + relay share one bus.
   - **Outbox name hardening (secondary, only meaningful after the single bus is wired):** change `getEventTypeName` (`outbox.go:67`) to consult `events.EventTypeOf[ev]` first, falling back to the current `fmt.Sprintf("%T")` only for unknown types. So stored `event_type` becomes `"booking.confirmed"`. This removes the latent drift noted in 0.2; it is **not** the root-cause fix (the duplicated bus is) — but it must ship alongside the bus unification so outbox-sourced events match the catalog subscribers.

### 5.2 Bus fixes
`internal/events/bus.go`:
- **Errors not swallowed:** `Publish` must collect handler errors and return them (signature becomes `Publish(ctx, e) error`) OR, to stay compatible, log+aggregate and panic-safe. Recommended: change to `Publish(ctx context.Context, e Event) error` that runs handlers and returns the first error (after running all). Update all call sites (`booking_service.go`, `trip_service.go`, `service.go`) to check the error.
- **Unsubscribe by identity:**
```go
func (b *InMemoryBus) Subscribe(eventType string, h Handler) (unsubscribe func()) {
    b.mu.Lock()
    b.subs[eventType] = append(b.subs[eventType], h)
    b.mu.Unlock()
    return func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        for i, existing := range b.subs[eventType] {
            if &existing != nil && sameHandler(existing, h) { // compare by pointer/identity
                b.subs[eventType] = append(b.subs[eventType][:i], b.subs[eventType][i+1:]...)
                break
            }
        }
    }
}
```
  Simpler: store handlers with an id and filter by `== h`. (Handler is a `func` value; `==` compares function pointers — valid for closures created once.)
- **Dead-letter (real):** extend `Relay.markFailed` (`relay.go:137`) to `UPDATE outbox_events SET status='failed', attempts=?, last_error=?`. On reaching `maxAttempts`, `UPDATE ... SET status='dead', dead_lettered_at=?` and emit a `founder`/log alert; stop retrying (already stops because `shouldSkip` only delays, add a `status='dead'` guard in `dispatch` to permanently skip). Add a `SELECT ... WHERE published_at IS NULL AND status <> 'dead'` to the poll query (`relay.go:71`).

### 5.3 Booking hardening
- **Immutable Confirmed:** in `booking_aggregate.go:139 Update`, change guard to:
  ```go
  if b.Status != BookingPending { // and not draft
      return errors.New("only pending bookings can be updated")
  }
  ```
  So `confirmed`/`cancelled`/`completed` are locked. (Keeps `draft` dead path irrelevant since NewBookingAggregate starts at `pending`.)
- **Delete guard:** fix `internal/domain/booking/entity.go:63 CanDelete()`:
  ```go
  func (b Booking) CanDelete() error {
      if b.Status != BookingPending { // business-rule: only pending/draft deletable
          return fmt.Errorf("only pending bookings can be deleted; current status: %s", b.Status)
      }
      return nil
  }
  ```
  (Remove `BookingConfirmed` from allowed set.)
- **Existence checks:** `CreateBooking` already checks customer/route (`booking_service.go:43-49`). Add, in `ConfirmBooking`, a re-fetch + `CanConfirm` via the aggregate path, and record history (below). Keep `ErrBookingNotFound` on missing id.
- **Reverse-fare parity (shared `PricingService`):** create `internal/service/pricing.go`:
  ```go
  type PricingService struct{ store Store }
  func (p *PricingService) Quote(ctx context.Context, routeID domain.RouteID, reverse bool) (price float64, source string, err error) {
      r, err := p.store.GetRouteByID(ctx, routeID)
      if err != nil { return 0, "", err }
      if reverse && r.ReverseStandardFare != nil {
          return *r.ReverseStandardFare, "reverse", nil
      }
      return r.StandardFare, "standard", nil
  }
  ```
  - `CreateBooking` (`booking_service.go:56-59`): replace `req.Price = route.StandardFare` with `price, src, _ := pricing.Quote(ctx, req.RouteID, false); req.Price = price` and persist `fare_source = src`.
  - Agent `get_quote` (`tools.go:105-117`): call the **same** `PricingService.Quote` so agent and service agree. Reverse determination stays (direction passed as `reverse`).
- **Status history:** every transition in `booking_service.go` (`Create/Confirm/Cancel/Complete`) appends a row to `booking_status_history` (transition + `changed_by` from `auth.ContextUser`, `reason`). `GetBooking`/`history` endpoint reads it.

Booking state machine (canonical, per `docs/business-rules/booking.md`):
```
pending ──confirm──▶ confirmed ──complete──▶ completed
   │                     │
 cancel                cancel (forbidden by BR? BR says cancel from any except completed;
   │                     │   keep: confirmed IS cancellable per BR line 31, but NOT editable/deletable)
   ▼                     ▼
cancelled            (confirmed immutable: no update/delete)
```
> NOTE conflict to resolve (section 11): BR line 31 says cancel allowed from any except completed; BR line 48 says only pending/draft deletable. Confirmed = cancellable but not deletable, not editable. Implement exactly that.

### 5.4 Trip POD hardening (canonical = DDD aggregate)
- **Persist full e-POD:** add fields to `internal/domain/trip/entity.go` `Trip` struct AND `SQLTripModel` (`trip_converter.go:11`): `PodPhotoURL, PodSignatureURL, PodConsigneeName, PodConsigneePhone *string; PodOTPVerified bool; PodCapturedAt, PodLat, PodLng, PodNotes, PodOTP揭Hash, PodOTPIssuedAt, PodVerifiedAt *...`. Map both directions in `MapToAggregate`/`MapToPersistence`.
- **OTP issuance + verify:**
  - `IssuePODOTP(ctx, tripID)` → generate 6-digit, `bcrypt.GenerateFromPassword`, store `pod_otp_hash`, `pod_otp_issued_at`, `pod_issued_by`; TTL 15 min (reject verify if expired).
  - `VerifyPODOTP` compares bcrypt; on success mark `pod_otp_verified=1`, `pod_verified_at`. Client `otp` field is **never trusted** as a boolean.
- **Unify on DDD aggregate:** replace legacy `entity.CanDeliver/CanComplete` usage in `trip_service.go` with aggregate transitions (`Start→ReachPickup→StartTransit→Deliver→Complete`). `DeliverTripWithPOD` must call `agg.Deliver(now)` requiring `in_transit`. Conflicting legacy guard (`entity.go:69` allowing `started`) is removed.
- **Re-home deliver-pod:** move handler from `internal/handlers/kharcha.go` (`main.go:797`) to `internal/handlers/trips.go` `DeliverPOD` and route `r.Post("/trips/{id}/deliver-pod", app.Trips.DeliverPOD)` in `TripHandlers.Routes`. Implementation calls `svcs.Trips.DeliverWithPOD` (full payload), not the legacy single-URL method.
- **Time-window conflict:** on `DeliverWithPOD`, call `store.CheckDriverConflict`/`CheckVehicleConflict` style overlap check: reject if the same `driver_id`/`vehicle_id` has another trip with `status IN (started, reached_pickup, in_transit)` whose `departure_time < now < arrival_time` (or `delivered_at`). Reuse existing `CheckDriverConflict`/`CheckVehicleConflict` (`trip_service.go:188,254`). Also enforce a POD capture window: `pod_captured_at` must be within `[departure_time, departure_time + company_config.pod_window_hours]` (config, section 6) — else `409 outside_window`.

Trip canonical state machine:
```
draft → scheduled → assigned → started → reached_pickup → in_transit → delivered → completed
                                                                          ↘ (cancel from any except completed)
```

---

## 6. Config / env

| Var | Default | Purpose | Package |
|-----|---------|---------|---------|
| `EVENT_BUS_SYNC` | `true` | Keep in-memory sync bus (no broker yet) | `internal/events` |
| `OUTBOX_POLL_INTERVAL` | `5s` | Relay poll interval (`relay.go:15`) | `internal/shared/outbox` |
| `OUTBOX_MAX_ATTEMPTS` | `5` | Dead-letter threshold (`relay.go:16`) | `internal/shared/outbox` |
| `POD_OTP_TTL_MINUTES` | `15` | OTP validity window | `internal/service` |
| `POD_WINDOW_HOURS` | `72` | Allowed e-POD capture window after departure | `internal/service` (reads `company_config`) |
| `BOOKING_HISTORY_RETENTION_DAYS` | `365` | History retention (housekeeping) | `internal/service` |

No external creds required; all adapters (founder Telegram, OTP SMS) already flag-gated.

---

## 7. Tests

Coverage gate: new code ≥ 80% unit coverage; all e2e tests in `test/` must pass before merge.

**7.1 Event flow e2e (`test/eventbus_test.go` — new)**
- Single bus: publish `booking.confirmed` → assert automation handler (auto-create trip) fires AND outbox relay (separate goroutine reading DB) re-publishes to same bus → founder handler receives. Use one shared `InMemoryBus`.
- Name-mismatch regression: write an outbox row with `event_type="booking.confirmed"` (via `OutboxWriter.SaveEvents` using a `BookingConfirmedEvent`) → `Relay.Run` once → assert subscriber on `events.BookingConfirmed` received exactly 1 event. (Proves `getEventTypeName` now returns canonical string.)
- Dead-letter: make a handler return error; after `OUTBOX_MAX_ATTEMPTS` polls assert `outbox_events.status='dead'` and `last_error` set, and event no longer retried.
- Unsubscribe: subscribe two handlers, unsubscribe the first, publish → assert only second runs.

**7.2 Immutability (`internal/booking/..._test.go`)**
- `Confirmed` booking → `Update` returns error; `DeleteBooking` returns `ErrInvalidState` (or new `ErrConfirmedImmutable`).
- `Pending` booking → delete ok.
- `PricingService.Quote(reverse=true)` equals `Route.ReverseStandardFare`; `CreateBooking(price=0)` stores `fare_source='standard'`. Agent `get_quote` reverse path matches service.

**7.3 POD persistence (`test/kharcha_epod_test.go` — extend)**
- POD-1..POD-6 extended to assert `pod_photo_url`, `pod_consignee_name`, `pod_otp_verified`, `pod_captured_at` are persisted (not just status). Add `setupInTransitTrip` reaches `in_transit` (call `ReachPickup` then `StartTransit`).
- New `TestEPOD_OTP`: `IssuePODOTP` → verify correct OTP → `pod_otp_verified=1`; wrong OTP → `401`/error, not verified.
- New `TestEPOD_TimeWindow`: deliver outside `POD_WINDOW_HOURS` → `409 outside_window`.
- New `TestEPOD_Conflict`: two overlapping in-transit trips for same driver → second deliver rejected.

**7.4 Tenant (`test/tenant_test.go` — new)**
- Handler tests pass a context via `shared.ContextWithTenantID(ctx, "7")` → assert query hits `WHERE tenant_id='7'`, not `'1'`. Grep-assert zero remaining `TenantID: "1"` literals after refactor (CI lint).

**Pass-before-merge checklist:** `go build ./...`, `go test ./...`, `goose up` 00053/00054/00055 + `goose down` round-trip, `grep -rn 'TenantID: "1"' internal/handlers` returns nothing.

---

## 8. Future / GPS-provider

- **Geo-fenced auto-POD:** when a telematics provider (spec 01/04 `TelematicsProvider` interface) reports the vehicle inside the destination geofence (spec 02), auto-trigger `DeliverWithPOD` on the **same unified bus** via a `telemetry.position.inside_geofence` event → handler issues POD. Keep manual OTP verify as the trust gate.
- **Ignition-based trip start:** `telemetry.ignition.on` event on the same bus → `trip.started` transition, replacing manual "Start" button; conflicts checked via the same `CheckDriverConflict`.
- Dead-letter → a real `dead_letter` table + founder alert channel (already flag-gated). Outbox could later fan out to NATS/Kafka behind the same `EventBus` interface (no caller change).

---

## 9. Edge cases

- Outbox `event_type` for pre-existing rows stored as `"BookingConfirmedEvent"`: migration 00053 does **not** rewrite history; add a one-time backfill `UPDATE outbox_events SET event_type='booking.confirmed' WHERE event_type='BookingConfirmedEvent' AND status='pending'` in 00053 Up (idempotent; safe because those rows were never delivered — the relay published them onto bus #1, which had no subscriber, and the name also drifted from the `"BookingConfirmed"` subscribers, per 0.1/0.2).
- Handler panic in `Publish`: wrap each `h(ctx,e)` in `func()` recover to avoid taking down the caller; record as error, continue to next handler.
- Two subscribers unsubscribe concurrently: `InMemoryBus` uses `sync.RWMutex` already; identity-based removal is concurrency-safe.
- `Confirmed` booking cancel: allowed per BR line 31; ensure `Cancel` does not violate immutability (cancel is a status change, not an edit/delete).
- POD with neither photo nor signature → rejected before any state change.
- OTP issued but trip delivered without OTP field → `pod_otp_verified=0`; allowed only if no OTP was issued (track `pod_otp_issued_at IS NULL`).
- Tenant missing in context → `TenantIDFromContext` defaults to `"1"` (`tenant.go:33`) — acceptable fallback, but handlers must still call it (not hardcode).

---

## 10. Phased rollout (build order)

1. **00053 + bus**: create `events.go` catalog; fix `bus.go` (errors, unsubscribe); fix `outbox.go` `getEventTypeName`; extend `relay.go` (dead-letter + status query); wire single bus in `main.go` + `service.NewServices`. Migrate 00053. → unblocks event flow.
2. **Tenant**: replace 37 `TenantID:"1"` literals (section 2.4). No migration.
3. **00054 booking**: add `PricingService`; fix `Update`/`CanDelete`; persist `fare_source`/`tenant_id`; add `booking_status_history` + endpoint. Migrate 00054.
4. **00055 trip POD**: extend entity + converter + persistence; OTP issue/verify; re-home `/deliver-pod`; unify on DDD aggregate; conflict + window checks. Migrate 00055.
5. **Tests + CI grep gate** for residual `"1"` literals and event-name regression.

---

## 11. Open items / VERIFY before coding

- **BR conflict (cancel vs delete):** `docs/business-rules/booking.md:31` (cancel from any except completed) vs `:48` (delete only pending/draft). Confirmed = cancellable, not deletable. Implement as specified in 5.3. **VERIFY** with product owner.
- **`routes.reverse_standard_fare` existence:** `tools.go:110` reads `r.ReverseStandardFare`; confirm the `routes` table/struct has it. If missing, add column in 00054 Up (and update `Route` entity + converter). **VERIFY** by grepping `ReverseStandardFare`.
- **`company_config` tenant_id column:** 00042 owns `company_config` per `00-migration-ownership-index.md`, but it is **NOT present** in `db/migrations/` (head `00039`). The `tenant_id` FK referenced in 00054 is therefore invalid; 00054 now scopes booking by a plain `tenant_id TEXT` column and defers the FK. **VERIFIED-WRONG** (was assumed taken). Revisit FK after 00042 merges.
- **`booking.Status` vs aggregate:** `booking_service.go` uses legacy `domain.Booking` (`entity.go`); the DDD `booking_aggregate.go` is currently only used in tests. Decide: route all transitions through the aggregate (preferred) or just fix the legacy guards. This spec fixes both; **VERIFY** which is the runtime path to avoid double-enforcement.
- **`DeliverWithPOD` vs `DeliverTripWithPOD`:** legacy single-URL method retained only as internal helper; public entrypoint is `DeliverWithPOD` with full payload. **VERIFY** no other caller uses the legacy signature.

---

## 12. File list

**Create**
- `internal/events/events.go` — event-type catalog + `EventTypeOf` map (fixes 0.5; hardens the 0.2 latent drift).
- `test/eventbus_test.go` — event flow e2e, dead-letter, unsubscribe, name-mismatch (7.1).
- `test/tenant_test.go` — tenant resolution (7.4).
- `internal/service/pricing.go` — shared `PricingService` (5.3).
- `db/migrations/00053_eventbus_outbox.sql` (3.1)
- `db/migrations/00054_booking_hardening.sql` (3.2)
- `db/migrations/00055_trip_pod.sql` (3.3)

**Modify**
- `internal/events/bus.go` — return errors from `Publish`; identity-based `Unsubscribe` (0.3, 0.4).
- `internal/shared/outbox/outbox.go` — `getEventTypeName` uses `events.EventTypeOf` (0.2).
- `internal/shared/outbox/relay.go` — dead-letter status/attempts/last_error; skip `dead` (0.3, 5.2).
- `cmd/server/main.go:819-826` — build ONE bus, inject into `NewServices`, relay, founder (0.1).
- `internal/service/service.go:75,103,111` — accept injected bus; remove local `NewInMemoryBus()` (0.1).
- `internal/service/booking_service.go` — `PricingService` fare (0.6), status history, immutable confirm (5.3).
- `internal/booking/domain/aggregate/booking_aggregate.go:139` — `Update` only when pending (0.6).
- `internal/domain/booking/entity.go:63` — `CanDelete` only pending (0.6).
- `internal/service/trip_service.go` — full POD persist; OTP; DDD aggregate; conflict/window (0.7, 5.4).
- `internal/domain/trip/entity.go` — add `reached_pickup`; remove conflicting `CanDeliver` (0.7).
- `internal/trip/domain/aggregate/trip_aggregate.go` — canonical (already has reached_pickup; keep).
- `internal/trip/infrastructure/persistence/sql/converters/trip_converter.go` — map POD + OTP cols (0.7).
- `internal/handlers/trips.go` — add `DeliverPOD`, `IssuePODOTP`, `GetTrip` POD; replace `TenantID:"1"` (0.8, 5.4).
- `internal/handlers/{drivers,vehicles,invoices,payments}.go` — replace `TenantID:"1"` (0.8).
- `internal/handlers/kharcha.go` — remove `DeliverWithPOD` POD handler (re-homed).
- `cmd/server/main.go:797` — re-route `/trips/{id}/deliver-pod` to Trip handlers.
- `internal/agent/tools.go:105` — use `PricingService.Quote` for parity (0.6).
- `web/templates/trip_view.html`, `trip_edit.html`, `booking_view.html`, `booking_edit.html` — POD/history UI (section 4).
- `test/kharcha_epod_test.go` — extend POD assertions + new OTP/window/conflict tests (7.3).
