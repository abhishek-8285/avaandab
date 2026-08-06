# Trip Business Rules

## Status Lifecycle

```text
Draft → Scheduled → Assigned → Started → Completed
                    ↓            ↓
                Cancelled   -
```

## Rules

### Creation
* A trip is created in `Draft` status by default.
* The `RouteID` must reference an existing route.
* `DepartureTime` must be in the future.

### Scheduling
* Only `Draft` trips can be scheduled.
* Scheduling moves the status from `Draft` to `Scheduled`.

### Driver Assignment
* Only `Draft`, `Scheduled`, or `Assigned` trips (not `Started` or `Completed`) can have a driver assigned.
* Assigning a driver moves the status from `Scheduled` to `Assigned`.
* **No overlapping driver assignments**: a driver cannot be assigned to two trips with overlapping time windows.

### Vehicle Assignment
* Only `Draft`, `Scheduled`, or `Assigned` trips can have a vehicle assigned.
* Assigning a vehicle moves the status from `Assigned` to `Assigned` (status stays the same, vehicle is recorded).
* **No overlapping vehicle assignments**: a vehicle cannot be assigned to two trips with overlapping time windows.
* A trip must have a driver assigned before a vehicle can be assigned.

### Starting
* Only `Scheduled` or `Assigned` trips can be started.
* Starting moves the status to `Started`.

### Completion
* Only `Started` trips can be completed.
* **Completed trips are immutable** — no further edits, assignments, or cancellations are allowed.

### Cancellation
* A trip can be cancelled from any status except `Completed`.
* Once cancelled, the trip is immutable.
* Cancelled trips cannot be started, completed, or assigned.

### Conflict Detection
* Before assigning a driver or vehicle, the system checks for time-window conflicts.
* A conflict occurs when the driver/vehicle is already assigned to another trip whose `[departure_time, completion_time]` overlaps with the new trip's time window.
