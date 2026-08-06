package service

import (
	"context"
	"fmt"

	"transport-app/internal/domain"
)

// CustomerService handles customer management.
type CustomerService struct {
	baseService
}

// CreateCustomer creates a new customer.
func (s *CustomerService) CreateCustomer(ctx context.Context, name, company, phone, email, gst, address, notes string) (domain.Customer, error) {
	if name == "" || phone == "" {
		return domain.Customer{}, fmt.Errorf("name and phone are required")
	}

	// Check phone uniqueness
	if _, err := s.store.GetCustomerByPhone(ctx, phone); err == nil {
		return domain.Customer{}, fmt.Errorf("customer with phone number %s already exists", phone)
	}

	customer := domain.Customer{
		ID:      domain.CustomerID(generateID()),
		Name:    name,
		Company: strPtr(company),
		Phone:   phone,
		Email:   strPtr(email),
		GST:     strPtr(gst),
		Address: strPtr(address),
		Notes:   strPtr(notes),
	}

	created, err := s.store.CreateCustomer(ctx, customer)
	if err != nil {
		return domain.Customer{}, err
	}

	s.log.Info("customer created", "customer_id", created.ID)
	return created, nil
}

// GetCustomer retrieves a customer by ID.
func (s *CustomerService) GetCustomer(ctx context.Context, id domain.CustomerID) (domain.Customer, error) {
	return s.store.GetCustomerByID(ctx, id)
}

// ListCustomers retrieves customers with search and pagination.
func (s *CustomerService) ListCustomers(ctx context.Context, query string, limit, offset int) ([]domain.Customer, int64, error) {
	customers, err := s.store.SearchCustomers(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountCustomers(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return customers, total, nil
}

// UpdateCustomer updates an existing customer.
func (s *CustomerService) UpdateCustomer(ctx context.Context, id domain.CustomerID, name, company, phone, email, gst, address, notes string) (domain.Customer, error) {
	customer, err := s.store.GetCustomerByID(ctx, id)
	if err != nil {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}

	// Check phone uniqueness for other customers
	if existing, err := s.store.GetCustomerByPhone(ctx, phone); err == nil && existing.ID != id {
		return domain.Customer{}, fmt.Errorf("customer with phone number %s already exists", phone)
	}

	customer.Name = name
	customer.Company = strPtr(company)
	customer.Phone = phone
	customer.Email = strPtr(email)
	customer.GST = strPtr(gst)
	customer.Address = strPtr(address)
	customer.Notes = strPtr(notes)

	updated, err := s.store.UpdateCustomer(ctx, customer)
	if err != nil {
		return domain.Customer{}, err
	}

	s.log.Info("customer updated", "customer_id", id)
	return updated, nil
}

// DeleteCustomer deletes a customer.
func (s *CustomerService) DeleteCustomer(ctx context.Context, id domain.CustomerID) error {
	if err := s.store.DeleteCustomer(ctx, id); err != nil {
		return err
	}
	s.log.Info("customer deleted", "customer_id", id)
	return nil
}
