-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Asia/Kolkata';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN timezone;
-- +goose StatementEnd

