package audit

import (
	"context"

	"transport-app/internal/domain/types"
)

// AuditLogService defines the interface for audit log operations.
type AuditLogService interface {
	CreateLog(ctx context.Context, action string, table string, recordID *string, oldValues, newValues *string, ip *string) (AuditLog, error)
	ListLogs(ctx context.Context, limit, offset int) ([]AuditLogWithUser, int64, error)
}

// FileService defines the interface for file management operations.
type FileService interface {
	CreateFile(ctx context.Context, file types.File) (types.File, error)
	GetFile(ctx context.Context, id types.FileID) (types.File, error)
	GetFilesByUploadable(ctx context.Context, uploadableType, uploadableID string) ([]types.File, error)
	DeleteFile(ctx context.Context, id types.FileID) error
	DeleteFilesByUploadable(ctx context.Context, uploadableType, uploadableID string) error
}
