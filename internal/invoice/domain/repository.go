package domain

import (
	"context"
	"time"

	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
)

// InvoiceReadModel optimized for read operations.
type InvoiceReadModel struct {
	ID            string
	InvoiceNumber string
	BookingID     string
	CustomerID    string
	TripID        *string
	Subtotal      float64
	Tax           float64
	Discount      float64
	Total         float64
	PaymentStatus string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// InvoiceRepository defines the persistence contract for invoices.
type InvoiceRepository interface {
	Save(ctx context.Context, inv *aggregate.InvoiceAggregate) error
	Find(ctx context.Context, id aggregate.InvoiceID, tenantID shared.TenantID) (*aggregate.InvoiceAggregate, error)
	FindByBookingID(ctx context.Context, bookingID string, tenantID shared.TenantID) (*aggregate.InvoiceAggregate, error)
	GetReadModel(ctx context.Context, id aggregate.InvoiceID, tenantID shared.TenantID) (InvoiceReadModel, error)
	SearchReadModels(ctx context.Context, tenantID shared.TenantID, query string, status string, limit int, offset int) ([]InvoiceReadModel, int64, error)
}
