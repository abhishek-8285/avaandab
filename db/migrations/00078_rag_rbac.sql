-- +goose Up
-- Security audit fix M2: RAG API endpoints previously required only
-- authentication. Seed rag:read / rag:write permissions so
-- RequirePermission("rag", ...) gates /api/rag/* (search/stats vs
-- index/reindex/teach/upload).

INSERT OR IGNORE INTO permissions (name, description) VALUES
('rag:read', 'Search knowledge base and view index stats'),
('rag:write', 'Teach, index, reindex and upload knowledge base content');

-- Admin (role id 1) gets both.
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions WHERE name IN ('rag:read','rag:write');

-- Dispatcher (role id 2) gets read-only access.
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE name IN ('rag:read');

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN
(SELECT id FROM permissions WHERE name IN ('rag:read','rag:write'));
DELETE FROM permissions WHERE name IN ('rag:read','rag:write');
