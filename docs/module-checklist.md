# Definition of Done — Module Checklist

Every new feature module (Booking, Trip, Vehicle, Driver, Invoice, Payment, etc.) must satisfy **all** items below before considered complete.

---

## 1. Domain Layer

- [ ] **Aggregate** implemented — aggregate root holds internal entities, protects invariants, and collects domain events.
- [ ] **Workflow** implemented — domain workflow/orchestration logic defined in `domain/workflow/`.
- [ ] **Repository contract** defined as an interface inside the domain package (`domain/repository.go`).
- [ ] **Domain events** declared and published from within the aggregate on state transitions.
- [ ] **Value Objects and Domain Policies** defined where applicable.

## 2. Infrastructure Layer

- [ ] **SQL migration** authored and placed in `db/migrations/` (idempotent, with `down`).
- [ ] **sqlc queries** written with generated type-safe query code.
- [ ] **Persistence adapter** implemented under `infrastructure/persistence/sql/`.
- [ ] **Converters** mapping Aggregate roots ⇄ DB Persistence Models implemented.
- [ ] **Outbox entries** integrated within the database transaction (outbox pattern).

## 3. Application Layer

- [ ] **Application use cases** (Commands & Queries) implemented as distinct handlers under `application/`.
- [ ] **Unit of Work** integration used for all write operations.
- [ ] **DTOs** designed — command inputs, query outputs, and read models separated from domain entities.

## 4. Presentation Layer

- [ ] **Web handlers** created under `presentation/web/`.
- [ ] **API handlers** and **DTOs** created under `presentation/api/`.
- [ ] **Templates** authored under `presentation/web/templates/`.
- [ ] **View models** designed for presentation view rendering under `presentation/web/viewmodels/`.
- [ ] **Validation** applied to all commands/queries.
- [ ] **Authorization** constraints verified at the Handler/Application boundary.
- [ ] **Audit logging** captured for critical write actions.

## 5. Testing

- [ ] **Unit tests** — domain logic, value objects, workflows, and use cases.
- [ ] **Integration tests** — repository + database (real or containerized Postgres).
- [ ] **HTTP tests** — web handlers and REST API endpoints (end-to-end through chi router).

## 6. Documentation

- [ ] Module README or inline doc comments covering purpose, key types, and usage.
