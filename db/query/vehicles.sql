-- name: CreateVehicle :one
INSERT INTO vehicles (id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage, tenant_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at;

-- name: GetVehicleByID :one
SELECT id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at
FROM vehicles WHERE id = ? AND tenant_id = ?;

-- name: GetVehicleByRegistration :one
SELECT id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at
FROM vehicles WHERE registration_number = ? AND tenant_id = ?;

-- name: UpdateVehicle :one
UPDATE vehicles
SET registration_number = ?, vehicle_number = ?, vehicle_type = ?, capacity = ?,
    fuel_type = ?, insurance_expiry = ?, fitness_expiry = ?, permit_expiry = ?,
    status = ?, current_mileage = ?, updated_at = datetime('now')
WHERE id = ? AND tenant_id = ?
RETURNING id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at;

-- name: DeleteVehicle :exec
DELETE FROM vehicles WHERE id = ? AND tenant_id = ?;

-- name: SearchVehicles :many
SELECT id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at
FROM vehicles
WHERE tenant_id = ?
  AND (registration_number LIKE '%' || ? || '%' OR vehicle_number LIKE '%' || ? || '%' OR vehicle_type LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountVehicles :one
SELECT COUNT(*) AS count
FROM vehicles
WHERE tenant_id = ?
  AND (registration_number LIKE '%' || ? || '%' OR vehicle_number LIKE '%' || ? || '%' OR vehicle_type LIKE '%' || ? || '%')
  AND (? = '' OR status = ?);

-- name: GetAvailableVehicles :many
SELECT id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at
FROM vehicles
WHERE status = 'available' AND tenant_id = ?
ORDER BY created_at ASC;

-- name: GetIdleVehicles :many
SELECT id, registration_number, vehicle_number, vehicle_type, capacity,
    fuel_type, insurance_expiry, fitness_expiry, permit_expiry, status, current_mileage,
    tenant_id, created_at, updated_at
FROM vehicles
WHERE status = 'available' AND tenant_id = ? AND updated_at < datetime('now', '-2 hours')
ORDER BY created_at ASC
LIMIT 10;
