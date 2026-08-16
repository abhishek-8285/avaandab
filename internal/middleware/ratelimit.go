package middleware

import (
	"net/http"
	"sync"
	"time"

	"transport-app/internal/auth"
)

// ipBucket tracks requests for a single client IP within a time window.
type ipBucket struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

// rateLimiter is a fixed-window per-IP limiter with periodic cleanup.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*ipBucket
}

// RateLimit returns middleware that limits requests per client IP to
// `limit` per `window` (default window: 1 minute).
func RateLimit(limit int) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = 10
	}
	rl := &rateLimiter{
		limit:   limit,
		window:  time.Minute,
		buckets: make(map[string]*ipBucket),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(auth.ClientIP(r), time.Now()) {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *rateLimiter) allow(ip string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.sweep(now)

	b, ok := rl.buckets[ip]
	if !ok || now.After(b.resetAt) {
		rl.buckets[ip] = &ipBucket{count: 1, resetAt: now.Add(rl.window), lastSeen: now}
		return true
	}

	b.lastSeen = now
	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

// sweep removes buckets that have been idle longer than the window.
func (rl *rateLimiter) sweep(now time.Time) {
	for ip, b := range rl.buckets {
		if now.Sub(b.lastSeen) > rl.window {
			delete(rl.buckets, ip)
		}
	}
}
