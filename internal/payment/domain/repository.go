package domain

import (
	"context"
	"time"

	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
)

// PaymentReadModel optimized for read queries.
type PaymentReadModel struct {
	ID            string
	InvoiceID     string
	InvoiceNumber string
	PaymentDate   time.Time
	Amount        float64
	Method        string
	Reference     *string
	Remarks       *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PaymentRepository defines the persistence contract for payments.
type PaymentRepository interface {
	Save(ctx context.Context, p *aggregate.PaymentAggregate) error
	Find(ctx context.Context, id aggregate.PaymentID, tenantID shared.TenantID) (*aggregate.PaymentAggregate, error)
	FindByReference(ctx context.Context, reference string, tenantID shared.TenantID) (aggregate.PaymentID, error)
	GetReadModel(ctx context.Context, id aggregate.PaymentID, tenantID shared.TenantID) (PaymentReadModel, error)
	GetPaymentsByInvoice(ctx context.Context, invoiceID string, tenantID shared.TenantID) ([]PaymentReadModel, error)
	SearchReadModels(ctx context.Context, tenantID shared.TenantID, method string, limit int, offset int) ([]PaymentReadModel, int64, error)
}
