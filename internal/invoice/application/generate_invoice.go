package application

import (
	"context"
	"errors"

	"transport-app/internal/invoice/domain"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// GenerateInvoiceCommand contains parameters to create a new invoice.
type GenerateInvoiceCommand struct {
	TenantID   shared.TenantID
	BookingID  string
	CustomerID string
	TripID     *string
	Subtotal   float64
	Tax        float64
	Discount   float64
	Total      float64
}

// GenerateInvoiceUseCase generates an invoice aggregate.
type GenerateInvoiceUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

// NewGenerateInvoiceUseCase constructs a new GenerateInvoiceUseCase.
func NewGenerateInvoiceUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *GenerateInvoiceUseCase {
	return &GenerateInvoiceUseCase{uow: uow, idGen: idGen, clock: clock}
}

// Execute performs the generation and transaction commit.
func (uc *GenerateInvoiceUseCase) Execute(ctx context.Context, cmd GenerateInvoiceCommand) (aggregate.InvoiceID, error) {
	if cmd.BookingID == "" {
		return "", errors.New("booking ID is required")
	}
	if cmd.CustomerID == "" {
		return "", errors.New("customer ID is required")
	}
	if cmd.Total < 0 {
		return "", errors.New("total cannot be negative")
	}

	var existingID aggregate.InvoiceID

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Invoices().(domain.InvoiceRepository)
		if !ok {
			return errors.New("failed to retrieve invoice repository")
		}

		existing, err := repo.FindByBookingID(txCtx, cmd.BookingID, cmd.TenantID)
		if err == nil && existing != nil {
			existingID = existing.ID
			return nil
		}

		id := aggregate.InvoiceID(uc.idGen.GenerateUUID())
		num := uc.idGen.GenerateDisplayID("INV")

		inv := aggregate.NewInvoiceAggregate(
			id,
			cmd.TenantID,
			num,
			cmd.BookingID,
			cmd.CustomerID,
			cmd.TripID,
			cmd.Subtotal,
			cmd.Tax,
			cmd.Discount,
			cmd.Total,
			aggregate.PaymentStatusPending,
			uc.clock.Now(),
		)

		if err := repo.Save(txCtx, inv); err != nil {
			return err
		}
		existingID = id
		return nil
	})

	if err != nil {
		return "", err
	}

	return existingID, nil
}
