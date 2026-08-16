-- +goose Up
-- Extend the payments.method CHECK constraint to allow 'razorpay' (webhook
-- payments). SQLite cannot ALTER a CHECK constraint, so rebuild the table.
PRAGMA foreign_keys=OFF;
ALTER TABLE payments RENAME TO payments_old;

CREATE TABLE payments (
    id              TEXT PRIMARY KEY,
    invoice_id      TEXT NOT NULL,
    payment_date    DATETIME NOT NULL,
    amount          REAL NOT NULL,
    method          TEXT NOT NULL CHECK (method IN ('cash', 'upi', 'bank_transfer', 'cheque', 'razorpay')),
    reference       TEXT,
    remarks         TEXT,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    tenant_id       TEXT DEFAULT '1' NOT NULL,
    idempotency_key TEXT,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);

INSERT INTO payments (id, invoice_id, payment_date, amount, method, reference, remarks, created_at, updated_at, tenant_id, idempotency_key)
    SELECT id, invoice_id, payment_date, amount, method, reference, remarks, created_at, updated_at, tenant_id, idempotency_key
    FROM payments_old;

DROP TABLE payments_old;

CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE UNIQUE INDEX idx_payments_idempotency ON payments(tenant_id, idempotency_key);

PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
ALTER TABLE payments RENAME TO payments_old;

CREATE TABLE payments (
    id          TEXT PRIMARY KEY,
    invoice_id  TEXT NOT NULL,
    payment_date DATETIME NOT NULL,
    amount      REAL NOT NULL,
    method      TEXT NOT NULL CHECK (method IN ('cash', 'upi', 'bank_transfer', 'cheque')),
    reference   TEXT,
    remarks     TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    tenant_id   TEXT DEFAULT '1' NOT NULL,
    idempotency_key TEXT,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);

INSERT INTO payments (id, invoice_id, payment_date, amount, method, reference, remarks, created_at, updated_at, tenant_id, idempotency_key)
    SELECT id, invoice_id, payment_date, amount, method, reference, remarks, created_at, updated_at, tenant_id, idempotency_key
    FROM payments_old;

DROP TABLE payments_old;

CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE UNIQUE INDEX idx_payments_idempotency ON payments(tenant_id, idempotency_key);

PRAGMA foreign_keys=ON;
