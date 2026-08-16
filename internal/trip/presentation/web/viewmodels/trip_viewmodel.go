package viewmodels

import (
	"fmt"
	"strings"
	"time"

	"transport-app/internal/trip/application"
)

type TripSummaryViewModel struct {
	ID            string
	TripNumber    string
	DriverName    string
	VehicleNumber string
	Route         string
	Status        string
	StatusBadge   string
	DepartureTime time.Time
	ArrivalTime   *time.Time
}

type TripDetailViewModel struct {
	TripSummaryViewModel
	BookingID        *string
	DriverID         *string
	VehicleID        *string
	RouteID          string
	RouteSource      string
	RouteDestination string
	Remarks          string
	StartedAt        *time.Time
	ReachedPickupAt  *time.Time
	InTransitAt      *time.Time
	DeliveredAt      *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func FromDTO(dto application.TripResponseDTO) TripDetailViewModel {
	return TripDetailViewModel{
		TripSummaryViewModel: TripSummaryViewModel{
			ID:            dto.ID,
			TripNumber:    dto.TripNumber,
			DriverName:    strings.TrimSpace(fmt.Sprintf("%s %s", dto.DriverFirstName, dto.DriverLastName)),
			VehicleNumber: dto.VehicleNumber,
			Route:         fmt.Sprintf("%s → %s", dto.RouteSource, dto.RouteDestination),
			Status:        dto.Status,
			StatusBadge:   statusBadge(dto.Status),
			DepartureTime: dto.DepartureTime,
			ArrivalTime:   dto.ArrivalTime,
		},
		BookingID:        dto.BookingID,
		DriverID:         dto.DriverID,
		VehicleID:        dto.VehicleID,
		RouteID:          dto.RouteID,
		RouteSource:      dto.RouteSource,
		RouteDestination: dto.RouteDestination,
		Remarks:          dto.Remarks,
		StartedAt:        dto.StartedAt,
		ReachedPickupAt:  dto.ReachedPickupAt,
		InTransitAt:      dto.InTransitAt,
		DeliveredAt:      dto.DeliveredAt,
		CompletedAt:      dto.CompletedAt,
		CreatedAt:        dto.CreatedAt,
		UpdatedAt:        dto.UpdatedAt,
	}
}

func FromDTOs(dtos []application.TripResponseDTO) []TripSummaryViewModel {
	out := make([]TripSummaryViewModel, len(dtos))
	for i, dto := range dtos {
		d := FromDTO(dto)
		out[i] = d.TripSummaryViewModel
	}
	return out
}

func statusBadge(status string) string {
	switch status {
	case "scheduled":
		return "info"
	case "assigned":
		return "warning"
	case "started", "in_transit":
		return "primary"
	case "delivered", "completed":
		return "success"
	case "cancelled":
		return "danger"
	default:
		return "secondary"
	}
}

func (t TripSummaryViewModel) FormattedDeparture() string {
	return t.DepartureTime.Format("2006-01-02 15:04")
}
