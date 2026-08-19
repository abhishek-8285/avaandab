# GraphQL + gRPC Read/Stream Layer — Implementation Spec v1

Status: ready
Depends-on: none (no new migration; reuses `telemetry_snapshots` from `00031_avandab_critical_fixes.sql`)
Migration owner: none — spec explicitly adds NO new DB table or migration.

> DECISION (under review — see §13): Implement **REAL read-only GraphQL** via `gqlgen` **only if a typed client is committed** (mutations stay on REST); **gRPC is recommended for REMOVAL**, not implementation, because no current consumer exists (mobile uses REST/MQTT/GraphQL; GPS ingest already covered by MQTT + REST telemetry). The existing mock GraphQL handler and gRPC stub are replaced/removed, not extended. If the team commits to an external gRPC contract, see §5/§7 for the build path; otherwise see §11/§13.1 for the removal path.

---

## 0. Verified ground truth (file:line)

Current state is MOCK/STUB. Every fact below is copy-pasted from the repo.

- **GraphQL is mocked.** `internal/graphqlservice/handler.go:25` `ServeHTTP` ignores the request body's `query`/`variables` entirely:
  ```go
  var q GraphQLQuery
  _ = json.NewDecoder(r.Body).Decode(&q)          // decoded then never used
  res, err := h.listTripsUC.Execute(...)           // hardcoded ListTripsQuery Page:1 Limit:50 Status:""
  ```
  It then hand-builds a fixed shape (`activeTrips`, `serverTime`) at `internal/graphqlservice/handler.go:43-65`. `gqlgen` is present but **indirect/unused**: `go.mod:19` `github.com/99designs/gqlgen v0.17.94 // indirect`.
- **gRPC is a stub.** `internal/grpcservice/server.go:23` builds `s := grpc.NewServer()` with **ZERO registered services** and logs `"[gRPC WARNING] gRPC server is a stub: no services are registered"` (`internal/grpcservice/server.go:24`).
- **`GetDriverLocation` is dead code.** `internal/grpcservice/server.go:32` `func (s *Server) GetDriverLocation(ctx, driverID string)` returns hardcoded Pune coords (`18.5204, 73.8567`) and is **never registered** with any gRPC server.
- **`dispatch.proto` is uncompiled.** `internal/grpcservice/dispatch.proto` defines `DispatchService` (`GetDriverLocation`, `UpdateTripStatus`) but there is **no generated `.pb.go`** (confirmed: `internal/grpcservice/` has only `server.go` + `dispatch.proto`).
- **Wiring in `main.go`.** gRPC started at `cmd/server/main.go:428` (`grpcservice.StartGRPCServer(grpcPort)`, port from `GRPC_PORT` default `50051` at `cmd/server/main.go:424-427`). GraphQL handler built at `cmd/server/main.go:431` (`graphqlservice.NewGraphQLHandler(listTrips)`) and mounted at `cmd/server/main.go:446-447` (`/query` + `/graphql`), behind `middleware.RequireAPIAuth` (cookie or bearer).
- **Smoke test is shallow.** `test/smoke_new_endpoints_test.go:226` `TestSmoke_GraphQLRealData` posts `{ activeTrips { id } }` and only asserts `200` + `serverTime` present (line 262). It does NOT verify field values, filtering, or that the query string is honored — because the handler ignores it.
- **Telemetry store exists.** `telemetry_snapshots` table created at `db/migrations/00031_avandab_critical_fixes.sql:4-16`:
  ```sql
  CREATE TABLE telemetry_snapshots (
    id TEXT PRIMARY KEY,
    trip_id TEXT,
    vehicle_id TEXT,
    timestamp DATETIME NOT NULL,
    latitude REAL, longitude REAL, speed REAL, fuel_level REAL, odometer REAL,
    FOREIGN KEY (trip_id) REFERENCES trips(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
  );
  CREATE INDEX idx_telemetry_snapshots_trip ON telemetry_snapshots(trip_id, timestamp);
  ```
  Written by `internal/telemetry/sync.go:72` `snapshotHandler` (`INSERT OR REPLACE`).
- **Real GPS ingest path exists.** `internal/mqttservice/mqtt.go:46` subscribes `avandab/telemetry/drivers/+/gps` and decodes `GPSTelemetryPayload{DriverID, Latitude, Longitude, Timestamp}` (`internal/mqttservice/mqtt.go:12-17`). Today it only logs (`internal/mqttservice/mqtt.go:49`). This is the future write-path for `TelemetryIngest` (see §8).
- **Trip read model links driver→vehicle.** `internal/trip/application/list_trips.go:62-87` `TripResponseDTO` exposes `DriverID`, `VehicleID`, `VehicleRegistrationNumber`, `Status`. Active trips are non-`completed`/non-`cancelled` (current filter at `internal/graphqlservice/handler.go:45`).
- **Auth model.** Web/API auth via `middleware.RequireAPIAuth(authStore, apiSecret)` (`cmd/server/main.go:445`); bearer-token secret from `config.APITokenSecret` (`internal/config/config.go:19`), gated in production at `cmd/server/main.go:405-412`. gRPC has no auth today.

