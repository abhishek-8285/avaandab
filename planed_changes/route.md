You're right — the route model is one of the weakest parts of the system. Here's my analysis.

## What's wrong with the current route model

### 1. Source/Destination are free-text strings
```go
// internal/domain/route/entity.go:11-23
type Route struct {
    Source              string    // "Pune"
    Destination         string    // "Mumbai"
    ...
}
```
- No geocoding, no lat/lng, no address hierarchy.
- "Pune" / "pune" / "Pune, MH" / "PUNE" = four different routes. Typos create duplicates.
- No way to show a route on a map.
- No link to customer warehouses or driver locations.

### 2. Point-to-point only — no multi-stop
A route is strictly `Source → Destination`. Real transport has:
- Multi-stop: `Pune → Lonavala → Mumbai → Bhiwandi`
- Hub-spoke: `Hub → spoke1 → spoke2 → Hub`
- Round trips with intermediate drops.

Current model can't represent any of these. You'd have to create N separate routes and N separate trips.

### 3. Fare is a single fixed number
```go
StandardFare        float64
ReverseStandardFare *float64
```
- Same fare regardless of vehicle type (truck vs tempo vs bike).
- Same fare regardless of customer (regular vs one-off).
- Same fare regardless of cargo weight or passengers.
- No surge/seasonal/weekend pricing.
- No per-km rate — only lump sum.

Real pricing needs a **matrix**: route × vehicle type × customer tier → rate.

### 4. Reverse logic is bolted on
```go
ReverseDistance     *float64
ReverseStandardFare *float64
```
`GetDistanceAndFare` does string matching to detect direction (`entity.go:26-41`). This is fragile:
- Case sensitivity handled with `EqualFold` but typos break it.
- Return trip often has different distance (one-way streets, highway exits) but the model forces you to shoehorn it into the same row.
- A route should be **directional** or **bidirectional** as a first-class concept, not an optional override.

### 5. Route is required for every booking
```go
// internal/service/booking_service.go:36-38
if req.RouteID == "" {
    return domain.Booking{}, fmt.Errorf("route is required")
}
```
- Ad-hoc bookings (one-off customer request to a new destination) are impossible without first creating a route.
- Owner has to leave booking form, go to /routes, create route, come back. Friction.

### 6. Distance is static and disconnected
- `distance` is a number entered by the owner. No link to:
  - Actual GPS path (telemetry/odometer).
  - Map API (Google Maps, OSRM).
  - Real-time traffic.
- Trip's actual distance comes from telemetry (`pnl/service.go:61-67`), but route's `distance` is never reconciled with it.

### 7. No route variants
Same source-destination can have multiple paths:
- Highway vs expressway (different toll, different distance, different time).
- Fast route vs cheap route.
Current model allows only one row per source-destination pair (`route_service.go:34` rejects duplicates).

### 8. No tenant isolation on routes
```sql
-- db/migrations/00005_routes.sql
CREATE TABLE routes (
    id, source, destination, distance, estimated_hours, standard_fare, ...
    -- no tenant_id
);
```
Every other table has `tenant_id`; routes don't. If multi-tenancy is added, routes leak across companies.

### 9. Route conflates three concepts
Today `Route` is simultaneously:
- A **place pair** (where → where).
- A **pricing rule** (how much).
- A **distance/time estimate** (how far).

These change at different rates and for different reasons. Bundling them makes the model rigid.

---

## Proposed model

### Separate Place, Route, and Pricing

```
Place (master data)
├── id, tenant_id
├── name (canonical: "Pune")
├── type (city | warehouse | hub | customer_site)
├── address, city, state, country, pincode
├── latitude, longitude
├── parent_id (→ Place, e.g. Pune → Maharashtra)
└── owner_customer_id (nullable, if it's a customer site)

Route (the path)
├── id, tenant_id
├── name ("Pune-Mumbai Express")
├── direction (oneway | bidirectional)
├── stops[] (ordered: RouteStop → Place)
│   ├── sequence (1, 2, 3...)
│   ├── place_id
│   └── leg_distance, leg_duration (from this stop to next)
├── total_distance, total_duration
├── polyline (encoded geometry, nullable)
└── is_active

RoutePricing (the money)
├── id, tenant_id
├── route_id
├── vehicle_type (truck_32ft, tempo, bike, * = all)
├── customer_tier (default, vip, * = all)
├── pricing_model (fixed | per_km | per_kg | per_kg_km)
├── base_rate, per_km_rate, min_charge
├── effective_from, effective_to
└── priority (most specific wins)
```

### How booking uses it

```
Booking
├── route_id (nullable — allows ad-hoc)
├── pickup_place_id (required)
├── dropoff_place_id (required)
├── intermediate_stops[] (optional, for multi-drop)
├── vehicle_type
├── price (auto-calculated from RoutePricing, or manual override)
```

- If `route_id` set → pull stops and pricing from route.
- If `route_id` null → owner picks pickup/dropoff places; system auto-creates a one-time route or leaves ad-hoc.
- Price auto-suggested from `RoutePricing` matrix; owner can override.

### How trip uses it

```
Trip
├── route_id (inherited from booking, or set directly)
├── actual_stops[] (can deviate from planned route)
├── actual_distance (from telemetry/odometer)
├── planned_distance (from route, for variance check)
```

### Bidirectional done right
- `direction = bidirectional` on Route means same stops in reverse order are valid.
- No `ReverseDistance` hack — reverse leg distances are just the stops reversed.
- Pricing can differ per direction: add `direction` to `RoutePricing` if needed.

### Multi-stop done right
- `RouteStop` is an ordered list. A route can have 2 stops (simple) or 10 (multi-drop).
- Trip can mark which stops are actual pickup/dropoff vs pass-through.

### Pricing done right
- Most specific match wins: `route + vehicle_type + customer_tier` beats `route + * + *`.
- Supports per-km for freight, fixed for passenger, per-kg for cargo.
- Time-windowed: `effective_from/to` for seasonal rates.
- Owner can set a default rate and override per customer.

---

## Migration path (minimal disruption)

| Step | Change | Backward compatible |
|---|---|---|
| 1 | Add `tenant_id` to `routes` table | yes, backfill "1" |
| 2 | Add `places` table + seed common Indian cities | new table, no break |
| 3 | Add `route_stops` table (route_id, place_id, sequence) | new table, no break |
| 4 | Make `routes.source/destination` nullable, add `pickup_place_id/dropoff_place_id` | old routes still work via string lookup |
| 5 | Add `route_pricing` table | new table, old `standard_fare` still used as fallback |
| 6 | Make `bookings.route_id` nullable, add `pickup_place_id/dropoff_place_id` | ad-hoc bookings enabled |
| 7 | Migrate existing `routes.source/destination` → Places (geocode or match by name) | one-time script |
| 8 | Deprecate `routes.standard_fare` once pricing table populated | read fallback until removed |

Phase 1 of owner onboarding can use the **old model** (strings) because owners just need to type "Pune → Mumbai". The new Place/Route model can be introduced in parallel and the wizard can start writing to both.

---

## What this fixes for owners

| Today | After |
|---|---|
| Must create route before booking | Ad-hoc booking with pickup/dropoff places |
| "Pune" typo = duplicate route | Canonical Place, dedup automatic |
| Same fare for truck and bike | Pricing matrix by vehicle type |
| Can't do multi-drop | Ordered stops |
| Return trip is a hack | Bidirectional flag, stops reverse cleanly |
| No map | Places have lat/lng; route has polyline |
| Distance never reconciled | Planned vs actual distance variance |

Want me to draft the schema migration and Place/Route domain entities next?