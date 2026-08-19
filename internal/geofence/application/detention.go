// Detention admin use cases: manual invoice attach and waiver (Spec 02 §6).
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"transport-app/internal/auth"
	"transport-app/internal/domain/audit"
	"transport-app/internal/domain/types"
	"transport-app/internal/geofence/domain"
	invoiceApp "transport-app/internal/invoice/application"
	invoiceaggregate "transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// DetentionTableName is the audit log table identifier for detentions.
const DetentionTableName = "trip_detentions"

// Action constants for detention admin operations.
const (
	ActionAttachDetention = "attach_detention"
	ActionWaiveDetention  = "waive_detention"
)

// ErrInvoicePaid is returned when auto/manual attach targets a paid invoice.
// Mapped to HTTP 409 by the API handler.
var ErrInvoicePaid = errors.New("invoice for this trip is already paid")

// AttachDetentionCommand targets one closed detention.
type AttachDetentionCommand struct {
	DetentionID string
	TenantID    shared.TenantID
}

// AttachDetentionUseCase attaches a closed detention as an invoice line item.
// Trips without bookings get a detention-only invoice (booking_id empty).
type AttachDetentionUseCase struct {
	uow   ports.UnitOfWork
	logs  domain.EventLogRepository
	invUC *invoiceApp.GenerateInvoiceUseCase
}

// NewAttachDetentionUseCase constructs an AttachDetentionUseCase.
func NewAttachDetentionUseCase(uow ports.UnitOfWork, logs domain.EventLogRepository, invUC *invoiceApp.GenerateInvoiceUseCase) *AttachDetentionUseCase {
	return &AttachDetentionUseCase{uow: uow, logs: logs, invUC: invUC}
}

// Execute attaches the detention line in one transaction with the invoice
// update and the detention status flip.
func (uc *AttachDetentionUseCase) Execute(ctx context.Context, cmd AttachDetentionCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		d, err := uc.logs.Find(txCtx, string(cmd.TenantID), cmd.DetentionID)
		if err != nil {
			return err
		}
		if d.Status != domain.DetentionClosed {
			return errors.New("only closed detentions can be attached")
		}
		if d.Amount <= 0 {
			return errors.New("detention has no billable amount")
		}

		tripID := d.TripID
		description := "Detention"
		if d.ZoneName != "" {
			description = "Detention at " + d.ZoneName
		}

		_, attached, err := uc.invUC.GenerateInTx(txCtx, invoiceApp.GenerateInvoiceCommand{
			TenantID: cmd.TenantID,
			TripID:   &tripID,
			LineItems: []invoiceApp.InvoiceLineItemInput{{
				TripID:      &tripID,
				LineType:    invoiceaggregate.LineTypeDetention,
				Description: description,
				Quantity:    float64(d.BillableSeconds) / 3600.0,
				UnitPrice:   d.RatePerHour,
				RefID:       &d.ID,
			}},
		})
		if err != nil {
			return err
		}
		if !attached {
			return ErrInvoicePaid
		}

		if err := uc.logs.MarkAttached(txCtx, d.ID); err != nil {
			return err
		}
		logGeofenceAudit(txCtx, ActionAttachDetention, d.ID, nil, nil)
		return nil
	})
}

// WaiveDetentionCommand targets one detention to waive.
type WaiveDetentionCommand struct {
	DetentionID string
	TenantID    shared.TenantID
}

// WaiveDetentionUseCase zeroes the amount and marks the detention waived.
type WaiveDetentionUseCase struct {
	uow  ports.UnitOfWork
	logs domain.EventLogRepository
}

// NewWaiveDetentionUseCase constructs a WaiveDetentionUseCase.
func NewWaiveDetentionUseCase(uow ports.UnitOfWork, logs domain.EventLogRepository) *WaiveDetentionUseCase {
	return &WaiveDetentionUseCase{uow: uow, logs: logs}
}

// Execute waives a detention (status=waived, amount=0).
func (uc *WaiveDetentionUseCase) Execute(ctx context.Context, cmd WaiveDetentionCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		d, err := uc.logs.Find(txCtx, string(cmd.TenantID), cmd.DetentionID)
		if err != nil {
			return err
		}
		if d.Status == domain.DetentionWaived || d.Status == domain.DetentionInvoiced {
			return errors.New("detention already resolved")
		}
		if err := uc.logs.Waive(txCtx, d.ID); err != nil {
			return err
		}
		logGeofenceAudit(txCtx, ActionWaiveDetention, d.ID, nil, nil)
		return nil
	})
}

// logGeofenceAudit writes an audit log entry within the transaction context.
func logGeofenceAudit(txCtx ports.TxContext, action, recordID string, oldValues, newValues *string) {
	auditRepo, ok := txCtx.Repositories().AuditLogs().(audit.AuditLogRepository)
	if !ok {
		return
	}
	_, _ = auditRepo.CreateAuditLog(txCtx, audit.AuditLog{
		ID:        types.FileID(uuid.NewString()),
		UserID:    getUserID(txCtx),
		Action:    action,
		TableName: DetentionTableName,
		RecordID:  &recordID,
		OldValues: oldValues,
		NewValues: newValues,
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
