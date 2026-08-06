package sql

import (
	"context"
	"database/sql"
	"errors"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/repository"
	"transport-app/internal/shared"
	"transport-app/internal/shared/outbox"
	"transport-app/internal/vehicle/domain"
	"transport-app/internal/vehicle/domain/aggregate"
	"transport-app/internal/vehicle/infrastructure/persistence/sql/converters"
)

type vehicleRepository struct {
	dbConn *sql.DB
	q      *db.Queries
	outbox *outbox.OutboxWriter
}

// NewVehicleRepository creates a SQLite-backed implementation of VehicleRepository.
func NewVehicleRepository(dbConn *sql.DB) domain.VehicleRepository {
	return &vehicleRepository{
		dbConn: dbConn,
		q:      db.New(dbConn),
		outbox: outbox.NewOutboxWriter(dbConn),
	}
}

func (r *vehicleRepository) Q(ctx context.Context) *db.Queries {
	if tx := repository.TxFromContext(ctx); tx != nil {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *vehicleRepository) Save(ctx context.Context, v *aggregate.VehicleAggregate) error {
	var currentMileage sql.NullFloat64
	if v.CurrentMileage != nil {
		currentMileage = sql.NullFloat64{Float64: *v.CurrentMileage, Valid: true}
	}

	_, err := r.Q(ctx).GetVehicleByID(ctx, db.GetVehicleByIDParams{
		ID:       string(v.ID),
		TenantID: string(v.TenantID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = r.Q(ctx).CreateVehicle(ctx, db.CreateVehicleParams{
				ID:                 string(v.ID),
				RegistrationNumber: v.RegistrationNumber,
				VehicleNumber:      v.VehicleNumber,
				VehicleType:        string(v.VehicleType),
				Capacity:           v.Capacity,
				FuelType:           string(v.FuelType),
				InsuranceExpiry:    v.InsuranceExpiry,
				FitnessExpiry:      v.FitnessExpiry,
				PermitExpiry:       v.PermitExpiry,
				Status:             string(v.Status),
				CurrentMileage:     currentMileage,
				TenantID:           string(v.TenantID),
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		_, err = r.Q(ctx).UpdateVehicle(ctx, db.UpdateVehicleParams{
			RegistrationNumber: v.RegistrationNumber,
			VehicleNumber:      v.VehicleNumber,
			VehicleType:        string(v.VehicleType),
			Capacity:           v.Capacity,
			FuelType:           string(v.FuelType),
			InsuranceExpiry:    v.InsuranceExpiry,
			FitnessExpiry:      v.FitnessExpiry,
			PermitExpiry:       v.PermitExpiry,
			Status:             string(v.Status),
			CurrentMileage:     currentMileage,
			ID:                 string(v.ID),
			TenantID:           string(v.TenantID),
		})
		if err != nil {
			return err
		}
	}

	err = r.outbox.SaveEvents(ctx, string(v.ID), "Vehicle", v.Events())
	if err != nil {
		return err
	}
	v.ClearEvents()
	return nil
}

func (r *vehicleRepository) Find(ctx context.Context, id aggregate.VehicleID, tenantID shared.TenantID) (*aggregate.VehicleAggregate, error) {
	row, err := r.Q(ctx).GetVehicleByID(ctx, db.GetVehicleByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return nil, err
	}
	v := db.Vehicle{
		ID:                 row.ID,
		RegistrationNumber: row.RegistrationNumber,
		VehicleNumber:      row.VehicleNumber,
		VehicleType:        row.VehicleType,
		Capacity:           row.Capacity,
		FuelType:           row.FuelType,
		InsuranceExpiry:    row.InsuranceExpiry,
		FitnessExpiry:      row.FitnessExpiry,
		PermitExpiry:       row.PermitExpiry,
		Status:             row.Status,
		CurrentMileage:     row.CurrentMileage,
		TenantID:           row.TenantID,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	return converters.ToDomain(v), nil
}

func (r *vehicleRepository) GetReadModel(ctx context.Context, id aggregate.VehicleID, tenantID shared.TenantID) (domain.VehicleReadModel, error) {
	row, err := r.Q(ctx).GetVehicleByID(ctx, db.GetVehicleByIDParams{
		ID:       string(id),
		TenantID: string(tenantID),
	})
	if err != nil {
		return domain.VehicleReadModel{}, err
	}
	v := db.Vehicle{
		ID:                 row.ID,
		RegistrationNumber: row.RegistrationNumber,
		VehicleNumber:      row.VehicleNumber,
		VehicleType:        row.VehicleType,
		Capacity:           row.Capacity,
		FuelType:           row.FuelType,
		InsuranceExpiry:    row.InsuranceExpiry,
		FitnessExpiry:      row.FitnessExpiry,
		PermitExpiry:       row.PermitExpiry,
		Status:             row.Status,
		CurrentMileage:     row.CurrentMileage,
		TenantID:           row.TenantID,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	return converters.ToReadModel(v), nil
}

func (r *vehicleRepository) SearchReadModels(ctx context.Context, tenantID shared.TenantID, query string, status string, limit int, offset int) ([]domain.VehicleReadModel, int64, error) {
	rows, err := r.Q(ctx).SearchVehicles(ctx, db.SearchVehiclesParams{
		TenantID: string(tenantID),
		Column2:  sql.NullString{String: query, Valid: true},
		Column3:  sql.NullString{String: query, Valid: true},
		Column4:  sql.NullString{String: query, Valid: true},
		Column5:  status,
		Status:   status,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.Q(ctx).CountVehicles(ctx, db.CountVehiclesParams{
		TenantID: string(tenantID),
		Column2:  sql.NullString{String: query, Valid: true},
		Column3:  sql.NullString{String: query, Valid: true},
		Column4:  sql.NullString{String: query, Valid: true},
		Column5:  status,
		Status:   status,
	})
	if err != nil {
		return nil, 0, err
	}

	readModels := make([]domain.VehicleReadModel, len(rows))
	for i, row := range rows {
		v := db.Vehicle{
			ID:                 row.ID,
			RegistrationNumber: row.RegistrationNumber,
			VehicleNumber:      row.VehicleNumber,
			VehicleType:        row.VehicleType,
			Capacity:           row.Capacity,
			FuelType:           row.FuelType,
			InsuranceExpiry:    row.InsuranceExpiry,
			FitnessExpiry:      row.FitnessExpiry,
			PermitExpiry:       row.PermitExpiry,
			Status:             row.Status,
			CurrentMileage:     row.CurrentMileage,
			TenantID:           row.TenantID,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		}
		readModels[i] = converters.ToReadModel(v)
	}

	return readModels, total, nil
}
