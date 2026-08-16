package auth

import (
	"sync"
	"time"
)

// ResetTokenStore issues single-use password-reset tokens with an expiry.
// Only the HMAC hash of a token is kept in memory, so a memory dump does not
// leak usable reset links.
type ResetTokenStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]resetTokenEntry
}

type resetTokenEntry struct {
	email   string
	expires time.Time
}

// NewResetTokenStore creates a store with the given token lifetime
// (defaults to 30 minutes when ttl <= 0).
func NewResetTokenStore(ttl time.Duration) *ResetTokenStore {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &ResetTokenStore{
		ttl:     ttl,
		entries: make(map[string]resetTokenEntry),
	}
}

// Create generates a fresh reset token for an email and returns the raw token
// to embed in a reset link. Only the hash is stored.
func (s *ResetTokenStore) Create(email string) (string, error) {
	token, err := GenerateSecureToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(time.Now())
	s.entries[HashToken(token)] = resetTokenEntry{
		email:   email,
		expires: time.Now().Add(s.ttl),
	}
	return token, nil
}

// Consume redeems a reset token exactly once, returning the associated email.
// Expired or unknown tokens report a miss and are purged.
func (s *ResetTokenStore) Consume(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.sweepLocked(now)

	key := HashToken(token)
	entry, ok := s.entries[key]
	delete(s.entries, key)
	if !ok || now.After(entry.expires) {
		return "", false
	}
	return entry.email, true
}

func (s *ResetTokenStore) sweepLocked(now time.Time) {
	for key, entry := range s.entries {
		if now.After(entry.expires) {
			delete(s.entries, key)
		}
	}
}
