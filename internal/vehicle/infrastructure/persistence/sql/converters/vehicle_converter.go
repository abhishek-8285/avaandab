package converters

import (
	"database/sql"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/shared"
	"transport-app/internal/vehicle/domain"
	"transport-app/internal/vehicle/domain/aggregate"
)

// ToDomain converts db.Vehicle to *aggregate.VehicleAggregate.
func ToDomain(v db.Vehicle) *aggregate.VehicleAggregate {
	return aggregate.NewVehicleAggregate(
		aggregate.VehicleID(v.ID),
		shared.TenantID(v.TenantID),
		v.RegistrationNumber,
		v.VehicleNumber,
		aggregate.VehicleType(v.VehicleType),
		v.Capacity,
		aggregate.FuelType(v.FuelType),
		v.InsuranceExpiry,
		v.FitnessExpiry,
		v.PermitExpiry,
		aggregate.VehicleStatus(v.Status),
		getFloat64Pointer(v.CurrentMileage),
		v.CreatedAt,
	)
}

// ToReadModel converts db.Vehicle to domain.VehicleReadModel.
func ToReadModel(v db.Vehicle) domain.VehicleReadModel {
	return domain.VehicleReadModel{
		ID:                 v.ID,
		RegistrationNumber: v.RegistrationNumber,
		VehicleNumber:      v.VehicleNumber,
		VehicleType:        v.VehicleType,
		Capacity:           v.Capacity,
		FuelType:           v.FuelType,
		InsuranceExpiry:    v.InsuranceExpiry,
		FitnessExpiry:      v.FitnessExpiry,
		PermitExpiry:       v.PermitExpiry,
		Status:             v.Status,
		CurrentMileage:     getFloat64Pointer(v.CurrentMileage),
		CreatedAt:          v.CreatedAt,
		UpdatedAt:          v.UpdatedAt,
	}
}

func getFloat64Pointer(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}
