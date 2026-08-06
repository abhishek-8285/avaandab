package application

import (
	"context"
	"errors"
	"time"

	"transport-app/internal/payment/domain"
	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// RecordPaymentCommand contains parameters to register a payment.
type RecordPaymentCommand struct {
	TenantID    shared.TenantID
	InvoiceID   string
	PaymentDate time.Time
	Amount      float64
	Method      aggregate.PaymentMethod
	Reference   *string
	Remarks     *string
}

// RecordPaymentUseCase records a payment.
type RecordPaymentUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

// NewRecordPaymentUseCase constructs a new RecordPaymentUseCase.
func NewRecordPaymentUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *RecordPaymentUseCase {
	return &RecordPaymentUseCase{uow: uow, idGen: idGen, clock: clock}
}

// Execute performs validation, creates the aggregate, and saves it.
func (uc *RecordPaymentUseCase) Execute(ctx context.Context, cmd RecordPaymentCommand) (aggregate.PaymentID, error) {
	if cmd.InvoiceID == "" {
		return "", errors.New("invoice ID is required")
	}
	if cmd.Amount <= 0 {
		return "", errors.New("payment amount must be greater than zero")
	}

	id := aggregate.PaymentID(uc.idGen.GenerateUUID())

	payment := aggregate.NewPaymentAggregate(
		id,
		cmd.TenantID,
		cmd.InvoiceID,
		cmd.PaymentDate,
		cmd.Amount,
		cmd.Method,
		cmd.Reference,
		cmd.Remarks,
		uc.clock.Now(),
	)

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Payments().(domain.PaymentRepository)
		if !ok {
			return errors.New("failed to retrieve payment repository")
		}
		return repo.Save(txCtx, payment)
	})

	if err != nil {
		return "", err
	}

	return id, nil
}
