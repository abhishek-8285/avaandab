package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/domain"
	"transport-app/internal/shared"
)

// ─── Test 1: Settlement persistence + bonus ───────────────────────────────────

func TestSettlement_PersistenceAndBonus(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	// Seed driver
	seedScorecardDriver(t, dbConn, "d-st-1", "DRV-ST1", "+919800000001", futureDate)
	_, err := dbConn.Exec(`UPDATE drivers SET score = 90.0, tier = 'A' WHERE id = 'd-st-1'`)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		seedBehaviourEvent(t, dbConn, "be-st-1-"+string(rune('a'+i)), "d-st-1", "speeding", "low", 1.0, i, "")
	}

	// Seed route & trip
	_, err = dbConn.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
		VALUES ('r-st-1','Mumbai','Pune',150.0,4.0,5000.0,'tenant-1')`)
	require.NoError(t, err)
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
		VALUES ('t-st-1','TRP-ST-01','r-st-1',NULL,'d-st-1',datetime('now','-2 hours'),'in_transit','tenant-1')`)
	require.NoError(t, err)

	// Create settlement: Gross 5000, Advances 500, Deductions 500
	// Pre-bonus Net = 5000 - 500 - 500 = 4000
	// Tier A bonus = 5% of 4000 = 200
	// Net Payout = 4000 + 200 = 4200
	record, err := svcs.Settlements.CreateSettlementForTrip(ctx, domain.TripID("t-st-1"), 5000.0, 500.0, 500.0)
	require.NoError(t, err)
	assert.Equal(t, domain.TripID("t-st-1"), record.TripID)
	assert.Equal(t, domain.DriverID("d-st-1"), record.DriverID)
	assert.InDelta(t, 5000.0, record.GrossFare, 0.01)
	assert.InDelta(t, 500.0, record.AdvancesKharcha, 0.01)
	assert.InDelta(t, 500.0, record.Deductions, 0.01)
	assert.InDelta(t, 200.0, record.PerformanceBonus, 0.01)
	assert.InDelta(t, 4200.0, record.NetPayout, 0.01)
	assert.Equal(t, "pending", record.Status)

	// Verify persistence in DB
	var grossFare, advances, deductions, bonus, netPayout float64
	var status string
	err = dbConn.QueryRow(`
		SELECT gross_fare, advances_kharcha, deductions, performance_bonus, net_payout, status
		FROM driver_settlements WHERE trip_id = 't-st-1'`).
		Scan(&grossFare, &advances, &deductions, &bonus, &netPayout, &status)
	require.NoError(t, err)
	assert.InDelta(t, 5000.0, grossFare, 0.01)
	assert.InDelta(t, 500.0, advances, 0.01)
	assert.InDelta(t, 500.0, deductions, 0.01)
	assert.InDelta(t, 200.0, bonus, 0.01)
	assert.InDelta(t, 4200.0, netPayout, 0.01)
	assert.Equal(t, "pending", status)
}

// ─── Test 2: Idempotency on trip_id UNIQUE ────────────────────────────────────

func TestSettlement_Idempotent(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-st-2", "DRV-ST2", "+919800000002", futureDate)
	_, err := dbConn.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
		VALUES ('r-st-2','Mumbai','Pune',150.0,4.0,5000.0,'tenant-1')`)
	require.NoError(t, err)
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
		VALUES ('t-st-2','TRP-ST-02','r-st-2',NULL,'d-st-2',datetime('now','-2 hours'),'in_transit','tenant-1')`)
	require.NoError(t, err)

	first, err := svcs.Settlements.CreateSettlementForTrip(ctx, domain.TripID("t-st-2"), 3000.0, 200.0, 100.0)
	require.NoError(t, err)

	second, err := svcs.Settlements.CreateSettlementForTrip(ctx, domain.TripID("t-st-2"), 3000.0, 200.0, 100.0)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "idempotent call must return existing settlement")

	var count int
	require.NoError(t, dbConn.QueryRow(`SELECT COUNT(*) FROM driver_settlements WHERE trip_id = 't-st-2'`).Scan(&count))
	assert.Equal(t, 1, count, "exact 1 settlement row must exist for the trip")
}

// ─── Test 3: Kharcha approval updates settlement ──────────────────────────────

