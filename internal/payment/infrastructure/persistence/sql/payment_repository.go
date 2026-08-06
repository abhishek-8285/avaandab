package sql

import (
	"context"
	"database/sql"
	"errors"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/payment/domain"
	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/payment/infrastructure/persistence/sql/converters"
	"transport-app/internal/repository"
	"transport-app/internal/shared"
	"transport-app/internal/shared/outbox"
)

type paymentRepository struct {
	dbConn *sql.DB
	q      *db.Queries
	outbox *outbox.OutboxWriter
}

// NewPaymentRepository creates a SQLite-backed implementation of PaymentRepository.
func NewPaymentRepository(dbConn *sql.DB) domain.PaymentRepository {
	return &paymentRepository{
		dbConn: dbConn,
		q:      db.New(dbConn),
		outbox: outbox.NewOutboxWriter(dbConn),
	}
}

func (r *paymentRepository) Q(ctx context.Context) *db.Queries {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *paymentRepository) Save(ctx context.Context, p *aggregate.PaymentAggregate) error {
	var reference, remarks sql.NullString
	if p.Reference != nil {
		reference = sql.NullString{String: *p.Reference, Valid: true}
	}
	if p.Remarks != nil {
		remarks = sql.NullString{String: *p.Remarks, Valid: true}
	}

	_, err := r.Q(ctx).GetPaymentByID(ctx, db.GetPaymentByIDParams{
		ID:       string(p.ID),
		TenantID: string(p.TenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = r.Q(ctx).CreatePayment(ctx, db.CreatePaymentParams{
				ID:          string(p.ID),
				InvoiceID:   p.InvoiceID,
				PaymentDate: p.PaymentDate,
				Amount:      p.Amount,
				Method:      string(p.Method),
				Reference:   reference,
				Remarks:     remarks,
				TenantID:    string(p.TenantID),
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		return errors.New("updating payments is not allowed (immutable transaction records)")
	}

	err = r.outbox.SaveEvents(ctx, string(p.ID), "Payment", p.Events())
	if err != nil {
		return err
	}
	p.ClearEvents()
	return nil
}

func (r *paymentRepository) Find(ctx context.Context, id aggregate.PaymentID, tenantID shared.TenantID) (*aggregate.PaymentAggregate, error) {
	row, err := r.Q(ctx).GetPaymentByID(ctx, db.GetPaymentByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return nil, err
	}
	p := db.Payment{
		ID:          row.ID,
		InvoiceID:   row.InvoiceID,
		PaymentDate: row.PaymentDate,
		Amount:      row.Amount,
		Method:      row.Method,
		Reference:   row.Reference,
		Remarks:     row.Remarks,
		TenantID:    row.TenantID,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	return converters.ToDomain(p), nil
}

func (r *paymentRepository) GetReadModel(ctx context.Context, id aggregate.PaymentID, tenantID shared.TenantID) (domain.PaymentReadModel, error) {
	row, err := r.Q(ctx).GetPaymentByID(ctx, db.GetPaymentByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return domain.PaymentReadModel{}, err
	}
	var ref, rem *string
	if row.Reference.Valid {
		ref = &row.Reference.String
	}
	if row.Remarks.Valid {
		rem = &row.Remarks.String
	}
	return domain.PaymentReadModel{
		ID:            row.ID,
		InvoiceID:     row.InvoiceID,
		InvoiceNumber: row.InvoiceNumber,
		PaymentDate:   row.PaymentDate,
		Amount:        row.Amount,
		Method:        row.Method,
		Reference:     ref,
		Remarks:       rem,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *paymentRepository) GetPaymentsByInvoice(ctx context.Context, invoiceID string, tenantID shared.TenantID) ([]domain.PaymentReadModel, error) {
	rows, err := r.Q(ctx).GetPaymentsByInvoice(ctx, db.GetPaymentsByInvoiceParams{
		InvoiceID: invoiceID,
		TenantID:  string(tenantID),
	})
	if err != nil {
		return nil, err
	}

	readModels := make([]domain.PaymentReadModel, len(rows))
	for i, row := range rows {
		var ref, rem *string
		if row.Reference.Valid {
			ref = &row.Reference.String
		}
		if row.Remarks.Valid {
			rem = &row.Remarks.String
		}
		readModels[i] = domain.PaymentReadModel{
			ID:            row.ID,
			InvoiceID:     row.InvoiceID,
			InvoiceNumber: row.InvoiceNumber,
			PaymentDate:   row.PaymentDate,
			Amount:        row.Amount,
			Method:        row.Method,
			Reference:     ref,
			Remarks:       rem,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
	}

	return readModels, nil
}

func (r *paymentRepository) SearchReadModels(ctx context.Context, tenantID shared.TenantID, method string, limit int, offset int) ([]domain.PaymentReadModel, int64, error) {
	rows, err := r.Q(ctx).SearchPayments(ctx, db.SearchPaymentsParams{
		TenantID: string(tenantID),
		Column2:  method,
		Method:   method,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.Q(ctx).CountPayments(ctx, db.CountPaymentsParams{
		TenantID: string(tenantID),
		Column2:  method,
		Method:   method,
	})
	if err != nil {
		return nil, 0, err
	}

	readModels := make([]domain.PaymentReadModel, len(rows))
	for i, row := range rows {
		var ref, rem *string
		if row.Reference.Valid {
			ref = &row.Reference.String
		}
		if row.Remarks.Valid {
			rem = &row.Remarks.String
		}
		readModels[i] = domain.PaymentReadModel{
			ID:            row.ID,
			InvoiceID:     row.InvoiceID,
			InvoiceNumber: row.InvoiceNumber,
			PaymentDate:   row.PaymentDate,
			Amount:        row.Amount,
			Method:        row.Method,
			Reference:     ref,
			Remarks:       rem,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
	}

	return readModels, total, nil
}
