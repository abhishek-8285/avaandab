-- +goose Up
-- Add missing bookings:approve permission and assign to roles
INSERT OR IGNORE INTO permissions (name, description)
VALUES ('bookings:approve', 'Approve bookings');

-- Assign to admin role (id: 1) - gets all permissions
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name = 'bookings:approve';

-- Assign to dispatcher role (id: 2) - gets all bookings permissions
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name = 'bookings:approve';

-- +goose Down
DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE name = 'bookings:approve');

DELETE FROM permissions WHERE name = 'bookings:approve';
