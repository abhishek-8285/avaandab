# MVTMS Architecture Guide

> Multi-Vehicle Transport Management System (branded as "Avandab")

This document describes the full architecture: backend, database, web UI, mobile app, auth, and deployment.

---

## 1. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     cmd/server/main.go (Entry)                   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Vertical Slice Modules (new / Sprint-based)             │   │
│  │  each: application/ (use cases) + presentation/           │   │
│  │        + domain/ (entity, events, repo interface)          │   │
│  │  booking/  trip/  invoice/  payment/  auth/                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Traditional Clean Architecture (established)              │   │
│  │  handlers → service → repository → domain                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Cross-cutting Infrastructure                              │   │
│  │  auth (sessions, Casbin, API tokens)                     │   │
│  │  events (in-memory bus)                                  │   │
│  │  middleware (auth, logging, recovery, rate limiting)     │   │
│  │  config / logging / telemetry / migrations               │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

The project uses a **hybrid architecture**: an established Clean/Hexagonal layer (`internal/domain`, `internal/service`, `internal/repository`, `internal/handlers`) alongside newer **DDD vertical slices** (`internal/booking`, `internal/trip`, etc.). Both coexist and are wired together in `cmd/server/main.go`.

---

## 2. Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| Router | chi/v5 |
| Database | SQLite (modernc.org/sqlite — pure Go, no CGO) |
| Migrations | goose/v3 (embedded via `go:embed`) |
| ORM/Query | Raw SQL + sqlc query files in `db/query/` |
| GraphQL | gqlgen (mock handler currently active) |
| gRPC | google.golang.org/grpc |
| MQTT | eclipse/paho.mqtt.golang |
| WebSocket | gorilla/websocket |
| Auth | Casbin v2 (RBAC), gorilla/securecookie |
| Password Hashing | golang.org/x/crypto (bcrypt) |
| Logging | log/slog (JSON in production) |
| Telemetry | OpenTelemetry + Prometheus |
| Web Frontend | Go `html/template` + Datastar + Tailwind CSS |
| Mobile Frontend | React Native (Expo) + TypeScript + Zustand |
| Container | Docker multi-stage (golang → distroless) |

---

## 3. Directory Structure

```
cmd/
├── server/          # Main server entry point (main.go)
├── agent/           # Android OTA auto-update agent
└── migrate/         # Database migration CLI

internal/
├── domain/          # Core domain entities, value objects, types
│   ├── booking/     # entity.go, service.go, events.go, repository.go, errors.go
│   ├── trip/        # entity.go, service.go, events.go, repository.go
│   ├── driver/      # entity.go
│   ├── vehicle/     # entity.go
│   ├── customer/    # entity.go
│   ├── route/       # entity.go
│   ├── invoice/     # entity.go, service.go, repository.go
│   ├── payment/     # entity.go, service.go, repository.go
│   ├── dispatch/    # entity.go
│   ├── expense/     # entity.go (DriverExpense / Kharcha)
│   ├── ewaybill/    # entity.go (E-Way Bill)
│   ├── user/        # entity.go, repository.go (User, Role, Session)
│   ├── company/     # entity.go (CompanySettings)
│   ├── audit/       # entity.go (AuditLog)
│   ├── notification/ # entity.go (Notification)
│   ├── types/       # ids.go (typed IDs), file.go, session.go, uow.go, clock.go
│   └── aliases.go   # Re-exports for backward compatibility
├── service/         # Application services (use-case orchestrators)
│   ├── service.go   # Services struct, baseService, Store interface, event wiring
│   ├── booking_service.go
│   ├── trip_service.go
│   ├── invoice_service.go
│   ├── payment_service.go
│   ├── driver_service.go
│   ├── vehicle_service.go
│   ├── customer_service.go
│   ├── route_service.go
│   ├── user_service.go
│   ├── auth_service.go
│   ├── company_service.go
│   ├── dashboard_service.go
│   ├── contact_service.go
│   ├── report_service.go
│   ├── telemetry_service.go
│   ├── settlement_service.go
│   └── kharcha_service.go
├── repository/      # Central repository interfaces
│   └── sqlite/      # SQLite implementation (SqliteStore)
├── handlers/        # HTTP handlers (18 files, one per module)
│   ├── app.go       # App struct, router setup, template rendering
│   ├── auth.go      # Login, register, logout, profile
│   ├── dashboard.go # KPI dashboard + WebSocket
│   ├── bookings.go
│   ├── trips.go
│   ├── invoices.go
│   ├── payments.go
│   ├── drivers.go
│   ├── vehicles.go
│   ├── customers.go
│   ├── routes.go
│   ├── users.go
│   ├── settings.go
│   ├── audit_logs.go
│   ├── reports.go
│   ├── contact.go
│   ├── kharcha.go
│   └── middleware.go # Auth middleware, SPA detection
├── auth/            # Authentication infrastructure
│   ├── session.go   # SessionStore (securecookie-based)
│   ├── casbin.go    # DBAdapter for Casbin RBAC
│   ├── apitoken.go  # API token generation/validation
│   └── password.go  # bcrypt hashing
├── events/          # In-memory event bus
│   └── bus.go       # Event, Handler, EventBus, InMemoryBus
├── config/          # Environment-based configuration
├── logging/         # slog setup
├── telemetry/       # OpenTelemetry setup
├── middleware/      # HTTP middleware (require auth, logging, recovery, SPAM)
├── graphqlservice/  # GraphQL endpoint
├── grpcservice/     # gRPC server
├── mqttservice/     # MQTT client
├── templates/       # HTML templates (layout, auth_layout, pages, partials)
├── static/          # CSS, JS, images
├── shared/          # clock, id, uow (shared infrastructure for vertical slices)
├── booking/         # Vertical slice: booking module
│   ├── application/  # Use cases (Create, Confirm, Cancel, etc.)
│   └── presentation/api/handlers/
├── trip/            # Vertical slice: trip module
├── invoice/         # Vertical slice: invoice module
├── payment/         # Vertical slice: payment module
├── auth/presentation/ # Vertical slice: auth API handlers
├── pdf/             # PDF generation (fpdf)
├── founder/         # Alert/notification system
└── operations/      # Internal tooling / health checks

db/
├── migrations/      # 32 SQL migration files (goose format)
└── query/           # sqlc query files (auth.sql, bookings.sql, etc.)

mobile/              # React Native app
├── src/
│   ├── components/    # Reusable UI components
│   ├── constants/     # App constants
│   ├── services/      # API clients (analytics, graphql, mqtt, storage, syncEngine, telemetry)
│   ├── stores/        # Zustand state management
│   ├── types/         # TypeScript type definitions
│   └── App.tsx        # Root component
├── package.json

web/                 # (if separate from internal/static)
```

