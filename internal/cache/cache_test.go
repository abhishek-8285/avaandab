package cache_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"transport-app/internal/cache"
)

type testSettings struct {
	driver   string
	addr     string
	password string
	db       int
	ttl      time.Duration
	prefix   string
}

func (t *testSettings) GetDriver() string            { return t.driver }
func (t *testSettings) GetRedisAddr() string         { return t.addr }
func (t *testSettings) GetRedisPassword() string     { return t.password }
func (t *testSettings) GetRedisDB() int              { return t.db }
func (t *testSettings) GetDefaultTTL() time.Duration { return t.ttl }
func (t *testSettings) GetKeyPrefix() string         { return t.prefix }

func TestNew_None(t *testing.T) {
	c, err := cache.New(context.Background(), &testSettings{driver: "none"}, nil)
	if err != nil {
		t.Fatalf("New(none) = %v, want nil", err)
	}
	if _, ok := c.(cache.Noop); !ok {
		t.Errorf("New(none) returned %T, want cache.Noop", c)
	}
}

func TestNew_EmptyDriverDefaultsToNoop(t *testing.T) {
	c, err := cache.New(context.Background(), &testSettings{}, nil)
	if err != nil {
		t.Fatalf("New(\"\") = %v, want nil", err)
	}
	if _, ok := c.(cache.Noop); !ok {
		t.Errorf("New(\"\") returned %T, want cache.Noop", c)
	}
}

func TestNew_Memory(t *testing.T) {
	s := &testSettings{driver: "memory", ttl: time.Hour, prefix: "mvtms:"}
	c, err := cache.New(context.Background(), s, slog.Default())
	if err != nil {
		t.Fatalf("New(memory) = %v, want nil", err)
	}
	defer func() { _ = c.(cache.Closer).Close() }()

	ctx := context.Background()

	if _, ok, err := c.Get(ctx, "k"); ok || err != nil {
		t.Fatalf("Get on empty cache = (%v, %v), want (false, nil)", ok, err)
	}

	if err := c.Set(ctx, "k", []byte("v1"), 0); err != nil {
		t.Fatalf("Set = %v, want nil", err)
	}
	got, ok, err := c.Get(ctx, "k")
	if !ok || err != nil || string(got) != "v1" {
		t.Fatalf("Get = (%q, %v, %v), want (v1, true, nil)", got, ok, err)
	}

	// Mutating the returned bytes must not corrupt the cached value.
	got[0] = 'X'
	got2, _, _ := c.Get(ctx, "k")
	if string(got2) != "v1" {
		t.Errorf("cached value mutated through returned slice: %q", got2)
	}

	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete = %v, want nil", err)
	}
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("Get after Delete = hit, want miss")
	}
	// Deleting a missing key is not an error.
	if err := c.Delete(ctx, "missing"); err != nil {
		t.Errorf("Delete(missing) = %v, want nil", err)
	}
}

func TestNew_MemoryExpiry(t *testing.T) {
	c, err := cache.New(context.Background(), &testSettings{
		driver: "memory",
		ttl:    30 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("New(memory) = %v", err)
	}
	defer func() { _ = c.(cache.Closer).Close() }()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatalf("Set = %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("Get after TTL = hit, want miss")
	}
}

func TestNew_UnsupportedDriver(t *testing.T) {
	if _, err := cache.New(context.Background(), &testSettings{driver: "memcached"}, nil); err == nil {
		t.Fatal("New(memcached) = nil error, want error")
	}
}

func TestNew_RedisUnreachable(t *testing.T) {
	s := &testSettings{driver: "redis", addr: "127.0.0.1:1"}
	if _, err := cache.New(context.Background(), s, nil); err == nil {
		t.Fatal("New(redis, unreachable) = nil error, want error")
	}
}

// TestNew_RedisLive runs only when REDIS_TEST_ADDR points at a real server:
// go test ./internal/cache/ -run RedisLive
func TestNew_RedisLive(t *testing.T) {
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("set REDIS_TEST_ADDR=host:port to run live redis tests")
	}
	s := &testSettings{driver: "redis", addr: addr, ttl: time.Minute, prefix: "test:"}
	c, err := cache.New(context.Background(), s, nil)
	if err != nil {
		t.Fatalf("New(redis) = %v", err)
	}
	defer func() { _ = c.(cache.Closer).Close() }()

	ctx := context.Background()
	if err := c.Set(ctx, "live", []byte("ok"), time.Minute); err != nil {
		t.Fatalf("Set = %v", err)
	}
	got, ok, err := c.Get(ctx, "live")
	if !ok || err != nil || string(got) != "ok" {
		t.Fatalf("Get = (%q, %v, %v), want (ok, true, nil)", got, ok, err)
	}
	if err := c.Delete(ctx, "live"); err != nil {
		t.Fatalf("Delete = %v", err)
	}
}

func TestMustNew_FallsBackToNoopOnError(t *testing.T) {
	c := cache.MustNew(context.Background(), &testSettings{driver: "redis", addr: "127.0.0.1:1"}, nil)
	if _, ok := c.(cache.Noop); !ok {
		t.Errorf("MustNew(unreachable redis) returned %T, want cache.Noop fallback", c)
	}
}
