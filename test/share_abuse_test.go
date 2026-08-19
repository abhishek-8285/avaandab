package test

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"transport-app/internal/config"
	"transport-app/internal/handlers"
	"transport-app/internal/middleware"
)

func newShareTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("test_share_abuse_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	require.NoError(t, err)

	cwd, _ := os.Getwd()
	migrationsDir := "../db/migrations"
	if filepath.Base(cwd) == "basic" {
		migrationsDir = "db/migrations"
	}

	_ = goose.SetDialect("sqlite")
	require.NoError(t, goose.Up(db, migrationsDir))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newShareApp(t *testing.T, db *sql.DB) *handlers.App {
	t.Helper()
	cfg := &config.Config{
		CookieSecret: "test-cookie-secret-32-chars-long!",
		LiveMap: config.LiveMapConfig{
			ShareLinkTTLHours:    24,
			ShareLinkMaxTTLHours: 168,
			ShareLinkMaxActive:   20,
			CSPEnabled:           true,
		},
	}
	tmpl := template.New("share_public.html")
	_, err := tmpl.Parse(`<html><body>{{.TripNumber}}</body></html>`)
	require.NoError(t, err)
	_, err = tmpl.New("share_pin_form.html").Parse(`<html><body>PIN Form</body></html>`)
	require.NoError(t, err)

	app := &handlers.App{
		DB:        db,
		Config:    cfg,
		Templates: tmpl,
	}
	app.Share = handlers.NewShareHandlers(app, db)
	return app
}

func seedShareTrip(t *testing.T, db *sql.DB, tripID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO trips
		(id, trip_number, booking_id, route_id, status, departure_time, arrival_time, tenant_id)
		VALUES (?, 'TRIP-ABUSE', 'b-1', 'r-1', 'started', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, '1')`,
		tripID)
	require.NoError(t, err)
}

func TestShareLink_RateLimitAbuse(t *testing.T) {
	db := newShareTestDB(t)
	app := newShareApp(t, db)
	seedShareTrip(t, db, "trip-rl")

	// Create share link directly in DB
	rawToken := "abuse-rate-limit-token-12345"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	_, err := db.Exec(`INSERT INTO share_links
		(id, trip_id, token_hash, created_by, created_at, expires_at)
		VALUES ('share-rl', 'trip-rl', ?, 'user-1', CURRENT_TIMESTAMP, datetime('now', '+24 hours'))`,
		tokenHash)
	require.NoError(t, err)

	// Router with rate limiter (20 requests per minute)
	r := chi.NewRouter()
	r.With(middleware.RateLimit(20)).Get("/share/{token}", app.Share.ViewShare)

	// Send 25 requests rapidly from same IP
	rateLimited := 0
	successCount := 0
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest("GET", "/share/"+rawToken, nil)
		req.RemoteAddr = "192.168.1.50:1234"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusTooManyRequests {
			rateLimited++
		} else if rr.Code == http.StatusOK || rr.Code == http.StatusSeeOther {
			successCount++
		}
	}

	assert.Equal(t, 20, successCount, "first 20 requests should succeed")
	assert.Equal(t, 5, rateLimited, "requests 21-25 should be 429 Too Many Requests")
}

func TestShareLink_PINBruteForce(t *testing.T) {
	db := newShareTestDB(t)
	app := newShareApp(t, db)
	seedShareTrip(t, db, "trip-pin")

	// Generate share with PIN "4321"
	rawToken := "pin-brute-force-token-999"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Compute salted PIN hash
	salt := "test-salt-12345"
	pinHash := sha256.Sum256([]byte(salt + "4321"))
	pinHashStr := hex.EncodeToString(pinHash[:])

	_, err := db.Exec(`INSERT INTO share_links
		(id, trip_id, token_hash, pin_salt, pin_hash, created_by, created_at, expires_at)
		VALUES ('share-pin', 'trip-pin', ?, ?, ?, 'user-1', CURRENT_TIMESTAMP, datetime('now', '+24 hours'))`,
		tokenHash, salt, pinHashStr)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Post("/share/{token}/verify", app.Share.VerifyPIN)

	// 1. Submit 4 incorrect PINs -> should return 403 (or 401) with remaining attempts
	for i := 1; i <= 4; i++ {
		form := url.Values{"pin": {fmt.Sprintf("000%d", i)}}
		req := httptest.NewRequest("POST", "/share/"+rawToken+"/verify", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code, "attempt %d should be 403", i)
		assert.Contains(t, rr.Body.String(), "attempt(s) remaining")
	}

	// 2. 5th incorrect PIN -> triggers lockout
	form5 := url.Values{"pin": {"9999"}}
	req5 := httptest.NewRequest("POST", "/share/"+rawToken+"/verify", strings.NewReader(form5.Encode()))
	req5.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr5 := httptest.NewRecorder()
	r.ServeHTTP(rr5, req5)
	assert.Equal(t, http.StatusForbidden, rr5.Code, "5th failed attempt should trigger lockout")
	assert.Equal(t, "900", rr5.Header().Get("Retry-After"), "must include Retry-After: 900 header")

	// Verify locked_until is set in database
	var lockedUntil sql.NullTime
	var failedAttempts int
	err = db.QueryRow("SELECT locked_until, failed_pin_attempts FROM share_links WHERE id = 'share-pin'").Scan(&lockedUntil, &failedAttempts)
	require.NoError(t, err)
	assert.True(t, lockedUntil.Valid)
	assert.Equal(t, 5, failedAttempts)
	assert.True(t, lockedUntil.Time.After(time.Now()))

	// 3. 6th attempt (even with CORRECT PIN) -> blocked with 403 while locked
	formCorrect := url.Values{"pin": {"4321"}}
	req6 := httptest.NewRequest("POST", "/share/"+rawToken+"/verify", strings.NewReader(formCorrect.Encode()))
	req6.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr6 := httptest.NewRecorder()
	r.ServeHTTP(rr6, req6)
	assert.Equal(t, http.StatusForbidden, rr6.Code)

	retrySecs := 0
	_, _ = fmt.Sscanf(rr6.Header().Get("Retry-After"), "%d", &retrySecs)
	assert.Greater(t, retrySecs, 800, "Retry-After should be approximately 900s")
	assert.LessOrEqual(t, retrySecs, 900)
}

func TestShareLink_SlidingExpiryThrasher(t *testing.T) {
	db := newShareTestDB(t)
	app := newShareApp(t, db)
	seedShareTrip(t, db, "trip-slide")

	rawToken := "slide-expiry-token-777"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Initial created 10 hours ago, expires in 14 hours (TTL=24)
	created := time.Now().UTC().Add(-10 * time.Hour).Format("2006-01-02 15:04:05")
	initialExpires := time.Now().UTC().Add(14 * time.Hour).Format("2006-01-02 15:04:05")

	_, err := db.Exec(`INSERT INTO share_links
		(id, trip_id, token_hash, created_by, created_at, expires_at)
		VALUES ('share-slide', 'trip-slide', ?, 'user-1', ?, ?)`,
		tokenHash, created, initialExpires)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/share/{token}/data", app.Share.ShareData)
	r.Get("/share/{token}", app.Share.ViewShare)

	// 1. Anti-Thrash: Poll /data 50 times -> expires_at MUST NOT change
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/share/"+rawToken+"/data", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	var expiresAfterData time.Time
	err = db.QueryRow("SELECT expires_at FROM share_links WHERE id = 'share-slide'").Scan(&expiresAfterData)
	require.NoError(t, err)
	initParsed, _ := time.Parse("2006-01-02 15:04:05", initialExpires)
	assert.Equal(t, initParsed.Unix(), expiresAfterData.Unix(), "/data poll must not extend expires_at")

	// 2. Full Page View -> triggers sliding expiry extension
	reqView := httptest.NewRequest("GET", "/share/"+rawToken, nil)
	rrView := httptest.NewRecorder()
	r.ServeHTTP(rrView, reqView)
	assert.Equal(t, http.StatusOK, rrView.Code)

	var expiresAfterView string
	err = db.QueryRow("SELECT expires_at FROM share_links WHERE id = 'share-slide'").Scan(&expiresAfterView)
	require.NoError(t, err)
	assert.NotEqual(t, initialExpires, expiresAfterView, "page view must extend expires_at")

	// 3. Hard Cap: Share created 160 hours ago with MaxTTL = 168 hours
	// Even on view, expires_at must not exceed created_at + 168 hours
	oldCreated := time.Now().UTC().Add(-160 * time.Hour).Format("2006-01-02 15:04:05")
	oldExpires := time.Now().UTC().Add(8 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.Exec(`UPDATE share_links SET created_at = ?, expires_at = ? WHERE id = 'share-slide'`, oldCreated, oldExpires)
	require.NoError(t, err)

	reqView2 := httptest.NewRequest("GET", "/share/"+rawToken, nil)
	rrView2 := httptest.NewRecorder()
	r.ServeHTTP(rrView2, reqView2)

	var cappedExpires time.Time
	err = db.QueryRow("SELECT expires_at FROM share_links WHERE id = 'share-slide'").Scan(&cappedExpires)
	require.NoError(t, err)

	createdAtParsed, _ := time.Parse("2006-01-02 15:04:05", oldCreated)
	maxAllowed := createdAtParsed.Add(168 * time.Hour)
	assert.True(t, cappedExpires.Before(maxAllowed) || cappedExpires.Equal(maxAllowed),
		"expires_at (%v) must not exceed created_at + 168h (%v)", cappedExpires, maxAllowed)
}