### 0.1 Verification Log (principal-engineer QA pass)

Every §0 claim was read against real source. Verdicts below. Severity/Effort refer to the **spec correction** required.

| # | Claim | Verdict | Correction / Evidence | Sev | Eff |
|---|-------|---------|-----------------------|-----|-----|
| C1 | GraphQL handler ignores `query`/`variables` | TRUE | `handler.go:28` decodes into `q`, never reads `q.Query`/`q.Variables`; `ListTripsQuery` hardcoded Page:1/Limit:50/Status:"" at `:31-36`; shape built `:43-65` | — | — |
| C2 | `gqlgen` present but `// indirect`/unused | TRUE | `go.mod:19` `github.com/99designs/gqlgen v0.17.94 // indirect`; no import in tree | — | — |
| C3 | gRPC stub: `grpc.NewServer()` + zero services | TRUE | `server.go:23-24`; no `Register…Server` call; warns "no services are registered" | — | — |
| C4 | `GetDriverLocation` hardcoded Pune, never registered | TRUE | `server.go:32-38` returns `18.5204,73.8567`; signature `(ctx, driverID string)` is NOT a gRPC method shape, so unregistrable as-is | — | — |
| C5 | `dispatch.proto` uncompiled, no `.pb.go` | TRUE | `internal/grpcservice/` contains only `server.go` + `dispatch.proto` (glob); no `pb/` | — | — |
| C6 | main.go wiring :428/:431/:446-447 | TRUE | `main.go:428` `StartGRPCServer(grpcPort)`; `:431` `NewGraphQLHandler`; `:446-447` `/query`+`/graphql` under `RequireAPIAuth` `:445` | — | — |
| C7 | Smoke test asserts only 200 + `serverTime` | TRUE | `test/smoke_new_endpoints_test.go:226` (`TestSmoke_GraphQLRealData`), `:262` asserts `ServerTime` only; query `{ activeTrips { id } }` not validated | — | — |
| C8 | `telemetry_snapshots` has `trip_id`/`vehicle_id` | TRUE | `00031_avandab_critical_fixes.sql:4-16`, index `:17` | — | — |
| C9 | `sync.go:72` `INSERT OR REPLACE` into `telemetry_snapshots` | TRUE | `sync.go:72` exact columns `(id,trip_id,vehicle_id,timestamp,lat,long,speed,fuel_level,odometer)` | — | — |
| C10 | MQTT subscribes `…/+/gps`, logs only | TRUE | `mqtt.go:45` topic, `:46-49` decode + `log.Printf`, no persistence | — | — |
| C11 | `TripResponseDTO` exposes DriverID/VehicleID/VehicleRegistrationNumber/Status | TRUE | `list_trips.go:62-87` (Status at `:78`) | — | — |
| C12 | Auth: `RequireAPIAuth` + prod-gated secret | TRUE | `main.go:405-412` (API_SECRET guard), `:445`; `config.go:19` `APITokenSecret` | — | — |
| C13 | `127.0.0.1` is at `cmd/server/main.go:416` (§11) | **FALSE** | Host literal lives in `internal/grpcservice/server.go:16`, not `main.go`. `main.go:416` is the MQTT comment block. Corrected in §11. | Low | Trivial |

> §5 SQL validity cross-check: `trips` (`00007_trips.sql:6,7,9,11`) has `driver_id`, `vehicle_id`, `departure_time`, `status` — matches the §5.2/§5.3 resolution queries. No schema fiction.

---

## 1. Overview / goal

Replace the mock GraphQL handler and the empty gRPC stub with **real, working, read-only** transports that read from the existing domain use cases and the existing `telemetry_snapshots` table. GraphQL exposes `Trip` / `Booking` / `Invoice` **queries only** (mutations remain REST). gRPC exposes `DispatchService.GetDriverLocation` (driver→active trip→vehicle→latest snapshot) and `TelemetryIngest.PushGPS` (unary write into `telemetry_snapshots`), with `StreamGPS` left as a documented future stub.

