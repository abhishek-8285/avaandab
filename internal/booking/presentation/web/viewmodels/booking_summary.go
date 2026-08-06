package viewmodels

import "time"

// BookingSummaryViewModel represents the presentation-optimized read model for HTML templates.
type BookingSummaryViewModel struct {
	ID               string
	BookingNumber    string
	CustomerID       string
	CustomerName     string
	CustomerCompany  string
	RouteID          string
	RouteSource      string
	RouteDestination string
	PickupDate       time.Time
	VehicleType      string
	Passengers       int64
	CargoWeight      *float64
	Price            float64
	Notes            string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
