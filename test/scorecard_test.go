package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/service"
	"transport-app/internal/shared"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func seedScorecardDriver(t *testing.T, db *sql.DB, id, code, phone, futureDate string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT OR IGNORE INTO drivers (id, driver_id, first_name, last_name, phone,
		  license_number, license_expiry, status, tenant_id)
		VALUES (?,?,?,?,?,?,?,'available','tenant-1')`,
		id, code, "Test", code, phone, "LN-"+id, futureDate)
	require.NoError(t, err)
}

func seedBehaviourEvent(t *testing.T, db *sql.DB, id, driverID, evType, severity string, weight float64, daysAgo int, metadata string) {
	t.Helper()
	ts := time.Now().UTC().AddDate(0, 0, -daysAgo).Format("2006-01-02 15:04:05")
	if metadata == "" {
		metadata = "{}"
	}
	_, err := db.Exec(`
		INSERT INTO driver_behaviour_events
		  (id, driver_id, event_type, severity, weight, metadata, occurred_at)
		VALUES (?,?,?,?,?,?,?)`,
		id, driverID, evType, severity, weight, metadata, ts)
	require.NoError(t, err)
}

// ─── Test 1: Full flow — engine writes events → score computed ────────────────
// 3 speeding events (weight 8, severity medium) at 0/10/20 days ago →
//
//	penalty(0)  = 8 × 1.5 × 1.0      = 12.0
//	penalty(10) = 8 × 1.5 × (20/30)  = 8.0
//	penalty(20) = 8 × 1.5 × (10/30)  = 4.0
//	total = 24 → score = 76 → tier B
func TestScorecard_FullFlow(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-sc-1", "DRV-SC1", "+919200000001", futureDate)

	for i, daysAgo := range []int{0, 10, 20} {
		seedBehaviourEvent(t, dbConn,
			"be-sc-1-"+string(rune('a'+i)), "d-sc-1",
			"speeding", "medium", 8.0, daysAgo, "")
	}

	result, err := svcs.Scorecard.RecomputeDriverScore(ctx, "d-sc-1")
	require.NoError(t, err)
	assert.InDelta(t, 76.0, result.Score, 0.5, "score formula: 100 - 24 = 76")
	assert.Equal(t, "B", result.Tier)
	assert.False(t, result.InsufficientData)

	// drivers table denormalized
	var score float64
	var tier string
	require.NoError(t, dbConn.QueryRow(`SELECT score, tier FROM drivers WHERE id = 'd-sc-1'`).Scan(&score, &tier))
	assert.InDelta(t, 76.0, score, 0.5)
	assert.Equal(t, "B", tier)

	// driver_scores history row written
	var histCount int
	require.NoError(t, dbConn.QueryRow(`SELECT COUNT(*) FROM driver_scores WHERE driver_id = 'd-sc-1'`).Scan(&histCount))
	assert.GreaterOrEqual(t, histCount, 1)
}

// ─── Test 2: Fraud cap — score > 69 capped; resolve excludes event → 100 ─────
// Spec 03 §4.2, §11 item 6: fraud_cap=69 caps the score when ABOVE the cap.
// ResolveFraudEvent excludes the event entirely (not just lifts the cap).
//
// Event: fuel_theft_suspicion, severity=low, weight=25 → penalty = 25×1.0×1.0 = 25
// Raw score = 100 - 25 = 75 → 75 > 69 → capped at 69
// After resolve: event excluded → penalty = 0 → score = 100
func TestScorecard_FraudCap_E2E(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-sc-2", "DRV-SC2", "+919200000002", futureDate)

	// severity=low (mult=1.0): penalty = 25 × 1.0 × 1.0 = 25; raw score = 75; 75 > 69 → cap
	seedBehaviourEvent(t, dbConn, "be-sc-2-theft", "d-sc-2", "fuel_theft_suspicion", "low", 25.0, 0, `{}`)

	result, err := svcs.Scorecard.RecomputeDriverScore(ctx, "d-sc-2")
	require.NoError(t, err)
	assert.InDelta(t, 69.0, result.Score, 0.5, "raw=75 > fraud_cap=69 → must be capped at 69")
	assert.Equal(t, "C", result.Tier)

	// Resolve: event excluded on next recompute → penalty=0 → score=100
	require.NoError(t, svcs.Scorecard.ResolveFraudEvent(ctx, "be-sc-2-theft", "admin-01"))

	result2, err := svcs.Scorecard.RecomputeDriverScore(ctx, "d-sc-2")
	require.NoError(t, err)
	assert.InDelta(t, 100.0, result2.Score, 0.5, "resolved fraud excluded → score=100")

	// Metadata updated
	var metadata string
	require.NoError(t, dbConn.QueryRow(`SELECT metadata FROM driver_behaviour_events WHERE id = 'be-sc-2-theft'`).Scan(&metadata))
	assert.Contains(t, metadata, `"resolved":true`)
	assert.Contains(t, metadata, `"resolved_by":"admin-01"`)

	// Audit log written (action matches service constant)
	var auditCount int
	require.NoError(t, dbConn.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'fraud_event_resolved' AND record_id = 'be-sc-2-theft'`).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "audit log must record fraud resolution")
}

