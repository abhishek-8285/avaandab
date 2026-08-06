-- +goose Up
ALTER TABLE trips ADD COLUMN tenant_id TEXT DEFAULT '1' NOT NULL;

-- +goose Down
