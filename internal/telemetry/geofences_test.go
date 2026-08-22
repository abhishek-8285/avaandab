package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"transport-app/internal/shared"
)

func TestGeofencesHandler_ReturnsActiveZones(t *testing.T) {
	db := newTestIngestorDB(t)
	_, err := db.Exec(`INSERT INTO geofences
		(id, tenant_id, name, kind, shape, center_lat, center_lng, radius_m, is_active)
		VALUES ('gf1', '1', 'Depot A', 'depot', 'circle', 19.07, 72.87, 500, 1)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO geofences
		(id, tenant_id, name, kind, shape, polygon, is_active)
		VALUES ('gf2', '1', 'Pickup Zone', 'pickup', 'polygon', '[[19.1,72.9],[19.11,72.91],[19.12,72.9]]', 1)`)
	require.NoError(t, err)
	// Inactive zone must be excluded.
	_, err = db.Exec(`INSERT INTO geofences
		(id, tenant_id, name, kind, shape, center_lat, center_lng, radius_m, is_active)
		VALUES ('gf3', '1', 'Inactive', 'depot', 'circle', 19.0, 72.8, 500, 0)`)
	require.NoError(t, err)
	// Another tenant's active zone must not leak.
	_, err = db.Exec(`INSERT INTO geofences
		(id, tenant_id, name, kind, shape, center_lat, center_lng, radius_m, is_active)
		VALUES ('gf4', '2', 'Other Tenant', 'depot', 'circle', 19.0, 72.8, 500, 1)`)
	require.NoError(t, err)

	h := GeofencesHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/geofences", nil)
	req = req.WithContext(shared.ContextWithTenantID(req.Context(), "1"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var zones []GeofenceZone
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &zones))
	require.Len(t, zones, 2)

	byID := map[string]GeofenceZone{}
	for _, z := range zones {
		byID[z.ID] = z
	}
	assert.Equal(t, "circle", byID["gf1"].Shape)
	assert.Equal(t, 500.0, byID["gf1"].RadiusM)
	assert.Equal(t, "polygon", byID["gf2"].Shape)
	require.Len(t, byID["gf2"].Polygon, 3)
	assert.Equal(t, 19.1, byID["gf2"].Polygon[0].Lat)
}

func TestGeofencesHandler_TenantScoping(t *testing.T) {
	db := newTestIngestorDB(t)
	_, err := db.Exec(`INSERT INTO geofences
		(id, tenant_id, name, kind, shape, center_lat, center_lng, radius_m, is_active)
		VALUES ('gf1', '1', 'Depot A', 'depot', 'circle', 19.07, 72.87, 500, 1)`)
	require.NoError(t, err)

	h := GeofencesHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/geofences", nil)
	req = req.WithContext(shared.ContextWithTenantID(req.Context(), "2"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var zones []GeofenceZone
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &zones))
	assert.Empty(t, zones)
}
