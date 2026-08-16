package sql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/invoice/domain"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/repository"
	"transport-app/internal/shared"
	"transport-app/internal/shared/outbox"
)

const updateInvoiceExtraFieldsSQL = `
UPDATE invoices
SET status = ?, paid_amount = ?, due_date = ?, updated_at = datetime('now')
WHERE id = ? AND tenant_id = ?
`

const findInvoiceByIDSQL = `
SELECT id, invoice_number, booking_id, customer_id, trip_id,
    subtotal, tax, discount, total, payment_status,
    paid_amount, status, due_date, version,
    tenant_id, created_at, updated_at
FROM invoices
WHERE id = ? AND tenant_id = ?
`

const findInvoiceByBookingSQL = `
SELECT id, invoice_number, booking_id, customer_id, trip_id,
    subtotal, tax, discount, total, payment_status,
    paid_amount, status, due_date, version,
    tenant_id, created_at, updated_at
FROM invoices
WHERE booking_id = ? AND tenant_id = ?
`

type invoiceRepository struct {
	dbConn *sql.DB
	q      *db.Queries
	outbox *outbox.OutboxWriter
}

// NewInvoiceRepository creates a SQLite-backed implementation of InvoiceRepository.
func NewInvoiceRepository(dbConn *sql.DB) domain.InvoiceRepository {
	return &invoiceRepository{
		dbConn: dbConn,
		q:      db.New(dbConn),
		outbox: outbox.NewOutboxWriter(dbConn),
	}
}

func (r *invoiceRepository) Q(ctx context.Context) *db.Queries {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *invoiceRepository) exec(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
} {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.dbConn
}

func (r *invoiceRepository) Save(ctx context.Context, inv *aggregate.InvoiceAggregate) error {
	var tripID sql.NullString
	if inv.TripID != nil {
		tripID = sql.NullString{String: *inv.TripID, Valid: true}
	}

	var exists bool
	err := r.exec(ctx).QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM invoices WHERE id = ? AND tenant_id = ?)", string(inv.ID), string(inv.TenantID)).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = r.Q(ctx).CreateInvoice(ctx, db.CreateInvoiceParams{
			ID:            string(inv.ID),
			InvoiceNumber: inv.InvoiceNumber,
			BookingID:     inv.BookingID,
			CustomerID:    inv.CustomerID,
			TripID:        tripID,
			Subtotal:      inv.Subtotal,
			Tax:           inv.Tax,
			Discount:      inv.Discount,
			Total:         inv.Total,
			PaymentStatus: string(inv.PaymentStatus),
			TenantID:      string(inv.TenantID),
		})
		if err != nil {
			return err
		}
	} else {
		_, err = r.Q(ctx).UpdateInvoice(ctx, db.UpdateInvoiceParams{
			InvoiceNumber: inv.InvoiceNumber,
			BookingID:     inv.BookingID,
			CustomerID:    inv.CustomerID,
			TripID:        tripID,
			Subtotal:      inv.Subtotal,
			Tax:           inv.Tax,
			Discount:      inv.Discount,
			Total:         inv.Total,
			PaymentStatus: string(inv.PaymentStatus),
			ID:            string(inv.ID),
			TenantID:      string(inv.TenantID),
		})
		if err != nil {
			return err
		}
		var dueDate sql.NullTime
		if inv.DueDate != nil {
			dueDate = sql.NullTime{Time: *inv.DueDate, Valid: true}
		}
		_, err = r.exec(ctx).ExecContext(ctx, updateInvoiceExtraFieldsSQL,
			string(inv.Status), inv.PaidAmount, dueDate,
			string(inv.ID), string(inv.TenantID),
		)
		if err != nil {
			return err
		}
	}

	err = r.outbox.SaveEvents(ctx, string(inv.ID), "Invoice", inv.Events())
	if err != nil {
		return err
	}
	inv.ClearEvents()
	return nil
}

func (r *invoiceRepository) Find(ctx context.Context, id aggregate.InvoiceID, tenantID shared.TenantID) (*aggregate.InvoiceAggregate, error) {
	return r.findInvoiceBySQL(ctx, findInvoiceByIDSQL, string(id), string(tenantID))
}