---

## 4. Domain Model

All domain entities use **typed string IDs** (`internal/domain/types/ids.go`) for type safety. IDs are UUID-based strings prefixed by entity type.

### Typed IDs

```go
type UserID string        // "usr_..."
type DriverID string       // "drv_..."
type VehicleID string      // "veh_..."
type CustomerID string     // "cust_..."
type RouteID string        // "rout_..."
type BookingID string      // "book_..."
type TripID string         // "trip_..."
type DispatchID string     // "disp_..."
type InvoiceID string      // "inv_..."
type PaymentID string      // "pay_..."
type FileID string         // "file_..."
type SessionID string      // "sess_..."
```

### Entity Relationships

```
Customer ──< Booking >─── Route
                  │
                  └─> Trip ──> Driver, Vehicle
                          │
                          ├─> EWayBill
                          ├─> Invoice ──> Payment
                          ├─> DriverExpense (Kharcha)
                          └─> DriverSettlement
                          └─> TelemetrySnapshot, TelemetryAlert
                          └─> Dispatch (planned via dispatch record)

User ──< Session
User ──< AuditLog
User ──< Notification
```

### Entity Details

#### Booking (Aggregate Root)

**State machine**:
```
Draft → Pending → Confirmed → Completed
              ↓
          Cancelled (immutable)
```

| Field | Type | Description |
|---|---|---|
| ID | `types.BookingID` | PK |
| BookingNumber | string | Unique |
| CustomerID | `types.CustomerID` | FK → customers |
| PickupDate | time.Time | Scheduled pickup time |
| RouteID | `types.RouteID` | FK → routes |
| VehicleType | `vehicle.VehicleType` | truck, mini_truck, bus, van, pickup, tempo |
| Passengers | int64 | Must be >= 1 |
| CargoWeight | *float64 | Nullable |
| Price | float64 | >= 0 |
| Notes | *string | |
| Status | BookingStatus | draft, pending, confirmed, cancelled, completed |
| ApprovedBy | *string | Dispatcher who approved |
| ApprovedAt | *time.Time | |
| Version | int | Optimistic concurrency control |
| TenantID | string | Multi-tenancy |

**Domain methods**: `CanConfirm()`, `CanCancel()`, `CanDelete()`

**Events**: `BookingCreatedEvent`, `BookingConfirmedEvent`, `BookingCancelledEvent`, `BookingCompletedEvent`

#### Trip

**State machine**:
```
Draft → Scheduled → Assigned → Started → ReachedPickup → InTransit → Delivered → Completed
                                                                   ↓
                                                            Cancelled (immutable)
```

| Field | Type | Description |
|---|---|---|
| ID | `types.TripID` | PK |
| TripNumber | string | Unique |
| BookingID | *BookingID | FK → bookings |
| DriverID | *DriverID | FK → drivers |
| VehicleID | *VehicleID | FK → vehicles |
| RouteID | RouteID | FK → routes |
| DepartureTime | time.Time | |
| ArrivalTime | *time.Time | |
| Status | TripStatus | draft, scheduled, assigned, started, reached_pickup, in_transit, delivered, completed, cancelled |
| EWayBillRef | *string | |
| PODURL | *string | Proof of delivery |
| FinalSettlementAmount | float64 | |
| Remarks | *string | |
| Timeline fields | *time.Time | StartedAt, ReachedPickupAt, InTransitAt, DeliveredAt, CompletedAt |
| e-POD fields | — | PhotoURL, SignatureURL, ConsigneeName, ConsigneePhone, OtpVerified, CapturedAt, Lat, Lng, Notes |

**Validation methods**: `CanSchedule()`, `CanStart()`, `CanDeliver()`, `CanComplete()`, `CanCancel()`

#### Driver

| Field | Type |
|---|---|
| ID | `types.DriverID` |
| DriverID | string | Human-readable driver code |
| FirstName, LastName | string |
| Phone | string |
| Email | *string |
| Address | *string |
| LicenseNumber | string |
| LicenseExpiry | time.Time |
| ExperienceYears | int64 |
| Status | DriverStatus (available, on_trip, leave, inactive, blocked) |
| Blocked | bool |
| Aadhaar | *string |
| PAN | *string |
| BankDetails | *string |
| EmergencyContactName, EmergencyContactPhone | *string |
| Notes | *string |

#### Vehicle

| Field | Type |
|---|---|
| ID | `types.VehicleID` |
| RegistrationNumber | string |
| VehicleNumber | string |
| VehicleType | VehicleType (truck, mini_truck, bus, van, pickup, tempo) |
| Capacity | int64 |
| FuelType | FuelType (diesel, petrol, gas, electric, cng) |
| InsuranceExpiry, FitnessExpiry, PermitExpiry, RCExpiry | time.Time |
| Status | VehicleStatus (available, running, maintenance, inactive, blocked) |
| Blocked | bool |
| DriverID | *DriverID |
| Odometer | float64 |
| CurrentMileage | *float64 |

#### Customer

| Field | Type |
|---|---|
| ID | `types.CustomerID` |
| CustomerCode | string (unique) |
| Name | string |
| Company | *string |
| ContactPerson | *string |
| Phone | string |
| Email | *string |
| GST | *string |
| Address | *string |
| BillingAddress | *string |
| PaymentTermsDays | int |
| Type | string (individual, company) |
| Status | string (active, inactive) |

#### Route

| Field | Type |
|---|---|
| ID | `types.RouteID` |
| Source | string |
| Destination | string |
| Distance | float64 |
| EstimatedHours | float64 |
| StandardFare | float64 |
| ReverseDistance | *float64 |
| ReverseStandardFare | *float64 |
| Remarks | *string |

**Method**: `GetDistanceAndFare(from, to string)` — handles bidirectional lookup.

#### Invoice

