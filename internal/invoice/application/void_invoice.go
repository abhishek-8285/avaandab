package application

import (
	"context"
	"database/sql"
	"errors"

	"transport-app/internal/invoice/domain"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

var ErrInvoiceAlreadyCancelled = errors.New("invoice already cancelled")
var ErrInvoiceCannotVoid = errors.New("invoice cannot be voided")

type VoidInvoiceCommand struct {
	TenantID shared.TenantID
	ID       aggregate.InvoiceID
}

type VoidInvoiceUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

func NewVoidInvoiceUseCase(uow ports.UnitOfWork, clock ports.Clock) *VoidInvoiceUseCase {
	return &VoidInvoiceUseCase{uow: uow, clock: clock}
}

func (uc *VoidInvoiceUseCase) Execute(ctx context.Context, cmd VoidInvoiceCommand) error {
	if cmd.ID == "" {
		return errors.New("invoice ID is required")
	}
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Invoices().(domain.InvoiceRepository)
		if !ok {
			return errors.New("failed to retrieve invoice repository")
		}
		inv, err := repo.Find(txCtx, cmd.ID, cmd.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("invoice not found")
			}
			return err
		}
		if err := inv.Void(uc.clock.Now()); err != nil {
			if err.Error() == "invoice already cancelled" {
				return ErrInvoiceAlreadyCancelled
			}
			return ErrInvoiceCannotVoid
		}
		return repo.Save(txCtx, inv)
	})
}
