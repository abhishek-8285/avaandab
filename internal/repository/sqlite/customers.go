package sqlite

import (
	"context"
	"database/sql"

	"transport-app/internal/domain"

	db "transport-app/db/generated/sqlite"
)

// CustomerRepository implementation

func (r *SQLRepository) CreateCustomer(ctx context.Context, customer domain.Customer) (domain.Customer, error) {
	created, err := r.Q(ctx).CreateCustomer(ctx, db.CreateCustomerParams{
		ID:      string(customer.ID),
		Name:    customer.Name,
		Company: nullString(customer.Company),
		Phone:   customer.Phone,
		Email:   nullString(customer.Email),
		Gst:     nullString(customer.GST),
		Address: nullString(customer.Address),
		Notes:   nullString(customer.Notes),
	})
	if err != nil {
		return domain.Customer{}, err
	}
	return toDomainCustomer(created), nil
}

func (r *SQLRepository) GetCustomerByID(ctx context.Context, id domain.CustomerID) (domain.Customer, error) {
	c, err := r.Q(ctx).GetCustomerByID(ctx, string(id))
	if err != nil {
		return domain.Customer{}, err
	}
	return toDomainCustomer(c), nil
}

func (r *SQLRepository) GetCustomerByPhone(ctx context.Context, phone string) (domain.Customer, error) {
	c, err := r.Q(ctx).GetCustomerByPhone(ctx, phone)
	if err != nil {
		return domain.Customer{}, err
	}
	return toDomainCustomer(c), nil
}

func (r *SQLRepository) UpdateCustomer(ctx context.Context, customer domain.Customer) (domain.Customer, error) {
	updated, err := r.Q(ctx).UpdateCustomer(ctx, db.UpdateCustomerParams{
		Name:    customer.Name,
		Company: nullString(customer.Company),
		Phone:   customer.Phone,
		Email:   nullString(customer.Email),
		Gst:     nullString(customer.GST),
		Address: nullString(customer.Address),
		Notes:   nullString(customer.Notes),
		ID:      string(customer.ID),
	})
	if err != nil {
		return domain.Customer{}, err
	}
	return toDomainCustomer(updated), nil
}

func (r *SQLRepository) DeleteCustomer(ctx context.Context, id domain.CustomerID) error {
	return r.Q(ctx).DeleteCustomer(ctx, string(id))
}

func (r *SQLRepository) SearchCustomers(ctx context.Context, query string, limit, offset int) ([]domain.Customer, error) {
	rows, err := r.Q(ctx).SearchCustomers(ctx, db.SearchCustomersParams{
		Column1: sql.NullString{String: query, Valid: true},
		Column2: sql.NullString{String: query, Valid: true},
		Column3: sql.NullString{String: query, Valid: true},
		Column4: sql.NullString{String: query, Valid: true},
		Limit:   int64(limit),
		Offset:  int64(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Customer, len(rows))
	for i, c := range rows {
		result[i] = toDomainCustomer(c)
	}
	return result, nil
}

func (r *SQLRepository) CountCustomers(ctx context.Context, query string) (int64, error) {
	count, err := r.Q(ctx).CountCustomers(ctx, db.CountCustomersParams{
		Column1: sql.NullString{String: query, Valid: true},
		Column2: sql.NullString{String: query, Valid: true},
		Column3: sql.NullString{String: query, Valid: true},
		Column4: sql.NullString{String: query, Valid: true},
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