func TestSettlement_KharchaApproval(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-st-3", "DRV-ST3", "+919800000003", futureDate)
	_, err := dbConn.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
		VALUES ('r-st-3','Mumbai','Pune',150.0,4.0,5000.0,'tenant-1')`)
	require.NoError(t, err)
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
		VALUES ('t-st-3','TRP-ST-03','r-st-3',NULL,'d-st-3',datetime('now','-2 hours'),'in_transit','tenant-1')`)
	require.NoError(t, err)

	// Create settlement: Fare 5000, Advances 500, Deductions 500 → Net 4000
	_, err = svcs.Settlements.CreateSettlementForTrip(ctx, domain.TripID("t-st-3"), 5000.0, 500.0, 500.0)
	require.NoError(t, err)

	// Create kharcha expense: 500
	expID, err := svcs.Kharcha.CreateExpense(ctx, "t-st-3", "d-st-3", "toll", 500.0, "Toll plaza", "", 0.0)
	require.NoError(t, err)

	// Approve expense
	err = svcs.Kharcha.ApproveExpense(ctx, expID, "admin-01")
	require.NoError(t, err)

	// Check settlement in DB: advances_kharcha should be 500 + 500 = 1000, net_payout = 4000 - 500 = 3500
	var advances, netPayout float64
	err = dbConn.QueryRow(`
		SELECT advances_kharcha, net_payout
		FROM driver_settlements WHERE trip_id = 't-st-3'`).Scan(&advances, &netPayout)
	require.NoError(t, err)
	assert.InDelta(t, 1000.0, advances, 0.01)
	assert.InDelta(t, 3500.0, netPayout, 0.01)
}

// ─── Test 4: Bonus tiers ──────────────────────────────────────────────────────

func TestSettlement_BonusTiers(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	tiers := []struct {
		driverID    string
		code        string
		score       float64
		tier        string
		events      int
		expectBonus float64 // on 1000 net
	}{
		{"d-tier-a", "DRV-TA", 90.0, "A", 3, 50.0},    // 5% of 1000
		{"d-tier-b", "DRV-TB", 75.0, "B", 3, 20.0},    // 2% of 1000
		{"d-tier-c", "DRV-TC", 50.0, "C", 3, 0.0},     // 0%
		{"d-tier-cold", "DRV-TD", 100.0, "A", 1, 0.0}, // <3 events -> cold start = 0%
	}

	for i, tc := range tiers {
		tripID := domain.TripID("t-tier-" + string(rune('1'+i)))
		seedScorecardDriver(t, dbConn, tc.driverID, tc.code, "+91980000001"+string(rune('0'+i)), futureDate)
		_, err := dbConn.Exec(`UPDATE drivers SET score=?, tier=? WHERE id=?`, tc.score, tc.tier, tc.driverID)
		require.NoError(t, err)
		for j := 0; j < tc.events; j++ {
			seedBehaviourEvent(t, dbConn, "be-"+tc.driverID+"-"+string(rune('a'+j)), tc.driverID, "speeding", "low", 1.0, j, "")
		}

		_, err = dbConn.Exec(`INSERT OR IGNORE INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
			VALUES (?,'Mumbai','Pune',150.0,4.0,5000.0,'tenant-1')`, "r-"+string(tripID))
		require.NoError(t, err)
		_, err = dbConn.Exec(`
			INSERT INTO trips (id, trip_number, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
			VALUES (?,'TRP-'||?,? ,NULL,?,datetime('now','-2 hours'),'in_transit','tenant-1')`,
			string(tripID), string(tripID), "r-"+string(tripID), tc.driverID)
		require.NoError(t, err)

		rec, err := svcs.Settlements.CreateSettlementForTrip(ctx, tripID, 1000.0, 0.0, 0.0)
		require.NoError(t, err)
		assert.InDelta(t, tc.expectBonus, rec.PerformanceBonus, 0.01, "tier %s bonus check", tc.tier)
		assert.InDelta(t, 1000.0+tc.expectBonus, rec.NetPayout, 0.01)
	}
}

// ─── Test 5: ProcessFinancialSettlement uses real data ─────────────────────────

