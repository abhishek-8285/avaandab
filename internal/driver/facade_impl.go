package driver

import (
	"context"

	"transport-app/internal/driver/application"
	"transport-app/internal/driver/domain/aggregate"
)

type driverFacadeImpl struct {
	createUC *application.CreateDriverUseCase
	updateUC *application.UpdateDriverUseCase
}

// NewDriverFacade constructs a new DriverFacade implementation.
func NewDriverFacade(
	createUC *application.CreateDriverUseCase,
	updateUC *application.UpdateDriverUseCase,
) DriverFacade {
	return &driverFacadeImpl{
		createUC: createUC,
		updateUC: updateUC,
	}
}

func (f *driverFacadeImpl) CreateDriver(ctx context.Context, cmd CreateDriverCommand) (aggregate.DriverID, error) {
	return f.createUC.Execute(ctx, application.CreateDriverCommand(cmd))
}

func (f *driverFacadeImpl) UpdateDriver(ctx context.Context, cmd UpdateDriverCommand) error {
	return f.updateUC.Execute(ctx, application.UpdateDriverCommand(cmd))
}
