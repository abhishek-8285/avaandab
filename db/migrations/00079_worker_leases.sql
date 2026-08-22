-- +goose Up
-- Worker leader-election leases so background jobs run on exactly one replica.
-- expires_at is BIGINT epoch-millis for cross-engine portability (SQLite/PG/MySQL).
CREATE TABLE worker_leases (
    name       TEXT PRIMARY KEY,
    holder     TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX idx_worker_leases_expiry ON worker_leases(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_worker_leases_expiry;
DROP TABLE IF EXISTS worker_leases;
