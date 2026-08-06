package sqlite

import (
	"context"
	"database/sql"

	"transport-app/internal/domain"

	db "transport-app/db/generated/sqlite"

	"transport-app/internal/shared"
)

// VehicleRepository implementation

func (r *SQLRepository) CreateVehicle(ctx context.Context, vehicle domain.Vehicle) (domain.Vehicle, error) {
	created, err := r.Q(ctx).CreateVehicle(ctx, db.CreateVehicleParams{
		ID:                 string(vehicle.ID),
		RegistrationNumber: vehicle.RegistrationNumber,
		VehicleNumber:      vehicle.VehicleNumber,
		VehicleType:        string(vehicle.VehicleType),
		Capacity:           vehicle.Capacity,
		FuelType:           string(vehicle.FuelType),
		InsuranceExpiry:    vehicle.InsuranceExpiry,
		FitnessExpiry:      vehicle.FitnessExpiry,
		PermitExpiry:       vehicle.PermitExpiry,
		Status:             string(vehicle.Status),
		CurrentMileage:     nullFloat(vehicle.CurrentMileage),
		TenantID: string(shared.TenantIDFromContext(ctx)),
	})
	if err != nil {
		return domain.Vehicle{}, err
	}
	v := db.Vehicle{
		ID:                 created.ID,
		RegistrationNumber: created.RegistrationNumber,
		VehicleNumber:      created.VehicleNumber,
		VehicleType:        created.VehicleType,
		Capacity:           created.Capacity,
		FuelType:           created.FuelType,
		InsuranceExpiry:    created.InsuranceExpiry,
		FitnessExpiry:      created.FitnessExpiry,
		PermitExpiry:       created.PermitExpiry,
		Status:             created.Status,
		CurrentMileage:     created.CurrentMileage,
		CreatedAt:          created.CreatedAt,
		UpdatedAt:          created.UpdatedAt,
	}
	return toDomainVehicle(v), nil
}

func (r *SQLRepository) GetVehicleByID(ctx context.Context, id domain.VehicleID) (domain.Vehicle, error) {
	row, err := r.Q(ctx).GetVehicleByID(ctx, db.GetVehicleByIDParams{
		ID:       string(id),
		TenantID: string(shared.TenantIDFromContext(ctx)),
	})
	if err != nil {
		return domain.Vehicle{}, err
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
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	return toDomainVehicle(v), nil
}

func (r *SQLRepository) GetVehicleByRegistration(ctx context.Context, regNum string) (domain.Vehicle, error) {
	row, err := r.Q(ctx).GetVehicleByRegistration(ctx, db.GetVehicleByRegistrationParams{
		RegistrationNumber: regNum,
		TenantID: string(shared.TenantIDFromContext(ctx)),
	})
	if err != nil {
		return domain.Vehicle{}, err
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
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	return toDomainVehicle(v), nil
}

func (r *SQLRepository) UpdateVehicle(ctx context.Context, vehicle domain.Vehicle) (domain.Vehicle, error) {
	updated, err := r.Q(ctx).UpdateVehicle(ctx, db.UpdateVehicleParams{
		RegistrationNumber: vehicle.RegistrationNumber,
		VehicleNumber:      vehicle.VehicleNumber,
		VehicleType:        string(vehicle.VehicleType),
		Capacity:           vehicle.Capacity,
		FuelType:           string(vehicle.FuelType),
		InsuranceExpiry:    vehicle.InsuranceExpiry,
		FitnessExpiry:      vehicle.FitnessExpiry,
		PermitExpiry:       vehicle.PermitExpiry,
		Status:             string(vehicle.Status),
		CurrentMileage:     nullFloat(vehicle.CurrentMileage),
		ID:                 string(vehicle.ID),
		TenantID: string(shared.TenantIDFromContext(ctx)),
	})
	if err != nil {
		return domain.Vehicle{}, err
	}
	v := db.Vehicle{
		ID:                 updated.ID,
		RegistrationNumber: updated.RegistrationNumber,
		VehicleNumber:      updated.VehicleNumber,
		VehicleType:        updated.VehicleType,
		Capacity:           updated.Capacity,
		FuelType:           updated.FuelType,
		InsuranceExpiry:    updated.InsuranceExpiry,
		FitnessExpiry:      updated.FitnessExpiry,
		PermitExpiry:       updated.PermitExpiry,
		Status:             updated.Status,
		CurrentMileage:     updated.CurrentMileage,
		CreatedAt:          updated.CreatedAt,
		UpdatedAt:          updated.UpdatedAt,
	}
	return toDomainVehicle(v), nil
}

func (r *SQLRepository) DeleteVehicle(ctx context.Context, id domain.VehicleID) error {
	return r.Q(ctx).DeleteVehicle(ctx, db.DeleteVehicleParams{
		ID:       string(id),
		TenantID: string(shared.TenantIDFromContext(ctx)),
	})
}

func (r *SQLRepository) SearchVehicles(ctx context.Context, query string, status string, limit, offset int) ([]domain.Vehicle, error) {
	rows, err := r.Q(ctx).SearchVehicles(ctx, db.SearchVehiclesParams{
		TenantID: string(shared.TenantIDFromContext(ctx)),
		Column2:  sql.NullString{String: query, Valid: true},
		Column3:  sql.NullString{String: query, Valid: true},
		Column4:  sql.NullString{String: query, Valid: true},
		Column5:  status,
		Status:   status,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Vehicle, len(rows))
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
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		}
		result[i] = toDomainVehicle(v)
	}
	return result, nil
}

func (r *SQLRepository) CountVehicles(ctx context.Context, query string, status string) (int64, error) {
	count, err := r.Q(ctx).CountVehicles(ctx, db.CountVehiclesParams{
		TenantID: string(shared.TenantIDFromContext(ctx)),
		Column2:  sql.NullString{String: query, Valid: true},
		Column3:  sql.NullString{String: query, Valid: true},
		Column4:  sql.NullString{String: query, Valid: true},
		Column5:  status,
		Status:   status,
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *SQLRepository) GetAvailableVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	rows, err := r.Q(ctx).GetAvailableVehicles(ctx, string(shared.TenantIDFromContext(ctx)))
	if err != nil {
		return nil, err
	}
	result := make([]domain.Vehicle, len(rows))
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
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		}
		result[i] = toDomainVehicle(v)
	}
	return result, nil
}
