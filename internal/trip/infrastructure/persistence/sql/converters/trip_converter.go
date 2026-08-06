package converters

import (
	"database/sql"
	"time"

	"transport-app/internal/shared"
	"transport-app/internal/trip/domain/aggregate"
)

type SQLTripModel struct {
	ID            string
	TripNumber    string
	BookingID     sql.NullString
	DriverID      sql.NullString
	VehicleID     sql.NullString
	RouteID       string
	DepartureTime time.Time
	ArrivalTime   sql.NullTime
	Status        string
	Remarks       sql.NullString
	TenantID      string
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func MapToAggregate(m SQLTripModel) *aggregate.TripAggregate {
	var bookingID *string
	if m.BookingID.Valid {
		val := m.BookingID.String
		bookingID = &val
	}

	var driverID *string
	if m.DriverID.Valid {
		val := m.DriverID.String
		driverID = &val
	}

	var vehicleID *string
	if m.VehicleID.Valid {
		val := m.VehicleID.String
		vehicleID = &val
	}

	var arrivalTime *time.Time
	if m.ArrivalTime.Valid {
		val := m.ArrivalTime.Time
		arrivalTime = &val
	}

	var remarks string
	if m.Remarks.Valid {
		remarks = m.Remarks.String
	}

	return &aggregate.TripAggregate{
		ID:            aggregate.TripID(m.ID),
		TenantID:      shared.TenantID(m.TenantID),
		TripNumber:    m.TripNumber,
		BookingID:     bookingID,
		DriverID:      driverID,
		VehicleID:     vehicleID,
		RouteID:       m.RouteID,
		DepartureTime: m.DepartureTime,
		ArrivalTime:   arrivalTime,
		Status:        aggregate.TripStatus(m.Status),
		Remarks:       remarks,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		Version:       m.Version,
	}
}

func MapToPersistence(agg *aggregate.TripAggregate) SQLTripModel {
	var bookingID sql.NullString
	if agg.BookingID != nil {
		bookingID = sql.NullString{String: *agg.BookingID, Valid: true}
	}

	var driverID sql.NullString
	if agg.DriverID != nil {
		driverID = sql.NullString{String: *agg.DriverID, Valid: true}
	}

	var vehicleID sql.NullString
	if agg.VehicleID != nil {
		vehicleID = sql.NullString{String: *agg.VehicleID, Valid: true}
	}

	var arrivalTime sql.NullTime
	if agg.ArrivalTime != nil {
		arrivalTime = sql.NullTime{Time: *agg.ArrivalTime, Valid: true}
	}

	var remarks sql.NullString
	if agg.Remarks != "" {
		remarks = sql.NullString{String: agg.Remarks, Valid: true}
	}

	return SQLTripModel{
		ID:            string(agg.ID),
		TripNumber:    agg.TripNumber,
		BookingID:     bookingID,
		DriverID:      driverID,
		VehicleID:     vehicleID,
		RouteID:       agg.RouteID,
		DepartureTime: agg.DepartureTime,
		ArrivalTime:   arrivalTime,
		Status:        string(agg.Status),
		Remarks:       remarks,
		TenantID:      string(agg.TenantID),
		Version:       agg.Version,
		CreatedAt:     agg.CreatedAt,
		UpdatedAt:     agg.UpdatedAt,
	}
}
