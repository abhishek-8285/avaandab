package uow

import (
	"context"
	"database/sql"

	bookingsql "transport-app/internal/booking/infrastructure/persistence/sql"
	driversql "transport-app/internal/driver/infrastructure/persistence/sql"
	invoicesql "transport-app/internal/invoice/infrastructure/persistence/sql"
	paymentsql "transport-app/internal/payment/infrastructure/persistence/sql"
	"transport-app/internal/repository"
	"transport-app/internal/repository/sqlite"
	"transport-app/internal/shared/ports"
	tripsql "transport-app/internal/trip/infrastructure/persistence/sql"
	vehiclesql "transport-app/internal/vehicle/infrastructure/persistence/sql"
)

type repositoryProvider struct {
	dbConn *sql.DB
}

func (p *repositoryProvider) Bookings() any {
	return bookingsql.NewBookingRepository(p.dbConn)
}

func (p *repositoryProvider) Trips() any {
	return tripsql.NewTripRepository(p.dbConn)
}

func (p *repositoryProvider) Drivers() any {
	return driversql.NewDriverRepository(p.dbConn)
}

func (p *repositoryProvider) Vehicles() any {
	return vehiclesql.NewVehicleRepository(p.dbConn)
}

func (p *repositoryProvider) Invoices() any {
	return invoicesql.NewInvoiceRepository(p.dbConn)
}

func (p *repositoryProvider) Payments() any {
	return paymentsql.NewPaymentRepository(p.dbConn)
}

type txContext struct {
	context.Context
	provider ports.RepositoryProvider
}

func (t *txContext) Repositories() ports.RepositoryProvider {
	return t.provider
}

type sqlUnitOfWork struct {
	db        *sql.DB
	txManager repository.TxManager
}

// NewSQLUnitOfWork constructs a SQLite-backed UnitOfWork implementation.
func NewSQLUnitOfWork(db *sql.DB) ports.UnitOfWork {
	return &sqlUnitOfWork{
		db:        db,
		txManager: repository.NewTxManager(sqlite.NewRepository(db)),
	}
}

func (u *sqlUnitOfWork) Execute(ctx context.Context, fn func(txCtx ports.TxContext) error) error {
	return u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		provider := &repositoryProvider{dbConn: u.db}
		tCtx := &txContext{
			Context:  txCtx,
			provider: provider,
		}
		return fn(tCtx)
	})
}