func (r *invoiceRepository) findInvoiceBySQL(ctx context.Context, querySQL string, args ...any) (*aggregate.InvoiceAggregate, error) {
	row := r.exec(ctx).QueryRowContext(ctx, querySQL, args...)
	var (
		id, invoiceNumber, bookingID, customerID, tenantID string
		tripID                                             sql.NullString
		subtotal, tax, discount, total, paidAmount         float64
		paymentStatus, status                              string
		dueDate                                            sql.NullTime
		version                                            int64
		createdAt, updatedAt                               time.Time
	)
	err := row.Scan(
		&id, &invoiceNumber, &bookingID, &customerID, &tripID,
		&subtotal, &tax, &discount, &total, &paymentStatus,
		&paidAmount, &status, &dueDate, &version,
		&tenantID, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	var tripPtr *string
	if tripID.Valid {
		tripPtr = &tripID.String
	}
	var dueDatePtr *time.Time
	if dueDate.Valid {
		dueDatePtr = &dueDate.Time
	}
	invStatus := aggregate.InvoiceStatus(status)
	if invStatus == "" {
		invStatus = aggregate.InvoiceStatusOutstanding
	}
	return aggregate.RehydrateInvoiceAggregate(
		aggregate.InvoiceID(id), shared.TenantID(tenantID), invoiceNumber,
		bookingID, customerID, tripPtr,
		subtotal, tax, discount, total,
		aggregate.PaymentStatus(paymentStatus), invStatus,
		paidAmount, 0, // creditBalance not in DB, default 0
		dueDatePtr, "", "", // financialYear, remarks not in invoices table
		createdAt, updatedAt, version,
	), nil
}

func (r *invoiceRepository) GetReadModel(ctx context.Context, id aggregate.InvoiceID, tenantID shared.TenantID) (domain.InvoiceReadModel, error) {
	row, err := r.Q(ctx).GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return domain.InvoiceReadModel{}, err
	}
	var tripID *string
	if row.TripID.Valid {
		tripID = &row.TripID.String
	}
	return domain.InvoiceReadModel{
		ID:              row.ID,
		InvoiceNumber:   row.InvoiceNumber,
		BookingID:       row.BookingID,
		BookingNumber:   row.BookingNumber.String,
		CustomerID:      row.CustomerID,
		CustomerName:    row.CustomerName,
		CustomerCompany: row.CustomerCompany.String,
		TripID:          tripID,
		TripNumber:      row.TripNumber.String,
		Subtotal:        row.Subtotal,
		Tax:             row.Tax,
		Discount:        row.Discount,
		Total:           row.Total,
		PaymentStatus:   row.PaymentStatus,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *invoiceRepository) SearchReadModels(ctx context.Context, tenantID shared.TenantID, query string, status string, limit int, offset int) ([]domain.InvoiceReadModel, int64, error) {
	rows, err := r.Q(ctx).SearchInvoices(ctx, db.SearchInvoicesParams{
		TenantID:      string(tenantID),
		Column2:       sql.NullString{String: query, Valid: true},
		Column3:       sql.NullString{String: query, Valid: true},
		Column4:       status,
		PaymentStatus: status,
		Limit:         int64(limit),
		Offset:        int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.Q(ctx).CountInvoices(ctx, db.CountInvoicesParams{
		TenantID:      string(tenantID),
		Column2:       sql.NullString{String: query, Valid: true},
		Column3:       sql.NullString{String: query, Valid: true},
		Column4:       status,
		PaymentStatus: status,
	})
	if err != nil {
		return nil, 0, err
	}

	readModels := make([]domain.InvoiceReadModel, len(rows))
	for i, row := range rows {
		var tripID *string
		if row.TripID.Valid {
			tripID = &row.TripID.String
		}
		readModels[i] = domain.InvoiceReadModel{
			ID:              row.ID,
			InvoiceNumber:   row.InvoiceNumber,
			BookingID:       row.BookingID,
			BookingNumber:   row.BookingNumber.String,
			CustomerID:      row.CustomerID,
			CustomerName:    row.CustomerName,
			CustomerCompany: row.CustomerCompany.String,
			TripID:          tripID,
			TripNumber:      row.TripNumber.String,
			Subtotal:        row.Subtotal,
			Tax:             row.Tax,
			Discount:        row.Discount,
			Total:           row.Total,
			PaymentStatus:   row.PaymentStatus,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
	}

	return readModels, total, nil
}

func (r *invoiceRepository) FindByBookingID(ctx context.Context, bookingID string, tenantID shared.TenantID) (*aggregate.InvoiceAggregate, error) {
	return r.findInvoiceBySQL(ctx, findInvoiceByBookingSQL, bookingID, string(tenantID))
}
