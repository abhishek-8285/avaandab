package dashboard

import "time"

// OperationalDashboardSummary represents live, modular dashboard stats (no fake cards).
type OperationalDashboardSummary struct {
	Timestamp         time.Time `json:"timestamp"`
	PendingBookings   int64     `json:"pending_bookings_count"`    // Booking module
	TodaysTripsCount  int64     `json:"todays_trips_count"`        // Trip module
	TotalOutstanding  float64   `json:"total_outstanding_amount"`  // Invoice module
	TodaysCollections float64   `json:"todays_collections_amount"` // Payment module
}