Non-goals:
- No mutations over GraphQL (create/update/delete stay on REST — preserves Casbin RBAC, approval gate, CSRF, outbox guarantees).
- No new database table, column, index migration, or migration number (reuses `00031`).
- No `UpdateTripStatus` gRPC write (out of scope; would bypass REST approval gate). Defined in proto only for forward-compat.
- No replacement of the MQTT ingress (MQTT stays the driver-facing live path; gRPC `PushGPS` is a server-to-server / test path).

---

## 2. API contract

### 2.1 GraphQL (read-only, gqlgen)

HTTP: `POST /query` and `GET /graphql` (already mounted `cmd/server/main.go:446-447`), behind `RequireAPIAuth`. `Content-Type: application/json`.

SDL (`internal/graphqlservice/gqlgen/gql_schema.graphqls`):

```graphql
# ── Scalars ──────────────────────────────────────────────
scalar Time   # RFC3339 string

# ── Entities ─────────────────────────────────────────────
type Trip {
  id: ID!
  tripNumber: String!
  bookingId: ID
  driverId: ID
  driverName: String
  vehicleId: ID
  vehicleNumber: String
  origin: String
  destination: String
  status: String!
  departureTime: Time
  arrivalTime: Time
}

type Booking {
  id: ID!
  customerId: ID
  routeId: ID
  pickupDate: Time
  vehicleType: String
  passengers: Int
  price: Float
  status: String
}

type Invoice {
  id: ID!
  bookingId: ID
  customerId: ID
  subtotal: Float
  tax: Float
  discount: Float
  total: Float
  status: String
}

type Query {
  # All non-completed/non-cancelled trips for the tenant (same semantics as today's activeTrips)
  trips(status: String, search: String, page: Int = 1, limit: Int = 50): [Trip!]!
  trip(id: ID!): Trip

  bookings(page: Int = 1, limit: Int = 50): [Booking!]!
  booking(id: ID!): Booking

  invoices(page: Int = 1, limit: Int = 50): [Invoice!]!
  invoice(id: ID!): Invoice
}

# No Mutation type — mutations stay on REST.
```

Request (real, honored):
```json
{ "query": "{ trips(status: \"in_transit\") { id tripNumber driverName vehicleNumber status } }", "variables": {} }
```

Success response (gqlgen standard envelope):
```json
{
  "data": {
    "trips": [
      { "id": "trip-abc", "tripNumber": "T-001", "driverName": "Amit Kumar",
        "vehicleNumber": "MH12AB1234", "status": "in_transit" }
    ]
  },
  "errors": null
}
```

Error response (e.g. invalid query or resolver failure):
```json
{ "data": null, "errors": [ { "message": "trip not found", "path": ["trip"] } ] }
```
HTTP codes: `200` for success or partial (`errors` present but data may be null); `400` for malformed GraphQL; `401` when auth missing (from `RequireAPIAuth`).

### 2.2 gRPC (real, compiled from proto)

Service definitions (`internal/grpcservice/dispatch.proto` — REPLACE current file):

```proto
syntax = "proto3";

package dispatch;

option go_package = "transport-app/internal/grpcservice/pb";

// ── DispatchService (read-only location lookup) ──────────
service DispatchService {
  rpc GetDriverLocation (LocationRequest) returns (LocationResponse);
  // UpdateTripStatus intentionally NOT implemented server-side (write gate).
  // rpc UpdateTripStatus (TripStatusRequest) returns (TripStatusResponse);
}

message LocationRequest {
  string driver_id = 1;
}

message LocationResponse {
  string driver_id   = 1;
  string vehicle_id  = 2;
  double latitude    = 3;
  double longitude   = 4;
  double speed       = 5;
  string last_updated = 6;   // RFC3339
}

message TripStatusRequest  { string trip_id = 1; string status = 2; }
message TripStatusResponse { bool success = 1; string message = 2; }

// ── TelemetryIngest (GPS write path) ────────────────────
service TelemetryIngest {
  // Unary: single GPS point. Implemented.
  rpc PushGPS (GPSPoint) returns (PushGPSResponse);
  // Client-streaming: batch of points. Defined now, implemented in §8.
  rpc StreamGPS (stream GPSPoint) returns (StreamGPSResponse);
}

message GPSPoint {
  string driver_id   = 1;
  string vehicle_id  = 2;   // optional; resolvable from active trip if empty
  string trip_id     = 3;   // optional; resolvable from active trip if empty
  double latitude    = 4;
  double longitude   = 5;
  double speed       = 6;
  double fuel_level  = 7;
  double odometer    = 8;
  string timestamp   = 9;   // RFC3339
}

message PushGPSResponse {
  bool   success      = 1;
  string snapshot_id  = 2;
  string server_time  = 3;
}

message StreamGPSResponse {
  int32  received = 1;
  string server_time = 2;
}
```

