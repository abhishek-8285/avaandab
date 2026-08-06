-- +goose Up
-- Add missing audit_logs and files permissions
INSERT OR IGNORE INTO permissions (name, description) VALUES
    ('audit_logs:read', 'Read audit logs'),
    ('files:create', 'Upload files'),
    ('files:read', 'Read files'),
    ('files:delete', 'Delete files');

-- Assign to admin (id: 1) - gets all permissions
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name = 'audit_logs:read';

-- Assign to dispatcher (id: 2) - can view audit logs
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name = 'audit_logs:read';

-- +goose Down
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions 
    WHERE name IN ('audit_logs:read', 'files:create', 'files:read', 'files:delete')
);
DELETE FROM permissions 
WHERE name IN ('audit_logs:read', 'files:create', 'files:read', 'files:delete');
