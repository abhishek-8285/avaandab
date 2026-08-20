package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
		TenantID:   string(tenantID),
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

func (r *tripRepository) CheckDriverConflict(ctx context.Context, driverID string, tenantID shared.TenantID, excludeTripID string) ([]domain.ConflictInfo, error) {
	rows, err := r.Q(ctx).CheckDriverConflict(ctx, db.CheckDriverConflictParams{
		DriverID: sql.NullString{String: driverID, Valid: driverID != ""},
		TenantID: string(tenantID),
		Column3:  excludeTripID,
		ID:       excludeTripID,
	})
	if err != nil {
		return nil, err
	}
	conflicts := make([]domain.ConflictInfo, 0, len(rows))
	for _, row := range rows {
		var arrival *time.Time
		if row.ArrivalTime.Valid {
			arrival = &row.ArrivalTime.Time
		}
		conflicts = append(conflicts, domain.ConflictInfo{
			ID:            row.ID,
			TripNumber:    row.TripNumber,
			Status:        row.Status,
			DepartureTime: row.DepartureTime,
			ArrivalTime:   arrival,
		})
	}
	return conflicts, nil
}

func (r *tripRepository) CheckVehicleConflict(ctx context.Context, vehicleID string, tenantID shared.TenantID, excludeTripID string) ([]domain.ConflictInfo, error) {
	rows, err := r.Q(ctx).CheckVehicleConflict(ctx, db.CheckVehicleConflictParams{
		VehicleID: sql.NullString{String: vehicleID, Valid: vehicleID != ""},
		TenantID:  string(tenantID),
		Column3:   excludeTripID,
		ID:        excludeTripID,
	})
	if err != nil {
		return nil, err
	}
	conflicts := make([]domain.ConflictInfo, 0, len(rows))
	for _, row := range rows {
		var arrival *time.Time
		if row.ArrivalTime.Valid {
			arrival = &row.ArrivalTime.Time
		}
		conflicts = append(conflicts, domain.ConflictInfo{
			ID:            row.ID,
			TripNumber:    row.TripNumber,
			Status:        row.Status,
			DepartureTime: row.DepartureTime,
			ArrivalTime:   arrival,
		})
	}
	return conflicts, nil
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
	qPattern := "%" + query + "%"

	querySQL := `
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.created_at, t.updated_at,
    t.started_at, t.reached_pickup_at, t.in_transit_at, t.delivered_at, t.completed_at,
    COALESCE(d.driver_id, '') AS driver_display_id,
    COALESCE(d.first_name, '') AS driver_first_name,
    COALESCE(d.last_name, '') AS driver_last_name,
    COALESCE(v.registration_number, '') AS vehicle_registration_number,
    COALESCE(v.vehicle_number, '') AS vehicle_number,
    COALESCE(r.source, '') AS route_source,
    COALESCE(r.destination, '') AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.tenant_id = ?
  AND (? = '' OR t.trip_number LIKE ? OR d.first_name LIKE ? OR d.last_name LIKE ? OR v.registration_number LIKE ? OR r.source LIKE ? OR r.destination LIKE ?)
  AND (? = '' OR t.status = ?)
ORDER BY t.departure_time DESC
LIMIT ? OFFSET ?`

	rows, err := r.dbConn.QueryContext(ctx, querySQL,
		string(tenantID),
		query, qPattern, qPattern, qPattern, qPattern, qPattern, qPattern,
		status, status,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var readModels []domain.TripReadModel
	for rows.Next() {
		var m domain.TripReadModel
		var bookingID, driverID, vehicleID sql.NullString
		var arrivalTime, startedAt, reachedPickupAt, inTransitAt, deliveredAt, completedAt sql.NullTime
		var remarks sql.NullString

		err := rows.Scan(
			&m.ID, &m.TripNumber, &bookingID, &driverID, &vehicleID, &m.RouteID,
			&m.DepartureTime, &arrivalTime, &m.Status, &remarks, &m.CreatedAt, &m.UpdatedAt,
			&startedAt, &reachedPickupAt, &inTransitAt, &deliveredAt, &completedAt,
			&m.DriverDisplayID, &m.DriverFirstName, &m.DriverLastName,
			&m.VehicleRegistrationNumber, &m.VehicleNumber,
			&m.RouteSource, &m.RouteDestination,
		)
		if err != nil {
			return nil, 0, err
		}

		if bookingID.Valid {
			m.BookingID = &bookingID.String
		}
		if driverID.Valid {
			m.DriverID = &driverID.String
		}
		if vehicleID.Valid {
			m.VehicleID = &vehicleID.String
		}
		if arrivalTime.Valid {
			m.ArrivalTime = &arrivalTime.Time
		}
		if remarks.Valid {
			m.Remarks = remarks.String
		}
		if startedAt.Valid {
			m.StartedAt = &startedAt.Time
		}
		if reachedPickupAt.Valid {
			m.ReachedPickupAt = &reachedPickupAt.Time
		}
		if inTransitAt.Valid {
			m.InTransitAt = &inTransitAt.Time
		}
		if deliveredAt.Valid {
			m.DeliveredAt = &deliveredAt.Time
		}
		if completedAt.Valid {
			m.CompletedAt = &completedAt.Time
		}

		readModels = append(readModels, m)
	}

	countSQL := `
SELECT COUNT(*)
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.tenant_id = ?
  AND (? = '' OR t.trip_number LIKE ? OR d.first_name LIKE ? OR d.last_name LIKE ? OR v.registration_number LIKE ? OR r.source LIKE ? OR r.destination LIKE ?)
  AND (? = '' OR t.status = ?)`

	var count int64
	err = r.dbConn.QueryRowContext(ctx, countSQL,
		string(tenantID),
		query, qPattern, qPattern, qPattern, qPattern, qPattern, qPattern,
		status, status,
	).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	return readModels, count, nil
}

func (r *tripRepository) SearchReadModelsByDriver(ctx context.Context, tenantID shared.TenantID, driverIDs []string, query string, status string, limit int, offset int) ([]domain.TripReadModel, int64, error) {
	if len(driverIDs) == 0 {
		return nil, 0, nil
	}

	allDriverIDs := make(map[string]struct{})
	for _, id := range driverIDs {
		if id != "" {
			allDriverIDs[id] = struct{}{}
		}
	}

	for _, id := range driverIDs {
		if id == "" {
			continue
		}
		var dID, dCode string
		_ = r.dbConn.QueryRowContext(ctx, `
			SELECT id, driver_id FROM drivers
			WHERE tenant_id = ? AND (id = ? OR email = (SELECT email FROM users WHERE id = ?))
			LIMIT 1
		`, string(tenantID), id, id).Scan(&dID, &dCode)
		if dID != "" {
			allDriverIDs[dID] = struct{}{}
		}
		if dCode != "" {
			allDriverIDs[dCode] = struct{}{}
		}
	}

	var resolvedIDs []string
	for id := range allDriverIDs {
		resolvedIDs = append(resolvedIDs, id)
	}
	if len(resolvedIDs) == 0 {
		return nil, 0, nil
	}

	qPattern := "%" + query + "%"

	placeholders := make([]string, len(resolvedIDs))
	for i := range resolvedIDs {
		placeholders[i] = "?"
	}
	driverClause := fmt.Sprintf("t.driver_id IN (%s)", strings.Join(placeholders, ","))

	querySQL := fmt.Sprintf(`
SELECT t.id, t.trip_number, t.booking_id, t.driver_id, t.vehicle_id, t.route_id,
    t.departure_time, t.arrival_time, t.status, t.remarks, t.created_at, t.updated_at,
    t.started_at, t.reached_pickup_at, t.in_transit_at, t.delivered_at, t.completed_at,
    COALESCE(d.driver_id, '') AS driver_display_id,
    COALESCE(d.first_name, '') AS driver_first_name,
    COALESCE(d.last_name, '') AS driver_last_name,
    COALESCE(v.registration_number, '') AS vehicle_registration_number,
    COALESCE(v.vehicle_number, '') AS vehicle_number,
    COALESCE(r.source, '') AS route_source,
    COALESCE(r.destination, '') AS route_destination
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.tenant_id = ?
  AND %s
  AND (? = '' OR t.trip_number LIKE ? OR d.first_name LIKE ? OR d.last_name LIKE ? OR v.registration_number LIKE ? OR r.source LIKE ? OR r.destination LIKE ?)
  AND (? = '' OR t.status = ?)
ORDER BY t.departure_time DESC
LIMIT ? OFFSET ?`, driverClause)

	args := []interface{}{string(tenantID)}
	for _, id := range resolvedIDs {
		args = append(args, id)
	}
	args = append(args, query, qPattern, qPattern, qPattern, qPattern, qPattern, qPattern, status, status, limit, offset)

	rows, err := r.dbConn.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var readModels []domain.TripReadModel
	for rows.Next() {
		var m domain.TripReadModel
		var bookingID, driverID, vehicleID sql.NullString
		var arrivalTime, startedAt, reachedPickupAt, inTransitAt, deliveredAt, completedAt sql.NullTime
		var remarks sql.NullString

		err := rows.Scan(
			&m.ID, &m.TripNumber, &bookingID, &driverID, &vehicleID, &m.RouteID,
			&m.DepartureTime, &arrivalTime, &m.Status, &remarks, &m.CreatedAt, &m.UpdatedAt,
			&startedAt, &reachedPickupAt, &inTransitAt, &deliveredAt, &completedAt,
			&m.DriverDisplayID, &m.DriverFirstName, &m.DriverLastName,
			&m.VehicleRegistrationNumber, &m.VehicleNumber,
			&m.RouteSource, &m.RouteDestination,
		)
		if err != nil {
			return nil, 0, err
		}

		if bookingID.Valid {
			m.BookingID = &bookingID.String
		}
		if driverID.Valid {
			m.DriverID = &driverID.String
		}
		if vehicleID.Valid {
			m.VehicleID = &vehicleID.String
		}
		if arrivalTime.Valid {
			m.ArrivalTime = &arrivalTime.Time
		}
		if remarks.Valid {
			m.Remarks = remarks.String
		}
		if startedAt.Valid {
			m.StartedAt = &startedAt.Time
		}
		if reachedPickupAt.Valid {
			m.ReachedPickupAt = &reachedPickupAt.Time
		}
		if inTransitAt.Valid {
			m.InTransitAt = &inTransitAt.Time
		}
		if deliveredAt.Valid {
			m.DeliveredAt = &deliveredAt.Time
		}
		if completedAt.Valid {
			m.CompletedAt = &completedAt.Time
		}

		readModels = append(readModels, m)
	}

	countSQL := fmt.Sprintf(`
SELECT COUNT(*)
FROM trips t
LEFT JOIN drivers d ON t.driver_id = d.id
LEFT JOIN vehicles v ON t.vehicle_id = v.id
LEFT JOIN routes r ON t.route_id = r.id
WHERE t.tenant_id = ?
  AND %s
  AND (? = '' OR t.trip_number LIKE ? OR d.first_name LIKE ? OR d.last_name LIKE ? OR v.registration_number LIKE ? OR r.source LIKE ? OR r.destination LIKE ?)
  AND (? = '' OR t.status = ?)`, driverClause)

	countArgs := []interface{}{string(tenantID)}
	for _, id := range resolvedIDs {
		countArgs = append(countArgs, id)
	}
	countArgs = append(countArgs, query, qPattern, qPattern, qPattern, qPattern, qPattern, qPattern, status, status)

	var count int64
	err = r.dbConn.QueryRowContext(ctx, countSQL, countArgs...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	return readModels, count, nil
}
