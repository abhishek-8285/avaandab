-- +goose Up
-- Spec 16 §5: RBAC permissions for the A/B experiments service + feature flag API.
INSERT OR IGNORE INTO permissions (name, description) VALUES
('experiments:read', 'View experiments, assignments and feature flag evaluation'),
('experiments:write', 'Create and manage experiments (lifecycle, metrics)');

-- Assign to admin role (role id 1 per 00012 pattern).
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name IN ('experiments:read','experiments:write');

-- Assign read to dispatcher (role id 2) and accountant (role id 3) where present.
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name IN ('experiments:read');
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions WHERE name IN ('experiments:read');

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN
(SELECT id FROM permissions WHERE name IN ('experiments:read','experiments:write'));
DELETE FROM permissions WHERE name IN ('experiments:read','experiments:write');
