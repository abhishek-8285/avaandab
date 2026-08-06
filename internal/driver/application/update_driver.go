package application

import (
	"context"
	"errors"
	"time"

	"transport-app/internal/driver/domain"
	"transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

type UpdateDriverCommand struct {
	ID                    aggregate.DriverID
	TenantID              shared.TenantID
	FirstName             string
	LastName              string
	Phone                 string
	Email                 *string
	Address               *string
	LicenseNumber         string
	LicenseExpiry         time.Time
	ExperienceYears       int64
	Status                aggregate.DriverStatus
	EmergencyContactName  *string
	EmergencyContactPhone *string
	Notes                 *string
}

type UpdateDriverUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

func NewUpdateDriverUseCase(uow ports.UnitOfWork, clock ports.Clock) *UpdateDriverUseCase {
	return &UpdateDriverUseCase{uow: uow, clock: clock}
}

func (uc *UpdateDriverUseCase) Execute(ctx context.Context, cmd UpdateDriverCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Drivers().(domain.DriverRepository)
		if !ok {
			return errors.New("failed to retrieve driver repository")
		}

		d, err := repo.Find(txCtx, cmd.ID, cmd.TenantID)
		if err != nil {
			return err
		}

		err = d.UpdateDetails(
			cmd.FirstName,
			cmd.LastName,
			cmd.Phone,
			cmd.Email,
			cmd.Address,
			cmd.LicenseNumber,
			cmd.LicenseExpiry,
			cmd.ExperienceYears,
			cmd.Status,
			cmd.EmergencyContactName,
			cmd.EmergencyContactPhone,
			cmd.Notes,
			uc.clock.Now(),
		)
		if err != nil {
			return err
		}

		return repo.Save(txCtx, d)
	})
}
