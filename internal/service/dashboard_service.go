package service

import (
	"context"
	"fmt"
	"time"

	"transport-app/internal/domain"
	"transport-app/internal/repository"
)

// DashboardData contains all the data needed to render the dashboard.
type DashboardData struct {
	// Cards
	TodaysTripsCount       int64
	ActiveTripsCount       int64
	CompletedTripsCount    int64
	CancelledTripsCount    int64
	AvailableVehiclesCount int64
	AvailableDriversCount  int64
	PendingPaymentsCount   int
	MonthlyRevenue         float64

	// Tables
	UpcomingTrips  []repository.TripWithJoins
	RecentBookings []repository.BookingWithJoins
	RecentPayments []repository.PaymentWithInvoice
	RecentActivity []repository.AuditLogWithUser
}

// DashboardService provides dashboard data aggregation.
type DashboardService struct {
	baseService
}

// GetDashboardData returns aggregated data for the dashboard.
func (s *DashboardService) GetDashboardData(ctx context.Context) (DashboardData, error) {
	data := DashboardData{}
	today := time.Now().Format("2006-01-02")

	// Today's trips by status
	statusCounts, err := s.store.CountTripsByStatusForDate(ctx, today)
	if err != nil {
		// Table might not exist yet, handle gracefully
		statusCounts = make(map[domain.TripStatus]int64)
	}

	data.TodaysTripsCount = statusCounts[domain.TripScheduled] + statusCounts[domain.TripAssigned] +
		statusCounts[domain.TripStarted] + statusCounts[domain.TripCompleted] + statusCounts[domain.TripCancelled] +
		statusCounts[domain.TripDraft]
	data.ActiveTripsCount = statusCounts[domain.TripScheduled] + statusCounts[domain.TripAssigned] + statusCounts[domain.TripStarted]
	data.CompletedTripsCount = statusCounts[domain.TripCompleted]
	data.CancelledTripsCount = statusCounts[domain.TripCancelled]

	// Available vehicles
	vehicles, err := s.store.GetAvailableVehicles(ctx)
	if err == nil {
		data.AvailableVehiclesCount = int64(len(vehicles))
	}

	// Available drivers
	drivers, err := s.store.GetAvailableDrivers(ctx)
	if err == nil {
		data.AvailableDriversCount = int64(len(drivers))
	}

	// Pending invoices count
	pendingInvoices, err := s.store.GetPendingInvoices(ctx)
	if err == nil {
		data.PendingPaymentsCount = len(pendingInvoices)
	}

	// Monthly revenue
	monthlyRev, err := s.store.GetMonthlyRevenue(ctx)
	if err == nil {
		currentMonth := time.Now().Format("2006-01")
		for _, rev := range monthlyRev {
			if rev.Month == currentMonth {
				data.MonthlyRevenue = rev.Total
				break
			}
		}
	}

	// Upcoming trips (next 7 days, not cancelled)
	// For simplicity, we get today's trips
	upcomingTrips, err := s.store.GetTripsByDate(ctx, today)
	if err != nil {
		data.UpcomingTrips = []repository.TripWithJoins{}
	} else {
		data.UpcomingTrips = upcomingTrips
	}

	// Recent bookings (last 10)
	recentBookings, err := s.store.SearchBookings(ctx, "", "", 10, 0)
	if err != nil {
		data.RecentBookings = []repository.BookingWithJoins{}
	} else {
		data.RecentBookings = recentBookings
	}

	// Recent payments (last 10)
	recentPayments, err := s.store.SearchPayments(ctx, "", 10, 0)
	if err != nil {
		data.RecentPayments = []repository.PaymentWithInvoice{}
	} else {
		data.RecentPayments = recentPayments
	}

	// Recent activity (last 10 audit logs)
	recentLogs, err := s.store.ListAuditLogs(ctx, 10, 0)
	if err != nil {
		data.RecentActivity = []repository.AuditLogWithUser{}
	} else {
		data.RecentActivity = recentLogs
	}

	return data, nil
}

// GetUpcomingTrips returns trips starting today.
func (s *DashboardService) GetUpcomingTrips(ctx context.Context, date string) ([]repository.TripWithJoins, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	return s.store.GetTripsByDate(ctx, date)
}

// GetAvailableDriversForDashboard returns available drivers count.
func (s *DashboardService) GetAvailableDriversForDashboard(ctx context.Context) (int64, error) {
	drivers, err := s.store.GetAvailableDrivers(ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(drivers)), nil
}

// GetAvailableVehiclesForDashboard returns available vehicles count.
func (s *DashboardService) GetAvailableVehiclesForDashboard(ctx context.Context) (int64, error) {
	vehicles, err := s.store.GetAvailableVehicles(ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(vehicles)), nil
}

// GetTodayTripsSummary returns counts of trips by status for a given date.
func (s *DashboardService) GetTodayTripsSummary(ctx context.Context, date string) (map[domain.TripStatus]int64, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	return s.store.CountTripsByStatusForDate(ctx, date)
}

// GetPendingPaymentsCount returns the count of invoices with pending/partial payments.
func (s *DashboardService) GetPendingPaymentsCount(ctx context.Context) (int, error) {
	pending, err := s.store.GetPendingInvoices(ctx)
	if err != nil {
		return 0, err
	}
	return len(pending), nil
}

// GetMonthlyRevenueSummary returns the total revenue for the current month.
func (s *DashboardService) GetMonthlyRevenueSummary(ctx context.Context) (float64, error) {
	monthlyRev, err := s.store.GetMonthlyRevenue(ctx)
	if err != nil {
		return 0, err
	}
	currentMonth := time.Now().Format("2006-01")
	for _, rev := range monthlyRev {
		if rev.Month == currentMonth {
			return rev.Total, nil
		}
	}
	return 0, nil
}

func (s *DashboardService) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)
	today := time.Now().Format("2006-01-02")

	statusCounts, err := s.store.CountTripsByStatusForDate(ctx, today)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip stats: %w", err)
	}

	for status, count := range statusCounts {
		stats[string(status)] = count
	}

	vehicles, _ := s.store.GetAvailableVehicles(ctx)
	stats["available_vehicles"] = int64(len(vehicles))

	drivers, _ := s.store.GetAvailableDrivers(ctx)
	stats["available_drivers"] = int64(len(drivers))

	return stats, nil
}
