package dto

import "time"

// InvoiceDTO represents the JSON body returned by invoice API endpoints.
type InvoiceDTO struct {
	ID            string     `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	BookingID     string     `json:"booking_id"`
	CustomerID    string     `json:"customer_id"`
	TripID        *string    `json:"trip_id"`
	Subtotal      float64    `json:"subtotal"`
	Tax           float64    `json:"tax"`
	Discount      float64    `json:"discount"`
	Total         float64    `json:"total"`
	PaymentStatus string     `json:"payment_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
