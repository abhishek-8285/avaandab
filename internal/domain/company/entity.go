package company

import "time"

// CompanySettings holds global company configuration.
type CompanySettings struct {
	ID            int64
	CompanyName   string
	LogoPath      *string
	Currency      string
	Timezone      string
	GSTEnabled    bool
	GSTRate       float64
	BookingPrefix string
	TripPrefix    string
	InvoicePrefix string
	Address       *string
	Phone         *string
	Email         *string
	GSTNumber     *string
	StateCode     string
	FinancialYear *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
