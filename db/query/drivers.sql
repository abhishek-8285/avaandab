-- name: CreateDriver :one
INSERT INTO drivers (id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at;

-- name: GetDriverByID :one
SELECT id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at
FROM drivers WHERE id = ? AND tenant_id = ?;

-- name: GetDriverByDriverID :one
SELECT id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at
FROM drivers WHERE driver_id = ? AND tenant_id = ?;

-- name: UpdateDriver :one
UPDATE drivers
SET driver_id = ?, first_name = ?, last_name = ?, phone = ?, email = ?, address = ?,
    license_number = ?, license_expiry = ?, experience_years = ?, status = ?,
    emergency_contact_name = ?, emergency_contact_phone = ?, notes = ?,
    updated_at = datetime('now')
WHERE id = ? AND tenant_id = ?
RETURNING id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at;

-- name: DeleteDriver :exec
DELETE FROM drivers WHERE id = ? AND tenant_id = ?;

-- name: SearchDrivers :many
SELECT id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at
FROM drivers
WHERE tenant_id = ?
  AND (first_name LIKE '%' || ? || '%' OR last_name LIKE '%' || ? || '%' OR phone LIKE '%' || ? || '%' OR license_number LIKE '%' || ? || '%')
  AND (? = '' OR status = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountDrivers :one
SELECT COUNT(*) AS count
FROM drivers
WHERE tenant_id = ?
  AND (first_name LIKE '%' || ? || '%' OR last_name LIKE '%' || ? || '%' OR phone LIKE '%' || ? || '%' OR license_number LIKE '%' || ? || '%')
  AND (? = '' OR status = ?);

-- name: GetAvailableDrivers :many
SELECT id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at
FROM drivers
WHERE status = 'available' AND tenant_id = ?
ORDER BY created_at ASC;

-- name: GetDriverByPhone :one
SELECT id, driver_id, first_name, last_name, phone, email, address,
    license_number, license_expiry, experience_years, status, emergency_contact_name,
    emergency_contact_phone, notes, tenant_id, created_at, updated_at
FROM drivers WHERE phone = ? AND tenant_id = ?;