Auth: every gRPC call requires metadata `authorization: Bearer <GRPC_AUTH token>`; rejected with `codes.Unauthenticated` otherwise (interceptor, §5).

Error codes:
- `GetDriverLocation` → `codes.NotFound` if no active trip / no snapshot; `codes.Unauthenticated` if token missing/invalid.
- `PushGPS` → `codes.InvalidArgument` if lat/lng missing; `codes.Unauthenticated` if token invalid.

---

## 3. DB contract

**NO new migration.** Reuse `telemetry_snapshots` (`db/migrations/00031_avandab_critical_fixes.sql:4`).

Recommended (optional, non-blocking) index to make `GetDriverLocation` fast — add ONLY if a perf issue appears; do NOT create a new migration number for it without updating `00-migration-ownership-index.md`. If added later, it is a pure additive index:

```sql
-- +goose Up
CREATE INDEX IF NOT EXISTS idx_telemetry_snapshots_vehicle
  ON telemetry_snapshots(vehicle_id, timestamp DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_telemetry_snapshots_vehicle;
```

`GetDriverLocation` resolution chain (read-only SELECTs, no writes):
1. Resolve the driver's **active trip**: `SELECT id, vehicle_id FROM trips WHERE driver_id = ? AND status NOT IN ('completed','cancelled') ORDER BY departure_time DESC LIMIT 1;`
2. Use that `vehicle_id` (fallback: join `vehicles` if needed).
3. Latest snapshot: `SELECT latitude, longitude, speed, timestamp FROM telemetry_snapshots WHERE vehicle_id = ? ORDER BY timestamp DESC LIMIT 1;`
4. If no snapshot yet → `NotFound`. (Do NOT fall back to hardcoded Pune — that was the stub behavior and is removed.)

`PushGPS` write reuses `internal/telemetry/sync.go:72` insert semantics: `INSERT OR REPLACE INTO telemetry_snapshots (id, trip_id, vehicle_id, timestamp, latitude, longitude, speed, fuel_level, odometer) VALUES (?,?,?,?,?,?,?,?,?)` with `id = 'snap-' + RFC3339Nano(now)`.

---

## 4. UI / mobile

- **Mobile (driver app / trackers):** reads trips via GraphQL `POST /query` with a bearer token (same `RequireAPIAuth` path). Example query for a live board:
  ```graphql
  { trips(status: "in_transit") { id driverName vehicleNumber origin destination status } }
  ```
- **Live driver GPS** continues to arrive over **MQTT** (`avandab/telemetry/drivers/{driverId}/gps`, `internal/mqttservice/mqtt.go:45`). `gRPC TelemetryIngest.PushGPS` is a server-to-server / test ingest, not the driver-facing path.
- **Web admin:** no new pages. Existing `/trips`, `/bookings`, `/invoices` pages unchanged. GraphQL is an API-only addition (mobile + external integrators).
- **gRPC consumers:** dispatch/ops dashboards or external microservices call `DispatchService.GetDriverLocation` over `:50051`.

---

## 5. Business logic

### 5.1 gqlgen resolver → existing use cases

Build resolvers that call the already-wired vertical-slice use cases (no new domain logic):

- `trips(...)` / `trip(id)` → `tripApp.ListTripsUseCase` (`internal/trip/application/list_trips.go`) and `tripApp.GetTripUseCase` (wired `cmd/server/main.go:231-232`). Map `ListTripsResponse.Trips` DTO → GraphQL `Trip` (driverName = `DriverFirstName + " " + DriverLastName`, origin/destination = `RouteSource`/`RouteDestination`).
- `bookings(...)` / `booking(id)` → `bookingApp.ListBookingsUseCase` / `GetBookingUseCase` (wired `cmd/server/main.go:217-218`).
- `invoices(...)` / `invoice(id)` → `invoiceApp.ListInvoicesUseCase` / `GetInvoiceUseCase` (wired `cmd/server/main.go:236-237`).

Tenant isolation: every resolver reads `tenantID` from `auth.ContextUser`/context (per AGENTS.md: never trust client `tenant_id`). gqlgen `Context` → extract from the HTTP request context already populated by `RequireAPIAuth`.

> The old `internal/graphqlservice/handler.go` (mock, hardcoded) is **deleted** and replaced by the gqlgen HTTP handler mounted at the same routes (`cmd/server/main.go:446-447`).

### 5.2 gRPC DispatchService implementation

File: `internal/grpcservice/dispatch_server.go` (new). `DispatchServer` holds `*sql.DB` (or a repository) + clock.

