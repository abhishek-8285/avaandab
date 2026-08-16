package viewmodels

import (
	"fmt"
	"time"

	"transport-app/internal/invoice/application"
)

type InvoiceSummaryViewModel struct {
	ID            string
	InvoiceNumber string
	BookingNumber string
	CustomerName  string
	TripNumber    string
	Total         float64
	PaidAmount    float64
	Outstanding   float64
	Status        string
	StatusBadge   string
	DueDate       *time.Time
	CreatedAt     time.Time
}

type InvoiceDetailViewModel struct {
	InvoiceSummaryViewModel
	CustomerCompany string
	BookingID       string
	CustomerID      string
	TripID          *string
	Subtotal        float64
	Tax             float64
	Discount        float64
	UpdatedAt       time.Time
}

func FromDTO(dto application.InvoiceResponseDTO) InvoiceDetailViewModel {
	return InvoiceDetailViewModel{
		InvoiceSummaryViewModel: InvoiceSummaryViewModel{
			ID:            dto.ID,
			InvoiceNumber: dto.InvoiceNumber,
			BookingNumber: dto.BookingNumber,
			CustomerName:  dto.CustomerName,
			TripNumber:    dto.TripNumber,
			Total:         dto.Total,
			Status:        dto.PaymentStatus,
			StatusBadge:   statusBadge(dto.PaymentStatus),
			CreatedAt:     dto.CreatedAt,
		},
		CustomerCompany: dto.CustomerCompany,
		BookingID:       dto.BookingID,
		CustomerID:      dto.CustomerID,
		TripID:          dto.TripID,
		Subtotal:        dto.Subtotal,
		Tax:             dto.Tax,
		Discount:        dto.Discount,
		UpdatedAt:       dto.UpdatedAt,
	}
}

func FromDTOs(dtos []application.InvoiceResponseDTO) []InvoiceSummaryViewModel {
	result := make([]InvoiceSummaryViewModel, len(dtos))
	for i, dto := range dtos {
		result[i] = InvoiceSummaryViewModel{
			ID:            dto.ID,
			InvoiceNumber: dto.InvoiceNumber,
			BookingNumber: dto.BookingNumber,
			CustomerName:  dto.CustomerName,
			TripNumber:    dto.TripNumber,
			Total:         dto.Total,
			Status:        dto.PaymentStatus,
			StatusBadge:   statusBadge(dto.PaymentStatus),
			CreatedAt:     dto.CreatedAt,
		}
	}
	return result
}

func statusBadge(status string) string {
	switch status {
	case "paid":
		return "badge-success"
	case "partially_paid":
		return "badge-warning"
	case "pending":
		return "badge-danger"
	default:
		return "badge-secondary"
	}
}

func (v InvoiceSummaryViewModel) FormattedTotal() string {
	return fmt.Sprintf("₹%.2f", v.Total)
}

func (v InvoiceSummaryViewModel) FormattedOutstanding() string {
	return fmt.Sprintf("₹%.2f", v.Outstanding)
}
