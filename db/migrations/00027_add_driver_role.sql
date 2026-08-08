-- +goose Up
-- Add 'driver' role if not exists
INSERT OR IGNORE INTO roles (id, name, description) VALUES (5, 'driver', 'Driver access for assigned trips and status updates');

-- Driver permissions: read assigned trips, update trip status
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 5, id FROM permissions 
WHERE name IN ('trips:read', 'trips:update', 'routes:read', 'vehicles:read');

-- +goose Down
DELETE FROM role_permissions WHERE role_id = 5;
DELETE FROM roles WHERE id = 5;
