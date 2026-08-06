-- +goose Up
CREATE TABLE vehicles (
    id                TEXT PRIMARY KEY,
    registration_number TEXT NOT NULL UNIQUE,
    vehicle_number    TEXT NOT NULL,
    vehicle_type      TEXT NOT NULL CHECK (vehicle_type IN ('truck', 'mini_truck', 'bus', 'van', 'pickup', 'tempo')),
    capacity          INTEGER NOT NULL,
    fuel_type         TEXT NOT NULL DEFAULT 'diesel' CHECK (fuel_type IN ('diesel', 'petrol', 'gas', 'electric', 'cng')),
    insurance_expiry  DATE NOT NULL,
    fitness_expiry    DATE NOT NULL,
    permit_expiry     DATE NOT NULL,
    status            TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'running', 'maintenance', 'inactive')),
    current_mileage   REAL,
    created_at        DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at        DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE IF EXISTS vehicles;
