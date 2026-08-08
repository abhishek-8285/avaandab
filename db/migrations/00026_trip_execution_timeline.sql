-- +goose Up
-- Add timeline columns to trips table for tracking execution milestones.
-- Also update the CHECK constraint to include the new execution statuses.
-- SQLite does not support ALTER TABLE to modify constraints, so we recreate the table.
-- +goose StatementBegin
CREATE TABLE trips_new (
    id            TEXT PRIMARY KEY,
    trip_number   TEXT NOT NULL UNIQUE,
    booking_id    TEXT,
    driver_id     TEXT,
    vehicle_id    TEXT,
    route_id      TEXT NOT NULL,
    departure_time DATETIME NOT NULL,
    arrival_time   DATETIME,
    status        TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'assigned', 'started', 'reached_pickup', 'in_transit', 'delivered', 'completed', 'cancelled')),
    remarks       TEXT,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    version       INTEGER NOT NULL DEFAULT 1,
    tenant_id     TEXT NOT NULL DEFAULT '1',
    started_at    DATETIME,
    reached_pickup_at DATETIME,
    in_transit_at DATETIME,
    delivered_at  DATETIME,
    completed_at  DATETIME,
    FOREIGN KEY (booking_id) REFERENCES bookings(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (route_id) REFERENCES routes(id)
);
INSERT INTO trips_new
    (id, trip_number, booking_id, driver_id, vehicle_id, route_id, departure_time, arrival_time, status, remarks, created_at, updated_at, version, tenant_id)
SELECT id, trip_number, booking_id, driver_id, vehicle_id, route_id, departure_time, arrival_time, status, remarks, created_at, updated_at, version, tenant_id
FROM trips;
DROP TABLE trips;
ALTER TABLE trips_new RENAME TO trips;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE trips_old (
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
INSERT INTO trips_old
    (id, trip_number, booking_id, driver_id, vehicle_id, route_id, departure_time, arrival_time, status, remarks, created_at, updated_at, version, tenant_id)
SELECT id, trip_number, booking_id, driver_id, vehicle_id, route_id, departure_time, arrival_time, status, remarks, created_at, updated_at, version, tenant_id
FROM trips;
DROP TABLE trips;
ALTER TABLE trips_old RENAME TO trips;
-- +goose StatementEnd
