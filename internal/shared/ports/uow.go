package ports

import (
	"context"
)

// RepositoryProvider exposes command repositories inside a transaction context.
type RepositoryProvider interface {
	// We can extend this with specific repository interfaces as they are built.
	// For V1 Booking, it returns our new BookingRepository contract.
	Bookings() any
	Trips() any
	Drivers() any
	Vehicles() any
	Invoices() any
	Payments() any
	AuditLogs() any
}

// TxContext represents a transactional context execution environment.
type TxContext interface {
	context.Context
	Repositories() RepositoryProvider
}

// UnitOfWork coordinates transaction boundaries across repository operations.
type UnitOfWork interface {
	Execute(ctx context.Context, fn func(txCtx TxContext) error) error
}
