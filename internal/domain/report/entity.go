package report

import (
	"time"

	"transport-app/internal/domain/types"
)

// DateRange represents the filter window for operational reporting.
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// TripsReport aggregates operational trip execution counts and statuses.
type TripsReport struct {
	Period            DateRange `json:"period"`
	TotalTrips        int64     `json:"total_trips"`
	ScheduledTrips    int64     `json:"scheduled_trips"`
	InTransitTrips    int64     `json:"in_transit_trips"`
	CompletedTrips    int64     `json:"completed_trips"`
	CancelledTrips    int64     `json:"cancelled_trips"`
	CompletionRatePct float64   `json:"completion_rate_pct"`
}

// RevenueReport summarizes billing totals and financial breakdown.
type RevenueReport struct {
	Period         DateRange `json:"period"`
	TotalInvoiced  float64   `json:"total_invoiced"`
	TotalCollected float64   `json:"total_collected"`
	TotalTax       float64   `json:"total_tax"`
	TotalDiscount  float64   `json:"total_discount"`
}

// OutstandingReport details uncollected balances across customers.
type OutstandingReport struct {
	TotalOutstanding  float64             `json:"total_outstanding"`
	OverdueAmount     float64             `json:"overdue_amount"`
	OutstandingCount  int64               `json:"outstanding_count"`
	OverdueCount      int64               `json:"overdue_count"`
	CustomerBreakdown []CustomerBalDetail `json:"customer_breakdown"`
}

type CustomerBalDetail struct {
	CustomerID   types.CustomerID `json:"customer_id"`
	CustomerName string           `json:"customer_name"`
	InvoiceCount int64            `json:"invoice_count"`
	Balance      float64          `json:"balance"`
}

// FleetUtilizationReport details vehicle usage and status breakdown.
type FleetUtilizationReport struct {
	Period                DateRange `json:"period"`
	TotalVehicles         int64     `json:"total_vehicles"`
	ActiveVehicles        int64     `json:"active_vehicles"`
	InMaintenanceVehicles int64     `json:"in_maintenance_vehicles"`
	IdleVehicles          int64     `json:"idle_vehicles"`
	UtilizationRatePct    float64   `json:"utilization_rate_pct"`
}

// DriverUtilizationReport details driver duty and assignment metrics.
type DriverUtilizationReport struct {
	Period             DateRange `json:"period"`
	TotalDrivers       int64     `json:"total_drivers"`
	ActiveOnTrip       int64     `json:"active_on_trip"`
	AvailableDrivers   int64     `json:"available_drivers"`
	OnLeaveDrivers     int64     `json:"on_leave_drivers"`
	UtilizationRatePct float64   `json:"utilization_rate_pct"`
}

// OperationalReportSummary combines all 5 required operational reports.
type OperationalReportSummary struct {
	GeneratedAt       time.Time               `json:"generated_at"`
	Trips             TripsReport             `json:"trips"`
	Revenue           RevenueReport           `json:"revenue"`
	Outstanding       OutstandingReport       `json:"outstanding"`
	FleetUtilization  FleetUtilizationReport  `json:"fleet_utilization"`
	DriverUtilization DriverUtilizationReport `json:"driver_utilization"`
}