// ─── Test 3: Nightly sweep covers drivers with and without events ─────────────

func TestScorecard_NightlySweep_E2E(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	// Driver 1: 3 events in window → gets computed score
	seedScorecardDriver(t, dbConn, "d-sw-1", "DRV-SW1", "+919300000001", futureDate)
	seedBehaviourEvent(t, dbConn, "be-sw-1a", "d-sw-1", "speeding", "low", 5.0, 1, "")
	seedBehaviourEvent(t, dbConn, "be-sw-1b", "d-sw-1", "harsh_braking", "low", 4.0, 2, "")
	seedBehaviourEvent(t, dbConn, "be-sw-1c", "d-sw-1", "speeding", "medium", 8.0, 3, "")

	// Driver 2: no events, but previously had score 55 → sweep must decay to 100
	seedScorecardDriver(t, dbConn, "d-sw-2", "DRV-SW2", "+919300000002", futureDate)
	_, err := dbConn.Exec(`UPDATE drivers SET score = 55.0, tier = 'C' WHERE id = 'd-sw-2'`)
	require.NoError(t, err)

	// Driver 3: 4 events
	seedScorecardDriver(t, dbConn, "d-sw-3", "DRV-SW3", "+919300000003", futureDate)
	for i := 0; i < 4; i++ {
		seedBehaviourEvent(t, dbConn,
			"be-sw-3-"+string(rune('a'+i)), "d-sw-3",
			"idling", "low", 3.0, i*3, "")
	}

	require.NoError(t, svcs.Scorecard.RecomputeAllDrivers(ctx))

	// Drivers 1 & 3 must have scores
	for _, dID := range []string{"d-sw-1", "d-sw-3"} {
		var score float64
		require.NoError(t, dbConn.QueryRow(`SELECT COALESCE(score,0) FROM drivers WHERE id = ?`, dID).Scan(&score))
		assert.Greater(t, score, 0.0, "driver %s must have a computed score", dID)
	}

	// Driver 2 (no events) must decay to 100 (all penalties expired)
	var score2 float64
	require.NoError(t, dbConn.QueryRow(`SELECT COALESCE(score,0) FROM drivers WHERE id = 'd-sw-2'`).Scan(&score2))
	assert.InDelta(t, 100.0, score2, 0.1, "driver with no window events must decay to 100")
}

// ─── Test 4: Leaderboard ranking ──────────────────────────────────────────────

