-- +goose Up
-- Add timeline columns to trips table for tracking execution milestones.
ALTER TABLE trips ADD COLUMN started_at DATETIME;
ALTER TABLE trips ADD COLUMN reached_pickup_at DATETIME;
ALTER TABLE trips ADD COLUMN in_transit_at DATETIME;
ALTER TABLE trips ADD COLUMN delivered_at DATETIME;
ALTER TABLE trips ADD COLUMN completed_at DATETIME;

-- +goose Down
-- SQLite does not support dropping columns in older versions;
-- timeline columns were added in this migration only.
-- If downgrade is needed, recreate the table without these columns.
-- +goose StatementBegin
CREATE TABLE trips_backup AS SELECT id, trip_number, booking_id, driver_id, vehicle_id, route_id,
       departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at
FROM trips;
DROP TABLE trips;
CREATE TABLE trips (
    id            TEXT PRIMARY KEY,
    trip_number   TEXT NOT NULL UNIQUE,
    booking_id    TEXT,
    driver_id     TEXT,
    vehicle_id    TEXT,
    route_id      TEXT NOT NULL,
    departure_time DATETIME NOT NULL,
    arrival_time   DATETIME,
    status        TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'assigned', 'started', 'completed', 'cancelled')),
    remarks       TEXT,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    version       INTEGER NOT NULL DEFAULT 1,
    tenant_id     TEXT NOT NULL DEFAULT '1',
    FOREIGN KEY (booking_id) REFERENCES bookings(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (route_id) REFERENCES routes(id)
);
INSERT INTO trips SELECT id, trip_number, booking_id, driver_id, vehicle_id, route_id,
     departure_time, arrival_time, status, remarks, tenant_id, version, created_at, updated_at
FROM trips_backup;
DROP TABLE trips_backup;
-- +goose StatementEnd
