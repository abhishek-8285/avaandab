package middleware

import (
	"hash/fnv"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"transport-app/internal/auth"
	"transport-app/internal/cache"
)

// ipBucket tracks requests for a single client IP within a time window.
type ipBucket struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

// rateLimiterShard guards the buckets assigned to it. Sharding avoids a single
// global mutex becoming a bottleneck under high request concurrency.
type rateLimiterShard struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
}

// rateLimiter is a fixed-window per-IP limiter with periodic cleanup,
// sharded into rateLimiterShards independent locks keyed by hash of the IP.
type rateLimiter struct {
	shards [rateLimiterShards]*rateLimiterShard
	limit  int
	window time.Duration
}

// rateLimiterShards is the number of independent lock domains in the limiter.
const rateLimiterShards = 16

// newRateLimiter builds a sharded limiter with the given per-IP limit and
// window (window <= 0 defaults to 1 minute, limit <= 0 to 10).
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	rl := &rateLimiter{
		limit:  limit,
		window: window,
	}
	for i := range rl.shards {
		rl.shards[i] = &rateLimiterShard{buckets: make(map[string]*ipBucket)}
	}
	return rl
}

// RateLimit returns middleware that limits requests per client IP to
// `limit` per `window` (default window: 1 minute).
//
// In-memory: each replica counts independently, so effective per-IP limit is
// limit × replicas. Use RateLimitDistributed for exact cross-replica limits.
func RateLimit(limit int) func(http.Handler) http.Handler {
	rl := newRateLimiter(limit, time.Minute)
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

// RateLimitDistributed limits per client IP across all replicas using an
// atomic cache counter (Redis INCR or memory-cache counters). When the
// backend does not support atomic increments (CACHE_DRIVER=none), it falls
// back to the local in-memory limiter — never to unlimited.
func RateLimitDistributed(c cache.Cache, limit int) func(http.Handler) http.Handler {
	incr, ok := c.(cache.Incrementer)
	if !ok {
		return RateLimit(limit)
	}
	window := time.Minute
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "ratelimit:" + auth.ClientIP(r)
			n, err := incr.Increment(r.Context(), key, window)
			if err != nil {
				// Backend failure must not open the floodgates: fail closed
				// but log loudly so the cache outage is visible.
				slog.Error("distributed rate limit backend error; rejecting request", "error", err)
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
			if n > int64(limit) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *rateLimiter) shardFor(ip string) *rateLimiterShard {
	h := fnv.New32a()
	h.Write([]byte(ip))
	return rl.shards[h.Sum32()%rateLimiterShards]
}

func (rl *rateLimiter) allow(ip string, now time.Time) bool {
	shard := rl.shardFor(ip)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.sweep(now, rl.window)

	b, ok := shard.buckets[ip]
	if !ok || now.After(b.resetAt) {
		shard.buckets[ip] = &ipBucket{count: 1, resetAt: now.Add(rl.window), lastSeen: now}
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
func (sh *rateLimiterShard) sweep(now time.Time, window time.Duration) {
	for ip, b := range sh.buckets {
		if now.Sub(b.lastSeen) > window {
			delete(sh.buckets, ip)
		}
	}
}
