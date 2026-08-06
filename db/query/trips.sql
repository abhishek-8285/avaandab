-- name: CreateTrip :one
INSERT INTO trips (id, trip_number, booking_id, driver_id, vehicle_id, route_id,
    departure_time, arrival_time, status, remarks, tenant_id, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
RETURNING id, trip_number, booking_id, driver_id, vehicle_id, route_id,
    departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at;

-- name: GetTripByID :one
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.tenant_id, t.version, t.created_at, t.updated_at,
    d.driver_id AS driver_display_id, d.first_name AS driver_first_name, d.last_name AS driver_last_name,
    v.registration_number AS vehicle_registration_number, v.vehicle_number AS vehicle_number,
    r.source AS route_source, r.destination AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.id = ? AND t.tenant_id = ?;

-- name: GetTripByNumber :one
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.tenant_id, t.version, t.created_at, t.updated_at,
    d.driver_id AS driver_display_id, d.first_name AS driver_first_name, d.last_name AS driver_last_name,
    v.registration_number AS vehicle_registration_number, v.vehicle_number AS vehicle_number,
    r.source AS route_source, r.destination AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.trip_number = ? AND t.tenant_id = ?;

-- name: GetTripByBookingID :one
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.tenant_id, t.version, t.created_at, t.updated_at,
    d.driver_id AS driver_display_id, d.first_name AS driver_first_name, d.last_name AS driver_last_name,
    v.registration_number AS vehicle_registration_number, v.vehicle_number AS vehicle_number,
    r.source AS route_source, r.destination AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.booking_id = ? AND t.tenant_id = ?;

-- name: UpdateTrip :one
UPDATE trips
SET trip_number = ?, booking_id = ?, driver_id = ?, vehicle_id = ?, route_id = ?,
    departure_time = ?, arrival_time = ?, status = ?, remarks = ?,
    version = version + 1,
    updated_at = datetime('now')
WHERE id = ? AND tenant_id = ? AND version = ?
RETURNING id, trip_number, booking_id, driver_id, vehicle_id, route_id,
    departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at;

-- name: UpdateTripStatus :one
UPDATE trips
SET status = ?, version = version + 1, updated_at = datetime('now')
WHERE id = ? AND tenant_id = ? AND version = ?
RETURNING id, trip_number, booking_id, driver_id, vehicle_id, route_id,
    departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at;

-- name: AssignDriverToTrip :one
UPDATE trips
SET driver_id = ?, version = version + 1, updated_at = datetime('now')
WHERE id = ? AND tenant_id = ? AND version = ?
RETURNING id, trip_number, booking_id, driver_id, vehicle_id, route_id,
    departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at;

-- name: AssignVehicleToTrip :one
UPDATE trips
SET vehicle_id = ?, version = version + 1, updated_at = datetime('now')
WHERE id = ? AND tenant_id = ? AND version = ?
RETURNING id, trip_number, booking_id, driver_id, vehicle_id, route_id,
    departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at;

-- name: DeleteTrip :exec
DELETE FROM trips WHERE id = ? AND tenant_id = ?;

-- name: SearchTrips :many
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.tenant_id, t.created_at, t.updated_at,
    d.driver_id AS driver_display_id, d.first_name AS driver_first_name, d.last_name AS driver_last_name,
    v.registration_number AS vehicle_registration_number, v.vehicle_number AS vehicle_number,
    r.source AS route_source, r.destination AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.tenant_id = ?
  AND (t.trip_number LIKE '%' || ? || '%' OR d.first_name LIKE '%' || ? || '%' OR d.last_name LIKE '%' || ? || '%' OR v.registration_number LIKE '%' || ? || '%' OR r.source LIKE '%' || ? || '%' OR r.destination LIKE '%' || ? || '%')
  AND (? = '' OR t.status = ?)
ORDER BY t.departure_time DESC
LIMIT ? OFFSET ?;

-- name: CountTrips :one
SELECT COUNT(*) AS count
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.tenant_id = ?
  AND (t.trip_number LIKE '%' || ? || '%' OR d.first_name LIKE '%' || ? || '%' OR d.last_name LIKE '%' || ? || '%' OR v.registration_number LIKE '%' || ? || '%' OR r.source LIKE '%' || ? || '%' OR r.destination LIKE '%' || ? || '%')
  AND (? = '' OR t.status = ?);

-- name: CheckVehicleConflict :many
SELECT id, trip_number, status, departure_time, arrival_time
FROM trips
WHERE vehicle_id = ? AND tenant_id = ?
  AND status IN ('scheduled', 'assigned', 'started')
  AND (? = '' OR id != ?);

-- name: CheckDriverConflict :many
SELECT id, trip_number, status, departure_time, arrival_time
FROM trips
WHERE driver_id = ? AND tenant_id = ?
  AND status IN ('scheduled', 'assigned', 'started')
  AND (? = '' OR id != ?);

-- name: GetTripsByDate :many
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.tenant_id, t.created_at, t.updated_at,
    d.driver_id AS driver_display_id, d.first_name AS driver_first_name, d.last_name AS driver_last_name,
    v.registration_number AS vehicle_registration_number, v.vehicle_number AS vehicle_number,
    r.source AS route_source, r.destination AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE date(t.departure_time) = ? AND t.tenant_id = ?
ORDER BY t.departure_time ASC;

-- name: CountTripsByStatus :many
SELECT status, COUNT(*) AS count
FROM trips
WHERE date(departure_time) = ? AND tenant_id = ?
GROUP BY status;
