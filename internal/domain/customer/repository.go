package customer

import (
	"context"

	"transport-app/internal/domain/types"
)

// CustomerRepository defines the interface for customer persistence.
type CustomerRepository interface {
	CreateCustomer(ctx context.Context, customer Customer) (Customer, error)
	GetCustomerByID(ctx context.Context, id types.CustomerID) (Customer, error)
	GetCustomerByPhone(ctx context.Context, phone string) (Customer, error)
	UpdateCustomer(ctx context.Context, customer Customer) (Customer, error)
	DeleteCustomer(ctx context.Context, id types.CustomerID) error
	SearchCustomers(ctx context.Context, query string, limit, offset int) ([]Customer, error)
	CountCustomers(ctx context.Context, query string) (int64, error)
}
