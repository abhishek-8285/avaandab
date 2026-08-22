-- +goose Up
-- 00081: Realign trips.status CHECK with the domain lifecycle (Spec 09).
-- 00007's CHECK predates 'reached_pickup' / 'in_transit' / 'delivered', so ANY
-- delivery write (TripDelivered via UpdateTripStatus, DeliverWithPOD, mobile
-- e-POD) failed with `CHECK constraint failed` once the trip left 'started'.
-- Verified live 2026-08-22: POST deliver-pod on a started trip → 500 (275).
-- SQLite cannot ALTER a CHECK: rebuild the table, preserving every column.

CREATE TABLE trips_rebuild_00081 (
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
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now')), tenant_id TEXT DEFAULT '1' NOT NULL, version INTEGER NOT NULL DEFAULT 1, started_at DATETIME, reached_pickup_at DATETIME, in_transit_at DATETIME, delivered_at DATETIME, completed_at DATETIME, eway_bill_ref TEXT, pod_url TEXT, final_settlement_amount REAL NOT NULL DEFAULT 0.0, pod_photo_url TEXT, pod_signature_url TEXT, pod_consignee_name TEXT, pod_consignee_phone TEXT, pod_otp_verified INTEGER NOT NULL DEFAULT 0, pod_captured_at DATETIME, pod_lat REAL, pod_lng REAL, pod_notes TEXT, estimated_margin REAL NOT NULL DEFAULT 0, fuel_consumed_liters REAL NOT NULL DEFAULT 0, toll_costs REAL NOT NULL DEFAULT 0, last_pnl_update DATETIME, fuel_cost_low REAL NOT NULL DEFAULT 0, fuel_cost_high REAL NOT NULL DEFAULT 0, margin_low REAL NOT NULL DEFAULT 0, margin_high REAL NOT NULL DEFAULT 0, pnl_confidence TEXT NOT NULL DEFAULT 'unavailable', fuel_cost_status TEXT NOT NULL DEFAULT 'pending_verification', pod_signature_data TEXT, pod_quantity_short REAL DEFAULT 0, pod_damage_qty REAL DEFAULT 0, pod_refusal_reason TEXT,
    FOREIGN KEY (booking_id) REFERENCES bookings(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (route_id) REFERENCES routes(id)
);

INSERT INTO trips_rebuild_00081 SELECT * FROM trips;
DROP TABLE trips;
ALTER TABLE trips_rebuild_00081 RENAME TO trips;

-- +goose Down
-- Map newer statuses back to the legacy enum before restoring the old CHECK.
UPDATE trips SET status = 'completed' WHERE status IN ('reached_pickup', 'in_transit', 'delivered');

ALTER TABLE trips RENAME TO trips_down_00081;
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
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now')), tenant_id TEXT DEFAULT '1' NOT NULL, version INTEGER NOT NULL DEFAULT 1, started_at DATETIME, reached_pickup_at DATETIME, in_transit_at DATETIME, delivered_at DATETIME, completed_at DATETIME, eway_bill_ref TEXT, pod_url TEXT, final_settlement_amount REAL NOT NULL DEFAULT 0.0, pod_photo_url TEXT, pod_signature_url TEXT, pod_consignee_name TEXT, pod_consignee_phone TEXT, pod_otp_verified INTEGER NOT NULL DEFAULT 0, pod_captured_at DATETIME, pod_lat REAL, pod_lng REAL, pod_notes TEXT, estimated_margin REAL NOT NULL DEFAULT 0, fuel_consumed_liters REAL NOT NULL DEFAULT 0, toll_costs REAL NOT NULL DEFAULT 0, last_pnl_update DATETIME, fuel_cost_low REAL NOT NULL DEFAULT 0, fuel_cost_high REAL NOT NULL DEFAULT 0, margin_low REAL NOT NULL DEFAULT 0, margin_high REAL NOT NULL DEFAULT 0, pnl_confidence TEXT NOT NULL DEFAULT 'unavailable', fuel_cost_status TEXT NOT NULL DEFAULT 'pending_verification', pod_signature_data TEXT, pod_quantity_short REAL DEFAULT 0, pod_damage_qty REAL DEFAULT 0, pod_refusal_reason TEXT,
    FOREIGN KEY (booking_id) REFERENCES bookings(id),
    FOREIGN KEY (driver_id) REFERENCES drivers(id),
    FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    FOREIGN KEY (route_id) REFERENCES routes(id)
);

INSERT INTO trips SELECT * FROM trips_down_00081;
DROP TABLE trips_down_00081;
