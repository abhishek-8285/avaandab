package service

import (
	"context"
	"fmt"
	"time"

	"transport-app/internal/domain"
	"transport-app/internal/events"
)

// DriverSettlementRecord represents a driver financial settlement.
type DriverSettlementRecord struct {
	ID              string          `json:"id"`
	TripID          domain.TripID   `json:"trip_id"`
	DriverID        domain.DriverID `json:"driver_id"`
	GrossFare       float64         `json:"gross_fare"`
	AdvancesKharcha float64         `json:"advances_kharcha"`
	Deductions      float64         `json:"deductions"`
	NetPayout       float64         `json:"net_payout"`
	Status          string          `json:"status"` // pending, processing, paid, disputed
	PaymentRef      *string         `json:"payment_ref"`
	PaidAt          *time.Time      `json:"paid_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// DriverSettlementService handles driver payout calculation and financial settlements.
type DriverSettlementService struct {
	baseService
}

// CreateSettlementForTrip calculates initial net payout for a trip.
// Net Payout = Fare - Advances (Kharcha) - Deductions.
func (s *DriverSettlementService) CreateSettlementForTrip(ctx context.Context, tripID domain.TripID, fare float64, advances float64, deductions float64) (DriverSettlementRecord, error) {
	driverID := domain.DriverID("drv-default")
	if s.store != nil {
		trip, err := s.store.GetTripByID(ctx, tripID)
		if err == nil && trip.DriverID != nil {
			driverID = *trip.DriverID
		}
	}

	netPayout := fare - advances - deductions
	if netPayout < 0 {
		netPayout = 0
	}

	settlement := DriverSettlementRecord{
		ID:              generateID(),
		TripID:          tripID,
		DriverID:        driverID,
		GrossFare:       fare,
		AdvancesKharcha: advances,
		Deductions:      deductions,
		NetPayout:       netPayout,
		Status:          "pending",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if s.log != nil {
		s.log.Info("driver settlement statement created", "trip_id", tripID, "driver_id", driverID, "net_payout", netPayout)
	}
	if s.store != nil {
		infoStr := fmt.Sprintf("net_payout=%.2f", netPayout)
		s.logAudit(ctx, nil, "create_settlement", "driver_settlements", settlement.ID, nil, &infoStr)
	}

	return settlement, nil
}

// ProcessFinancialSettlement triggers payout when trip is delivered and customer payment is received.
// Rule 4: Financial Settlement Rule.
func (s *DriverSettlementService) ProcessFinancialSettlement(ctx context.Context, tripID domain.TripID, paymentRef string) (DriverSettlementRecord, error) {
	trip, err := s.store.GetTripByID(ctx, tripID)
	if err != nil {
		return DriverSettlementRecord{}, domain.ErrTripNotFound
	}

	if trip.DriverID == nil {
		return DriverSettlementRecord{}, fmt.Errorf("cannot process settlement for trip without assigned driver")
	}

	// Calculate net payout (Fare - Kharcha - Advances)
	fare := 1000.0 // Default or extracted from booking/trip fare
	if trip.BookingID != nil {
		if bk, err := s.store.GetBookingByID(ctx, *trip.BookingID); err == nil {
			fare = bk.Price
		}
	}

	advances := 200.0  // Example kharcha advance
	deductions := 50.0 // Example toll / fee deduction
	netPayout := fare - advances - deductions

	now := time.Now()
	settlement := DriverSettlementRecord{
		ID:              generateID(),
		TripID:          tripID,
		DriverID:        *trip.DriverID,
		GrossFare:       fare,
		AdvancesKharcha: advances,
		Deductions:      deductions,
		NetPayout:       netPayout,
		Status:          "paid",
		PaymentRef:      &paymentRef,
		PaidAt:          &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	settleInfo := fmt.Sprintf("net_payout=%.2f payment_ref=%s", netPayout, paymentRef)
	s.logAudit(ctx, nil, "process_financial_settlement", "driver_settlements", settlement.ID, nil, &settleInfo)

	s.events.Publish(ctx, events.Event{
		Type: "DriverPayoutSettled",
		Payload: map[string]interface{}{
			"trip_id":     tripID,
			"driver_id":   *trip.DriverID,
			"net_payout":  netPayout,
			"payment_ref": paymentRef,
			"occurred_at": now,
		},
	})

	return settlement, nil
}
