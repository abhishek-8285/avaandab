-- +goose Up
ALTER TABLE invoices ADD COLUMN tenant_id TEXT DEFAULT '1' NOT NULL;
ALTER TABLE payments ADD COLUMN tenant_id TEXT DEFAULT '1' NOT NULL;

-- +goose Down