| Field | Type |
|---|---|
| ID | `types.InvoiceID` |
| InvoiceNumber | string |
| BookingID | BookingID |
| CustomerID | CustomerID |
| TripID | *TripID |
| Subtotal, Tax, Discount, Total | float64 |
| PaidAmount | float64 |
| Status | InvoiceStatus (draft, issued, outstanding, paid, cancelled) |
| PaymentStatus | PaymentStatus (pending, paid, partially_paid) |
| DueDate | *time.Time |
| FinancialYear | string |

**Methods**: `OutstandingBalance()`, `MarkIssued(dueDate)`, `ApplyPayment(amount)`

#### Payment

| Field | Type |
|---|---|
| ID | `types.PaymentID` |
| InvoiceID | InvoiceID |
| PaymentDate | time.Time |
| Amount | float64 |
| Method | PaymentMethod (cash, bank_transfer, upi, cheque) |
| Reference | *string |
| Remarks | *string |

#### Dispatch

| Field | Type |
|---|---|
| ID | `types.DispatchID` |
| DispatchNo | string (unique) |
| DispatcherID | UserID |
| BookingID | BookingID |
| DriverID, VehicleID | *DriverID, *VehicleID |
| ScheduledAt | time.Time |
| Status | DispatchStatus (draft, assigned, converted, cancelled) |
| TripID | *TripID |

#### Driver Expense (Kharcha)

| Field | Type |
|---|---|
| ID | string |
| TripID | TripID |
| DriverID | DriverID |
| ExpenseType | fuel, toll, food, repair, advance |
| Amount | float64 |
| Status | pending, approved, rejected, settled |
| Category | advance, fuel, toll, food, repair, other |
| RequestedBy | *DriverID |
| ApprovedBy | *UserID |
| ApprovedAt | *time.Time |
| RejectedReason | *string |
| ReceiptURL | *string |
| Description | *string |

#### E-Way Bill

| Field | Type |
|---|---|
| ID | string |
| TripID | TripID (unique) |
| EWBNumber | string (unique) |
| IRN | *string (unique) |
| GenerationDate | time.Time |
| ValidUntil | time.Time |
| TransporterID | *string |
| VehicleNumber | *string |
| Status | active, cancelled, expired |
| RawResponse | *string |

#### User & RBAC

| Field | Type |
|---|---|
| ID | `types.UserID` |
| Email | string (unique) |
| PasswordHash | string (bcrypt) |
| Name | string |
| Phone | *string |
| Timezone | string |
| Role | Role (embedded struct) |
| Status | UserStatus (active, inactive, suspended) |
| LastLoginAt | *time.Time |

**Roles**:
| Role | Description |
|---|---|
| admin (id=1) | Full access — all permissions |
| dispatcher (id=2) | Manage drivers, vehicles, customers, routes, bookings, trips; view reports/settings |
| accountant (id=3) | Invoices, payments, reports |
| viewer (id=4) | Read-only access |
| driver (id=5) | Driver-specific actions |

**Permission model**: Resource-action pairs (e.g., `bookings:create`, `trips:assign`, `invoices:export`). Permissions are assigned to roles via `role_permissions` table. Policy stored in DB, loaded by Casbin DBAdapter.

---

## 5. Database Schema

SQLite database with 32 goose migrations. PRAGMAs: `journal_mode=WAL`, `synchronous=OFF`, `cache_size=-131072` (128MB), `mmap_size=536870912` (512MB), `foreign_keys=ON`, `temp_store=MEMORY`.

### Core Auth Tables

**`roles`**
| Column | Type | Constraints |
|---|---|---|
| id | INTEGER | PK AUTOINCREMENT |
| name | TEXT | UNIQUE, NOT NULL |
| description | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`permissions`**
| Column | Type | Constraints |
|---|---|---|
| id | INTEGER | PK AUTOINCREMENT |
| name | TEXT | UNIQUE, NOT NULL |
| description | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`role_permissions`** (many-to-many: roles ↔ permissions)
| Column | Type | Constraints |
|---|---|---|
| role_id | INTEGER | PK, FK→roles, CASCADE DELETE |
| permission_id | INTEGER | PK, FK→permissions, CASCADE DELETE |

**`user_roles`** (many-to-many: users ↔ roles)
| Column | Type | Constraints |
|---|---|---|
| user_id | TEXT | PK, FK→users, CASCADE DELETE |
| role_id | INTEGER | PK, FK→roles, CASCADE DELETE |

**`users`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| email | TEXT | UNIQUE, NOT NULL |
| password_hash | TEXT | NOT NULL |
| name | TEXT | NOT NULL |
| phone | TEXT | |
| role_id | INTEGER | FK→roles, DEFAULT 2 |
| status | TEXT | DEFAULT 'active', CHECK(active, inactive, suspended) |
| last_login_at | DATETIME | |
| timezone | TEXT | DEFAULT 'Asia/Kolkata' |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`sessions`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| user_id | TEXT | FK→users, ON DELETE CASCADE |
| token_hash | TEXT | NOT NULL |
| expires_at | DATETIME | NOT NULL |
| user_agent | TEXT | |
| ip_address | TEXT | |
| created_at | DATETIME | DEFAULT now |

**`api_tokens`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| token_hash | TEXT | UNIQUE, NOT NULL |
| user_id | TEXT | FK→users |
| name | TEXT | |
| created_at | DATETIME | DEFAULT now |
| expires_at | DATETIME | |

**`company_settings`** (single-row table, id=1)
| Column | Type | Constraints |
|---|---|---|
| id | INTEGER | PK CHECK=1 |
| company_name | TEXT | NOT NULL |
| logo_path | TEXT | |
| address | TEXT | |
| phone | TEXT | |
| email | TEXT | |
| gst_number | TEXT | |
| currency | TEXT | DEFAULT 'INR' |
| timezone | TEXT | DEFAULT 'Asia/Kolkata' |
| gst_enabled | BOOLEAN | DEFAULT 0 |
| gst_rate | REAL | DEFAULT 0.0 |
| booking_prefix | TEXT | DEFAULT 'BK' |
| trip_prefix | TEXT | DEFAULT 'TR' |
| invoice_prefix | TEXT | DEFAULT 'INV' |
| financial_year | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

### Operational Tables

