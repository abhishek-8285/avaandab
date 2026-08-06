package trip

import (
	"context"
	"transport-app/internal/shared"
	"transport-app/internal/trip/application"
	"transport-app/internal/trip/domain/aggregate"
)

type tripFacadeImpl struct {
	createUC   *application.CreateTripUseCase
	startUC    *application.StartTripUseCase
	completeUC *application.CompleteTripUseCase
	cancelUC   *application.CancelTripUseCase
}

// NewTripFacade constructs a concrete implementation of TripFacade.
func NewTripFacade(
	createUC *application.CreateTripUseCase,
	startUC *application.StartTripUseCase,
	completeUC *application.CompleteTripUseCase,
	cancelUC *application.CancelTripUseCase,
) TripFacade {
	return &tripFacadeImpl{
		createUC:   createUC,
		startUC:    startUC,
		completeUC: completeUC,
		cancelUC:   cancelUC,
	}
}

func (f *tripFacadeImpl) CreateTrip(ctx context.Context, cmd CreateTripCommand) (aggregate.TripID, error) {
	return f.createUC.Execute(ctx, application.CreateTripCommand(cmd))
}

func (f *tripFacadeImpl) StartTrip(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) error {
	return f.startUC.Execute(ctx, application.StartTripCommand{
		TripID:   id,
		TenantID: tenantID,
	})
}

func (f *tripFacadeImpl) CompleteTrip(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) error {
	return f.completeUC.Execute(ctx, application.CompleteTripCommand{
		TripID:   id,
		TenantID: tenantID,
	})
}

func (f *tripFacadeImpl) CancelTrip(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) error {
	return f.cancelUC.Execute(ctx, application.CancelTripCommand{
		TripID:   id,
		TenantID: tenantID,
	})
}
