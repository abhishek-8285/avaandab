package customer_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/customer"
	"transport-app/internal/domain/types"
)

func TestCustomer_Validate(t *testing.T) {
	company := "Tata Logistics"
	email := "contact@tata.com"
	gst := "27AAAAA0000A1Z5"
	contact := "Rajesh Sharma"

	c := customer.Customer{
		ID:               types.CustomerID("cust-1"),
		CustomerCode:     "CUST-001",
		Name:             "Ratan Tata",
		Company:          &company,
		ContactPerson:    &contact,
		Phone:            "+91-9999988888",
		Email:            &email,
		GST:              &gst,
		PaymentTermsDays: 30,
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid customer, got %v", err)
	}

	// Missing customer code
	cInvalidCode := c
	cInvalidCode.CustomerCode = ""
	if err := cInvalidCode.Validate(); err != customer.ErrInvalidCustomerCode {
		t.Fatalf("expected ErrInvalidCustomerCode, got %v", err)
	}

	// Missing phone
	cInvalidPhone := c
	cInvalidPhone.Phone = ""
	if err := cInvalidPhone.Validate(); err != customer.ErrInvalidPhone {
		t.Fatalf("expected ErrInvalidPhone, got %v", err)
	}

	// Missing name & company
	cInvalidCompany := c
	cInvalidCompany.Name = ""
	cInvalidCompany.Company = nil
	if err := cInvalidCompany.Validate(); err != customer.ErrInvalidCompanyName {
		t.Fatalf("expected ErrInvalidCompanyName, got %v", err)
	}
}