**`drivers`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| driver_id | TEXT | NOT NULL |
| first_name | TEXT | NOT NULL |
| last_name | TEXT | NOT NULL |
| phone | TEXT | NOT NULL |
| email | TEXT | |
| address | TEXT | |
| license_number | TEXT | NOT NULL |
| license_expiry | DATETIME | NOT NULL |
| experience_years | INTEGER | DEFAULT 0 |
| status | TEXT | DEFAULT 'available' |
| blocked | BOOLEAN | DEFAULT 0 |
| blocked_reason | TEXT | |
| aadhaar | TEXT | |
| pan | TEXT | |
| bank_details | TEXT | |
| emergency_contact_name | TEXT | |
| emergency_contact_phone | TEXT | |
| notes | TEXT | |
| version | INTEGER | DEFAULT 1 |
| tenant_id | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`vehicles`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| registration_number | TEXT | NOT NULL |
| vehicle_number | TEXT | |
| model | TEXT | |
| make | TEXT | |
| year | INTEGER | |
| capacity | INTEGER | DEFAULT 1 |
| vehicle_type | TEXT | CHECK(truck, mini_truck, bus, van, pickup, tempo) |
| fuel_type | TEXT | |
| insurance_expiry | DATETIME | |
| fitness_expiry | DATETIME | |
| permit_expiry | DATETIME | |
| rc_expiry | DATE | |
| status | TEXT | DEFAULT 'available' |
| blocked | BOOLEAN | DEFAULT 0 |
| blocked_reason | TEXT | |
| driver_id | TEXT | FK→drivers |
| odometer | REAL | DEFAULT 0.0 |
| current_mileage | REAL | |
| version | INTEGER | DEFAULT 1 |
| tenant_id | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`customers`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| customer_code | TEXT | UNIQUE, NOT NULL |
| name | TEXT | NOT NULL |
| company | TEXT | |
| contact_person | TEXT | |
| phone | TEXT | NOT NULL |
| email | TEXT | |
| gst | TEXT | |
| address | TEXT | |
| billing_address | TEXT | |
| payment_terms_days | INTEGER | DEFAULT 0 |
| type | TEXT | DEFAULT 'individual', CHECK(individual, company) |
| status | TEXT | DEFAULT 'active', CHECK(active, inactive) |
| notes | TEXT | |
| tenant_id | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`routes`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| source | TEXT | NOT NULL |
| destination | TEXT | NOT NULL |
| distance | REAL | NOT NULL |
| estimated_hours | REAL | |
| base_price | REAL | DEFAULT 0 |
| reverse_distance | REAL | |
| reverse_standard_fare | REAL | |
| remarks | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`bookings`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| booking_number | TEXT | UNIQUE, NOT NULL |
| customer_id | TEXT | FK→customers, NOT NULL |
| pickup_date | DATETIME | NOT NULL |
| route_id | TEXT | FK→routes, NOT NULL |
| vehicle_type | TEXT | CHECK(all vehicle types) |
| passengers | INTEGER | DEFAULT 1 |
| cargo_weight | REAL | |
| price | REAL | DEFAULT 0 |
| notes | TEXT | |
| status | TEXT | DEFAULT 'draft', CHECK(draft, pending, confirmed, cancelled, completed) |
| approved_by | TEXT | |
| approved_at | DATETIME | |
| version | INTEGER | DEFAULT 1 |
| tenant_id | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`dispatches`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| dispatch_no | TEXT | UNIQUE, NOT NULL |
| dispatcher_id | TEXT | FK→users, NOT NULL |
| booking_id | TEXT | FK→bookings, NOT NULL |
| driver_id | TEXT | FK→drivers |
| vehicle_id | TEXT | FK→vehicles |
| scheduled_at | DATETIME | NOT NULL |
| status | TEXT | DEFAULT 'draft' |
| trip_id | TEXT | FK→trips |
| notes | TEXT | |
| tenant_id | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`trips`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| trip_number | TEXT | UNIQUE, NOT NULL |
| booking_id | TEXT | FK→bookings |
| driver_id | TEXT | FK→drivers |
| vehicle_id | TEXT | FK→vehicles |
| route_id | TEXT | FK→routes, NOT NULL |
| departure_time | DATETIME | NOT NULL |
| arrival_time | DATETIME | |
| status | TEXT | DEFAULT 'draft' |
| eway_bill_ref | TEXT | |
| pod_url | TEXT | |
| final_settlement_amount | REAL | DEFAULT 0 |
| remarks | TEXT | |
| version | INTEGER | DEFAULT 1 |
| tenant_id | TEXT | DEFAULT '1' |
| started_at | DATETIME | |
| reached_pickup_at | DATETIME | |
| in_transit_at | DATETIME | |
| delivered_at | DATETIME | |
| completed_at | DATETIME | |
| pod_photo_url | TEXT | |
| pod_signature_url | TEXT | |
| pod_consignee_name | TEXT | |
| pod_consignee_phone | TEXT | |
| pod_otp_verified | INTEGER | DEFAULT 0 |
| pod_captured_at | DATETIME | |
| pod_lat | REAL | |
| pod_lng | REAL | |
| pod_notes | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`trips_odometer`**
| Column | Type |
|---|---|
| id | TEXT PK |
| trip_id | TEXT FK→trips |
| timestamp | DATETIME NOT NULL |
| odometer | REAL |
| latitude | REAL |
| longitude | REAL |
| recorded_at | DATETIME |

**`invoices`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| invoice_number | TEXT | UNIQUE, NOT NULL |
| booking_id | TEXT | FK→bookings, NOT NULL |
| customer_id | TEXT | FK→customers, NOT NULL |
| trip_id | TEXT | FK→trips |
| subtotal | REAL | DEFAULT 0 |
| tax | REAL | DEFAULT 0 |
| discount | REAL | DEFAULT 0 |
| total | REAL | DEFAULT 0 |
| paid_amount | REAL | DEFAULT 0 |
| status | TEXT | DEFAULT 'draft' |
| payment_status | TEXT | DEFAULT 'pending', CHECK(pending, paid, partially_paid) |
| due_date | DATETIME | |
| financial_year | TEXT | |
| remarks | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`payments`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| invoice_id | TEXT | FK→invoices, NOT NULL |
| payment_date | DATETIME | NOT NULL |
| amount | REAL | NOT NULL |
| method | TEXT | CHECK(cash, bank_transfer, upi, cheque) |
| reference | TEXT | |
| remarks | TEXT | |
| tenant_id | TEXT | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`driver_expenses`** (Kharcha)
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| trip_id | TEXT | FK→trips |
| driver_id | TEXT | FK→drivers |
| expense_type | TEXT | CHECK(fuel, toll, food, repair, advance) |
| amount | REAL | NOT NULL |
| description | TEXT | |
| receipt_url | TEXT | |
| approved | INTEGER | DEFAULT 0 |
| status | TEXT | DEFAULT 'pending', CHECK(pending, approved, rejected, settled) |
| category | TEXT | DEFAULT 'advance', CHECK(advance, fuel, toll, food, repair, other) |
| requested_by | TEXT | FK→drivers |
| approved_by | TEXT | FK→users |
| rejected_reason | TEXT | |
| approved_at | DATETIME | |
| created_at | DATETIME | DEFAULT now |

