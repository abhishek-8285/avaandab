package customer

import (
	"context"

	"transport-app/internal/domain/types"
)

// CustomerService defines the interface for customer business operations.
type CustomerService interface {
	CreateCustomer(ctx context.Context, name, company, phone, email, gst, address, notes string) (Customer, error)
	GetCustomer(ctx context.Context, id types.CustomerID) (Customer, error)
	ListCustomers(ctx context.Context, query string, limit, offset int) ([]Customer, int64, error)
	UpdateCustomer(ctx context.Context, id types.CustomerID, name, company, phone, email, gst, address, notes string) (Customer, error)
	DeleteCustomer(ctx context.Context, id types.CustomerID) error
}
