# Module Checklist

Every new feature module (Booking, Trip, Vehicle, Driver, Invoice, Payment) must implement the following checklist to ensure consistency and prevent architectural drift.

---

## 1. Domain Layer
- [ ] Aggregate Root defined (holds internal business entities and collects domain events).
- [ ] Invariants and state transitions protected inside the aggregate root.
- [ ] Value Objects and Domain Policies defined.
- [ ] Repository Contract defined as an interface inside the domain package (`domain/repository.go`).

---

## 2. Infrastructure Layer
- [ ] Database repository implemented under `infrastructure/persistence/sql/`.
- [ ] Converters mapping Aggregate roots ⇄ DB Persistence Models implemented.
- [ ] Outbox events integrated within the database transaction.

---

## 3. Application Layer
- [ ] Use Cases (Commands & Queries) implemented as distinct handlers under `application/`.
- [ ] Unit of Work integration used for all write operations.
- [ ] Response DTOs and Read Models separated from domain/database entities.

---

## 4. Presentation Layer
- [ ] View Models designed for presentation view rendering.
- [ ] Web handlers and Datastar templates created under `presentation/web/`.
- [ ] REST API handlers and DTOs created under `presentation/api/`.

---

## 5. Security & Governance
- [ ] Input validation applied to all commands/queries.
- [ ] Permission check constraints verified at the Handler/Application boundary.
- [ ] Audit logs captured for critical write actions.
