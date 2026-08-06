package vehicle

import (
	"context"

	"transport-app/internal/vehicle/application"
	"transport-app/internal/vehicle/domain/aggregate"
)

type vehicleFacadeImpl struct {
	createUC *application.CreateVehicleUseCase
	updateUC *application.UpdateVehicleUseCase
}

// NewVehicleFacade constructs a concrete implementation of VehicleFacade.
func NewVehicleFacade(
	createUC *application.CreateVehicleUseCase,
	updateUC *application.UpdateVehicleUseCase,
) VehicleFacade {
	return &vehicleFacadeImpl{
		createUC: createUC,
		updateUC: updateUC,
	}
}

func (f *vehicleFacadeImpl) CreateVehicle(ctx context.Context, cmd CreateVehicleCommand) (aggregate.VehicleID, error) {
	return f.createUC.Execute(ctx, application.CreateVehicleCommand(cmd))
}

func (f *vehicleFacadeImpl) UpdateVehicle(ctx context.Context, cmd UpdateVehicleCommand) error {
	return f.updateUC.Execute(ctx, application.UpdateVehicleCommand(cmd))
}