func TestSettlement_ProcessFinancial(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	// Seed customer (customers table has no tenant_id column)
	_, err := dbConn.Exec(`
		INSERT INTO customers (id, name, phone, email)
		VALUES ('c-pfs-1','Customer 1','+919800000099','c1@test.com')`)
	require.NoError(t, err)

	// Seed driver
	seedScorecardDriver(t, dbConn, "d-pfs-1", "DRV-PFS1", "+919800000055", futureDate)
	_, err = dbConn.Exec(`UPDATE drivers SET score = 90.0, tier = 'A' WHERE id = 'd-pfs-1'`)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		seedBehaviourEvent(t, dbConn, "be-pfs-1-"+string(rune('a'+i)), "d-pfs-1", "speeding", "low", 1.0, i, "")
	}

	// Seed route
	_, err = dbConn.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
		VALUES ('r-pfs-1','Delhi','Jaipur',250.0,5.0,8000.0,'tenant-1')`)
	require.NoError(t, err)

	// Seed booking with price = 8000.0
	_, err = dbConn.Exec(`
		INSERT INTO bookings (id, booking_number, customer_id, route_id, vehicle_type, pickup_date, price, status, tenant_id)
		VALUES ('b-pfs-1','BKG-PFS-01','c-pfs-1','r-pfs-1','truck',datetime('now','-3 hours'),8000.0,'confirmed','tenant-1')`)
	require.NoError(t, err)

	// Seed trip linked to booking
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, booking_id, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
		VALUES ('t-pfs-1','TRP-PFS-01','b-pfs-1','r-pfs-1',NULL,'d-pfs-1',datetime('now','-2 hours'),'completed','tenant-1')`)
	require.NoError(t, err)

	// Seed approved kharcha of 1200.0
	_, err = dbConn.Exec(`
		INSERT INTO driver_expenses (id, trip_id, driver_id, expense_type, category, amount, description, status)
		VALUES ('exp-pfs-1','t-pfs-1','d-pfs-1','fuel','fuel',1200.0,'Diesel refill','approved')`)
	require.NoError(t, err)

	// Process financial settlement
	record, err := svcs.Settlements.ProcessFinancialSettlement(ctx, domain.TripID("t-pfs-1"), "PAY-REF-999")
	require.NoError(t, err)

	// Gross fare = 8000.0 (from booking, not hardcoded 1000.0)
	assert.InDelta(t, 8000.0, record.GrossFare, 0.01)
	// Advances = 1200.0 (from approved kharcha, not hardcoded 200.0)
	assert.InDelta(t, 1200.0, record.AdvancesKharcha, 0.01)
	assert.Equal(t, "paid", record.Status)
	assert.Equal(t, "PAY-REF-999", *record.PaymentRef)

	// Pre-bonus net = 8000 - 1200 - 50 = 6750
	// Tier A bonus = 5% of 6750 = 337.5
	// Final net = 6750 + 337.5 = 7087.5
	assert.InDelta(t, 337.5, record.PerformanceBonus, 0.01)
	assert.InDelta(t, 7087.5, record.NetPayout, 0.01)

	// DB row should be status = 'paid'
	var dbStatus, dbPayRef string
	err = dbConn.QueryRow(`SELECT status, payment_ref FROM driver_settlements WHERE trip_id = 't-pfs-1'`).Scan(&dbStatus, &dbPayRef)
	require.NoError(t, err)
	assert.Equal(t, "paid", dbStatus)
	assert.Equal(t, "PAY-REF-999", dbPayRef)
}

// ─── Test 6: Negative net payout floored at 0 ──────────────────────────────────

func TestSettlement_NegativeFloor(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-st-neg", "DRV-NEG", "+919800000077", futureDate)
	_, err := dbConn.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
		VALUES ('r-st-neg','Mumbai','Pune',150.0,4.0,5000.0,'tenant-1')`)
	require.NoError(t, err)
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
		VALUES ('t-st-neg','TRP-ST-NEG','r-st-neg',NULL,'d-st-neg',datetime('now','-2 hours'),'in_transit','tenant-1')`)
	require.NoError(t, err)

	// Fare: 100, Advances: 500, Deductions: 200 -> Raw net = -600 -> Clamped to 0
	rec, err := svcs.Settlements.CreateSettlementForTrip(ctx, domain.TripID("t-st-neg"), 100.0, 500.0, 200.0)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, rec.NetPayout, 0.01, "negative net payout must clamp to 0")
}

// ─── Test 7: Full Phase 5 regression ──────────────────────────────────────────

