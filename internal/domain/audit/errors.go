package audit

import "errors"

// Audit-specific domain errors.
var (
	ErrAuditLogNotFound = errors.New("audit log not found")
	ErrFileNotFound     = errors.New("file not found")
)
