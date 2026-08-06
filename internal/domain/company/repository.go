package company

import "context"

// CompanySettingsRepository defines the interface for company settings persistence.
type CompanySettingsRepository interface {
	GetCompanySettings(ctx context.Context) (CompanySettings, error)
	UpdateCompanySettings(ctx context.Context, settings CompanySettings) (CompanySettings, error)
}
