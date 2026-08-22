-- +goose Up
-- Capture driver GPS with expense claims (Spec 13 mobile kharcha flow).
ALTER TABLE driver_expenses ADD COLUMN latitude REAL;
ALTER TABLE driver_expenses ADD COLUMN longitude REAL;

-- +goose Down
ALTER TABLE driver_expenses DROP COLUMN longitude;
ALTER TABLE driver_expenses DROP COLUMN latitude;
