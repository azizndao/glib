package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/azizndao/glib/ratelimit"
	"github.com/azizndao/glib/router"
	"github.com/azizndao/glib/slog"
	"github.com/azizndao/glib/validation"
)

func ExampleRateLimit_perRoute() {
	validator := validation.New(validation.DefaultValidatorConfig())
	r := router.New(slog.Create(), validator)

	// Global rate limit: 1000 requests per minute
	r.Use(ratelimit.RateLimit(ratelimit.Config{
		Max:    1000,
		Window: time.Minute,
	}))

	// API routes with stricter limit: 10 requests per minute
	apiGroup := r.SubRouter("/api")
	apiGroup.Use(ratelimit.RateLimit(ratelimit.Config{
		Max:    10,
		Window: time.Minute,
		KeyGenerator: func(c *router.Ctx) string {
			return "api:" + c.IP()
		},
	}))

	apiGroup.Get("/sensitive", func(c *router.Ctx) error {
		return c.JSON(map[string]string{"data": "sensitive"})
	})
}

// Test basic rate limiting functionality
func TestMemoryStore_Increment(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	window := time.Minute

	// First increment
	count, ttl, err := store.Increment(ctx, key, window)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
	if ttl != window {
		t.Errorf("expected ttl %v, got %v", window, ttl)
	}

	// Second increment
	count, ttl, err = store.Increment(ctx, key, window)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// TTL should be less than window now
	if ttl >= window {
		t.Errorf("expected ttl < %v, got %v", window, ttl)
	}
}

// Test window expiration
func TestMemoryStore_WindowExpiration(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	window := 100 * time.Millisecond

	// First increment
	count, _, err := store.Increment(ctx, key, window)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should reset to 1
	count, _, err = store.Increment(ctx, key, window)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1 after expiration, got %d", count)
	}
}

// Test Get method
func TestMemoryStore_Get(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	window := time.Minute

	// Non-existent key
	count, _, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 for non-existent key, got %d", count)
	}

	// Increment and get
	store.Increment(ctx, key, window)
	store.Increment(ctx, key, window)

	count, _, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

// Test Reset method
func TestMemoryStore_Reset(t *testing.T) {
	store := ratelimit.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	window := time.Minute

	// Increment a few times
	store.Increment(ctx, key, window)
	store.Increment(ctx, key, window)

	// Reset
	err := store.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be back to 0
	count, _, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 after reset, got %d", count)
	}
}

// Benchmark memory store increment
func BenchmarkMemoryStore_Increment(b *testing.B) {
	store := ratelimit.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	window := time.Minute

	for b.Loop() {
		key := "bench-key"
		store.Increment(ctx, key, window)
	}
}

// Benchmark memory store with different keys (simulates real usage)
func BenchmarkMemoryStore_IncrementDifferentKeys(b *testing.B) {
	store := ratelimit.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	window := time.Minute

	for i := 0; b.Loop(); i++ {
		key := "bench-key-" + string(rune(i%100))
		store.Increment(ctx, key, window)
	}
}
