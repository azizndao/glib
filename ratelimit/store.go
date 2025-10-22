package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/azizndao/glib/router"
)

const (
	// DefaultCleanupInterval is the interval at which the memory store cleanup runs
	DefaultCleanupInterval = time.Minute

	// DefaultMaxAge is the maximum age of entries before they are removed during cleanup
	DefaultMaxAge = 10 * time.Minute
)

// Store is the interface for rate limit storage backends
// Implementations can use in-memory maps, Redis, Memcached, etc.
type Store interface {
	// Increment increments the counter for the given key and returns the new count
	// If the key doesn't exist or has expired, it creates a new counter starting at 1
	// Returns: (current count, time until window expires, error)
	Increment(ctx context.Context, key string, window time.Duration) (int, time.Duration, error)

	// Decrement decrements the counter for the given key
	// Returns error if key doesn't exist
	Decrement(ctx context.Context, key string) error

	// Get returns the current count for the given key
	// Returns: (current count, time until window expires, error)
	Get(ctx context.Context, key string) (int, time.Duration, error)

	// Reset resets the counter for the given key
	Reset(ctx context.Context, key string) error

	// Close closes the store and cleans up resources
	Close() error
}

// Config holds configuration for the RateLimit middleware
type Config struct {
	// Max is the maximum number of requests allowed in the time window
	Max int

	// Window is the time window for rate limiting
	Window time.Duration

	// Store is the storage backend for rate limit counters
	// Default: NewMemoryStore()
	Store Store

	// KeyGenerator is a function that generates a unique key for each client
	// Default: uses IP address
	KeyGenerator func(*router.Ctx) string

	// Handler is called when rate limit is exceeded
	// Default: returns 429 Too Many Requests
	Handler router.Handler

	// SkipFailedRequests determines if failed requests should be counted
	// Default: false
	SkipFailedRequests bool

	// SkipSuccessfulRequests determines if successful requests should be counted
	// Default: false
	SkipSuccessfulRequests bool

	// HeaderPrefix is the prefix for rate limit headers
	// Default: "X-RateLimit-"
	HeaderPrefix string
}

// RateLimitConfig is an alias for Config (for backwards compatibility)
type RateLimitConfig = Config

// MemoryStore implements Store interface using an in-memory map
type MemoryStore struct {
	entries map[string]*memoryEntry
	mu      sync.RWMutex
	cleanup *time.Ticker
	done    chan struct{}
}

// memoryEntry tracks request count and window start time for a client
type memoryEntry struct {
	count       int
	windowStart time.Time
	mu          sync.Mutex
}

// NewMemoryStore creates a new in-memory store for rate limiting
// The cleanup runs every DefaultCleanupInterval to remove entries older than DefaultMaxAge
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		entries: make(map[string]*memoryEntry),
		done:    make(chan struct{}),
		cleanup: time.NewTicker(DefaultCleanupInterval),
	}

	// Start cleanup goroutine
	go store.cleanupRoutine()

	return store
}

// Increment increments the counter for the given key
func (m *MemoryStore) Increment(ctx context.Context, key string, window time.Duration) (int, time.Duration, error) {
	now := time.Now()

	m.mu.Lock()
	entry, exists := m.entries[key]
	if !exists {
		entry = &memoryEntry{
			count:       1,
			windowStart: now,
		}
		m.entries[key] = entry
		m.mu.Unlock()
		return 1, window, nil
	}
	m.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if window has expired
	elapsed := now.Sub(entry.windowStart)
	if elapsed > window {
		// Reset window
		entry.count = 1
		entry.windowStart = now
		return 1, window, nil
	}

	// Increment count
	entry.count++
	ttl := window - elapsed
	return entry.count, ttl, nil
}

// Decrement decrements the counter for the given key
func (m *MemoryStore) Decrement(ctx context.Context, key string) error {
	m.mu.RLock()
	entry, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		return nil // Key doesn't exist, nothing to decrement
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.count > 0 {
		entry.count--
	}
	return nil
}

// Get returns the current count for the given key
func (m *MemoryStore) Get(ctx context.Context, key string) (int, time.Duration, error) {
	m.mu.RLock()
	entry, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		return 0, 0, nil
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	return entry.count, 0, nil
}

// Reset resets the counter for the given key
func (m *MemoryStore) Reset(ctx context.Context, key string) error {
	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()
	return nil
}

// Close stops the cleanup goroutine and cleans up resources
func (m *MemoryStore) Close() error {
	close(m.done)
	return nil
}

// cleanupRoutine periodically removes expired entries
func (m *MemoryStore) cleanupRoutine() {
	for {
		select {
		case <-m.cleanup.C:
			now := time.Now()

			m.mu.Lock()
			for key, entry := range m.entries {
				entry.mu.Lock()
				if now.Sub(entry.windowStart) > DefaultMaxAge {
					delete(m.entries, key)
				}
				entry.mu.Unlock()
			}
			m.mu.Unlock()
		case <-m.done:
			m.cleanup.Stop()
			return
		}
	}
}
