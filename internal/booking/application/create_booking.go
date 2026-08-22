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

// CreateBookingCommand defines the parameters required to request a new booking.
type CreateBookingCommand struct {
	TenantID    shared.TenantID
	CustomerID  string
	RouteID     string
	PickupDate  string
	VehicleType string
	Passengers  int64
	CargoWeight *float64
	Price       float64
	Notes       string
}

// CreateBookingUseCase orchestrates the validation and persistence of a new booking.
type CreateBookingUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

// NewCreateBookingUseCase creates a new CreateBookingUseCase.
func NewCreateBookingUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *CreateBookingUseCase {
	return &CreateBookingUseCase{uow: uow, idGen: idGen, clock: clock}
}

// Execute performs the creation of the booking aggregate within transactional boundaries.
func (uc *CreateBookingUseCase) Execute(ctx context.Context, cmd CreateBookingCommand) (aggregate.BookingID, error) {
	if cmd.CustomerID == "" {
		return "", errors.New("customer ID is required")
	}
	if cmd.RouteID == "" {
		return "", errors.New("route ID is required")
	}
	if cmd.VehicleType == "" {
		return "", errors.New("vehicle type is required")
	}
	if cmd.Passengers < 1 {
		return "", errors.New("passengers must be at least 1")
	}
	if cmd.Price < 0 {
		return "", errors.New("price cannot be negative")
	}

	pickupDate, err := time.Parse("2006-01-02", cmd.PickupDate)
	if err != nil {
		pickupDate, err = time.Parse(time.RFC3339, cmd.PickupDate)
		if err != nil {
			pickupDate, err = time.Parse("2006-01-02 15:04", cmd.PickupDate)
			if err != nil {
				pickupDate, err = time.Parse("2006-01-02 15:04:05", cmd.PickupDate)
				if err != nil {
					return "", errors.New("invalid pickup date format")
				}
			}
		}
	}

	bookingID := aggregate.BookingID(uc.idGen.GenerateUUID())
	bookingNumber := uc.idGen.GenerateDisplayID("BK")
	priceMoney := shared.FloatToMoney(cmd.Price, "INR")

	booking := aggregate.NewBookingAggregate(
		bookingID,
		cmd.TenantID,
		bookingNumber,
		cmd.CustomerID,
		cmd.RouteID,
		pickupDate,
		cmd.VehicleType,
		cmd.Passengers,
		cmd.CargoWeight,
		priceMoney,
		cmd.Notes,
		uc.clock.Now(),
	)

	err = uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}
		if err := repo.Save(txCtx, booking); err != nil {
			return err
		}
		logAudit(txCtx, ActionCreate, string(booking.ID), nil, nil)
		return nil
	})

	if err != nil {
		return "", err
	}

	return bookingID, nil
}
