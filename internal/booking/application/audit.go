package application

import (
	"context"

	"github.com/google/uuid"

	"transport-app/internal/auth"
	"transport-app/internal/domain/audit"
	"transport-app/internal/domain/types"
	"transport-app/internal/shared/ports"
)

// AuditAction constants for booking operations.
const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionConfirm  = "confirm"
	ActionComplete = "complete"
	ActionCancel   = "cancel"
	ActionDelete   = "delete"
)

// BookingTableName is the audit log table identifier for bookings.
const BookingTableName = "bookings"

// logAudit writes an audit log entry within the transaction context.
// It retrieves the AuditLogRepository from the transaction's repository provider.
func logAudit(txCtx ports.TxContext, action, recordID string, oldValues, newValues *string) {
	auditRepo, ok := txCtx.Repositories().AuditLogs().(audit.AuditLogRepository)
	if !ok {
		return
	}
	_, _ = auditRepo.CreateAuditLog(txCtx, audit.AuditLog{
		ID:        types.FileID(uuid.NewString()),
		UserID:    getUserID(txCtx),
		Action:    action,
		TableName: BookingTableName,
		RecordID:  &recordID,
		OldValues: oldValues,
		NewValues: newValues,
		IPAddress: getUserIP(txCtx),
	})
}

// getUserID extracts the user ID from the session context, if present.
func getUserID(ctx context.Context) *types.UserID {
	session, ok := ctx.Value(auth.ContextUser).(*auth.SessionData)
	if !ok || session == nil || session.UserID == "" {
		return nil
	}
	uid := types.UserID(session.UserID)
	return &uid
}

// getUserIP extracts the user IP from the request context, if present.
func getUserIP(ctx context.Context) *string {
	if ip, ok := ctx.Value(auth.ContextIP).(string); ok && ip != "" {
		return &ip
	}
	return nil
}
