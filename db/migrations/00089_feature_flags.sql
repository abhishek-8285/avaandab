-- +goose Up
-- Per-org feature flags (plugin-style enable/disable). Catalog lives in
-- internal/features/features.go; this table holds explicit per-tenant
-- grants/revocations that override tier defaults. RBAC: features:update
-- (admin-only) gates the /settings/features toggle UI.

CREATE TABLE IF NOT EXISTS feature_flags (
    tenant_id  TEXT NOT NULL,
    feature    TEXT NOT NULL,
    enabled    INTEGER NOT NULL,
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, feature)
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_tenant ON feature_flags (tenant_id);

INSERT OR IGNORE INTO permissions (name, description) VALUES
    ('features:update', 'Enable and disable product features for the organisation');

INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name = 'features:update';

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN
(SELECT id FROM permissions WHERE name = 'features:update');
DELETE FROM permissions WHERE name = 'features:update';
DROP TABLE IF EXISTS feature_flags;
