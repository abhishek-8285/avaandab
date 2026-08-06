-- +goose Up
-- Base schema: roles, users, permissions, user_roles, company_settings, sessions, files, audit_logs

CREATE TABLE roles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE permissions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE users (
    id             TEXT PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    name           TEXT NOT NULL,
    phone          TEXT,
    role_id        INTEGER NOT NULL DEFAULT 2,
    status         TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    last_login_at  DATETIME,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

CREATE TABLE company_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    company_name    TEXT NOT NULL,
    logo_path       TEXT,
    currency        TEXT NOT NULL DEFAULT 'INR',
    timezone        TEXT NOT NULL DEFAULT 'Asia/Kolkata',
    gst_enabled     BOOLEAN NOT NULL DEFAULT 0,
    gst_rate        REAL NOT NULL DEFAULT 0.0,
    booking_prefix  TEXT NOT NULL DEFAULT 'BK',
    trip_prefix     TEXT NOT NULL DEFAULT 'TR',
    invoice_prefix  TEXT NOT NULL DEFAULT 'INV',
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    user_agent TEXT,
    ip_address TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE files (
    id              TEXT PRIMARY KEY,
    filename        TEXT NOT NULL,
    original_name   TEXT NOT NULL,
    path            TEXT NOT NULL,
    size            INTEGER NOT NULL,
    mime_type       TEXT NOT NULL,
    uploadable_type TEXT NOT NULL CHECK (uploadable_type IN ('driver_license', 'vehicle_insurance', 'vehicle_permit', 'company_logo')),
    uploadable_id   TEXT,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE audit_logs (
    id          TEXT PRIMARY KEY,
    user_id     TEXT,
    action      TEXT NOT NULL,
    table_name  TEXT NOT NULL,
    record_id   TEXT,
    old_values TEXT,
    new_values  TEXT,
    ip_address  TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS company_settings;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
