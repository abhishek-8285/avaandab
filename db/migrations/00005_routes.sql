-- +goose Up
CREATE TABLE routes (
    id              TEXT PRIMARY KEY,
    source          TEXT NOT NULL,
    destination     TEXT NOT NULL,
    distance        REAL NOT NULL,
    estimated_hours REAL NOT NULL,
    standard_fare   REAL NOT NULL,
    remarks         TEXT,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE IF EXISTS routes;
