-- name: CreateCustomer :one
INSERT INTO customers (id, name, company, phone, email, gst, address, notes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, company, phone, email, gst, address, notes, created_at, updated_at;

-- name: GetCustomerByID :one
SELECT id, name, company, phone, email, gst, address, notes, created_at, updated_at
FROM customers WHERE id = ?;

-- name: UpdateCustomer :one
UPDATE customers
SET name = ?, company = ?, phone = ?, email = ?, gst = ?, address = ?, notes = ?,
    updated_at = datetime('now')
WHERE id = ?
RETURNING id, name, company, phone, email, gst, address, notes, created_at, updated_at;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = ?;

-- name: SearchCustomers :many
SELECT id, name, company, phone, email, gst, address, notes, created_at, updated_at
FROM customers
WHERE (name LIKE '%' || ? || '%' OR company LIKE '%' || ? || '%' OR phone LIKE '%' || ? || '%' OR email LIKE '%' || ? || '%')
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountCustomers :one
SELECT COUNT(*) AS count
FROM customers
WHERE (name LIKE '%' || ? || '%' OR company LIKE '%' || ? || '%' OR phone LIKE '%' || ? || '%' OR email LIKE '%' || ? || '%');

-- name: GetCustomerByPhone :one
SELECT id, name, company, phone, email, gst, address, notes, created_at, updated_at
FROM customers WHERE phone = ?;
