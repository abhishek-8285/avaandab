-- +goose Up
ALTER TABLE drivers ADD COLUMN tenant_id TEXT DEFAULT '1' NOT NULL;
ALTER TABLE vehicles ADD COLUMN tenant_id TEXT DEFAULT '1' NOT NULL;

-- +goose Down
