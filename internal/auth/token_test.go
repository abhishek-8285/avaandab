package auth_test

import (
	"testing"
	"time"

	"transport-app/internal/auth"
)

func TestTokenHashAndCompare(t *testing.T) {
	auth.SetTokenSecret([]byte("unit-test-secret"))
	defer auth.SetTokenSecret([]byte("unit-test-secret"))

	token := "raw-token-123"
	hash := auth.HashToken(token)

	// The stored hash must never contain the raw token.
	if hash == token || len(hash) != 64 {
		t.Fatalf("expected 64-char hex digest, got %q", hash)
	}

	if !auth.CompareToken(token, hash) {
		t.Fatalf("expected valid token to compare")
	}
	if auth.CompareToken("other-token", hash) {
		t.Fatalf("expected tampered token to fail comparison")
	}
	if auth.CompareToken(token, "not-hex!") {
		t.Fatalf("expected malformed stored hash to fail comparison")
	}

	// Same token hashed under a different secret must not match.
	auth.SetTokenSecret([]byte("other-secret"))
	if auth.CompareToken(token, hash) {
		t.Fatalf("expected hash to change with a different secret")
	}
	auth.SetTokenSecret([]byte("unit-test-secret"))
}

func TestValidatePassword(t *testing.T) {
	valid := []string{"admin12345", "Passw0rd!", "a1bcdefgh"}
	for _, pw := range valid {
		if err := auth.ValidatePassword(pw); err != nil {
			t.Fatalf("expected %q to be valid, got %v", pw, err)
		}
	}

	invalid := []string{"", "short", "12345678", "abcdefgh", "123456789", "AaBbCcDdEe"}
	for _, pw := range invalid {
		if err := auth.ValidatePassword(pw); err == nil {
			t.Fatalf("expected %q to be rejected", pw)
		}
	}
}

func TestResetTokenStore(t *testing.T) {
	store := auth.NewResetTokenStore(0)

	email := "user@example.com"
	token, err := store.Create(email)
	if err != nil {
		t.Fatalf("failed to create reset token: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty reset token")
	}

	// Tokens are single-use: first consume succeeds, second fails.
	gotEmail, ok := store.Consume(token)
	if !ok || gotEmail != email {
		t.Fatalf("expected first consume to return %q, got %q ok=%v", email, gotEmail, ok)
	}
	if _, ok := store.Consume(token); ok {
		t.Fatalf("expected second consume to fail")
	}

	// Expired tokens must be rejected.
	short := auth.NewResetTokenStore(10 * time.Millisecond)
	token2, err := short.Create(email)
	if err != nil {
		t.Fatalf("failed to create reset token: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := short.Consume(token2); ok {
		t.Fatalf("expected expired token to be rejected")
	}
}
