package audit

import (
	"time"

	"transport-app/internal/domain/types"
)

// AuditLogCreated is emitted when an audit log entry is created.
type AuditLogCreated struct {
	ID        types.FileID
	UserID    *types.UserID
	Action    string
	TableName string
	OccurredAt time.Time
}