`GetDriverLocation(ctx, *LocationRequest)`:
```
1. token := grpc metadata authorization; interceptor already validated → ctx has subject.
2. driverID := req.DriverId
3. row := db.QueryRowContext(ctx,
     `SELECT t.id, t.vehicle_id FROM trips t
      WHERE t.driver_id = ? AND t.status NOT IN ('completed','cancelled')
      ORDER BY t.departure_time DESC LIMIT 1`, driverID)
   if no row → return nil, status.Error(codes.NotFound, "no active trip for driver")
4. var tripID, vehicleID string; scan.
5. snap := db.QueryRowContext(ctx,
     `SELECT latitude, longitude, speed, timestamp FROM telemetry_snapshots
      WHERE vehicle_id = ? ORDER BY timestamp DESC LIMIT 1`, vehicleID)
   if no row → return nil, status.Error(codes.NotFound, "no telemetry for vehicle")
6. return &LocationResponse{DriverId, VehicleId: vehicleID, Latitude, Longitude, Speed, LastUpdated: ts}, nil
```

`UpdateTripStatus` is **not registered** (write gate). The proto message stays for forward-compat only.

### 5.3 gRPC TelemetryIngest implementation

File: `internal/grpcservice/telemetry_server.go` (new).

`PushGPS(ctx, *GPSPoint)`:
```
1. if req.Latitude==0 && req.Longitude==0 → InvalidArgument (allow explicit 0,0? No: require both present & in range [-90,90]/[-180,180]).
2. resolve vehicle_id / trip_id if empty:
     if req.VehicleId=="" || req.TripId=="" →
       SELECT vehicle_id FROM trips WHERE driver_id=? AND status NOT IN ('completed','cancelled') LIMIT 1
3. ts := req.Timestamp; if "" → now RFC3339.
4. snapshotID := "snap-" + time.Now().UTC().Format("20060102150405.000000000")
5. INSERT OR REPLACE telemetry_snapshots (id, trip_id, vehicle_id, timestamp, latitude, longitude, speed, fuel_level, odometer)
   VALUES (?,?,?,?,?,?,?,?,?)
6. return &PushGPSResponse{Success:true, SnapshotId: snapshotID, ServerTime: now}
```

### 5.4 gRPC auth interceptor

`internal/grpcservice/auth.go` (new): unary + stream interceptor reading `md.Get("authorization")` → strip `Bearer ` → compare against `cfg.GRPC.AuthToken` (env `GRPC_AUTH`). On mismatch → `codes.Unauthenticated`. Per-call subject can be stuffed into context for audit.

### 5.5 Server wiring (replace stub)

`internal/grpcservice/server.go` rewritten:
```go
func StartGRPCServer(port string, db *sql.DB, authToken string) {
  lis, _ := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
  s := grpc.NewServer(
    grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(authToken)),
    grpc.StreamInterceptor(auth.StreamAuthInterceptor(authToken)),
  )
  pb.RegisterDispatchServiceServer(s, &DispatchServer{db: db})
  pb.RegisterTelemetryIngestServer(s, &TelemetryServer{db: db})
  go s.Serve(lis)
}
```
Update call site `cmd/server/main.go:428` → `grpcservice.StartGRPCServer(grpcPort, database, cfg.GRPC.AuthToken)`.

---

## 6. Config / env

Add a `GRPC` block to `internal/config/config.go`:

```go
type GRPCConfig struct {
  Enabled  bool
  Port     string
  AuthToken string
}
// inside Config: GRPC GRPCConfig
// inside Load():
cfg.GRPC = GRPCConfig{
  Enabled:   getEnv("GRPC_ENABLED", "true") == "true",
  Port:      getEnv("GRPC_PORT", "50051"),
  AuthToken: os.Getenv("GRPC_AUTH"),
}
```

GraphQL has no new flag (always on with `RequireAPIAuth`), but document the existing toggle implicitly via `GRAPHQL_ENABLED` for symmetry:

| Var | Default | Purpose | Read by |
|-----|---------|---------|---------|
| `GRAPHQL_ENABLED` | `true` | Enable gqlgen HTTP handler at `/query`,`/graphql` | `cmd/server/main.go` (gate mount at :446) |
| `GRPC_ENABLED` | `true` | Start gRPC server | `cmd/server/main.go` (:428) |
| `GRPC_PORT` | `50051` | gRPC listen port | `cmd/server/main.go` (:424-427) + `config.GRPC.Port` |
| `GRPC_AUTH` | `""` | Bearer token for gRPC calls; empty ⇒ reject all (fail-closed) | `internal/grpcservice/auth.go`, `config.GRPC.AuthToken` |

In production require `GRPC_AUTH` non-empty (mirror `cmd/server/main.go:407` API_SECRET guard).

---

## 7. Tests

