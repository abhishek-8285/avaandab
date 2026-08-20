package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/auth"
	"transport-app/internal/domain"
	"transport-app/internal/handlers"
	"transport-app/internal/service"
	"transport-app/internal/shared"
)

// ─── Test 1: Full HTTP API Lifecycle (Create, List, Get, Ack, Resolve, Dismiss) ───
func TestOpsAlerts_HTTPLifecycle(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)

	authAllow := &mockPhase6Auth{allowed: true}
	app := &handlers.App{DB: dbConn, AuthSrv: authAllow, Services: svcs}
	app.OpsAlerts = handlers.NewOpsAlertHandlers(app, svcs.OpsAlerts, authAllow)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.ContextUser, &auth.SessionData{
				UserID: "admin-1",
				Role:   "admin",
			})
			ctx = shared.ContextWithTenantID(ctx, "1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	app.OpsAlerts.RegisterRoutes(r)

	// 1. POST /api/v1/ops-alerts/generate (Manual creation)
	createBody, _ := json.Marshal(map[string]interface{}{
		"alert_type":  service.OpsAlertVehicleBreakdown,
		"severity":    service.OpsAlertSeverityHigh,
		"title":       "Vehicle broke down in transit",
		"description": "Clutch failure",
		"entity_type": "vehicle",
		"entity_id":   "v-100",
	})
	req := httptest.NewRequest("POST", "/api/v1/ops-alerts/generate", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &createResp)
	require.NoError(t, err)
	alertID, ok := createResp["alert_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, alertID)

	// 2. GET /api/v1/ops-alerts (List)
	req = httptest.NewRequest("GET", "/api/v1/ops-alerts?status=open", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var listResp struct {
		Alerts []service.OpsAlert `json:"alerts"`
		Total  int                `json:"total"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &listResp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, listResp.Total, 1)

	// 3. GET /api/v1/ops-alerts/{id}
	req = httptest.NewRequest("GET", "/api/v1/ops-alerts/"+alertID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var getAlert service.OpsAlert
	err = json.Unmarshal(rec.Body.Bytes(), &getAlert)
	require.NoError(t, err)
	assert.Equal(t, alertID, getAlert.ID)
	assert.Equal(t, "open", getAlert.Status)
	assert.Equal(t, "Vehicle broke down in transit", getAlert.Title)

	// 4. POST /api/v1/ops-alerts/{id}/acknowledge
	req = httptest.NewRequest("POST", "/api/v1/ops-alerts/"+alertID+"/acknowledge", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify acknowledged state
	req = httptest.NewRequest("GET", "/api/v1/ops-alerts/"+alertID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	err = json.Unmarshal(rec.Body.Bytes(), &getAlert)
	require.NoError(t, err)
	assert.Equal(t, "acknowledged", getAlert.Status)
	require.NotNil(t, getAlert.AcknowledgedBy)
	assert.Equal(t, "admin-1", *getAlert.AcknowledgedBy)

	// 5. POST /api/v1/ops-alerts/{id}/resolve
	resBody, _ := json.Marshal(map[string]string{"note": "Replacement vehicle dispatched"})
	req = httptest.NewRequest("POST", "/api/v1/ops-alerts/"+alertID+"/resolve", bytes.NewReader(resBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify resolved state
	req = httptest.NewRequest("GET", "/api/v1/ops-alerts/"+alertID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	err = json.Unmarshal(rec.Body.Bytes(), &getAlert)
	require.NoError(t, err)
	assert.Equal(t, "resolved", getAlert.Status)
	require.NotNil(t, getAlert.ResolutionNote)
	assert.Equal(t, "Replacement vehicle dispatched", *getAlert.ResolutionNote)
}

// ─── Test 2: Settlement Dispute Auto-creates Ops Alert ───
func TestOpsAlerts_SettlementDisputeTrigger(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "1")

	// Seed driver, vehicle, customer, route, booking, trip, settlement
	_, err := dbConn.Exec(`
		INSERT OR IGNORE INTO drivers (id, driver_id, first_name, last_name, phone, status, tenant_id)
		VALUES ('drv-oa-1', 'DRV-OA1', 'Ramesh', 'Kumar', '+919999999901', 'available', '1');

		INSERT OR IGNORE INTO vehicles (id, registration_number, vehicle_number, vehicle_type, capacity, insurance_expiry, fitness_expiry, permit_expiry, status, tenant_id)
		VALUES ('veh-oa-1', 'KA01OA1111', 'KA01OA1111', 'truck', 2500, '2028-01-01', '2028-01-01', '2028-01-01', 'available', '1');

		INSERT OR IGNORE INTO customers (id, name, phone, email)
		VALUES ('cust-oa-1', 'Test Customer', '+919999999902', 'cust@test.com');

		INSERT OR IGNORE INTO routes (id, source, destination, distance, estimated_hours, standard_fare)
		VALUES ('rt-oa-1', 'BLR', 'HYD', 500.0, 10.0, 15000.0);

		INSERT OR IGNORE INTO bookings (id, booking_number, customer_id, route_id, pickup_date, status, price, tenant_id)
		VALUES ('bk-oa-1', 'BK-OA1', 'cust-oa-1', 'rt-oa-1', '2026-08-20', 'confirmed', 15000.0, '1');

		INSERT OR IGNORE INTO trips (id, trip_number, booking_id, route_id, driver_id, vehicle_id, status, tenant_id)
		VALUES ('trip-oa-1', 'TR-OA1', 'bk-oa-1', 'rt-oa-1', 'drv-oa-1', 'veh-oa-1', 'completed', '1');

		INSERT OR IGNORE INTO driver_settlements
		  (id, trip_id, driver_id, gross_fare, commission_amount, advances_kharcha,
		   deductions, performance_bonus, tds_rate, tds_amount, net_payout, rate_model, status, created_at, updated_at)
		VALUES
		  ('stl-oa-1', 'trip-oa-1', 'drv-oa-1', 1000.0, 100.0, 200.0,
		   50.0, 0.0, 1.0, 10.0, 640.0, 'flat_per_trip', 'pending', datetime('now'), datetime('now'));
	`)
	require.NoError(t, err)

	// Dispute settlement
	_, err = svcs.Settlements.DisputeSettlement(ctx, "stl-oa-1", "Deduction calculation incorrect", 700.0)
	require.NoError(t, err)

	// Check ops_alerts for settlement_dispute alert
	alerts, total, err := svcs.OpsAlerts.ListAlerts(ctx, "1", service.OpsAlertFilters{Type: service.OpsAlertSettlementDispute})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, service.OpsAlertSettlementDispute, alerts[0].AlertType)
	assert.Equal(t, service.OpsAlertSeverityHigh, alerts[0].Severity)
	assert.Contains(t, alerts[0].Description, "Deduction calculation incorrect")
}

// ─── Test 3: Compliance Breach Auto-creates Ops Alert ───
func TestOpsAlerts_ComplianceBreachTrigger(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)
	ctx := shared.ContextWithTenantID(context.Background(), "1")

	// Seed driver with expired license
	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	_, err := dbConn.Exec(`
		INSERT OR IGNORE INTO drivers (id, driver_id, first_name, last_name, phone, license_number, license_expiry, status, tenant_id)
		VALUES ('drv-oa-exp', 'DRV-EXP', 'Suresh', 'Singh', '+919999999903', 'DL-EXP-1', ?, 'available', '1');
	`, pastDate)
	require.NoError(t, err)

	drvID := domain.DriverID("drv-oa-exp")
	err = svcs.Compliance.EnforceDispatchCompliance(ctx, &drvID, nil)
	require.Error(t, err, "Dispatch must be blocked for expired license")

	// Check ops_alerts for compliance_breach alert
	alerts, total, err := svcs.OpsAlerts.ListAlerts(ctx, "1", service.OpsAlertFilters{Type: service.OpsAlertComplianceBreach})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 1)
	assert.Equal(t, service.OpsAlertComplianceBreach, alerts[0].AlertType)
	assert.Equal(t, service.OpsAlertSeverityCritical, alerts[0].Severity)
	assert.Contains(t, alerts[0].Description, "expired")
}

// ─── Test 4: RBAC Permission Denied (403 Forbidden) ───
func TestOpsAlerts_RBAC_Forbidden(t *testing.T) {
	dbConn, svcs, _, _, _ := setupComplianceTestEnv(t)

	authDeny := &mockPhase6Auth{allowed: false}
	app := &handlers.App{DB: dbConn, AuthSrv: authDeny, Services: svcs}
	app.OpsAlerts = handlers.NewOpsAlertHandlers(app, svcs.OpsAlerts, authDeny)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.ContextUser, &auth.SessionData{
				UserID: "guest-1",
				Role:   "guest",
			})
			ctx = shared.ContextWithTenantID(ctx, "1")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	app.OpsAlerts.RegisterRoutes(r)

	// GET /api/v1/ops-alerts -> 403 Forbidden
	req := httptest.NewRequest("GET", "/api/v1/ops-alerts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// POST /api/v1/ops-alerts/generate -> 403 Forbidden
	req = httptest.NewRequest("POST", "/api/v1/ops-alerts/generate", bytes.NewReader([]byte(`{}`)))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