**`eway_bills`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| trip_id | TEXT | UNIQUE, FK→trips |
| ewb_number | TEXT | UNIQUE, NOT NULL |
| irn | TEXT | UNIQUE |
| generation_date | DATETIME | NOT NULL |
| valid_until | DATETIME | NOT NULL |
| transporter_id | TEXT | |
| vehicle_number | TEXT | |
| status | TEXT | DEFAULT 'active', CHECK(active, cancelled, expired) |
| raw_response | TEXT | |
| created_at | DATETIME | DEFAULT now |

**`driver_settlements`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| trip_id | TEXT | UNIQUE, FK→trips |
| driver_id | TEXT | FK→drivers |
| gross_fare | REAL | DEFAULT 0 |
| advances_kharcha | REAL | DEFAULT 0 |
| deductions | REAL | DEFAULT 0 |
| net_payout | REAL | DEFAULT 0 |
| status | TEXT | DEFAULT 'pending', CHECK(pending, processing, paid, disputed) |
| payment_ref | TEXT | |
| paid_at | DATETIME | |
| created_at | DATETIME | DEFAULT now |
| updated_at | DATETIME | DEFAULT now |

**`telemetry_snapshots`**
| Column | Type |
|---|---|
| id | TEXT PK |
| trip_id | TEXT FK→trips |
| vehicle_id | TEXT FK→vehicles |
| timestamp | DATETIME NOT NULL |
| latitude | REAL |
| longitude | REAL |
| speed | REAL |
| fuel_level | REAL |
| odometer | REAL |

**`telemetry_alerts`**
| Column | Type |
|---|---|
| id | TEXT PK |
| trip_id | TEXT FK→trips |
| vehicle_id | TEXT FK→vehicles |
| driver_id | TEXT FK→drivers |
| alert_type | TEXT CHECK(gps_deviation, fuel_theft, temp_breach, speeding) |
| severity | TEXT DEFAULT 'warning' |
| details | TEXT NOT NULL |
| latitude | REAL |
| longitude | REAL |
| resolved | INTEGER DEFAULT 0 |
| created_at | DATETIME DEFAULT now |

**`compliance_checks`**
| Column | Type |
|---|---|
| id | TEXT PK |
| entity_type | TEXT CHECK(driver, vehicle, cargo) |
| entity_id | TEXT NOT NULL |
| check_type | TEXT NOT NULL |
| status | TEXT CHECK(valid, expired, blocked, warning) |
| details | TEXT |
| created_at | DATETIME DEFAULT now |

### Supporting Tables

**`files`**
| Column | Type | Constraints |
|---|---|---|
| id | TEXT | PK |
| filename | TEXT | NOT NULL |
| original_name | TEXT | NOT NULL |
| path | TEXT | NOT NULL |
| size | INTEGER | NOT NULL |
| mime_type | TEXT | NOT NULL |
| uploadable_type | TEXT | CHECK(driver_license, vehicle_insurance, vehicle_permit, company_logo) |
| uploadable_id | TEXT | |
| category | TEXT | DEFAULT 'general' |
| created_at | DATETIME | DEFAULT now |

**`audit_logs`**
| Column | Type |
|---|---|
| id | TEXT PK |
| user_id | TEXT FK→users |
| action | TEXT NOT NULL |
| table_name | TEXT NOT NULL |
| record_id | TEXT |
| old_values | TEXT (JSON) |
| new_values | TEXT (JSON) |
| ip_address | TEXT |
| created_at | DATETIME DEFAULT now |

**`notifications`**
| Column | Type |
|---|---|
| id | TEXT PK |
| user_id | TEXT FK→users |
| title | TEXT NOT NULL |
| message | TEXT NOT NULL |
| channel | TEXT DEFAULT 'in_app' |
| status | TEXT DEFAULT 'unread' |
| link | TEXT |
| created_at | DATETIME DEFAULT now |
| read_at | DATETIME |

**`contact_submissions`**
| Column | Type |
|---|---|
| id | TEXT PK |
| name | TEXT NOT NULL |
| email | TEXT |
| phone | TEXT |
| subject | TEXT NOT NULL |
| message | TEXT NOT NULL |
| status | TEXT DEFAULT 'new' |
| created_at | DATETIME DEFAULT now |
| updated_at | DATETIME DEFAULT now |

**`outbox_events`** (for eventual consistency)
| Column | Type |
|---|---|
| id | TEXT PK |
| event_type | TEXT NOT NULL |
| payload | TEXT NOT NULL |
| aggregate_id | TEXT |
| created_at | DATETIME DEFAULT now |
| processed | INTEGER DEFAULT 0 |

**`vehicle_photos`**
| Column | Type |
|---|---|
| id | TEXT PK |
| vehicle_id | TEXT FK→vehicles |
| file_id | TEXT FK→files |
| side | TEXT |
| created_at | DATETIME DEFAULT now |

**`migrations`** (goose tracking)
| Column | Type |
|---|---|
| id | INTEGER PK AUTOINCREMENT |
| name | TEXT |
| checksum | TEXT |
| executed_at | DATETIME DEFAULT now |

---

## 6. Authentication & Authorization

### Auth Stack (`internal/auth/`)

**Session-based (web UI)**:
- Gorilla `securecookie` signs session cookies
- `SessionData`: `{UserID, Role, Name, Expires}`
- Cookie expiry: 24 hours (configurable via `SESSION_MAX_AGE`)
- `SessionStore.CreateSession()` → sets signed cookie
- `SessionStore.Validate()` → reads + verifies cookie from request
- `SessionStore.Destroy()` → deletes session cookie

