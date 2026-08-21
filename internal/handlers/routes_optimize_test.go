package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	dbmigr "transport-app/db"
	"transport-app/internal/auth"
	"transport-app/internal/config"
	"transport-app/internal/shared"
)

func newTestRouteDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("test_route_%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	migrations, err := fs.Sub(dbmigr.Migrations, "migrations")
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	require.NoError(t, err)
	_, err = provider.Up(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRouteOptimize_MockProvider(t *testing.T) {
	db := newTestRouteDB(t)
	cfg := &config.Config{}
	cfg.Routing.Provider = "mock"
	cfg.Routing.OSRMURL = ""

	app := &App{
		Config:  cfg,
		DB:      db,
		AuthSrv: &optimizeMockAuth{can: true},
	}
	h := &RouteHandlers{App: app}

	ctx := shared.ContextWithTenantID(context.Background(), shared.DefaultTenant)
	sess := &auth.SessionData{UserID: "u-admin", Role: "admin", Name: "Admin"}
	ctx = context.WithValue(ctx, auth.ContextUser, sess)

	body := map[string]interface{}{
		"shipments": []map[string]interface{}{
			{"id": "s1", "latitude": 19.0760, "longitude": 72.8777},
			{"id": "s2", "latitude": 19.2183, "longitude": 72.9781},
		},
		"vehicles": []map[string]interface{}{
			{"id": "v1", "start_lat": 19.0760, "start_lng": 72.8777, "capacity": 100},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/optimize", bytes.NewReader(b))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Optimize(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Code, rr.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp["job_id"])
	require.Equal(t, "completed", resp["status"])

	jobID := resp["job_id"].(string)
	var status string
	err := db.QueryRowContext(context.Background(), `SELECT status FROM route_optimization_jobs WHERE id=?`, jobID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "completed", status)

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/routes/optimize/jobs", nil)
	req3 = req3.WithContext(ctx)
	rr3 := httptest.NewRecorder()
	h.OptimizeJobs(rr3, req3)
	require.Equal(t, http.StatusOK, rr3.Code)
}

func TestRouteOptimize_Validation(t *testing.T) {
	db := newTestRouteDB(t)
	cfg := &config.Config{}
	cfg.Routing.Provider = "mock"
	app := &App{Config: cfg, DB: db, AuthSrv: &optimizeMockAuth{can: true}}
	h := &RouteHandlers{App: app}
	ctx := shared.ContextWithTenantID(context.Background(), shared.DefaultTenant)

	// Too many shipments
	shipments := make([]map[string]interface{}, 51)
	for i := range shipments {
		shipments[i] = map[string]interface{}{"id": fmt.Sprintf("s%d", i), "latitude": 19.0, "longitude": 72.0}
	}
	body := map[string]interface{}{
		"shipments": shipments,
		"vehicles":  []map[string]interface{}{{"id": "v1", "start_lat": 19.0, "start_lng": 72.0}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/optimize", bytes.NewReader(b))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.Optimize(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

type optimizeMockAuth struct {
	can bool
}

func (m *optimizeMockAuth) Can(userID, resource, action string) bool { return m.can }
func (m *optimizeMockAuth) Reload() error                            { return nil }
func (m *optimizeMockAuth) AddRoleForUser(userID, role string) error { return nil }
func (m *optimizeMockAuth) DeleteRolesForUser(userID string) error   { return nil }
