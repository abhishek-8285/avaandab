package viewmodels

import (
	"fmt"
	"time"

	"transport-app/internal/payment/application"
)

type PaymentSummaryViewModel struct {
	ID              string
	InvoiceNumber   string
	PaymentDate     time.Time
	Amount          float64
	FormattedAmount string
	Method          string
	MethodBadge     string
	Reference       *string
	CreatedAt       time.Time
}

func FromDTO(dto application.PaymentResponseDTO) PaymentSummaryViewModel {
	return PaymentSummaryViewModel{
		ID:              dto.ID,
		InvoiceNumber:   dto.InvoiceNumber,
		PaymentDate:     dto.PaymentDate,
		Amount:          dto.Amount,
		FormattedAmount: fmt.Sprintf("₹%.2f", dto.Amount),
		Method:          dto.Method,
		MethodBadge:     methodBadge(dto.Method),
		Reference:       dto.Reference,
		CreatedAt:       dto.CreatedAt,
	}
}

func FromDTOs(dtos []application.PaymentResponseDTO) []PaymentSummaryViewModel {
	out := make([]PaymentSummaryViewModel, len(dtos))
	for i, dto := range dtos {
		out[i] = FromDTO(dto)
	}
	return out
}

func methodBadge(method string) string {
	switch method {
	case "cash":
		return "secondary"
	case "upi":
		return "info"
	case "bank_transfer":
		return "primary"
	case "cheque":
		return "warning"
	default:
		return "secondary"
	}
}
