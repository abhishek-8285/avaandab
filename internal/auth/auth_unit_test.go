package auth_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"transport-app/internal/auth"
)

func TestPassword_HashAndCheck(t *testing.T) {
	pass := "SecurePass123!"
	hash, err := auth.HashPassword(pass)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if err := auth.CheckPassword(pass, hash); err != nil {
		t.Fatalf("expected password match, got %v", err)
	}

	if err := auth.CheckPassword("WrongPass!", hash); err == nil {
		t.Fatalf("expected password mismatch error")
	}
}

func TestAPIToken_IssueAndParse(t *testing.T) {
	secret := []byte("secret-key-12345")
	claims := auth.APITokenClaims{
		UserID:   "usr-99",
		Role:     "admin",
		TenantID: "tenant-1",
	}

	token, err := auth.IssueAPIToken(secret, claims)
	if err != nil {
		t.Fatalf("failed to issue API token: %v", err)
	}

	parsed, err := auth.ParseAPIToken(secret, token)
	if err != nil {
		t.Fatalf("failed to parse valid API token: %v", err)
	}

	if parsed.UserID != "usr-99" || parsed.Role != "admin" || parsed.TenantID != "tenant-1" {
		t.Fatalf("parsed claims mismatch")
	}

	// Invalid signature
	_, err = auth.ParseAPIToken([]byte("wrong-secret"), token)
	if err != auth.ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid for wrong secret, got %v", err)
	}

	// Invalid token format
	_, err = auth.ParseAPIToken(secret, "invalid-token")
	if err != auth.ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid for bad token format, got %v", err)
	}

	// Expired token
	expiredClaims := auth.APITokenClaims{
		UserID:    "usr-expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	expiredToken, _ := auth.IssueAPIToken(secret, expiredClaims)
	_, err = auth.ParseAPIToken(secret, expiredToken)
	if err != auth.ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired for expired token, got %v", err)
	}
}

func TestSessionStore_CreateValidateClear(t *testing.T) {
	secret := "test-secret-key-32-bytes-long-!"
	store := auth.NewSessionStore(secret, false)

	rec := httptest.NewRecorder()
	store.CreateSession(rec, "usr-1", "admin", "Admin User")

	res := rec.Result()
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie to be set")
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])

	data, ok := store.ValidateSession(req)
	if !ok || data.UserID != "usr-1" || data.Role != "admin" {
		t.Fatalf("session validation failed, got %+v", data)
	}

	clearRec := httptest.NewRecorder()
	store.ClearSession(clearRec)
	clearCookies := clearRec.Result().Cookies()
	if len(clearCookies) == 0 || clearCookies[0].MaxAge != -1 {
		t.Fatalf("expected clear session cookie with MaxAge=-1")
	}
}

func TestGenerateSecureTokenAndClientIPLocation(t *testing.T) {
	tok, err := auth.GenerateSecureToken()
	if err != nil || len(tok) == 0 {
		t.Fatalf("failed to generate secure token: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req.Header.Set("CF-IPCountry", "IN")
	req.Header.Set("CF-IPCity", "Mumbai")

	if ip := auth.ClientIP(req); ip != "203.0.113.1" {
		t.Fatalf("expected CF-Connecting-IP 203.0.113.1, got %s", ip)
	}

	if loc := auth.ClientLocation(req); loc != "Mumbai, IN" {
		t.Fatalf("expected location 'Mumbai, IN', got '%s'", loc)
	}

	// Test X-Real-IP fallback
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Real-IP", "198.51.100.2")
	if ip := auth.ClientIP(req2); ip != "198.51.100.2" {
		t.Fatalf("expected X-Real-IP 198.51.100.2, got %s", ip)
	}

	// Test X-Forwarded-For fallback
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.Header.Set("X-Forwarded-For", "198.51.100.3, 10.0.0.1")
	if ip := auth.ClientIP(req3); ip != "198.51.100.3" {
		t.Fatalf("expected X-Forwarded-For first IP 198.51.100.3, got %s", ip)
	}
}

func TestHasPermissionAndRoleMapping(t *testing.T) {
	if !auth.HasPermission(1, "users", "write") {
		t.Fatalf("admin (role 1) should have permission on users")
	}

	if auth.HasPermission(2, "users", "write") {
		t.Fatalf("dispatcher (role 2) should NOT have permission on users")
	}

	if !auth.HasPermission(4, "trips", "read") {
		t.Fatalf("viewer (role 4) should have read permission on trips")
	}

	if auth.RoleNameForID(1) != "admin" {
		t.Fatalf("expected role name admin for ID 1")
	}

	if auth.RoleIDForName("admin") != 1 {
		t.Fatalf("expected role ID 1 for admin")
	}
}
