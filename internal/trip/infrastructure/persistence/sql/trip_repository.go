package sql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/repository"
	"transport-app/internal/shared"
	"transport-app/internal/shared/outbox"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
	"transport-app/internal/trip/infrastructure/persistence/sql/converters"
)

type tripRepository struct {
	dbConn *sql.DB
	q      *db.Queries
	outbox *outbox.OutboxWriter
}

// NewTripRepository creates a SQLite-backed implementation of TripRepository.
func NewTripRepository(dbConn *sql.DB) domain.TripRepository {
	return &tripRepository{
		dbConn: dbConn,
		q:      db.New(dbConn),
		outbox: outbox.NewOutboxWriter(dbConn),
	}
}

// Q retrieves queries, using a transaction context if active.
func (r *tripRepository) Q(ctx context.Context) *db.Queries {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *tripRepository) Save(ctx context.Context, t *aggregate.TripAggregate) error {
	exists, err := r.Exists(ctx, t.ID, t.TenantID)
	if err != nil {
		return err
	}

	p := converters.MapToPersistence(t)

	if exists {
		_, err = r.Q(ctx).UpdateTrip(ctx, db.UpdateTripParams{
			TripNumber:      p.TripNumber,
			BookingID:       p.BookingID,
			DriverID:        p.DriverID,
			VehicleID:       p.VehicleID,
			RouteID:         p.RouteID,
			DepartureTime:   p.DepartureTime,
			ArrivalTime:     p.ArrivalTime,
			Status:          p.Status,
			Remarks:         p.Remarks,
			StartedAt:       p.StartedAt,
			ReachedPickupAt: p.ReachedPickupAt,
			InTransitAt:     p.InTransitAt,
			DeliveredAt:     p.DeliveredAt,
			CompletedAt:     p.CompletedAt,
			ID:              p.ID,
			TenantID:        p.TenantID,
			Version:         p.Version,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("concurrency conflict: trip modified by another process")
			}
			return err
		}
		t.Version++
	} else {
		_, err = r.Q(ctx).CreateTrip(ctx, db.CreateTripParams{
			ID:              p.ID,
			TripNumber:      p.TripNumber,
			BookingID:       p.BookingID,
			DriverID:        p.DriverID,
			VehicleID:       p.VehicleID,
			RouteID:         p.RouteID,
			DepartureTime:   p.DepartureTime,
			ArrivalTime:     p.ArrivalTime,
			Status:          p.Status,
			Remarks:         p.Remarks,
			TenantID:        p.TenantID,
			StartedAt:       p.StartedAt,
			ReachedPickupAt: p.ReachedPickupAt,
			InTransitAt:     p.InTransitAt,
			DeliveredAt:     p.DeliveredAt,
			CompletedAt:     p.CompletedAt,
		})
		if err != nil {
			return err
		}
		t.Version = 1
	}
	err = r.outbox.SaveEvents(ctx, string(t.ID), "Trip", t.Events())
	if err != nil {
		return err
	}
	t.ClearEvents()
	return nil
}

func (r *tripRepository) Find(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) (*aggregate.TripAggregate, error) {
	row, err := r.Q(ctx).GetTripByID(ctx, db.GetTripByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("trip not found")
		}
		return nil, err
	}

	m := converters.SQLTripModel{
		ID:              row.ID,
		TripNumber:      row.TripNumber,
		BookingID:       row.BookingID,
		DriverID:        row.DriverID,
		VehicleID:       row.VehicleID,
		RouteID:         row.RouteID,
		DepartureTime:   row.DepartureTime,
		ArrivalTime:     row.ArrivalTime,
		Status:          row.Status,
		Remarks:         row.Remarks,
		TenantID:        row.TenantID,
		Version:         row.Version,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		StartedAt:       row.StartedAt,
		ReachedPickupAt: row.ReachedPickupAt,
		InTransitAt:     row.InTransitAt,
		DeliveredAt:     row.DeliveredAt,
		CompletedAt:     row.CompletedAt,
	}
	return converters.MapToAggregate(m), nil
}