func TestPhase5_FullRegression(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	// 1. Setup vehicle with fuel sensor
	_, err := dbConn.Exec(`
		INSERT INTO vehicles (id, vehicle_number, registration_number, vehicle_type,
		  capacity, fuel_type, insurance_expiry, fitness_expiry, permit_expiry, puc_expiry,
		  fuel_sensor_fitted, tank_capacity_litres, current_mileage, status, tenant_id)
		VALUES ('v-reg-1','TRK-REG1','MH01REG001','truck',15.0,'diesel',?,?,?,?,1,200.0,4.0,'available','tenant-1')`,
		futureDate, futureDate, futureDate, futureDate)
	require.NoError(t, err)

	// 2. Setup driver
	seedScorecardDriver(t, dbConn, "d-reg-1", "DRV-REG1", "+919800000088", futureDate)

	// 3. Setup route and trip: distance 200 km @ 4.0 kmpl = 50.0 L expected
	_, err = dbConn.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare, tenant_id)
		VALUES ('r-reg-1','Mumbai','Pune',200.0,4.0,15000.0,'tenant-1')`)
	require.NoError(t, err)
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, route_id, vehicle_id, driver_id, departure_time, status, tenant_id)
		VALUES ('t-reg-1','TRP-REG-01','r-reg-1','v-reg-1','d-reg-1',datetime('now','-5 hours'),'in_transit','tenant-1')`)
	require.NoError(t, err)

	// 4. Ingest fuel refill event: 50L refill (from 30% to 55% of 200L tank = 50L)
	_, err = dbConn.Exec(`
		INSERT INTO fuel_events (id, vehicle_id, trip_id, driver_id, event_type,
		  fuel_level_before, fuel_level_after, odometer_before, odometer_after,
		  estimated_litres, confidence, occurred_at)
		VALUES ('fe-reg-1','v-reg-1','t-reg-1','d-reg-1','refill_detected',
		  30.0, 55.0, 1000.0, 1000.0, 50.0, 0.95, datetime('now','-2 hours'))`)
	require.NoError(t, err)

	// 5. Driver claims 50L fuel expense -> Audit pass -> passed
	expID, err := svcs.Kharcha.CreateExpense(ctx, "t-reg-1", "d-reg-1", "fuel", 5000.0, "Fuel fill 50L", "", 50.0)
	require.NoError(t, err)

	auditedCount, err := svcs.FuelAudit.AuditPendingClaims(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, auditedCount)

	var auditStatus string
	require.NoError(t, dbConn.QueryRow(`SELECT audit_status FROM driver_expenses WHERE id = ?`, expID).Scan(&auditStatus))
	assert.Equal(t, "passed", auditStatus)

	// 6. Record driver behaviour events and compute score -> Tier A
	for i := 0; i < 3; i++ {
		seedBehaviourEvent(t, dbConn, "be-reg-1-"+string(rune('a'+i)), "d-reg-1", "speeding", "low", 0.5, i, "")
	}
	scoreResult, err := svcs.Scorecard.RecomputeDriverScore(ctx, "d-reg-1")
	require.NoError(t, err)
	assert.Equal(t, "A", scoreResult.Tier)
	assert.GreaterOrEqual(t, scoreResult.Score, 85.0)

	// 7. Create initial settlement for trip: Gross 15000, Advances 0, Deductions 1000
	// Pre-bonus Net = 14000 -> 5% bonus = 700 -> Net = 14700
	settlement, err := svcs.Settlements.CreateSettlementForTrip(ctx, domain.TripID("t-reg-1"), 15000.0, 0.0, 1000.0)
	require.NoError(t, err)
	assert.InDelta(t, 700.0, settlement.PerformanceBonus, 0.01)
	assert.InDelta(t, 14700.0, settlement.NetPayout, 0.01)

	// 8. Approve the fuel expense (amount 5000) -> Kharcha deduction from settlement
	err = svcs.Kharcha.ApproveExpense(ctx, expID, "admin-01")
	require.NoError(t, err)

	// 9. Verify settlement row updated with kharcha deduction
	var advKharcha, finalNet float64
	err = dbConn.QueryRow(`
		SELECT advances_kharcha, net_payout
		FROM driver_settlements WHERE trip_id = 't-reg-1'`).Scan(&advKharcha, &finalNet)
	require.NoError(t, err)
	assert.InDelta(t, 5000.0, advKharcha, 0.01)
	assert.InDelta(t, 14700.0-5000.0, finalNet, 0.01)
}
