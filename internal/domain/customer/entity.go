package customer

import (
	"errors"
	"strings"
	"time"

	"transport-app/internal/domain/types"
)

var (
	ErrInvalidCustomerCode = errors.New("customer code is required")
	ErrInvalidCompanyName  = errors.New("company name is required")
	ErrInvalidPhone        = errors.New("phone number is required")
)

// Customer represents a corporate shipper customer who books transport services.
type Customer struct {
	ID               types.CustomerID
	CustomerCode     string
	Name             string
	Company          *string
	ContactPerson    *string
	Phone            string
	Email            *string
	GST              *string
	Address          *string
	BillingAddress   *string
	PaymentTermsDays int
	Status           string
	Notes            *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate validates the required customer fields.
func (c Customer) Validate() error {
	if strings.TrimSpace(c.CustomerCode) == "" {
		return ErrInvalidCustomerCode
	}
	if strings.TrimSpace(c.Name) == "" && (c.Company == nil || strings.TrimSpace(*c.Company) == "") {
		return ErrInvalidCompanyName
	}
	if strings.TrimSpace(c.Phone) == "" {
		return ErrInvalidPhone
	}
	return nil
}
