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

type CreateDriverCommand struct {
	TenantID              shared.TenantID
	FirstName             string
	LastName              string
	Phone                 string
	Email                 *string
	Address               *string
	LicenseNumber         string
	LicenseExpiry         time.Time
	ExperienceYears       int64
	EmergencyContactName  *string
	EmergencyContactPhone *string
	Notes                 *string
}

type CreateDriverUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

func NewCreateDriverUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *CreateDriverUseCase {
	return &CreateDriverUseCase{uow: uow, idGen: idGen, clock: clock}
}

func (uc *CreateDriverUseCase) Execute(ctx context.Context, cmd CreateDriverCommand) (aggregate.DriverID, error) {
	if cmd.FirstName == "" || cmd.LastName == "" {
		return "", errors.New("first and last name are required")
	}
	if cmd.Phone == "" {
		return "", errors.New("phone number is required")
	}

	id := aggregate.DriverID(uc.idGen.GenerateUUID())
	displayID := uc.idGen.GenerateDisplayID("DR")

	d := aggregate.NewDriverAggregate(
		id,
		cmd.TenantID,
		displayID,
		cmd.FirstName,
		cmd.LastName,
		cmd.Phone,
		cmd.Email,
		cmd.Address,
		cmd.LicenseNumber,
		cmd.LicenseExpiry,
		cmd.ExperienceYears,
		aggregate.DriverAvailable,
		cmd.EmergencyContactName,
		cmd.EmergencyContactPhone,
		cmd.Notes,
		uc.clock.Now(),
	)

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Drivers().(domain.DriverRepository)
		if !ok {
			return errors.New("failed to retrieve driver repository")
		}
		return repo.Save(txCtx, d)
	})

	if err != nil {
		return "", err
	}

	return id, nil
}
