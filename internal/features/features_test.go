package features

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func testRegistry(t *testing.T, env map[string]string) *Registry {
	t.Helper()
	db, err := sql.Open("sqlite", t.TempDir()+"/features_test.db")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE feature_flags (
		tenant_id TEXT NOT NULL,
		feature   TEXT NOT NULL,
		enabled   INTEGER NOT NULL,
		updated_by TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL,
		PRIMARY KEY (tenant_id, feature))`)
	require.NoError(t, err)
	return NewRegistry(db, func(k string) string { return env[k] })
}

func TestTierDefaults(t *testing.T) {
	reg := testRegistry(t, nil)
	ctx := context.Background()

	assert.True(t, reg.Enabled(ctx, "1", "ewaybill"), "core feature must default on")
	assert.False(t, reg.Enabled(ctx, "1", "fastag"), "addon must default off")
	assert.False(t, reg.Enabled(ctx, "1", "no-such-feature"), "unknown key must be off")
}

func TestEnvSwitch(t *testing.T) {
	ctx := context.Background()

	// env "false" disables process-wide, even for EnvDefaultOn addons
	reg := testRegistry(t, map[string]string{"TELEMETRY_ENABLED": "false"})
	assert.False(t, reg.Enabled(ctx, "1", "telemetry"))

	// env "true" enables process-wide (the pre-flags behaviour)
	reg2 := testRegistry(t, map[string]string{"TELEMETRY_ENABLED": "true"})
	assert.True(t, reg2.Enabled(ctx, "1", "telemetry"))
	assert.True(t, reg2.Enabled(ctx, "1", "scorecard"))

	// env unset: EnvDefaultOn keeps existing deployments working…
	reg3 := testRegistry(t, nil)
	assert.True(t, reg3.Enabled(ctx, "1", "telemetry"))
	// …while addons without a default stay off until granted
	assert.False(t, reg3.Enabled(ctx, "1", "fastag"))
	assert.False(t, reg3.Enabled(ctx, "1", "agent"))

	// explicit org revoke still wins over env=true
	require.NoError(t, reg2.Set(ctx, "1", "telemetry", false, "admin"))
	assert.False(t, reg2.Enabled(ctx, "1", "telemetry"))
}

func TestPerOrgGrantAndIsolation(t *testing.T) {
	reg := testRegistry(t, nil)
	ctx := context.Background()

	require.NoError(t, reg.Set(ctx, "7", "fastag", true, "admin-1"))
	assert.True(t, reg.Enabled(ctx, "7", "fastag"))
	assert.False(t, reg.Enabled(ctx, "8", "fastag"), "grant must not leak across orgs")

	// revoke is explicit and beats tier default for core too
	require.NoError(t, reg.Set(ctx, "7", "ewaybill", false, "admin-1"))
	assert.False(t, reg.Enabled(ctx, "7", "ewaybill"))
	assert.True(t, reg.Enabled(ctx, "8", "ewaybill"))

	err := reg.Set(ctx, "7", "not-a-feature", true, "x")
	assert.ErrorIs(t, err, ErrUnknownFeature)
}

func TestSnapshotAndCacheRefresh(t *testing.T) {
	reg := testRegistry(t, nil)
	ctx := context.Background()

	snap := reg.Snapshot(ctx, "1")
	require.NotEmpty(t, snap)
	byKey := map[string]SnapshotEntry{}
	for _, e := range snap {
		byKey[e.Key] = e
	}
	assert.True(t, byKey["customer_portal"].Enabled)
	assert.False(t, byKey["agent"].Enabled)
	assert.Equal(t, TierAddon, byKey["agent"].Tier)

	// Set invalidates the cache — next read sees the new state immediately.
	require.NoError(t, reg.Set(ctx, "1", "agent", true, "admin"))
	assert.True(t, reg.Enabled(ctx, "1", "agent"))
}