### 7.1 Real GraphQL field assertions (replace shallow smoke test)

Extend `test/smoke_new_endpoints_test.go` `TestSmoke_GraphQLRealData` to assert real data:
```go
rr := doSmoke(t, router, http.MethodPost, "/query", map[string]interface{}{
  "query": "{ trips { id status driverName vehicleNumber origin destination } }",
})
require.Equal(t, http.StatusOK, rr.Code)
var gqlResp struct {
  Data struct {
    Trips []struct {
      ID string `json:"id"`; Status string `json:"status"`
      DriverName string `json:"driverName"`; VehicleNumber string `json:"vehicleNumber"`
      Origin string `json:"origin"`; Destination string `json:"destination"`
    } `json:"trips"`
  } `json:"data"`
  Errors []interface{} `json:"errors"`
}
require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &gqlResp))
require.Empty(t, gqlResp.Errors)
require.Len(t, gqlResp.Data.Trips, 1)            // the trip seeded above
require.Equal(t, "smoke test trip", ...)         // via status/remarks exposure
// negative: bad query returns errors, not 200-with-data
rrBad := doSmoke(t, router, http.MethodPost, "/query", map[string]interface{}{"query":"{ notAField }"})
require.Contains(t, rrBad.Body.String(), "errors")
```

### 7.2 gRPC bufconn tests (no network)

`internal/grpcservice/dispatch_server_test.go`:
```go
func TestGetDriverLocation_NotFound(t *testing.T) {
  srv, conn := startBufconn(t)            // grpc.NewServer + bufconn, register DispatchServer with test DB
  c := pb.NewDispatchServiceClient(conn)
  _, err := c.GetDriverLocation(ctx, &pb.LocationRequest{DriverId: "ghost"})
  require.Error(t, err)
  require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetDriverLocation_RealCoords(t *testing.T) {
  // seed active trip for driver D with vehicle V, insert snapshot lat=19.0 lng=74.0
  resp, err := c.GetDriverLocation(ctx, &pb.LocationRequest{DriverId: "D"})
  require.NoError(t, err)
  require.Equal(t, 19.0, resp.Latitude)
  require.Equal(t, 74.0, resp.Longitude)
  require.Equal(t, "V", resp.VehicleId)
}

func TestPushGPS_WritesSnapshot(t *testing.T) {
  c := pb.NewTelemetryIngestClient(conn)
  r, err := c.PushGPS(ctx, &pb.GPSPoint{DriverId:"D", Latitude:20.0, Longitude:75.0})
  require.NoError(t, err); require.True(t, r.Success)
  // assert row in telemetry_snapshots via test DB
}

func TestGRPC_Unauthenticated(t *testing.T) {
  // dial WITHOUT auth metadata → expect codes.Unauthenticated on any RPC
}
```

### 7.3 Contract compile gate

Add `make proto` / CI step: `protoc --go_out=. --go-grpc_out=. internal/grpcservice/dispatch.proto` must succeed and `go build ./...` must pass. Add `gqlgen` generate step (`go run github.com/99designs/gqlgen generate`) to CI; remove the `// indirect` by adding a `tools.go` with `//go:build tools` importing gqlgen.

**Pass-before-merge checklist:**
- [ ] `go build ./...` clean
- [ ] `go test ./test/... ./internal/grpcservice/...` green
- [ ] proto regenerates (`pb` package present)
- [ ] `GetDriverLocation` returns `NotFound` (no hardcoded Pune)
- [ ] GraphQL test asserts real field values, not just `serverTime`

---

## 8. Future / GPS-provider

- **`StreamGPS` client-streaming** (defined in proto): accept a stream of `GPSPoint`, buffer, batch-`INSERT` per N points or per flush; return `StreamGPSResponse{received, server_time}`. Implement in `internal/grpcservice/telemetry_server.go`.
- **Server-streaming `SubscribeTripTelemetry`** (new service/method, future): `rpc SubscribeTripTelemetry(TripRequest) returns (stream GPSPoint)` pushes live points as they land. Back it by a pub/sub fan-out fed by both MQTT and `PushGPS` writes (shared write path below).
- **MQTT + gRPC share a write path.** Refactor `internal/mqttservice/mqtt.go:46` subscriber to call the **same** persistence function that `PushGPS` uses (extract `persistGPS(ctx, db, point)` into `internal/telemetry`). Then MQTT (driver-facing) and gRPC (server-to-server/test) both feed `telemetry_snapshots` identically — single source of truth, no divergence.
- **TelematicsProvider abstraction** (per AGENTS.md): define `type TelematicsProvider interface { Ingest(ctx, GPSPoint) error }`; MQTT and gRPC are two adapters behind it; third-party (LocoNav/WheelsEye/MapMyIndia) plug in later without touching `telemetry_snapshots`.
- **Scale:** `telemetry_snapshots` will grow; add the `idx_telemetry_snapshots_vehicle` index (§3) and consider partitioning/retention job later. Reads stay point-lookups on `vehicle_id + timestamp DESC`.

