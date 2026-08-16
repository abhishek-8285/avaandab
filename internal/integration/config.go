package integration

import (
	"os"
	"strconv"

	"transport-app/internal/integration/accounting"
	"transport-app/internal/integration/ewaybill"
	"transport-app/internal/integration/fastag"
	"transport-app/internal/integration/gstn"
)

// Config holds connection settings for all integration providers.
type Config struct {
	EWayBill   ewaybill.Config
	GSTN       gstn.Config
	FASTag     fastag.Config
	Accounting accounting.Config
}

// LoadConfig reads integration settings from environment variables.
func LoadConfig() Config {
	return Config{
		EWayBill: ewaybill.Config{
			Endpoint: os.Getenv("INTEGRATION_EWAYBILL_ENDPOINT"),
			APIKey:   os.Getenv("INTEGRATION_EWAYBILL_API_KEY"),
			Enabled:  parseBool(os.Getenv("INTEGRATION_EWAYBILL_ENABLED")),
		},
		GSTN: gstn.Config{
			Endpoint: os.Getenv("INTEGRATION_GSTN_ENDPOINT"),
			APIKey:   os.Getenv("INTEGRATION_GSTN_API_KEY"),
			Enabled:  parseBool(os.Getenv("INTEGRATION_GSTN_ENABLED")),
		},
		FASTag: fastag.Config{
			Endpoint: os.Getenv("INTEGRATION_FASTAG_ENDPOINT"),
			APIKey:   os.Getenv("INTEGRATION_FASTAG_API_KEY"),
			Enabled:  parseBool(os.Getenv("INTEGRATION_FASTAG_ENABLED")),
		},
		Accounting: accounting.Config{
			Endpoint: os.Getenv("INTEGRATION_ACCOUNTING_ENDPOINT"),
			APIKey:   os.Getenv("INTEGRATION_ACCOUNTING_API_KEY"),
			Enabled:  parseBool(os.Getenv("INTEGRATION_ACCOUNTING_ENABLED")),
		},
	}
}

func parseBool(v string) bool {
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