func TestScorecard_Leaderboard_E2E(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	for _, item := range []struct {
		id, code, phone string
		score           float64
		tier            string
		events          int
	}{
		{"d-lb-1", "DRV-LB1", "+919400000001", 92.0, "A", 5},
		{"d-lb-2", "DRV-LB2", "+919400000002", 75.0, "B", 3},
		{"d-lb-3", "DRV-LB3", "+919400000003", 60.0, "C", 1},
	} {
		seedScorecardDriver(t, dbConn, item.id, item.code, item.phone, futureDate)
		_, err := dbConn.Exec(`UPDATE drivers SET score = ?, tier = ? WHERE id = ?`, item.score, item.tier, item.id)
		require.NoError(t, err)
		for j := 0; j < item.events; j++ {
			seedBehaviourEvent(t, dbConn,
				"be-lb-"+item.id+"-"+string(rune('a'+j)),
				item.id, "speeding", "low", 2.0, j, "")
		}
	}

	rows, _, err := svcs.Scorecard.Leaderboard(ctx, "tenant-1", 20)
	require.NoError(t, err)

	found := map[string]service.LeaderboardRow{}
	for _, r := range rows {
		for _, id := range []string{"d-lb-1", "d-lb-2", "d-lb-3"} {
			if r.DriverID == id {
				found[id] = r
			}
		}
	}
	require.Len(t, found, 3, "all 3 seeded drivers must appear in leaderboard")

	// Verify rank order by score descending
	idxLB1, idxLB2, idxLB3 := -1, -1, -1
	for i, r := range rows {
		switch r.DriverID {
		case "d-lb-1":
			idxLB1 = i
		case "d-lb-2":
			idxLB2 = i
		case "d-lb-3":
			idxLB3 = i
		}
	}
	assert.Less(t, idxLB1, idxLB2, "score 92 must rank above 75")
	assert.Less(t, idxLB2, idxLB3, "score 75 must rank above 60")

	// insufficient_data: 1-event driver flagged, 5-event driver not
	assert.True(t, found["d-lb-3"].InsufficientData, "1-event driver must have insufficient_data")
	assert.False(t, found["d-lb-1"].InsufficientData, "5-event driver must not have insufficient_data")
}

// ─── Test 5: BonusForPayout per tier ──────────────────────────────────────────
// BonusForPayout returns 0 when driver has < minEvents (default 3) in window.
// Seed 3 events per driver to satisfy the cold-start guard.
func TestScorecard_BonusForPayout_E2E(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	for i, item := range []struct {
		id, code, phone string
		score           float64
		tier            string
	}{
		{"d-bp-1", "DRV-BP1", "+919500000001", 92.0, "A"},
		{"d-bp-2", "DRV-BP2", "+919500000002", 75.0, "B"},
		{"d-bp-3", "DRV-BP3", "+919500000003", 60.0, "C"},
	} {
		seedScorecardDriver(t, dbConn, item.id, item.code, item.phone, futureDate)
		_, err := dbConn.Exec(`UPDATE drivers SET score=?, tier=? WHERE id=?`, item.score, item.tier, item.id)
		require.NoError(t, err)
		// Seed 3 events to satisfy minEvents cold-start guard
		for j := 0; j < 3; j++ {
			seedBehaviourEvent(t, dbConn,
				"be-bp-"+item.id+"-"+string(rune('a'+j)),
				item.id, "speeding", "low", 1.0, j, "")
		}
		_ = i
	}

	// Defaults: A=5%, B=2%, C=0%
	assert.InDelta(t, 250.0, svcs.Scorecard.BonusForPayout(ctx, "d-bp-1", 5000.0), 0.01, "tier A 5%")
	assert.InDelta(t, 100.0, svcs.Scorecard.BonusForPayout(ctx, "d-bp-2", 5000.0), 0.01, "tier B 2%")
	assert.InDelta(t, 0.0, svcs.Scorecard.BonusForPayout(ctx, "d-bp-3", 5000.0), 0.01, "tier C 0%")
	assert.InDelta(t, 0.0, svcs.Scorecard.BonusForPayout(ctx, "nonexistent-driver", 5000.0), 0.01, "unknown driver → 0")
}

// ─── Test 6: DriverDetail breakdown ───────────────────────────────────────────

func TestScorecard_DriverDetail_E2E(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-dd-1", "DRV-DD1", "+919600000001", futureDate)
	seedBehaviourEvent(t, dbConn, "be-dd-1", "d-dd-1", "speeding", "medium", 8.0, 0, "")
	seedBehaviourEvent(t, dbConn, "be-dd-2", "d-dd-1", "speeding", "medium", 8.0, 5, "")
	seedBehaviourEvent(t, dbConn, "be-dd-3", "d-dd-1", "harsh_braking", "low", 4.0, 2, "")
	seedBehaviourEvent(t, dbConn, "be-dd-4", "d-dd-1", "fuel_theft_suspicion", "high", 25.0, 1, `{}`)

	_, err := svcs.Scorecard.RecomputeDriverScore(ctx, "d-dd-1")
	require.NoError(t, err)

	detail, err := svcs.Scorecard.DriverDetail(ctx, "d-dd-1")
	require.NoError(t, err)

	assert.Equal(t, "d-dd-1", detail.DriverID)
	assert.Equal(t, 4, detail.EventCount)
	assert.NotEmpty(t, detail.Breakdown, "event breakdown must be non-empty")
	assert.NotEmpty(t, detail.History, "score history must have at least 1 point")
	assert.GreaterOrEqual(t, len(detail.FraudEvents), 1, "theft event must appear in fraud events")
	assert.Equal(t, "fuel_theft_suspicion", detail.FraudEvents[0].EventType)
	assert.False(t, detail.FraudEvents[0].Resolved)
}

