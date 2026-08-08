package dto

import (
	"time"

	"transport-app/internal/payment/domain/aggregate"
)

// PaymentDTO represents the JSON body returned by payment API endpoints.
type PaymentDTO struct {
	ID          string                  `json:"id"`
	InvoiceID   string                  `json:"invoice_id"`
	PaymentDate time.Time               `json:"payment_date"`
	Amount      float64                 `json:"amount"`
	Method      aggregate.PaymentMethod `json:"method"`
	Reference   *string                 `json:"reference"`
	Remarks     *string                 `json:"remarks"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}
