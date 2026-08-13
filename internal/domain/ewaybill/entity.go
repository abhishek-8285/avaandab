package ewaybill

import (
	"errors"
	"strings"
	"time"

	"transport-app/internal/domain/types"
)

var (
	ErrInvalidEWBNumber = errors.New("eway bill number must be at least 12 characters")
	ErrEWBExpired       = errors.New("eway bill is expired")
)

// EWayBill represents a government GST E-Way Bill for a transport trip.
type EWayBill struct {
	ID             string
	TripID         types.TripID
	EWBNumber      string
	IRN            *string
	GenerationDate time.Time
	ValidUntil     time.Time
	TransporterID  *string
	VehicleNumber  *string
	Status         string
	RawResponse    *string
	CreatedAt      time.Time
}

// IsActive checks if the E-Way Bill is active and not expired.
func (e EWayBill) IsActive(now time.Time) bool {
	if e.Status == "cancelled" || e.Status == "expired" {
		return false
	}
	return now.Before(e.ValidUntil)
}

// Validate validates EWB fields.
func (e EWayBill) Validate(now time.Time) error {
	if len(strings.TrimSpace(e.EWBNumber)) < 12 {
		return ErrInvalidEWBNumber
	}
	if !now.Before(e.ValidUntil) {
		return ErrEWBExpired
	}
	return nil
}
