package sql

import (
	"context"
	"database/sql"
	"errors"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/invoice/domain"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/invoice/infrastructure/persistence/sql/converters"
	"transport-app/internal/repository"
	"transport-app/internal/shared"
	"transport-app/internal/shared/outbox"
)

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

func (r *invoiceRepository) Save(ctx context.Context, inv *aggregate.InvoiceAggregate) error {
	var tripID sql.NullString
	if inv.TripID != nil {
		tripID = sql.NullString{String: *inv.TripID, Valid: true}
	}

	_, err := r.Q(ctx).GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:       string(inv.ID),
		TenantID: string(inv.TenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	}

	err = r.outbox.SaveEvents(ctx, string(inv.ID), "Invoice", inv.Events())
	if err != nil {
		return err
	}
	inv.ClearEvents()
	return nil
}

func (r *invoiceRepository) Find(ctx context.Context, id aggregate.InvoiceID, tenantID shared.TenantID) (*aggregate.InvoiceAggregate, error) {
	row, err := r.Q(ctx).GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return nil, err
	}
	tripID := sql.NullString{String: row.TripID.String, Valid: row.TripID.Valid}
	inv := db.Invoice{
		ID:            row.ID,
		InvoiceNumber: row.InvoiceNumber,
		BookingID:     row.BookingID,
		CustomerID:    row.CustomerID,
		TripID:        tripID,
		Subtotal:      row.Subtotal,
		Tax:           row.Tax,
		Discount:      row.Discount,
		Total:         row.Total,
		PaymentStatus: row.PaymentStatus,
		TenantID:      row.TenantID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	return converters.ToDomain(inv), nil
}

func (r *invoiceRepository) GetReadModel(ctx context.Context, id aggregate.InvoiceID, tenantID shared.TenantID) (domain.InvoiceReadModel, error) {
	row, err := r.Q(ctx).GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return domain.InvoiceReadModel{}, err
	}
	tripID := sql.NullString{String: row.TripID.String, Valid: row.TripID.Valid}
	inv := db.Invoice{
		ID:            row.ID,
		InvoiceNumber: row.InvoiceNumber,
		BookingID:     row.BookingID,
		CustomerID:    row.CustomerID,
		TripID:        tripID,
		Subtotal:      row.Subtotal,
		Tax:           row.Tax,
		Discount:      row.Discount,
		Total:         row.Total,
		PaymentStatus: row.PaymentStatus,
		TenantID:      row.TenantID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	return converters.ToReadModel(inv), nil
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
		tripID := sql.NullString{String: row.TripID.String, Valid: row.TripID.Valid}
		inv := db.Invoice{
			ID:            row.ID,
			InvoiceNumber: row.InvoiceNumber,
			BookingID:     row.BookingID,
			CustomerID:    row.CustomerID,
			TripID:        tripID,
			Subtotal:      row.Subtotal,
			Tax:           row.Tax,
			Discount:      row.Discount,
			Total:         row.Total,
			PaymentStatus: row.PaymentStatus,
			TenantID:      row.TenantID,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
		readModels[i] = converters.ToReadModel(inv)
	}

	return readModels, total, nil
}

func (r *invoiceRepository) FindByBookingID(ctx context.Context, bookingID string, tenantID shared.TenantID) (*aggregate.InvoiceAggregate, error) {
	row, err := r.Q(ctx).GetInvoiceByBookingID(ctx, db.GetInvoiceByBookingIDParams{
		BookingID: bookingID,
		TenantID:  string(tenantID),
	})
	if err != nil {
		return nil, err
	}
	tripID := sql.NullString{String: row.TripID.String, Valid: row.TripID.Valid}
	inv := db.Invoice{
		ID:            row.ID,
		InvoiceNumber: row.InvoiceNumber,
		BookingID:     row.BookingID,
		CustomerID:    row.CustomerID,
		TripID:        tripID,
		Subtotal:      row.Subtotal,
		Tax:           row.Tax,
		Discount:      row.Discount,
		Total:         row.Total,
		PaymentStatus: row.PaymentStatus,
		TenantID:      row.TenantID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	return converters.ToDomain(inv), nil
}