func (r *tripRepository) FindByNumber(ctx context.Context, number string, tenantID shared.TenantID) (*aggregate.TripAggregate, error) {
	row, err := r.Q(ctx).GetTripByNumber(ctx, db.GetTripByNumberParams{
		TripNumber: number,
		TenantID:      string(tenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("trip not found")
		}
		return nil, err
	}

	m := converters.SQLTripModel{
		ID:              row.ID,
		TripNumber:      row.TripNumber,
		BookingID:       row.BookingID,
		DriverID:        row.DriverID,
		VehicleID:       row.VehicleID,
		RouteID:         row.RouteID,
		DepartureTime:   row.DepartureTime,
		ArrivalTime:     row.ArrivalTime,
		Status:          row.Status,
		Remarks:         row.Remarks,
		TenantID:        row.TenantID,
		Version:         row.Version,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		StartedAt:       row.StartedAt,
		ReachedPickupAt: row.ReachedPickupAt,
		InTransitAt:     row.InTransitAt,
		DeliveredAt:     row.DeliveredAt,
		CompletedAt:     row.CompletedAt,
	}
	return converters.MapToAggregate(m), nil
}

func (r *tripRepository) Exists(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) (bool, error) {
	_, err := r.Q(ctx).GetTripByID(ctx, db.GetTripByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *tripRepository) GetReadModel(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) (domain.TripReadModel, error) {
	row, err := r.Q(ctx).GetTripByID(ctx, db.GetTripByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TripReadModel{}, errors.New("trip not found")
		}
		return domain.TripReadModel{}, err
	}

	var bookingID *string
	if row.BookingID.Valid {
		bookingID = &row.BookingID.String
	}
	var driverID *string
	if row.DriverID.Valid {
		driverID = &row.DriverID.String
	}
	var vehicleID *string
	if row.VehicleID.Valid {
		vehicleID = &row.VehicleID.String
	}
	var arrivalTime *time.Time
	if row.ArrivalTime.Valid {
		arrivalTime = &row.ArrivalTime.Time
	}
	var remarks string
	if row.Remarks.Valid {
		remarks = row.Remarks.String
	}

	var startedAt *time.Time
	if row.StartedAt.Valid {
		startedAt = &row.StartedAt.Time
	}
	var reachedPickupAt *time.Time
	if row.ReachedPickupAt.Valid {
		reachedPickupAt = &row.ReachedPickupAt.Time
	}
	var inTransitAt *time.Time
	if row.InTransitAt.Valid {
		inTransitAt = &row.InTransitAt.Time
	}
	var deliveredAt *time.Time
	if row.DeliveredAt.Valid {
		deliveredAt = &row.DeliveredAt.Time
	}
	var completedAt *time.Time
	if row.CompletedAt.Valid {
		completedAt = &row.CompletedAt.Time
	}

	return domain.TripReadModel{
		ID:                        row.ID,
		TripNumber:                row.TripNumber,
		BookingID:                 bookingID,
		DriverID:                  driverID,
		DriverDisplayID:           row.DriverDisplayID.String,
		DriverFirstName:           row.DriverFirstName.String,
		DriverLastName:            row.DriverLastName.String,
		VehicleID:                 vehicleID,
		VehicleRegistrationNumber: row.VehicleRegistrationNumber.String,
		VehicleNumber:             row.VehicleNumber.String,
		RouteID:                   row.RouteID,
		RouteSource:               row.RouteSource.String,
		RouteDestination:          row.RouteDestination.String,
		DepartureTime:             row.DepartureTime,
		ArrivalTime:               arrivalTime,
		Status:                    row.Status,
		Remarks:                   remarks,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		StartedAt:                 startedAt,
		ReachedPickupAt:           reachedPickupAt,
		InTransitAt:               inTransitAt,
		DeliveredAt:               deliveredAt,
		CompletedAt:               completedAt,
	}, nil
}

func (r *tripRepository) SearchReadModels(ctx context.Context, tenantID shared.TenantID, query string, status string, limit int, offset int) ([]domain.TripReadModel, int64, error) {
	rows, err := r.Q(ctx).SearchTrips(ctx, db.SearchTripsParams{
		TenantID: string(tenantID),
		Column2:  sql.NullString{String: query, Valid: true},
		Column3:  sql.NullString{String: query, Valid: true},
		Column4:  sql.NullString{String: query, Valid: true},
		Column5:  sql.NullString{String: query, Valid: true},
		Column6:  sql.NullString{String: query, Valid: true},
		Column7:  sql.NullString{String: query, Valid: true},
		Column8:  status,
		Status:   status,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.Q(ctx).CountTrips(ctx, db.CountTripsParams{
		TenantID: string(tenantID),
		Column2:  sql.NullString{String: query, Valid: true},
		Column3:  sql.NullString{String: query, Valid: true},
		Column4:  sql.NullString{String: query, Valid: true},
		Column5:  sql.NullString{String: query, Valid: true},
		Column6:  sql.NullString{String: query, Valid: true},
		Column7:  sql.NullString{String: query, Valid: true},
		Column8:  status,
		Status:   status,
	})
	if err != nil {
		return nil, 0, err
	}

	readModels := make([]domain.TripReadModel, len(rows))
	for i, row := range rows {
		var bookingID *string
		if row.BookingID.Valid {
			bookingID = &row.BookingID.String
		}
		var driverID *string
		if row.DriverID.Valid {
			driverID = &row.DriverID.String
		}
		var vehicleID *string
		if row.VehicleID.Valid {
			vehicleID = &row.VehicleID.String
		}
		var arrivalTime *time.Time
		if row.ArrivalTime.Valid {
			arrivalTime = &row.ArrivalTime.Time
		}
		var remarks string
		if row.Remarks.Valid {
			remarks = row.Remarks.String
		}

		var startedAt *time.Time
		if row.StartedAt.Valid {
			startedAt = &row.StartedAt.Time
		}
		var reachedPickupAt *time.Time
		if row.ReachedPickupAt.Valid {
			reachedPickupAt = &row.ReachedPickupAt.Time
		}
		var inTransitAt *time.Time
		if row.InTransitAt.Valid {
			inTransitAt = &row.InTransitAt.Time
		}
		var deliveredAt *time.Time
		if row.DeliveredAt.Valid {
			deliveredAt = &row.DeliveredAt.Time
		}
		var completedAt *time.Time
		if row.CompletedAt.Valid {
			completedAt = &row.CompletedAt.Time
		}

		readModels[i] = domain.TripReadModel{
			ID:                        row.ID,
			TripNumber:                row.TripNumber,
			BookingID:                 bookingID,
			DriverID:                  driverID,
			DriverDisplayID:           row.DriverDisplayID.String,
			DriverFirstName:           row.DriverFirstName.String,
			DriverLastName:            row.DriverLastName.String,
			VehicleID:                 vehicleID,
			VehicleRegistrationNumber: row.VehicleRegistrationNumber.String,
			VehicleNumber:             row.VehicleNumber.String,
			RouteID:                   row.RouteID,
			RouteSource:               row.RouteSource.String,
			RouteDestination:          row.RouteDestination.String,
			DepartureTime:             row.DepartureTime,
			ArrivalTime:               arrivalTime,
			Status:                    row.Status,
			Remarks:                   remarks,
			CreatedAt:                 row.CreatedAt,
			UpdatedAt:                 row.UpdatedAt,
			StartedAt:                 startedAt,
			ReachedPickupAt:           reachedPickupAt,
			InTransitAt:               inTransitAt,
			DeliveredAt:               deliveredAt,
			CompletedAt:               completedAt,
		}
	}

	return readModels, count, nil
}
