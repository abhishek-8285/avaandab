-- +goose Up
ALTER TABLE payments ADD COLUMN idempotency_key TEXT;
CREATE UNIQUE INDEX idx_payments_idempotency ON payments(tenant_id, idempotency_key);

-- +goose Down
DROP INDEX IF EXISTS idx_payments_idempotency;
