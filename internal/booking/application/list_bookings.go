package application

import (
	"context"
	"errors"

	"transport-app/internal/booking/domain"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// ListBookingsQuery holds pagination, status, and search filters.
type ListBookingsQuery struct {
	TenantID shared.TenantID
	Page     int
	Limit    int
	Search   string
	Status   string
}

// ListBookingsResponse represents the paginated result.
type ListBookingsResponse struct {
	Bookings []BookingResponseDTO
	Total    int64
}

// ListBookingsUseCase orchestrates the paginated search query execution.
type ListBookingsUseCase struct {
	uow ports.UnitOfWork
}

// NewListBookingsUseCase creates a new ListBookingsUseCase.
func NewListBookingsUseCase(uow ports.UnitOfWork) *ListBookingsUseCase {
	return &ListBookingsUseCase{uow: uow}
}

// Execute performs retrieval of paginated booking DTOs.
func (uc *ListBookingsUseCase) Execute(ctx context.Context, q ListBookingsQuery) (ListBookingsResponse, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	offset := (q.Page - 1) * q.Limit

	var res ListBookingsResponse

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}

		readModels, total, err := repo.SearchReadModels(txCtx, q.TenantID, q.Search, q.Status, q.Limit, offset)
		if err != nil {
			return err
		}

		dtos := make([]BookingResponseDTO, len(readModels))
		for i, b := range readModels {
			dtos[i] = BookingResponseDTO{
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
		}

		res = ListBookingsResponse{
			Bookings: dtos,
			Total:    total,
		}
		return nil
	})

	return res, err
}