---

## 9. Edge cases

- **No active trip for driver** → `GetDriverLocation` returns `NotFound` (NOT Pune hardcoded). Mobile must handle "location unavailable".
- **Active trip but no GPS yet** → `NotFound` ("no telemetry for vehicle"); do not synthesize.
- **Multiple active trips for one driver** → pick latest `departure_time DESC LIMIT 1` (deterministic).
- **`PushGPS` with empty `vehicle_id`/`trip_id`** → resolved from active trip via `driver_id`; if unresolvable → `InvalidArgument`.
- **Out-of-range lat/lng** → reject `InvalidArgument`.
- **gRPC without `GRPC_AUTH` configured in prod** → server refuses to start (mirror API_SECRET guard).
- **gqlgen query error / resolver panic** → gqlgen returns `errors` array, HTTP 200 with `data:null`; `Recoverer` middleware is HTTP-only and does not cover gRPC — gRPC handlers must return errors, never panic.
- **Tenant isolation** → GraphQL resolvers and gRPC (when scoped) must filter by tenant from `ContextUser`, never client-supplied `tenant_id`.
- **Auth token rotation** → `GRPC_AUTH` is read at startup; restart required to rotate (documented limitation).

---

## 10. Phased rollout (build order)

1. **Proto + codegen:** rewrite `dispatch.proto`, run `protoc` → `internal/grpcservice/pb/`. Add `tools.go` for gqlgen.
2. **gqlgen bootstrap:** add `gqlgen.yml`, `gql_schema.graphqls`, generate `internal/graphqlservice/gqlgen/`. Implement resolvers calling existing use cases.
3. **Replace GraphQL handler:** delete `internal/graphqlservice/handler.go` mock; mount gqlgen `Handler` at `cmd/server/main.go:446-447`.
4. **gRPC real server:** rewrite `server.go`, add `dispatch_server.go`, `telemetry_server.go`, `auth.go`. Wire `StartGRPCServer(grpcPort, database, cfg.GRPC.AuthToken)`.
5. **Config:** add `GRPCConfig` to `config.go`, env table (§6), prod guard for `GRPC_AUTH`.
6. **Tests:** bufconn gRPC tests + real GraphQL field assertions; CI proto/gqlgen generate gates.
7. **Shared write path (future):** extract `persistGPS`, point MQTT subscriber at it (§8).

---

## 11. Open items / VERIFY

- **MUST:** confirm `trips` table has `driver_id`, `vehicle_id`, `status`, `departure_time` columns (verified via `TripResponseDTO` at `internal/trip/application/list_trips.go:66-75`; confirm column names in `db/migrations/00007_trips.sql` before SQL in §5).
- **MUST:** confirm `RequireAPIAuth` populates a context value gqlgen can read for tenant (verify `auth.ContextUser` key; adapt resolver context extraction).
- **DECISION point — keep or remove:** this spec implements the REAL version. If the team decides these transports are not needed, the removal path is: delete `internal/graphqlservice/handler.go`, `internal/grpcservice/server.go`, `dispatch.proto`; remove mounts at `cmd/server/main.go:428,431,446-447`; drop `gqlgen // indirect` from `go.mod`. No DB change.
- **VERIFY:** gqlgen v0.17.94 compatibility with Go version in `go.mod` (run `go mod tidy` after adding direct dep).
- **VERIFY:** `GRPC_PORT` 50051 not blocked in deploy; consider `0.0.0.0` vs `127.0.0.1` (the host literal is hardcoded as `127.0.0.1` inside `internal/grpcservice/server.go:16`, NOT `cmd/server/main.go` — external consumers would need `0.0.0.0` + `GRPC_AUTH`).

---

## 12. File list (create / modify)

**Create**
- `internal/grpcservice/dispatch.proto` (rewrite with DispatchService + TelemetryIngest + GPSPoint)
- `internal/grpcservice/pb/dispatch.pb.go` (generated)
- `internal/grpcservice/pb/dispatch_grpc.pb.go` (generated)
- `internal/grpcservice/dispatch_server.go` (GetDriverLocation impl)
- `internal/grpcservice/telemetry_server.go` (PushGPS impl; StreamGPS stub)
- `internal/grpcservice/auth.go` (unary+stream auth interceptors)
- `internal/grpcservice/dispatch_server_test.go` (bufconn tests)
- `internal/graphqlservice/gqlgen/gql_schema.graphqls` (SDL)
- `internal/graphqlservice/gqlgen/generated.go` (gqlgen generated)
- `internal/graphqlservice/gqlgen/models_gen.go` (generated)
- `internal/graphqlservice/resolvers.go` (resolver → use cases)
- `gqlgen.yml` (gqlgen config)
- `tools.go` (build-tag tools file pinning gqlgen)

