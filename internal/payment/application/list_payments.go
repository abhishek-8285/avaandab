package application

import (
	"context"
	"errors"

	"transport-app/internal/payment/domain"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

type ListPaymentsQuery struct {
	TenantID shared.TenantID
	Page     int
	Limit    int
	Method   string
}

type ListPaymentsResponse struct {
	Payments []PaymentResponseDTO
	Total    int64
}

type ListPaymentsUseCase struct {
	uow ports.UnitOfWork
}

func NewListPaymentsUseCase(uow ports.UnitOfWork) *ListPaymentsUseCase {
	return &ListPaymentsUseCase{uow: uow}
}

func (uc *ListPaymentsUseCase) Execute(ctx context.Context, q ListPaymentsQuery) (ListPaymentsResponse, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	offset := (q.Page - 1) * q.Limit

	var res ListPaymentsResponse

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Payments().(domain.PaymentRepository)
		if !ok {
			return errors.New("failed to retrieve payment repository")
		}

		rows, total, err := repo.SearchReadModels(txCtx, q.TenantID, q.Method, q.Limit, offset)
		if err != nil {
			return err
		}

		dtos := make([]PaymentResponseDTO, len(rows))
		for i, p := range rows {
			dtos[i] = PaymentResponseDTO{
				ID:          p.ID,
				InvoiceID:   p.InvoiceID,
				PaymentDate: p.PaymentDate,
				Amount:      p.Amount,
				Method:      p.Method,
				Reference:   p.Reference,
				Remarks:     p.Remarks,
				CreatedAt:   p.CreatedAt,
				UpdatedAt:   p.UpdatedAt,
			}
		}

		res = ListPaymentsResponse{
			Payments: dtos,
			Total:    total,
		}
		return nil
	})

	return res, err
}
