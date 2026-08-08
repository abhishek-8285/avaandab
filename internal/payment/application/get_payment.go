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

type PaymentResponseDTO struct {
	ID            string    `json:"id"`
	InvoiceID     string    `json:"invoice_id"`
	InvoiceNumber string    `json:"invoice_number"`
	PaymentDate   time.Time `json:"payment_date"`
	Amount        float64   `json:"amount"`
	Method        string    `json:"method"`
	Reference     *string   `json:"reference"`
	Remarks       *string   `json:"remarks"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GetPaymentQuery struct {
	ID       aggregate.PaymentID
	TenantID shared.TenantID
}

type GetPaymentUseCase struct {
	uow ports.UnitOfWork
}

func NewGetPaymentUseCase(uow ports.UnitOfWork) *GetPaymentUseCase {
	return &GetPaymentUseCase{uow: uow}
}

func (uc *GetPaymentUseCase) Execute(ctx context.Context, q GetPaymentQuery) (PaymentResponseDTO, error) {
	var dto PaymentResponseDTO
	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Payments().(domain.PaymentRepository)
		if !ok {
			return errors.New("failed to retrieve payment repository")
		}

		p, err := repo.GetReadModel(txCtx, q.ID, q.TenantID)
		if err != nil {
			return err
		}

		dto = PaymentResponseDTO{
			ID:            p.ID,
			InvoiceID:     p.InvoiceID,
			InvoiceNumber: p.InvoiceNumber,
			PaymentDate:   p.PaymentDate,
			Amount:        p.Amount,
			Method:        p.Method,
			Reference:     p.Reference,
			Remarks:       p.Remarks,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}
		return nil
	})
	return dto, err
}
