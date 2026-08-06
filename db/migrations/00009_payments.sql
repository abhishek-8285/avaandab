-- +goose Up
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
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);

CREATE INDEX idx_payments_invoice ON payments(invoice_id);

-- +goose Down
DROP TABLE IF EXISTS payments;
