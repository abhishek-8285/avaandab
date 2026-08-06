package company

import "time"

// CompanySettingsUpdated is emitted when company settings are updated.
type CompanySettingsUpdated struct {
	SettingsID  int64
	CompanyName string
	OccurredAt  time.Time
}
