-- +goose Up
-- The original 00011 migration (dev-only) seeded an admin with a known
-- password (admin@transport.local / admin123). Neutralize that account IF it
-- still uses the known default hash: block login and replace the usable
-- hash with a dead one. Accounts whose password was changed (i.e. a real
-- admin in use) are left untouched. Deleting the row is avoided because
-- audit_logs / dispatches may reference it via non-cascading FKs.
UPDATE users
SET status = 'inactive',
    password_hash = '$2a$04$LHwgzzaNLzgDanJ0lJl1RO.B3yYsbEdvpPSZ.D6IDXlqTsy64NFwC'
WHERE email = 'admin@transport.local'
  AND password_hash = '$2a$10$4iu79R3QI9SFJpbyyu5RN..eKYXNetETGcS8bQXFdsVJ5.CtjrMC.';

-- +goose Down
-- Intentionally not restored; the seed migration is a no-op now.
SELECT 1;
