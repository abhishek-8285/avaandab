package fuel

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration00043_RoundTrip(t *testing.T) {
	name := fmt.Sprintf("rt43_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	defer db.Close()

	_ = goose.SetDialect("sqlite")

	// Up to current
	require.NoError(t, goose.Up(db, "../../db/migrations"), "goose up failed")

	// Verify all 00043 tables exist
	for _, tbl := range []string{"fuel_events", "fuel_claim_audits", "driver_behaviour_events", "driver_scores"} {
		var n int
		db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&n)
		assert.Equal(t, 1, n, "table %s must exist after up", tbl)
	}

	// Verify company_config seeds are there
	var fuelKeys int
	db.QueryRow(`SELECT count(*) FROM company_config WHERE key LIKE 'fuel.%' OR key LIKE 'scorecard.%'`).Scan(&fuelKeys)
	assert.GreaterOrEqual(t, fuelKeys, 20, "fuel+scorecard config seeds must be present")

	// Verify INSERT OR IGNORE is idempotent
	_, err = db.Exec(`INSERT OR IGNORE INTO company_config (tenant_id, key, value) VALUES ('1', 'fuel.median_window', '99')`)
	require.NoError(t, err, "duplicate seed insert must not fail")
	// Value should still be original (7), not 99
	var val string
	db.QueryRow(`SELECT value FROM company_config WHERE tenant_id='1' AND key='fuel.median_window'`).Scan(&val)
	assert.Equal(t, "7", val, "INSERT OR IGNORE must not overwrite existing seed")

	// Verify company_config table was NOT created by 00043 (owned by 00042)
	var tableCount int
	db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='company_config'`).Scan(&tableCount)
	assert.Equal(t, 1, tableCount, "company_config must exist exactly once")

	// Down to 42 (reverses 00043)
	require.NoError(t, goose.DownTo(db, "../../db/migrations", 42), "goose down to 42 failed")

	for _, tbl := range []string{"fuel_events", "fuel_claim_audits", "driver_behaviour_events", "driver_scores"} {
		var n int
		db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&n)
		assert.Equal(t, 0, n, "table %s must be dropped after down", tbl)
	}

	// Up again — idempotent
	require.NoError(t, goose.Up(db, "../../db/migrations"), "goose up again failed")

	// Verify RBAC seeds
	var permCount int
	db.QueryRow(`SELECT count(*) FROM permissions WHERE name IN ('fuel:read','fuel:update','scorecard:read')`).Scan(&permCount)
	assert.Equal(t, 3, permCount)

	// Verify drivers got score/tier columns
	_, err = db.Exec(`INSERT OR IGNORE INTO drivers (id, driver_id, first_name, last_name, phone, license_number, license_expiry, status) VALUES ('d-rt1', 'DRT-1', 'A', 'B', '9000000001', 'LN-RT1', '2028-01-01', 'available')`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE drivers SET score = 88.5, tier = 'A' WHERE id = 'd-rt1'`)
	require.NoError(t, err)

	var score float64
	var tier string
	db.QueryRow(`SELECT score, tier FROM drivers WHERE id = 'd-rt1'`).Scan(&score, &tier)
	assert.InDelta(t, 88.5, score, 0.001)
	assert.Equal(t, "A", tier)

	// Verify driver_expenses got audit_status/fuel_litres columns
	colCheck := func(tbl, col string) {
		rows, _ := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, tbl))
		defer rows.Close()
		found := false
		for rows.Next() {
			var cid int
			var cname, ctype string
			var nn int
			var dflt, pk sql.NullString
			rows.Scan(&cid, &cname, &ctype, &nn, &dflt, &pk)
			if strings.EqualFold(cname, col) {
				found = true
			}
		}
		assert.True(t, found, "column %s.%s must exist", tbl, col)
	}
	colCheck("driver_expenses", "audit_status")
	colCheck("driver_expenses", "fuel_litres")
	colCheck("driver_settlements", "performance_bonus")
}
