-- name: CreateDriverExpense :one
INSERT INTO driver_expenses (id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by, approved_by, rejected_reason, approved_at,
    approved, created_at;

-- name: GetDriverExpenseByID :one
SELECT id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by, approved_by, rejected_reason, approved_at,
    approved, created_at
FROM driver_expenses WHERE id = ?;

-- name: ListDriverExpensesByTrip :many
SELECT id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by, approved_by, rejected_reason, approved_at,
    approved, created_at
FROM driver_expenses WHERE trip_id = ? ORDER BY created_at DESC;

-- name: ListDriverExpensesByDriver :many
SELECT id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by, approved_by, rejected_reason, approved_at,
    approved, created_at
FROM driver_expenses WHERE driver_id = ? ORDER BY created_at DESC;

-- name: ListDriverExpensesByStatus :many
SELECT id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by, approved_by, rejected_reason, approved_at,
    approved, created_at
FROM driver_expenses
WHERE COALESCE(status, 'pending') = ?
ORDER BY created_at ASC;

-- name: UpdateDriverExpenseStatus :one
UPDATE driver_expenses
SET status = ?, approved_by = ?, rejected_reason = ?, approved_at = datetime('now')
WHERE id = ?
RETURNING id, trip_id, driver_id, expense_type, amount, description, receipt_url,
    status, category, requested_by, approved_by, rejected_reason, approved_at,
    approved, created_at;

-- name: DeleteDriverExpense :exec
DELETE FROM driver_expenses WHERE id = ?;
