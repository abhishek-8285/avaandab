-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (id, aggregate_id, aggregate_type, event_type, payload, created_at)
VALUES (?, ?, ?, ?, ?, ?);
