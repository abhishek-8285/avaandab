-- +goose Up
CREATE TABLE customers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    company     TEXT,
    phone       TEXT NOT NULL,
    email       TEXT,
    gst         TEXT,
    address     TEXT,
    notes       TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE IF EXISTS customers;
