// Package cache abstracts the caching backend behind a small interface so
// switching from no-op → in-process → Redis is an env-only change
// (CACHE_DRIVER), never a code change. Callers depend only on Cache.
package cache

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Driver names accepted by CACHE_DRIVER.
const (
	DriverNone   = "none"
	DriverMemory = "memory"
	DriverRedis  = "redis"
)

// Cache is the minimal contract every backend fulfils. Values are opaque
// bytes; serialization stays with the caller.
type Cache interface {
	// Get returns the cached value for key, or ok=false on miss/expiry.
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	// Set stores value under key. ttl <= 0 means "use the configured default".
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Delete removes key. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
}

// Closer is implemented by backends holding resources that must be released
// on shutdown (janitor goroutines, redis connections). Use errors.As-free
// type assertion: if c, ok := cache.(Closer); ok { _ = c.Close() }.
type Closer interface {
	Close() error
}

// Incrementer is optionally implemented by backends that support atomic
// counters (Redis INCR). Used for cross-replica rate limiting and counters;
// Noop deliberately does NOT implement it so callers fall back to local
// behavior instead of silently allowing unlimited requests.
type Incrementer interface {
	// Increment atomically adds 1 to key, creating it with ttl on first hit.
	// Returns the new value. ttl <= 0 uses the configured default TTL.
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// Settings is the minimal view of config.CacheConfig this package needs.
type Settings interface {
	GetDriver() string
	GetRedisAddr() string
	GetRedisPassword() string
	GetRedisDB() int
	GetDefaultTTL() time.Duration
	GetKeyPrefix() string
}

// New builds the cache backend named by cfg.Driver. A connection failure or
// misconfiguration returns an error — callers decide whether to fall back to
// the no-op cache or abort startup.
func New(ctx context.Context, cfg Settings, logger *slog.Logger) (Cache, error) {
	prefix := cfg.GetKeyPrefix()
	switch strings.ToLower(cfg.GetDriver()) {
	case "", DriverNone:
		return Noop{}, nil
	case DriverMemory:
		return newMemoryCache(cfg.GetDefaultTTL()), nil
	case DriverRedis:
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.GetRedisAddr(),
			Password: cfg.GetRedisPassword(),
			DB:       cfg.GetRedisDB(),
		})
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := client.Ping(pingCtx).Err(); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("cache: redis ping %s: %w", cfg.GetRedisAddr(), err)
		}
		if logger != nil {
			logger.Info("Redis cache connected", "addr", cfg.GetRedisAddr(), "db", cfg.GetRedisDB())
		}
		return &redisCache{client: client, defaultTTL: cfg.GetDefaultTTL(), prefix: prefix}, nil
	default:
		return nil, fmt.Errorf("cache: unsupported driver %q (use none, memory or redis)", cfg.GetDriver())
	}
}

// MustNew is New but falls back to the no-op cache on error after logging —
// for call sites where caching is an optimization, never correctness.
func MustNew(ctx context.Context, cfg Settings, logger *slog.Logger) Cache {
	c, err := New(ctx, cfg, logger)
	if err != nil {
		if logger != nil {
			logger.Error("cache init failed; falling back to no-op cache", "error", err)
		}
		return Noop{}
	}
	return c
}

// Noop discards everything. Used when CACHE_DRIVER=none and as the safe
// fallback when the real backend cannot be reached.
type Noop struct{}

func (Noop) Get(context.Context, string) ([]byte, bool, error) { return nil, false, nil }
func (Noop) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}
func (Noop) Delete(context.Context, string) error { return nil }

// entry is one memory-cache record with absolute expiry.
type entry struct {
	value     []byte
	expiresAt time.Time // zero = no expiry
}

func (e entry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

// memoryCache is an in-process TTL cache with a periodic janitor sweep.
// Single-instance only — values are NOT shared across server replicas.
type memoryCache struct {
	mu         sync.RWMutex
	items      map[string]entry
	counters   map[string]*counter
	defaultTTL time.Duration
	stop       chan struct{}
	stopOnce   sync.Once
}

func newMemoryCache(defaultTTL time.Duration) *memoryCache {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	m := &memoryCache{
		items:      make(map[string]entry),
		counters:   make(map[string]*counter),
		defaultTTL: defaultTTL,
		stop:       make(chan struct{}),
	}
	go m.janitor()
	return m
}

func (m *memoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	e, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || e.expired(time.Now()) {
		return nil, false, nil
	}
	// Copy so callers cannot mutate cached bytes.
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, true, nil
}

func (m *memoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = m.defaultTTL
	}
	v := make([]byte, len(value))
	copy(v, value)
	e := entry{value: v}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[key] = e
	m.mu.Unlock()
	return nil
}

func (m *memoryCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
	return nil
}

// Close stops the janitor and drops all entries.
func (m *memoryCache) Close() error {
	m.stopOnce.Do(func() { close(m.stop) })
	m.mu.Lock()
	m.items = make(map[string]entry)
	m.mu.Unlock()
	return nil
}

// counter is a fixed-window atomic counter for Increment.
type counter struct {
	n         int64
	expiresAt time.Time // zero = no expiry (never happens: ttl always > 0)
}

func (c counter) expired(now time.Time) bool {
	return !c.expiresAt.IsZero() && now.After(c.expiresAt)
}

// Increment implements Incrementer with a fixed window per key. Single-
// process only; use the redis driver for cross-replica counting.
func (m *memoryCache) Increment(_ context.Context, key string, ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		ttl = m.defaultTTL
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.counters[key]
	if !ok || c.expired(now) {
		m.counters[key] = &counter{n: 1, expiresAt: now.Add(ttl)}
		return 1, nil
	}
	c.n++
	return c.n, nil
}

// janitor purges expired entries every minute until Close is called.
func (m *memoryCache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			now := time.Now()
			m.mu.Lock()
			for k, e := range m.items {
				if e.expired(now) {
					delete(m.items, k)
				}
			}
			for k, c := range m.counters {
				if c.expired(now) {
					delete(m.counters, k)
				}
			}
			m.mu.Unlock()
		}
	}
}

// redisCache fronts Redis with key prefixing so multiple environments can
// share one cluster without collisions.
type redisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	prefix     string
}

func (r *redisCache) fullKey(key string) string { return r.prefix + key }

func (r *redisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := r.client.Get(ctx, r.fullKey(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache: redis get: %w", err)
	}
	return val, true, nil
}

func (r *redisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = r.defaultTTL
	}
	if err := r.client.Set(ctx, r.fullKey(key), value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: redis set: %w", err)
	}
	return nil
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, r.fullKey(key)).Err(); err != nil {
		return fmt.Errorf("cache: redis delete: %w", err)
	}
	return nil
}

// Increment implements Incrementer via INCR; TTL is set only on the first
// hit of a window (when the counter was created) using redis.Nil detection.
func (r *redisCache) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		ttl = r.defaultTTL
	}
	full := r.fullKey(key)
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, full)
	pipe.ExpireNX(ctx, full, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("cache: redis increment: %w", err)
	}
	return incr.Val(), nil
}

// Close releases the Redis connection pool.
func (r *redisCache) Close() error { return r.client.Close() }
