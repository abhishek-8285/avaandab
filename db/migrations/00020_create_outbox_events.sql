-- +goose Up
CREATE TABLE outbox_events (
    id TEXT PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    published_at DATETIME
);

-- +goose Down
DROP TABLE outbox_events;
