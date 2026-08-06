package application

import (
	"context"
	"errors"
	"time"

	"transport-app/internal/booking/domain"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// BookingResponseDTO represents the read model of a booking returned to the callers.
type BookingResponseDTO struct {
	ID               string
	BookingNumber    string
	CustomerID       string
	CustomerName     string
	CustomerCompany  string
	RouteID          string
	RouteSource      string
	RouteDestination string
	PickupDate       time.Time
	VehicleType      string
	Passengers       int64
	CargoWeight      *float64
	Price            float64
	Notes            string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// GetBookingQuery parameters.
type GetBookingQuery struct {
	BookingID aggregate.BookingID
	TenantID  shared.TenantID
}

// GetBookingUseCase orchestrates retrieving a booking aggregate.
type GetBookingUseCase struct {
	uow ports.UnitOfWork
}

// NewGetBookingUseCase creates a new GetBookingUseCase.
func NewGetBookingUseCase(uow ports.UnitOfWork) *GetBookingUseCase {
	return &GetBookingUseCase{uow: uow}
}

// Execute retrieves a booking aggregate and returns its read model DTO.
func (uc *GetBookingUseCase) Execute(ctx context.Context, query GetBookingQuery) (BookingResponseDTO, error) {
	var dto BookingResponseDTO
	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}

		b, err := repo.GetReadModel(txCtx, query.BookingID, query.TenantID)
		if err != nil {
			return err
		}

		dto = BookingResponseDTO{
			ID:               b.ID,
			BookingNumber:    b.BookingNumber,
			CustomerID:       b.CustomerID,
			CustomerName:     b.CustomerName,
			CustomerCompany:  b.CustomerCompany,
			RouteID:          b.RouteID,
			RouteSource:      b.RouteSource,
			RouteDestination: b.RouteDestination,
			PickupDate:       b.PickupDate,
			VehicleType:      b.VehicleType,
			Passengers:       b.Passengers,
			CargoWeight:      b.CargoWeight,
			Price:            b.Price,
			Notes:            b.Notes,
			Status:           b.Status,
			CreatedAt:        b.CreatedAt,
			UpdatedAt:        b.UpdatedAt,
		}
		return nil
	})
	return dto, err
}
