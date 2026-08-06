package company

import "errors"

// Company-specific domain errors.
var (
	ErrCompanySettingsNotFound = errors.New("company settings not found")
)
