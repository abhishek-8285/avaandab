package service

import (
	"context"

	"transport-app/internal/domain"
)

// CompanySettingsService handles company configuration.
type CompanySettingsService struct {
	baseService
}

// GetSettings returns the company settings.
func (s *CompanySettingsService) GetSettings(ctx context.Context) (domain.CompanySettings, error) {
	return s.store.GetCompanySettings(ctx)
}

// UpdateSettings updates the company settings.
func (s *CompanySettingsService) UpdateSettings(ctx context.Context, companyName, currency, timezone string, gstEnabled bool, gstRate float64, bookingPrefix, tripPrefix, invoicePrefix, financialYear string, address, phone, email, gstNumber string, logoPath *string) (domain.CompanySettings, error) {
	settings := domain.CompanySettings{
		CompanyName:   companyName,
		Currency:      currency,
		Timezone:      timezone,
		GSTEnabled:    gstEnabled,
		GSTRate:       gstRate,
		BookingPrefix: bookingPrefix,
		TripPrefix:    tripPrefix,
		InvoicePrefix: invoicePrefix,
		FinancialYear: strPtr(financialYear),
		Address:       &address,
		Phone:         &phone,
		Email:         &email,
		GSTNumber:     &gstNumber,
		LogoPath:      logoPath,
	}

	updated, err := s.store.UpdateCompanySettings(ctx, settings)
	if err != nil {
		return domain.CompanySettings{}, err
	}

	s.log.Info("company settings updated")
	return updated, nil
}
