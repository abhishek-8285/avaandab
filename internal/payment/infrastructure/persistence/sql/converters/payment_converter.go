package converters

import (
	"database/sql"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/payment/domain"
	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
)

// ToDomain converts db.Payment to *aggregate.PaymentAggregate.
func ToDomain(p db.Payment) *aggregate.PaymentAggregate {
	return aggregate.NewPaymentAggregate(
		aggregate.PaymentID(p.ID),
		shared.TenantID(p.TenantID),
		p.InvoiceID,
		p.PaymentDate,
		p.Amount,
		aggregate.PaymentMethod(p.Method),
		getStringPointer(p.Reference),
		getStringPointer(p.Remarks),
		p.CreatedAt,
	)
}

// ToReadModel converts db.Payment to domain.PaymentReadModel.
func ToReadModel(p db.Payment) domain.PaymentReadModel {
	return domain.PaymentReadModel{
		ID:          p.ID,
		InvoiceID:   p.InvoiceID,
		PaymentDate: p.PaymentDate,
		Amount:      p.Amount,
		Method:      p.Method,
		Reference:   getStringPointer(p.Reference),
		Remarks:     getStringPointer(p.Remarks),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func getStringPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}
