-- +goose Up
-- Seed an admin user for initial login (dev only)
-- Password: admin123
INSERT OR IGNORE INTO users (id, email, password_hash, name, phone, role_id, status, created_at, updated_at)
VALUES (
    '765f6e4e-3b2a-4c1d-9e0f-1a2b3c4d5e6f',
    'admin@transport.local',
    '$2a$10$4iu79R3QI9SFJpbyyu5RN..eKYXNetETGcS8bQXFdsVJ5.CtjrMC.',
    'Admin User',
    NULL,
    1,
    'active',
    datetime('now'),
    datetime('now')
);

-- +goose Down
DELETE FROM users WHERE email = 'admin@transport.local';