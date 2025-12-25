package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
)

// item represents a cached item with value and expiration.
type item struct {
	value      any
	expiration time.Time
	forever    bool
}

func (i *item) isExpired() bool {
	return !i.forever && time.Now().After(i.expiration)
}

// MemoryStats holds statistics for the memory cache.
type MemoryStats struct {
	Hits        atomic.Int64
	Misses      atomic.Int64
	Writes      atomic.Int64
	Deletes     atomic.Int64
	Evictions   atomic.Int64
	CurrentSize atomic.Int64
}

// Memory implements an in-memory cache store with automatic cleanup and statistics.
type Memory struct {
	items        map[string]*item
	mu           sync.RWMutex
	cleanupTimer *time.Ticker
	stopCleanup  chan bool
	stats        *MemoryStats
}

// New creates a new in-memory cache store.
func New() *Memory {
	store := &Memory{
		items:        make(map[string]*item),
		cleanupTimer: time.NewTicker(1 * time.Minute),
		stopCleanup:  make(chan bool),
		stats:        &MemoryStats{},
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// cleanup removes expired items periodically.
func (m *Memory) cleanup() {
	for {
		select {
		case <-m.cleanupTimer.C:
			m.removeExpired()
		case <-m.stopCleanup:
			m.cleanupTimer.Stop()
			return
		}
	}
}

// removeExpired removes all expired items.
func (m *Memory) removeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	evicted := int64(0)
	for key, item := range m.items {
		if !item.forever && now.After(item.expiration) {
			delete(m.items, key)
			evicted++
		}
	}

	if evicted > 0 {
		m.stats.Evictions.Add(evicted)
		m.stats.CurrentSize.Add(-evicted)
	}
}

// Stop stops the cleanup goroutine. Call this when shutting down.
func (m *Memory) Stop() {
	m.stopCleanup <- true
}

// GetStats returns current cache statistics.
func (m *Memory) GetStats() cache.Statistics {
	return cache.Statistics{
		Hits:    m.stats.Hits.Load(),
		Misses:  m.stats.Misses.Load(),
		Writes:  m.stats.Writes.Load(),
		Deletes: m.stats.Deletes.Load(),
	}
}

// ResetStats resets all statistics counters.
func (m *Memory) ResetStats() {
	m.stats.Hits.Store(0)
	m.stats.Misses.Store(0)
	m.stats.Writes.Store(0)
	m.stats.Deletes.Store(0)
	m.stats.Evictions.Store(0)
}

// Size returns the current number of items in the cache.
func (m *Memory) Size() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.items))
}

// Get retrieves an item from cache.
func (m *Memory) Get(ctx context.Context, key string, dest any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.items[key]
	if !exists {
		m.stats.Misses.Add(1)
		return cache.ErrCacheMiss
	}

	if item.isExpired() {
		m.stats.Misses.Add(1)
		return cache.ErrCacheMiss
	}

	m.stats.Hits.Add(1)
	return internal.CopyValue(item.value, dest)
}

// Put stores an item in cache with TTL.
func (m *Memory) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, existed := m.items[key]

	m.items[key] = &item{
		value:      value,
		expiration: time.Now().Add(ttl),
		forever:    false,
	}

	m.stats.Writes.Add(1)
	if !existed {
		m.stats.CurrentSize.Add(1)
	}

	return nil
}

// Forever stores an item without expiration.
func (m *Memory) Forever(ctx context.Context, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, existed := m.items[key]

	m.items[key] = &item{
		value:   value,
		forever: true,
	}

	m.stats.Writes.Add(1)
	if !existed {
		m.stats.CurrentSize.Add(1)
	}

	return nil
}

// Has checks if an item exists in cache.
func (m *Memory) Has(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.items[key]
	if !exists {
		return false, nil
	}

	return !item.isExpired(), nil
}

// Missing checks if an item doesn't exist.
func (m *Memory) Missing(ctx context.Context, key string) (bool, error) {
	has, err := m.Has(ctx, key)
	return !has, err
}

// Increment increments a numeric value.
func (m *Memory) Increment(ctx context.Context, key string, value int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingItem, exists := m.items[key]
	if !exists || existingItem.isExpired() {
		// Initialize with the increment value
		m.items[key] = &item{
			value:   value,
			forever: true,
		}
		m.stats.Writes.Add(1)
		if !exists {
			m.stats.CurrentSize.Add(1)
		}
		return value, nil
	}

	// Try to get current value as int64
	var current int64
	switch v := existingItem.value.(type) {
	case int64:
		current = v
	case int:
		current = int64(v)
	case float64:
		current = int64(v)
	default:
		m.stats.Misses.Add(1)
		return 0, cache.ErrCacheMiss
	}

	newValue := current + value
	existingItem.value = newValue
	m.stats.Writes.Add(1)

	return newValue, nil
}

// Decrement decrements a numeric value.
func (m *Memory) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	return m.Increment(ctx, key, -value)
}

// Forget removes an item from cache.
func (m *Memory) Forget(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.items[key]; exists {
		delete(m.items, key)
		m.stats.Deletes.Add(1)
		m.stats.CurrentSize.Add(-1)
	}

	return nil
}

// Flush clears all items from cache.
func (m *Memory) Flush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := int64(len(m.items))
	m.items = make(map[string]*item)
	m.stats.Deletes.Add(count)
	m.stats.CurrentSize.Store(0)

	return nil
}

// Remember implements the cache-aside pattern.
func (m *Memory) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
	// Try to get from cache
	err := m.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != cache.ErrCacheMiss {
		return err
	}

	// Execute callback
	value, err := callback()
	if err != nil {
		return err
	}

	// Store in cache
	if err := m.Put(ctx, key, value, ttl); err != nil {
		return err
	}

	return internal.CopyValue(value, dest)
}

// RememberForever is like Remember but stores forever.
func (m *Memory) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
	err := m.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != cache.ErrCacheMiss {
		return err
	}

	value, err := callback()
	if err != nil {
		return err
	}

	if err := m.Forever(ctx, key, value); err != nil {
		return err
	}

	return internal.CopyValue(value, dest)
}

// Pull retrieves and deletes an item.
func (m *Memory) Pull(ctx context.Context, key string, dest any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.items[key]
	if !exists || item.isExpired() {
		m.stats.Misses.Add(1)
		return cache.ErrCacheMiss
	}

	if err := internal.CopyValue(item.value, dest); err != nil {
		return err
	}

	delete(m.items, key)
	m.stats.Hits.Add(1)
	m.stats.Deletes.Add(1)
	m.stats.CurrentSize.Add(-1)

	return nil
}

// Add stores an item only if it doesn't exist.
func (m *Memory) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingItem, exists := m.items[key]
	if exists && !existingItem.isExpired() {
		return false, nil
	}

	m.items[key] = &item{
		value:      value,
		expiration: time.Now().Add(ttl),
		forever:    false,
	}

	m.stats.Writes.Add(1)
	if !exists {
		m.stats.CurrentSize.Add(1)
	}

	return true, nil
}

// Tags returns a tagged cache instance.
func (m *Memory) Tags(names ...string) cache.TaggedStore {
	return newTaggedMemory(m, names)
}

// Lock creates a cache lock (in-memory implementation).
func (m *Memory) Lock(name string, ttl time.Duration) cache.Lock {
	return newMemoryLock(m, name, ttl)
}
