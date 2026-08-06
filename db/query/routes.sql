-- name: CreateRoute :one
INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, remarks)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, source, destination, distance, estimated_hours, standard_fare, remarks, created_at, updated_at;

-- name: GetRouteByID :one
SELECT id, source, destination, distance, estimated_hours, standard_fare, remarks, created_at, updated_at
FROM routes WHERE id = ?;

-- name: UpdateRoute :one
UPDATE routes
SET source = ?, destination = ?, distance = ?, estimated_hours = ?, standard_fare = ?, remarks = ?,
    updated_at = datetime('now')
WHERE id = ?
RETURNING id, source, destination, distance, estimated_hours, standard_fare, remarks, created_at, updated_at;

-- name: DeleteRoute :exec
DELETE FROM routes WHERE id = ?;

-- name: SearchRoutes :many
SELECT id, source, destination, distance, estimated_hours, standard_fare, remarks, created_at, updated_at
FROM routes
WHERE source LIKE '%' || ? || '%' OR destination LIKE '%' || ? || '%'
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountRoutes :one
SELECT COUNT(*) AS count
FROM routes
WHERE source LIKE '%' || ? || '%' OR destination LIKE '%' || ? || '%';

-- name: GetRouteBySourceAndDestination :one
SELECT id, source, destination, distance, estimated_hours, standard_fare, remarks, created_at, updated_at
FROM routes WHERE source = ? AND destination = ?;
