-- +goose Up
-- Seed roles
INSERT OR IGNORE INTO roles (name, description) VALUES ('admin', 'Full system administrator');
INSERT OR IGNORE INTO roles (name, description) VALUES ('dispatcher', 'Manages bookings and trips');
INSERT OR IGNORE INTO roles (name, description) VALUES ('accountant', 'Handles invoices and payments');
INSERT OR IGNORE INTO roles (name, description) VALUES ('viewer', 'Read-only access');

-- Seed default company settings
INSERT OR IGNORE INTO company_settings (id, company_name) VALUES (1, 'Transport Company');

-- +goose Down
DELETE FROM company_settings WHERE id = 1;
DELETE FROM roles WHERE name IN ('admin', 'dispatcher', 'accountant', 'viewer');
