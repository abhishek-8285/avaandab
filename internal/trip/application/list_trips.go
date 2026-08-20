package application

import (
	"context"
	"errors"

	driverdomain "transport-app/internal/driver/domain"
	driveragg "transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
)

// ListTripsQuery parameters.
type ListTripsQuery struct {
	TenantID   shared.TenantID
	Page       int
	Limit      int
	Search     string
	Status     string
	DriverID   string
	AuthUserID string
}

// ListTripsResponse paginated results.
type ListTripsResponse struct {
	Trips []TripResponseDTO
	Total int64
}

// ListTripsUseCase queries a list of trips.
type ListTripsUseCase struct {
	uow ports.UnitOfWork
}

// NewListTripsUseCase creates a new ListTripsUseCase.
func NewListTripsUseCase(uow ports.UnitOfWork) *ListTripsUseCase {
	return &ListTripsUseCase{uow: uow}
}

// Execute retrieves paginated trip read models.
func (uc *ListTripsUseCase) Execute(ctx context.Context, q ListTripsQuery) (ListTripsResponse, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	offset := (q.Page - 1) * q.Limit

	var res ListTripsResponse

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}

		var rows []domain.TripReadModel
		var total int64
		var err error

		if q.DriverID == "me" && q.AuthUserID != "" {
			driverIDs := []string{q.AuthUserID}
			if driverRepo, ok := txCtx.Repositories().Drivers().(driverdomain.DriverRepository); ok {
				if rm, dErr := driverRepo.GetReadModel(txCtx, driveragg.DriverID(q.AuthUserID), q.TenantID); dErr == nil {
					if rm.ID != "" && rm.ID != q.AuthUserID {
						driverIDs = append(driverIDs, rm.ID)
					}
					if rm.DriverDisplayID != "" && rm.DriverDisplayID != q.AuthUserID {
						driverIDs = append(driverIDs, rm.DriverDisplayID)
					}
				}
			}
			rows, total, err = repo.SearchReadModelsByDriver(txCtx, q.TenantID, driverIDs, q.Search, q.Status, q.Limit, offset)
		} else if q.DriverID != "" && q.DriverID != "me" {
			rows, total, err = repo.SearchReadModelsByDriver(txCtx, q.TenantID, []string{q.DriverID}, q.Search, q.Status, q.Limit, offset)
		} else {
			rows, total, err = repo.SearchReadModels(txCtx, q.TenantID, q.Search, q.Status, q.Limit, offset)
		}

		if err != nil {
			return err
		}

		dtos := make([]TripResponseDTO, len(rows))
		for i, t := range rows {
			dtos[i] = TripResponseDTO{
				ID:                        t.ID,
				TripNumber:                t.TripNumber,
				BookingID:                 t.BookingID,
				DriverID:                  t.DriverID,
				DriverDisplayID:           t.DriverDisplayID,
				DriverFirstName:           t.DriverFirstName,
				DriverLastName:            t.DriverLastName,
				VehicleID:                 t.VehicleID,
				VehicleRegistrationNumber: t.VehicleRegistrationNumber,
				VehicleNumber:             t.VehicleNumber,
				RouteID:                   t.RouteID,
				RouteSource:               t.RouteSource,
				RouteDestination:          t.RouteDestination,
				DepartureTime:             t.DepartureTime,
				ArrivalTime:               t.ArrivalTime,
				Status:                    t.Status,
				Remarks:                   t.Remarks,
				CreatedAt:                 t.CreatedAt,
				UpdatedAt:                 t.UpdatedAt,
				StartedAt:                 t.StartedAt,
				ReachedPickupAt:           t.ReachedPickupAt,
				InTransitAt:               t.InTransitAt,
				DeliveredAt:               t.DeliveredAt,
				CompletedAt:               t.CompletedAt,
			}
		}

		res = ListTripsResponse{
			Trips: dtos,
			Total: total,
		}
		return nil
	})

	return res, err
}
