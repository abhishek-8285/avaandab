package types

import "time"

// Typed IDs for domain entities
type UserID string
type DriverID string
type VehicleID string
type CustomerID string
type RouteID string
type BookingID string
type TripID string
type InvoiceID string
type PaymentID string
type FileID string
type SessionID string

// String converts a typed ID to its string representation.
func (id UserID) String() string    { return string(id) }
func (id DriverID) String() string  { return string(id) }
func (id VehicleID) String() string { return string(id) }
func (id CustomerID) String() string { return string(id) }
func (id RouteID) String() string    { return string(id) }
func (id BookingID) String() string  { return string(id) }
func (id TripID) String() string     { return string(id) }
func (id InvoiceID) String() string  { return string(id) }
func (id PaymentID) String() string  { return string(id) }
func (id FileID) String() string    { return string(id) }
func (id SessionID) String() string { return string(id) }

// File represents an uploaded file.
type File struct {
	ID             FileID
	Filename       string
	OriginalName   string
	Path           string
	Size           int64
	MimeType       string
	UploadableType string
	UploadableID   *string
	CreatedAt      time.Time
}

// Session represents an authenticated user session.
type Session struct {
	ID        SessionID
	UserID    UserID
	TokenHash string
	ExpiresAt time.Time
	UserAgent *string
	IPAddress *string
	CreatedAt time.Time
}
