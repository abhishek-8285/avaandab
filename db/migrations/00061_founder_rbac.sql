-- +goose Up
-- Spec 16 §6, §7: RBAC permissions for the founder visibility layer
-- (founder signals, audit trail, dashboard). Admin-only by default.
INSERT OR IGNORE INTO permissions (name, description) VALUES
('founder:read', 'View founder signals, audit trail, and dashboard'),
('founder:update', 'Acknowledge founder signals');

-- Admin role (role id 1 per 00012 pattern) gets full access.
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name IN ('founder:read','founder:update');

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN
(SELECT id FROM permissions WHERE name IN ('founder:read','founder:update'));
DELETE FROM permissions WHERE name IN ('founder:read','founder:update');
