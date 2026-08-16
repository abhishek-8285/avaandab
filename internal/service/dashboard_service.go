package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

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

// DashboardService provides dashboard data aggregation with high-performance memory caching.
type DashboardService struct {
	baseService
	cacheMu    sync.RWMutex
	cachedData DashboardData
	cachedAt   time.Time
	ttl        time.Duration
}

// GetDashboardData returns aggregated data for the dashboard with ultra-fast memory caching.
func (s *DashboardService) GetDashboardData(ctx context.Context) (DashboardData, error) {
	ttl := s.ttl
	if ttl == 0 {
		ttl = 3 * time.Second
	}

	s.cacheMu.RLock()
	if time.Since(s.cachedAt) < ttl {
		data := s.cachedData
		s.cacheMu.RUnlock()
		return data, nil
	}
	s.cacheMu.RUnlock()

	var (
		todaysTripsCount, activeTripsCount, completedTripsCount, cancelledTripsCount int64
		availVehiclesCount, availDriversCount                                        int64
		pendingPaymentsCount                                                         int
		monthlyRevenue                                                               float64
		upcomingTrips                                                                []repository.TripWithJoins
		recentBookings                                                               []repository.BookingWithJoins
		recentPayments                                                               []repository.PaymentWithInvoice
		recentActivity                                                               []repository.AuditLogWithUser
	)

	today := time.Now().Format("2006-01-02")
	g, ctx := errgroup.WithContext(ctx)

	// 1. Today's trips by status
	g.Go(func() error {
		statusCounts, err := s.store.CountTripsByStatusForDate(ctx, today)
		if err == nil {
			todaysTripsCount = statusCounts[domain.TripScheduled] + statusCounts[domain.TripAssigned] +
				statusCounts[domain.TripStarted] + statusCounts[domain.TripCompleted] + statusCounts[domain.TripCancelled] +
				statusCounts[domain.TripDraft]
			activeTripsCount = statusCounts[domain.TripScheduled] + statusCounts[domain.TripAssigned] + statusCounts[domain.TripStarted]
			completedTripsCount = statusCounts[domain.TripCompleted]
			cancelledTripsCount = statusCounts[domain.TripCancelled]
		}
		return nil
	})

	// 2. Available vehicles
	g.Go(func() error {
		vehicles, err := s.store.GetAvailableVehicles(ctx)
		if err == nil {
			availVehiclesCount = int64(len(vehicles))
		}
		return nil
	})

	// 3. Available drivers
	g.Go(func() error {
		drivers, err := s.store.GetAvailableDrivers(ctx)
		if err == nil {
			availDriversCount = int64(len(drivers))
		}
		return nil
	})

	// 4. Pending invoices count
	g.Go(func() error {
		pendingInvoices, err := s.store.GetPendingInvoices(ctx)
		if err == nil {
			pendingPaymentsCount = len(pendingInvoices)
		}
		return nil
	})

	// 5. Monthly revenue
	g.Go(func() error {
		monthlyRev, err := s.store.GetMonthlyRevenue(ctx)
		if err == nil {
			currentMonth := time.Now().Format("2006-01")
			for _, rev := range monthlyRev {
				if rev.Month == currentMonth {
					monthlyRevenue = rev.Total
					break
				}
			}
		}
		return nil
	})

	// 6. Upcoming trips
	g.Go(func() error {
		trips, err := s.store.GetTripsByDate(ctx, today)
		if err == nil {
			upcomingTrips = trips
		} else {
			upcomingTrips = []repository.TripWithJoins{}
		}
		return nil
	})

	// 7. Recent bookings
	g.Go(func() error {
		bookings, err := s.store.SearchBookings(ctx, "", "", 10, 0)
		if err == nil {
			recentBookings = bookings
		} else {
			recentBookings = []repository.BookingWithJoins{}
		}
		return nil
	})

	// 8. Recent payments
	g.Go(func() error {
		payments, err := s.store.SearchPayments(ctx, "", 10, 0)
		if err == nil {
			recentPayments = payments
		} else {
			recentPayments = []repository.PaymentWithInvoice{}
		}
		return nil
	})

	// 9. Recent activity
	g.Go(func() error {
		logs, err := s.store.ListAuditLogs(ctx, 10, 0)
		if err == nil {
			recentActivity = logs
		} else {
			recentActivity = []repository.AuditLogWithUser{}
		}
		return nil
	})

	_ = g.Wait()

	data := DashboardData{
		TodaysTripsCount:       todaysTripsCount,
		ActiveTripsCount:       activeTripsCount,
		CompletedTripsCount:    completedTripsCount,
		CancelledTripsCount:    cancelledTripsCount,
		AvailableVehiclesCount: availVehiclesCount,
		AvailableDriversCount:  availDriversCount,
		PendingPaymentsCount:   pendingPaymentsCount,
		MonthlyRevenue:         monthlyRevenue,
		UpcomingTrips:          upcomingTrips,
		RecentBookings:         recentBookings,
		RecentPayments:         recentPayments,
		RecentActivity:         recentActivity,
	}

	s.cacheMu.Lock()
	s.cachedData = data
	s.cachedAt = time.Now()
	s.cacheMu.Unlock()

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