**Modify**
- `internal/graphqlservice/handler.go` → DELETE (replaced by gqlgen handler)
- `internal/grpcservice/server.go` → rewrite `StartGRPCServer(port, db, authToken)` + register services
- `cmd/server/main.go` → :428 pass `database` + `cfg.GRPC.AuthToken`; :431 remove `NewGraphQLHandler`; :446-447 mount gqlgen `Handler`
- `internal/config/config.go` → add `GRPCConfig` + `GRPC` field + load block
- `test/smoke_new_endpoints_test.go` → strengthen `TestSmoke_GraphQLRealData` with real field asserts
- `go.mod` → promote `gqlgen` to direct; add `google.golang.org/grpc` (already present) + `google.golang.org/protobuf`

**No DB migration / No new table.**

---

## 13. Principal recommendation (build vs remove)

The spec was written as "locked: implement the REAL version." A neutral QA pass flips that default: nothing here is load-bearing for any **existing** client. Recommendations below are engineering judgments, not mandates — ownership is the team's.

### 13.1 gRPC — **RECOMMEND REMOVAL (do not build)**

- **Decision:** Delete `internal/grpcservice/server.go` + `dispatch.proto` (the stub) and drop `gqlgen // indirect` cleanup only if GraphQL also dropped. Do NOT invest in proto/codegen, auth interceptor, or bufconn tests.
- **Tradeoff:** We lose a typed, low-latency RPC surface. But no consumer exists today:
  - Mobile/driver app uses **REST + MQTT** (per AGENTS.md); live GPS already arrives over MQTT (`mqtt.go:45`).
  - `GetDriverLocation` is achievable over **GraphQL** (add a `location` field on `Trip` backed by the same `telemetry_snapshots` chain) or an existing REST route — no new protocol needed.
  - `TelemetryIngest.PushGPS` duplicates the **already-working** REST paths `POST /api/v1/telemetry/sync` and `POST /api/v1/telemetry/snapshots` (`sync.go:48-49`). A second ingest path will diverge from MQTT/REST and double the persistence surface.
- **Cost of building (avoided):** ~4-5 new files (`pb/` x2, `dispatch_server.go`, `telemetry_server.go`, `auth.go`, `dispatch_server_test.go`), a `protoc`/codegen CI toolchain, a new network listener (`GRPC_PORT`), and an auth interceptor — all for zero current callers. Also an added attack surface (a second open port + bearer scheme).
- **Cost of removal (now):** trivial — delete stub, remove mounts at `main.go:428,431,446-447` (§11 path). Revisit only when a concrete external microservice contract demands gRPC.

### 13.2 GraphQL (gqlgen) — **BUILD only if a typed consumer is committed; otherwise keep the existing minimal handler**

- **Decision:** The current mock already returns `activeTrips` + `serverTime` (C1). If only that shape is needed, a thin JSON handler suffices and gqlgen is overkill. Adopt gqlgen **only** when a mobile/external integrator commits to a typed GraphQL client.
- **Tradeoff (value):** Real query parsing, field selection, and a schema contract reduce over-fetching and give integrators a stable API. **vs (cost):** gqlgen build complexity — generated `generated.go`/`models_gen.go`, `gqlgen.yml`, `tools.go` build-tag, a `go run gqlgen generate` CI gate, and a larger dependency surface. The spec's §0 claim that it is `// indirect`/unused (C2) confirms it is not yet wired.
- **Effort:** Medium (codegen + resolvers + strengthened smoke test §7.1).

### 13.3 bufconn testing — **deferred with gRPC**

- **Decision:** Only relevant if §13.1 is reversed. If gRPC is ever built, bufconn (in-memory listener, no port, hermetic) is the correct choice for `dispatch_server_test.go` (§7.2) — no real network, fast, no `GRPC_PORT` conflict in CI.
- **Tradeoff:** None vs real TCP in tests; strictly better. Moot while gRPC is removed.

### 13.4 Bottom line

Ship **GraphQL only if a consumer exists**; **remove the gRPC stub now** rather than "implementing the real version." The locked §0 DECISION line should read: *GraphQL optional behind a committed client; gRPC removed until an external contract requires it.* This cuts ~5 files + a CI toolchain and one network listener with no loss of current functionality.
