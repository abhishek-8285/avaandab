package dto

import "time"

// BookingDTO represents the JSON body returned by API endpoints.
type BookingDTO struct {
	ID            string    `json:"id"`
	BookingNumber string    `json:"booking_number"`
	CustomerID    string    `json:"customer_id"`
	RouteID       string    `json:"route_id"`
	PickupDate    time.Time `json:"pickup_date"`
	VehicleType   string    `json:"vehicle_type"`
	Passengers    int64     `json:"passengers"`
	CargoWeight   *float64  `json:"cargo_weight"`
	Price         float64   `json:"price"`
	Notes         string    `json:"notes"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
