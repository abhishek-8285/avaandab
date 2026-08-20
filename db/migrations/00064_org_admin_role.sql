-- +goose Up
-- Add 'org_admin' role for tenant-scoped organization administration
INSERT OR IGNORE INTO roles (id, name, description) VALUES
(6, 'org_admin', 'Organization Administrator with full tenant-scoped management');

-- Grant all operational, financial, asset, and user permissions to org_admin
-- Excludes platform-wide founder signals and experiments
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 6, id FROM permissions
WHERE name NOT IN (
    'founder:read', 'founder:update',
    'experiments:read', 'experiments:write'
);

-- +goose Down
DELETE FROM role_permissions WHERE role_id = 6;
DELETE FROM roles WHERE id = 6;
