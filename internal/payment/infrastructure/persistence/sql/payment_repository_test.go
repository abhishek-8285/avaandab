package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"

	paymentagg "transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
)

const paymentSchema = `
CREATE TABLE invoices (
	id TEXT PRIMARY KEY,
	invoice_number TEXT NOT NULL,
	total REAL NOT NULL,
	payment_status TEXT NOT NULL DEFAULT 'pending',
	tenant_id TEXT NOT NULL DEFAULT '1'
);
CREATE TABLE payments (
	id TEXT PRIMARY KEY,
	invoice_id TEXT NOT NULL,
	payment_date DATETIME NOT NULL,
	amount REAL NOT NULL,
	method TEXT NOT NULL,
	reference TEXT,
	remarks TEXT,
	tenant_id TEXT NOT NULL DEFAULT '1',
	idempotency_key TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_payments_idempotency ON payments(tenant_id, idempotency_key);
CREATE TABLE outbox_events (
	id TEXT PRIMARY KEY,
	aggregate_id TEXT NOT NULL,
	aggregate_type TEXT NOT NULL,
	event_type TEXT NOT NULL,
	payload TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	published_at DATETIME
);
`

func setupPaymentDB(t *testing.T) *sql.DB {
	t.Helper()
	dbConn, err := sql.Open("sqlite", ":memory:")
	assert.NoError(t, err)
	_, err = dbConn.Exec(paymentSchema)
	assert.NoError(t, err)
	_, err = dbConn.Exec(`INSERT INTO invoices (id, invoice_number, total, payment_status, tenant_id) VALUES ('inv-1', 'INV-0001', 1100.0, 'pending', '1')`)
	assert.NoError(t, err)
	return dbConn
}

func TestPaymentRepository_SaveAndFind(t *testing.T) {
	dbConn := setupPaymentDB(t)
	defer func() { _ = dbConn.Close() }()

	repo := NewPaymentRepository(dbConn)
	ctx := context.Background()

	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	ref := "REF-001"
	agg := paymentagg.NewPaymentAggregate(
		"pay-1",
		shared.TenantID("1"),
		"inv-1",
		now,
		500.0,
		paymentagg.PaymentMethodUPI,
		&ref,
		nil,
		now,
	)

	err := repo.Save(ctx, agg)
	assert.NoError(t, err)

	found, err := repo.Find(ctx, "pay-1", shared.TenantID("1"))
	assert.NoError(t, err)
	assert.Equal(t, "pay-1", string(found.ID))
	assert.Equal(t, "inv-1", found.InvoiceID)
	assert.Equal(t, 500.0, found.Amount)
	assert.Equal(t, paymentagg.PaymentMethodUPI, found.Method)
	assert.NotNil(t, found.Reference)
	assert.Equal(t, "REF-001", *found.Reference)
}

func TestPaymentRepository_IdempotencyKey(t *testing.T) {
	dbConn := setupPaymentDB(t)
	defer func() { _ = dbConn.Close() }()

	repo := NewPaymentRepository(dbConn)
	ctx := context.Background()

	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	ref := "REF-DUP"
	p1 := paymentagg.NewPaymentAggregate(
		"pay-2a",
		shared.TenantID("1"),
		"inv-1",
		now,
		500.0,
		paymentagg.PaymentMethodCash,
		&ref,
		nil,
		now,
	)
	err := repo.Save(ctx, p1)
	assert.NoError(t, err)

	p2 := paymentagg.NewPaymentAggregate(
		"pay-2b",
		shared.TenantID("1"),
		"inv-1",
		now,
		500.0,
		paymentagg.PaymentMethodCash,
		&ref,
		nil,
		now,
	)
	err = repo.Save(ctx, p2)
	assert.NoError(t, err)

	assert.Equal(t, string(p1.ID), string(p2.ID))

	var count int
	err = dbConn.QueryRow(`SELECT COUNT(*) FROM payments WHERE idempotency_key = 'ref:REF-DUP'`).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPaymentRepository_FindNonExistent(t *testing.T) {
	dbConn := setupPaymentDB(t)
	defer func() { _ = dbConn.Close() }()

	repo := NewPaymentRepository(dbConn)
	ctx := context.Background()

	_, err := repo.Find(ctx, "does-not-exist", shared.TenantID("1"))
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
