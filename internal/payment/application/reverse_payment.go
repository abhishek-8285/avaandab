package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	invoiceDomain "transport-app/internal/invoice/domain"
	invoiceAgg "transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/payment/domain"
	paymentagg "transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

var ErrPaymentNotFound = errors.New("payment not found")

type ReversePaymentCommand struct {
	TenantID      shared.TenantID
	OriginalPayID paymentagg.PaymentID
	Reason        string
}

type ReversePaymentUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

func NewReversePaymentUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *ReversePaymentUseCase {
	return &ReversePaymentUseCase{uow: uow, idGen: idGen, clock: clock}
}

func (uc *ReversePaymentUseCase) Execute(ctx context.Context, cmd ReversePaymentCommand) (paymentagg.PaymentID, error) {
	if cmd.OriginalPayID == "" {
		return "", errors.New("original payment ID is required")
	}
	if cmd.Reason == "" {
		return "", errors.New("reversal reason is required")
	}

	var id paymentagg.PaymentID

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		payRepo, ok := txCtx.Repositories().Payments().(domain.PaymentRepository)
		if !ok {
			return errors.New("failed to retrieve payment repository")
		}

		original, err := payRepo.Find(txCtx, cmd.OriginalPayID, cmd.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrPaymentNotFound
			}
			return err
		}
		if original == nil {
			return ErrPaymentNotFound
		}

		reversalRef := fmt.Sprintf("REVERSAL:%s", string(original.ID))
		reversal := paymentagg.NewPaymentAggregate(
			paymentagg.PaymentID(uc.idGen.GenerateUUID()),
			cmd.TenantID,
			original.InvoiceID,
			uc.clock.Now(),
			-original.Amount,
			original.Method,
			&reversalRef,
			&cmd.Reason,
			uc.clock.Now(),
		)

		invoiceRepo, ok := txCtx.Repositories().Invoices().(invoiceDomain.InvoiceRepository)
		if !ok {
			return errors.New("failed to retrieve invoice repository")
		}

		inv, err := invoiceRepo.Find(txCtx, invoiceAgg.InvoiceID(original.InvoiceID), cmd.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvoiceNotFound
			}
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}

		if err := inv.ApplyPayment(-original.Amount, uc.clock.Now()); err != nil {
			return err
		}

		if err := invoiceRepo.Save(txCtx, inv); err != nil {
			return err
		}

		if err := payRepo.Save(txCtx, reversal); err != nil {
			return err
		}

		id = reversal.ID
		return nil
	})

	if err != nil {
		switch err.Error() {
		case ErrPaymentNotFound.Error():
			return "", ErrPaymentNotFound
		case ErrInvoiceNotFound.Error():
			return "", ErrInvoiceNotFound
		default:
			return "", err
		}
	}

	return id, nil
}