// ─── Test 7: ResolveFraudEvent — metadata JSON, audit log ────────────────────

func TestScorecard_ResolveFraud_E2E(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	seedScorecardDriver(t, dbConn, "d-rf-1", "DRV-RF1", "+919610000001", futureDate)
	seedBehaviourEvent(t, dbConn, "be-rf-1", "d-rf-1", "odometer_rollback", "high", 20.0, 0, `{}`)

	// Before resolve: metadata has no resolved key
	var before string
	require.NoError(t, dbConn.QueryRow(`SELECT metadata FROM driver_behaviour_events WHERE id = 'be-rf-1'`).Scan(&before))
	var m0 map[string]interface{}
	json.Unmarshal([]byte(before), &m0)
	assert.Nil(t, m0["resolved"])

	require.NoError(t, svcs.Scorecard.ResolveFraudEvent(ctx, "be-rf-1", "admin-rf"))

	var after string
	require.NoError(t, dbConn.QueryRow(`SELECT metadata FROM driver_behaviour_events WHERE id = 'be-rf-1'`).Scan(&after))
	assert.Contains(t, after, `"resolved":true`)
	assert.Contains(t, after, `"resolved_by":"admin-rf"`)
	assert.Contains(t, after, `"resolved_at"`)

	// Audit log entry
	var auditCount int
	require.NoError(t, dbConn.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'fraud_event_resolved' AND record_id = 'be-rf-1'`).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)
}

// ─── Test 8: HTTP routes — /scorecard registered ──────────────────────────────

func TestScorecard_HTTPRoutes(t *testing.T) {
	_, _, _, _, app := setupComplianceTestEnv(t)

	r := chi.NewRouter()
	r.Route("/scorecard", app.Scorecard.Routes)

	for _, path := range []string{"/scorecard/", "/scorecard/table"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code,
			"%s must be registered (got 404 = route missing)", path)
	}

	// POST resolve must not 404
	req := httptest.NewRequest(http.MethodPost, "/scorecard/drivers/some-id/resolve",
		strings.NewReader("event_id=some-event"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "/scorecard/drivers/{id}/resolve must be registered")
}

// ─── Test 9: Preferred drivers — tier A/B only, sorted by score ───────────────

func TestScorecard_PreferredDrivers(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	for _, item := range []struct {
		id, code, phone string
		score           float64
		tier            string
	}{
		{"d-pd-1", "DRV-PD1", "+919700000001", 92.0, "A"},
		{"d-pd-2", "DRV-PD2", "+919700000002", 74.0, "B"},
		{"d-pd-3", "DRV-PD3", "+919700000003", 55.0, "C"},
	} {
		seedScorecardDriver(t, dbConn, item.id, item.code, item.phone, futureDate)
		_, err := dbConn.Exec(`UPDATE drivers SET score=?, tier=? WHERE id=?`, item.score, item.tier, item.id)
		require.NoError(t, err)
	}

	preferred, err := svcs.Scorecard.PreferredDrivers(ctx, 10)
	require.NoError(t, err)

	// Tier C must not appear
	for _, p := range preferred {
		assert.NotEqual(t, "d-pd-3", p.DriverID, "tier C must not appear in preferred list")
	}

	// Tier A must outrank Tier B
	idxA, idxB := -1, -1
	for i, p := range preferred {
		switch p.DriverID {
		case "d-pd-1":
			idxA = i
		case "d-pd-2":
			idxB = i
		}
	}
	if idxA >= 0 && idxB >= 0 {
		assert.Less(t, idxA, idxB, "score 92 must rank above score 74")
	}
}
