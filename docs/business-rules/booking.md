# Booking Business Rules

## Status Lifecycle

```text
Draft → Pending → Confirmed → Completed
                ↓
              Cancelled
```

## Rules

### Creation

* A booking is created in `Pending` status by default.
* The `CustomerID` must reference an existing customer.
* The `RouteID` must reference an existing route.
* `VehicleType` must be specified.
* `Passengers` must be >= 1.
* `Price` must be >= 0.
* `PickupDate` must be a valid RFC3339 timestamp.

### Confirmation

* Only `Pending` bookings can be confirmed.
* Confirming moves the status from `Pending` to `Confirmed`.
* A `Cancelled` or `Completed` booking cannot be confirmed.

### Cancellation

* A booking can be cancelled from any status except `Completed`.
* Once cancelled, the booking is immutable — no further state changes are allowed.
* The user who cancels must have the `bookings:cancel` permission.

### Completion

* Only `Confirmed` bookings can be marked as `Completed`.
* `Completed` bookings are immutable — they cannot be edited, confirmed, or cancelled.

### Update / Edit

* Only `Pending` bookings can be updated.
* `Confirmed`, `Cancelled`, and `Completed` bookings cannot be edited.
* `Complete` is not a state transition from `Pending` — a booking must be confirmed first.

### Deletion

* Only `Pending` and `Draft`-equivalent bookings can be deleted.
* Deleting removes the booking record entirely (soft-delete is not used).

### Audit

* Every state transition (create, confirm, cancel, complete, update, delete) is logged to the `audit_logs` table.
* Each audit entry records the action, table name, record ID, old/new values, IP address, and timestamp.

### Permissions

| Action     | Permission          |
|------------|---------------------|
| Create     | `bookings:create`     |
| Read/List  | `bookings:read`      |
| Update     | `bookings:update`    |
| Confirm    | `bookings:approve`   |
| Cancel     | `bookings:cancel`    |
| Delete     | `bookings:delete`    |