**Bearer token (REST API v1)**:
- `middleware.RequireAPIAuth` validates `Authorization: Bearer <token>` header
- API tokens stored as hashes in `api_tokens` table
- Signed using cookie secret (JWT-style)

**Password hashing**: bcrypt via `golang.org/x/crypto`

### RBAC (Casbin)

**Model** (`internal/auth/casbin.go`):
```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

**DBAdapter** loads from two tables:
- `roles` + `role_permissions` (role → permission mapping)
- `user_roles` (user → role mapping)

**Permission system**: Resource-action pairs
```
drivers:create, drivers:read, drivers:update, drivers:delete, drivers:export
vehicles:create, vehicles:read, vehicles:update, vehicles:delete, vehicles:export
customers:create, customers:read, customers:update, customers:delete, customers:export
routes:create, routes:read, routes:update, routes:delete
bookings:create, bookings:read, bookings:update, bookings:delete, bookings:cancel, bookings:approve
trips:create, trips:read, trips:update, trips:delete, trips:assign
invoices:create, invoices:read, invoices:update, invoices:delete, invoices:export
payments:create, payments:read, payments:update, payments:delete
reports:read, reports:export
settings:read, settings:update
audit_logs:read
files:create, files:read, files:delete
users:create, users:read, users:update, users:delete
```

**Role → Permission assignments**:
| Role (id) | Permissions |
|---|---|
| admin (1) | All permissions |
| dispatcher (2) | drivers CRUD+export, vehicles CRUD+export, customers CRUD+export, routes CRUD, bookings CRUD+cancel+approve, trips CRUD+assign, reports read+export, settings read |
| accountant (3) | invoices CRUD+export, payments CRUD, reports read+export |
| viewer (4) | read-only on drivers, vehicles, customers, routes, bookings, trips, invoices, payments, reports |
| driver (5) | trips:read, trips:assign (own), files:upload (own) |

**Template-level RBAC**: The `can(user, resource, action)` template function checks permissions for UI element visibility.

---

## 7. Backend APIs

### HTTP Routing (chi v5)

```
Public (no auth):
  GET /                    → Marketing page (cached)
  GET /robots.txt          → SEO
  GET /sitemap.xml         → SEO
  GET /login               → Login page
  POST /login              → Authenticate
  GET /register            → Registration page
  POST /register           → Register
  GET /forgot-password     → Password reset form
  POST /forgot-password    → Send reset
  POST /logout             → End session
  GET /contact-us/*        → Contact form

Protected (RequireAuth):
  GET /dashboard           → KPI dashboard + WebSocket
  GET /files/{id}          → Download uploaded file
  /users                   → User management (admin)
  /drivers                 → Driver CRUD
  /vehicles                → Vehicle CRUD
  /customers               → Customer CRUD
  /routes                  → Route CRUD
  /bookings                → Booking CRUD + confirm/cancel/complete
  /trips                   → Trip CRUD + execution timeline + POD
  /invoices                → Invoice CRUD
  /payments                → Payment CRUD
  /reports                 → Revenue, trip, driver reports
  /settings                → Company settings
  /audit-logs              → Audit trail
  /kharcha                 → Driver expense approvals
  POST /trips/{id}/deliver-pod → e-POD delivery (mobile callback)
  /profile                 → User profile
  /change-password         → Password change

REST API v1 (Bearer token):
  POST /api/v1/token       → Issue token
  POST /api/v1/register    → Register new user/account
  POST /api/v1/refresh    → Refresh token
  POST /api/v1/bookings/*  → Booking CRUD
  POST /api/v1/trips/*     → Trip CRUD
  POST /api/v1/invoices/*  → Invoice CRUD
  POST /api/v1/payments/*  → Payment CRUD

GraphQL:
  POST /query              → GraphQL queries
  GET /graphql             → GraphQL playground

Telemetry:
  POST /api/v1/telemetry/sync → Mobile batch telemetry upload

Static & Uploads:
  GET /static/*            → CSS/JS/images (cache-busted via ?v=)
  GET /uploads/*           → User uploads (24hr cache)
```

### Middleware Chain

```
chi.RequestID → chi.Recoverer → chi.Compress(5) → chi.Timeout(60s)
  → middleware.SPAMiddleware
  → bodySizeLimit(32MB)
  → [route groups with auth]
```

**Auth middleware**:
- `middleware.RequireAuth(authStore)` — validates session cookie, populates `auth.ContextUser`
- `middleware.RequireAPIAuth(authStore, secret)` — validates Bearer token
- Template function `can(user, resource, action)` — checks Casbin permissions

### gRPC Server

- Port: `50051`
- `internal/grpcservice/` — Dispatch microservice RPC

### MQTT Client

- Broker: `tcp://localhost:1883`
- `internal/mqttservice/` — Real-time vehicle/driver communication

### WebSocket

- `/dashboard` — Real-time KPI updates via WebSocket hub

---

## 8. Event-Driven Orchestration

In-memory synchronous event bus (`internal/events/bus.go`):

```text
EventBus (in-memory, mutex-protected)
  │
  ├── Publish(ctx, Event{Type, Payload})
  └── Subscribe(eventType, Handler) → unsubscribe()
```

**Cross-service event wiring** (`internal/service/service.go → initEventHandlers()`):

| Trigger Event | Subscriber | Action |
|---|---|---|
| `BookingConfirmed` | Trips service | `CreateTrip` — auto-create trip from confirmed booking |
| `TripCompleted` | Invoices service | `GenerateInvoiceFromTrip` — auto-generate invoice |
| `TripDelivered` | Invoices service | `GenerateInvoiceFromTrip` — auto-generate GST invoice |
| `TripDelivered` | Settlements service | `CreateSettlementForTrip` — auto-create driver settlement |

**Domain event types** (strongly typed payloads):
- `BookingCreatedEvent{BookingID, BookingNumber, CustomerID, RouteID, PickupDate, OccurredAt}`
- `BookingConfirmedEvent{BookingID, ConfirmedAt, OccurredAt}`
- `BookingCancelledEvent{BookingID, CancelledAt, OccurredAt}`
- `BookingCompletedEvent{BookingID, CompletedAt, OccurredAt}`
- `TripCompletedEvent{TripID, CompletedAt, OccurredAt}`

**Outbox pattern**: `outbox_events` table for eventual consistency with async delivery.

---

## 9. Web Frontend (Server-Rendered)

### Tech Stack
- Go `html/template` (not React/Vue — pure server-side rendering)
- Datastar (HTMX-compatible AJAX partial updates)
- Tailwind CSS
- Partial template rendering for SPA-like UX

### Template System (`internal/handlers/app.go`)

**Layouts**:
- `layout.html` — Main layout (sidebar navigation, notifications, flash messages)
- `auth_layout.html` — Auth pages (login/register, no sidebar)

**Rendering modes**:
1. Full page — `renderPage()` — renders content template inside layout
2. Fragment — `renderFragment()` — raw template output for Datastar/AJAX
3. Form — `renderForm()` — auto-detects Datastar request, renders fragment or full page
4. Marketing — `Marketing()` — cached homepage (zero-alloc, `sync.Once`)

**Template functions**:
| Function | Purpose |
|---|---|
| `can(user, resource, action)` | RBAC permission check |
| `formatDateTime`, `formatDate` | Date formatting |
| `datetime`, `date_only` | Time formatting |
| `statusBadgeClass` | CSS classes by status |
| `priceFormat` | Currency formatting (2 decimal places) |
| `yesNo`, `nullString` | Display helpers |
| `safeHTML`, `lower`, `upper`, `join`, `abbr` | Utility functions |
| `add`, `sub`, `mul`, `div` | Math for templates |
| `fileExt`, `slice`, `derefTime` | More utilities |

**Datastar detection**: Requests with `Datastar-Request: true` or `HX-Request: true` or `?_fragment=true` get partial fragment responses.

### Static Assets

Served from `/static/*` with two cache strategies:
- **Versioned** (`?v=` param): `Cache-Control: public, max-age=31536000, immutable` (1 year)
- **Unversioned**: `Cache-Control: public, max-age=3600, must-revalidate` (1 hour)

Uploaded files served from `/uploads/*` with 24-hour cache.

### Pages

| Route | Template | Handler |
|---|---|---|
| `/` | `home.html` | `app.Marketing` |
| `/login` | `login_form.html` | `app.Auth.LoginPage` |
| `/register` | `register_form.html` | `app.Auth.RegisterPage` |
| `/dashboard` | `dashboard.html` | `app.Dashboard.Index` |
| `/drivers/*` | `drivers.html` + partials | `app.Drivers.Routes` |
| `/vehicles/*` | `vehicles.html` + partials | `app.Vehicles.Routes` |
| `/customers/*` | `customers.html` + partials | `app.Customers.Routes` |
| `/routes/*` | `routes.html` | `app.Routes.Routes` |
| `/bookings/*` | `bookings.html` + `booking_form.html` | `app.Bookings.Routes` |
| `/trips/*` | `trips.html` + `trip_form.html` | `app.Trips.Routes` |
| `/invoices/*` | `invoices.html` | `app.Invoices.Routes` |
| `/payments/*` | `payments.html` | `app.Payments.Routes` |
| `/reports/*` | `reports.html` | `app.Reports.Routes` |
| `/settings` | `settings.html` | `app.SettingsH.Routes` |
| `/audit-logs` | `audit_logs.html` | `app.AuditLogs.Routes` |
| `/kharcha/*` | `kharcha.html` | `app.Kharcha.Routes` |

### Pagination

Standard pagination for all list views:
- Default: 20 items per page
- Query params: `?q=<search>&status=<filter>&page=<n>&limit=<n>`
- `parsePaginationParams()` parses from URL
- `newPaginationData()` computes HasPrev/HasNext/TotalPages

---

## 10. Mobile Frontend (React Native / Expo)

### Architecture

```
mobile/
├── src/
│   ├── App.tsx           # Root component, navigation setup
│   ├── components/       # Reusable UI: TripCard, forms, maps
│   ├── constants/        # Colors, route names, config
│   ├── services/         # API clients (all async, offline-first)
│   │   ├── analytics.ts  # Telemetry data collection
│   │   ├── graphql.ts    # Apollo GraphQL client
│   │   ├── mqtt.ts       # MQTT real-time communication
│   │   ├── storage.ts    # AsyncStorage wrapper
│   │   ├── syncEngine.ts # Offline-first sync engine
│   │   └── telemetry.ts  # Telemetry batch processing
│   ├── stores/           # Zustand state management
│   └── types/            # TypeScript type definitions
```

### Key Features

- **Offline-first**: `syncEngine.ts` queues mutations locally, syncs to server when online
- **Real-time**: MQTT client for driver-server communication (trip updates, GPS)
- **Telemetry**: Batch sync to `/api/v1/telemetry/sync` for analytics
- **GraphQL**: Apollo Client for read queries
- **Screens**:
  - `LoginScreen` — Bearer token auth
  - `BookingScheduleScreen` — Schedule pickup, select route/vehicle
  - `DeliveryVerificationScreen` — e-POD capture (photo, signature, OTP)
  - `EarningsOverviewScreen` — Driver earnings summary
  - `LiveDriverTrackingMap` — Real-time GPS tracking
  - `TripCard` — Trip status card component
  - `FirstTimeSetupScreen` — Onboarding flow
  - `GetStartedScreen` / `OnboardingOverviewScreen` / `SplashScreen`

---

## 11. Infrastructure

### Repository Layer

- **`internal/repository/repository.go`** — Central `Store` interface combining all repository interfaces
- **`internal/repository/sqlite/`** — `SqliteStore` implementing all repositories
  - Uses `modernc.org/sqlite` (pure Go, no CGO dependency)
  - `baseRepository` struct with shared `*sql.DB` and logger
  - Each entity has its own repository file (`booking_repository.go`, `trip_repository.go`, etc.)

### Database Connection

```go
// From cmd/server/main.go
database, err := sql.Open("sqlite", cfg.DatabaseURL)
database.SetMaxOpenConns(64)
database.SetMaxIdleConns(32)
database.SetConnMaxLifetime(5 * time.Minute)

// PRAGMAs for high throughput
PRAGMA journal_mode=WAL;
PRAGMA synchronous=OFF;
PRAGMA busy_timeout=10000;
PRAGMA cache_size=-131072;   // 128MB
PRAGMA mmap_size=536870912;  // 512MB
PRAGMA foreign_keys=ON;
PRAGMA temp_store=MEMORY;
```

### Unit of Work

- `internal/shared/uow/` — `SQLUnitOfWork` for transactional vertical-slice operations
- Injected into use cases alongside `IDGenerator` and `Clock`

### Shared Infrastructure

| Package | Purpose |
|---|---|
| `internal/shared/clock` | Real clock (injectable for tests) |
| `internal/shared/id` | UUID ID generator (injectable) |
| `internal/shared/uow` | Unit of work pattern |
| `internal/shared/pagination` | Pagination utilities |

### Migrations

- 32 goose migrations in `db/migrations/`
- Format: `-- +goose Up` / `-- +goose Down`
- Embedded into binary via `go:embed` (`db/embed.go`)
- Automatically run on server startup via `goose.NewProvider`

### Telemetry & Observability

- **OpenTelemetry**: Traces and metrics setup
- **Prometheus**: Metrics endpoint
- **Structured logging**: slog with JSON output in production
- **Request tracing**: Per-request UUID, logged with duration and status

### File Storage

- Configurable upload directory (`UPLOAD_DIR` env, default `./uploads`)
- File metadata stored in `files` table
- Served via `/uploads/*` with 24-hour cache

---

## 12. Key Workflows

### Booking → Trip → Invoice Flow

```text
1. Booking created (status: draft → pending)
   → BookingCreatedEvent published

2. Dispatcher confirms booking (status: pending → confirmed)
   → BookingConfirmedEvent published
   → Event bus subscriber: Trips.CreateTrip() (auto-create trip)
   →     Trip created (status: draft → scheduled)

3. Dispatcher assigns driver + vehicle (trip: scheduled → assigned)

4. Driver starts trip (trip: assigned → started)
   → Timeline: started_at recorded

5. Driver marks in-transit (trip: started → in_transit)
   → Timeline: in_transit_at recorded

6. Driver delivers (trip: in_transit → delivered)
   → e-POD captured (photo, signature, OTP)
   → Timeline: delivered_at recorded
   → TripDeliveredEvent published
   → Event bus subscribers:
      a) Invoices.GenerateInvoiceFromTrip() (auto-generate invoice)
      b) Settlements.CreateSettlementForTrip() (auto-create driver settlement)

7. Customer pays invoice (payment recorded)
   → Invoice.ApplyPayment() updates paid_amount
   → If fully paid: status → paid

8. Driver completes trip (trip: delivered → completed)
   → TripCompletedEvent published
   → Event bus subscriber: Invoices.GenerateInvoiceFromTrip() (if not already generated)
```

### Kharcha (Driver Expense) Flow

```text
1. Driver submits expense (fuel, toll, food, repair, advance)
   → DriverExpense created (status: pending)

2. Accountant reviews in /kharcha page
   → Can approve (status: approved) or reject (status: rejected)
   → Approved expenses reduce driver settlement
```

### E-Way Bill Flow

```text
1. Trip assigned to vehicle with valid RC/permit
   → E-Way Bill generated (status: active)
   → Linked to trip via eway_bill_ref

2. On trip completion or expiry
   → E-Way Bill cancelled (status: cancelled/expired)
```

---

## 13. Deployment

### Docker

```dockerfile
# Multi-stage build
FROM golang:1.26-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/mvtms ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/mvtms /mvtms
COPY --from=builder /app/db/migrations /db/migrations
COPY --from=builder /app/internal/templates /internal/templates
COPY --from=builder /app/internal/static /internal/static
COPY --from=builder /app/.env.example /.env.example
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/mvtms"]
```

### Server Bootstrap (`cmd/server/main.go`)

1. Load `.env` → `config.Load()`
2. Setup logging (`slog`, JSON in production)
3. Open SQLite with performance PRAGMAs (64 max connections, WAL mode)
4. Run embedded migrations (goose)
5. Initialize `SqliteStore` (repository)
6. Initialize `Services` (service layer)
7. Initialize `SessionStore` (securecookie)
8. Initialize Casbin `AuthorizationService` (DB-backed)
9. Initialize `App` handlers (template parsing)
10. Wire vertical-slice use cases (UnitOfWork + IDGenerator + Clock)
11. Setup chi router with middleware chain
12. Register all route groups
13. Start HTTP server

### Android OTA Agent (`cmd/agent/main.go`)

- Runs on Android (Termux environment, `/data/local/tmp`)
- Polls GitHub releases for version manifest JSON
- On new version: download → SHA256 verify → backup old binary → extract → restart server
- **Automatic rollback**: if health check fails, restores previous binary
- Health check: HTTP GET to `127.0.0.1:8092/` with 10s timeout
- Uses `start.sh` if present, otherwise launches `./server` directly
- Environment: `PORT=8092`, `ENV=production`, `DATABASE_URL=file:mvtms.db?...`

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8090` (web), `8092` (agent) | Server port |
| `APP_ENV` | `development` | Environment: development, staging, production |
| `DATABASE_URL` | `file:mvtms.db?...` | SQLite connection string |
| `COOKIE_SECRET` | dev key | securecookie signing secret |
| `SESSION_MAX_AGE` | `24h` | Session cookie lifetime |
| `LOG_LEVEL` | `info` | slog level |
| `UPLOAD_DIR` | `./uploads` | File upload directory |
| `STATIC_DIR` | `internal/static` | Static asset directory |
| `MAX_UPLOAD_SIZE` | `10` (MB) | Max upload size |
| `TEMPLATES_DIR` | `internal/templates` | Template directory |
| `RAZORPAY_KEY_ID` | test key | Payment gateway key |
| `RAZORPAY_KEY_SECRET` | test key | Payment gateway secret |
| `APP_DOMAIN` | `avandab.com` | Application domain (sitemap, SEO) |
| `APP_VERSION` | timestamp | Build version (set via ldflags) |

### Deploy Scripts

| Script | Purpose |
|---|---|
| `deploy.sh` | Standard deployment |
| `deploy_remote.sh` | Remote server deployment |
| `deploy_avandab.sh` | Avandab-specific deployment |
| `backup_db.sh` | Database backup script |
| `publish_update.sh` | Publish new version/update |
| `setup_cloudflare_avandab.sh` | Cloudflare setup for Avandab |
| `Makefile` | Build, test, lint, run commands |
| `sqlc.yaml` | SQL code generation config (for type-safe query code) |
