# FlyFleet Developer Guidelines & Coding Standards

This document defines the governance rules, coding standards, and architectural fitness constraints for FlyFleet.

---

## 1. Package Dependency Boundaries

Dependencies flow strictly inward and downward. 

### Forbidden Imports
- **Domain (`*/domain/`)** must never import:
  - `database/sql`
  - `net/http`
  - `html/template`
  - `internal/platform/*`
- **Application (`*/application/`)** must never import:
  - `net/http`
  - `html/template`

---

## 2. Dependency Injection Policy

- Only the **Composition Root** (`cmd/server/bootstrap.go`) is allowed to construct and wire dependencies.
- Use cases and facades must never initialize services directly.

---

## 3. Aggregate Lifecycle & Repository Rules

- Repositories deal exclusively with aggregate roots.
- Allowed repository methods: `Find`, `Save`, `Exists`, `Delete`.
- **Forbidden**: Partial updates (`UpdateStatus`, `AssignVehicle`). Load the aggregate root, execute the state change within the aggregate boundary, and invoke `Save(aggregate)`.

---

## 4. Key Coding Conventions

- **Money**: Never use `float64`. Always use the `Money` value object (`Amount`, `Currency`).
- **Time/IDs**: Never call `time.Now()` or generator packages directly. Inject and use `shared/ports/Clock` and `shared/ports/IDGenerator`.
- **Immutable Commands**: Commands representing write actions must be read-only structures after instantiation.
- **Logging/Errors**: Domain models must never log; they return errors. Logging is handled at the Application and Infrastructure layers.
- **Module Data Isolation**: Direct SQL queries crossing module schemas (e.g. Trip accessing Booking tables) are strictly forbidden. Access data only via the target module's public `Facade`.
