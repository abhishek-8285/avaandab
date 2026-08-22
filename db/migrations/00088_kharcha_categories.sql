-- +goose Up
-- Kharcha category expansion: add rto, tyre, bhatta.
-- Why: the mobile app already offers RTO and TYRE categories (ExpenseScreen),
-- which the server CHECK rejects — a live bug. bhatta (daily allowance) is a
-- standard Indian fleet expense the settlement flow should capture.
-- SQLite cannot ALTER a CHECK constraint, so driver_expenses is rebuilt with
-- the identical schema except the widened category AND expense_type lists (the
-- service writes the category into both columns) (00031 + 00032 + 00043
-- + 00076 + 00082 columns preserved).

CREATE TABLE driver_expenses_new (
  id TEXT PRIMARY KEY,
  trip_id TEXT,
  driver_id TEXT,
  expense_type TEXT NOT NULL CHECK (expense_type IN ('fuel', 'toll', 'food', 'repair', 'advance', 'other', 'rto', 'tyre', 'bhatta')),
  amount REAL NOT NULL,
  description TEXT,
  receipt_url TEXT,
  approved INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'settled')),
  category TEXT NOT NULL DEFAULT 'advance' CHECK (category IN ('advance', 'fuel', 'toll', 'food', 'repair', 'other', 'rto', 'tyre', 'bhatta')),
  requested_by TEXT REFERENCES drivers(id),
  approved_by TEXT REFERENCES users(id),
  rejected_reason TEXT,
  approved_at DATETIME,
  audit_status TEXT NOT NULL DEFAULT 'pending' CHECK (audit_status IN ('pending','needs_review','passed','failed')),
  fuel_litres REAL,
  idempotency_key TEXT,
  latitude REAL,
  longitude REAL,
  FOREIGN KEY (trip_id) REFERENCES trips(id),
  FOREIGN KEY (driver_id) REFERENCES drivers(id)
);

INSERT INTO driver_expenses_new (id, trip_id, driver_id, expense_type, amount, description,
  receipt_url, approved, created_at, status, category, requested_by, approved_by,
  rejected_reason, approved_at, audit_status, fuel_litres, idempotency_key, latitude, longitude)
SELECT id, trip_id, driver_id, expense_type, amount, description,
  receipt_url, approved, created_at, status, category, requested_by, approved_by,
  rejected_reason, approved_at, audit_status, fuel_litres, idempotency_key, latitude, longitude
FROM driver_expenses;

DROP TABLE driver_expenses;
ALTER TABLE driver_expenses_new RENAME TO driver_expenses;

CREATE INDEX IF NOT EXISTS idx_driver_expenses_status ON driver_expenses(status);
CREATE INDEX IF NOT EXISTS idx_driver_expenses_trip ON driver_expenses(trip_id);
CREATE INDEX IF NOT EXISTS idx_driver_expenses_driver ON driver_expenses(driver_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_driver_expenses_idempotency ON driver_expenses(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key != '';

-- +goose Down
-- Rebuild with the original narrow category list; rto/tyre/bhatta rows fold
-- into the nearest original bucket so both CHECKs hold (rto/tyre → repair,
-- bhatta → advance, category → other). Lossy for detail — intentional.

CREATE TABLE driver_expenses_old (
  id TEXT PRIMARY KEY,
  trip_id TEXT,
  driver_id TEXT,
  expense_type TEXT NOT NULL CHECK (expense_type IN ('fuel', 'toll', 'food', 'repair', 'advance')),
  amount REAL NOT NULL,
  description TEXT,
  receipt_url TEXT,
  approved INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'settled')),
  category TEXT NOT NULL DEFAULT 'advance' CHECK (category IN ('advance', 'fuel', 'toll', 'food', 'repair', 'other')),
  requested_by TEXT REFERENCES drivers(id),
  approved_by TEXT REFERENCES users(id),
  rejected_reason TEXT,
  approved_at DATETIME,
  audit_status TEXT NOT NULL DEFAULT 'pending' CHECK (audit_status IN ('pending','needs_review','passed','failed')),
  fuel_litres REAL,
  idempotency_key TEXT,
  latitude REAL,
  longitude REAL,
  FOREIGN KEY (trip_id) REFERENCES trips(id),
  FOREIGN KEY (driver_id) REFERENCES drivers(id)
);

INSERT INTO driver_expenses_old (id, trip_id, driver_id, expense_type, amount, description,
  receipt_url, approved, created_at, status, category, requested_by, approved_by,
  rejected_reason, approved_at, audit_status, fuel_litres, idempotency_key, latitude, longitude)
SELECT id, trip_id, driver_id,
  CASE expense_type WHEN 'rto' THEN 'repair' WHEN 'tyre' THEN 'repair' WHEN 'bhatta' THEN 'advance' ELSE expense_type END,
  amount, description,
  receipt_url, approved, created_at, status,
  CASE category WHEN 'rto' THEN 'other' WHEN 'tyre' THEN 'other' WHEN 'bhatta' THEN 'other' ELSE category END,
  requested_by, approved_by,
  rejected_reason, approved_at, audit_status, fuel_litres, idempotency_key, latitude, longitude
FROM driver_expenses;

DROP TABLE driver_expenses;
ALTER TABLE driver_expenses_old RENAME TO driver_expenses;

CREATE INDEX IF NOT EXISTS idx_driver_expenses_status ON driver_expenses(status);
CREATE INDEX IF NOT EXISTS idx_driver_expenses_trip ON driver_expenses(trip_id);
CREATE INDEX IF NOT EXISTS idx_driver_expenses_driver ON driver_expenses(driver_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_driver_expenses_idempotency ON driver_expenses(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key != '';
