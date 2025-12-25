// Package cache provides a flexible caching layer with support for multiple drivers,
// cache tags, distributed locks, and the cache-aside (remember) pattern.
//
// Example usage:
//
//	// Create cache manager
//	manager := cache.NewManager()
//	manager.RegisterDriver("memory", func() cache.Store {
//	    return memory.New()
//	})
//
//	// Get default cache
//	cache, _ := manager.Default()
//
//	// Basic operations
//	cache.Put(ctx, "key", value, 5*time.Minute)
//	cache.Get(ctx, "key", &result)
//
//	// Remember pattern (cache-aside)
//	var user User
//	cache.Remember(ctx, "user:123", &user, 1*time.Hour, func() (any, error) {
//	    return db.FindUser(123)
//	})
//
//	// Cache tags
//	tagged := cache.Tags("users", "posts")
//	tagged.Put(ctx, "feed:123", feed, 1*time.Hour)
//	tagged.FlushTags(ctx) // Invalidate all user/post caches
//
//	// Distributed locks
//	lock := cache.Lock("process-job", 30*time.Second)
//	lock.Get(ctx, func() error {
//	    // Critical section
//	    return processJob()
//	})
package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss is returned when a cache key is not found
	ErrCacheMiss = errors.New("cache: key not found")

	// ErrLockNotAcquired is returned when a lock cannot be acquired
	ErrLockNotAcquired = errors.New("cache: lock not acquired")

	// ErrLockTimeout is returned when lock acquisition times out
	ErrLockTimeout = errors.New("cache: lock timeout")
)

// Store represents a cache store with support for tags and locks.
type Store interface {
	// Get retrieves an item from cache
	Get(ctx context.Context, key string, dest any) error

	// Put stores an item in cache with TTL
	Put(ctx context.Context, key string, value any, ttl time.Duration) error

	// Forever stores an item without expiration
	Forever(ctx context.Context, key string, value any) error

	// Has checks if item exists in cache
	Has(ctx context.Context, key string) (bool, error)

	// Missing checks if item doesn't exist
	Missing(ctx context.Context, key string) (bool, error)

	// Increment increments a numeric value
	Increment(ctx context.Context, key string, value int64) (int64, error)

	// Decrement decrements a numeric value
	Decrement(ctx context.Context, key string, value int64) (int64, error)

	// Forget removes an item from cache
	Forget(ctx context.Context, key string) error

	// Flush clears all items from cache
	Flush(ctx context.Context) error

	// Remember gets value or stores callback result (cache-aside pattern)
	Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error

	// RememberForever gets value or stores callback result forever
	RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error

	// Pull retrieves and deletes an item
	Pull(ctx context.Context, key string, dest any) error

	// Add stores item only if it doesn't exist
	Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)

	// Tags returns a tagged cache instance
	Tags(names ...string) TaggedStore

	// Lock creates a cache lock
	Lock(name string, ttl time.Duration) Lock
}

// TaggedStore represents a cache with tags for grouped invalidation.
type TaggedStore interface {
	Store

	// FlushTags flushes all items with these tags
	FlushTags(ctx context.Context) error
}

// Lock represents a distributed lock.
type Lock interface {
	// Acquire attempts to acquire the lock
	Acquire(ctx context.Context) (bool, error)

	// Release releases the lock
	Release(ctx context.Context) error

	// ForceRelease forcefully releases the lock
	ForceRelease(ctx context.Context) error

	// Owner returns the lock owner identifier
	Owner() string

	// Get executes callback while holding lock
	Get(ctx context.Context, callback func() error) error

	// Block waits to acquire lock then executes callback
	Block(ctx context.Context, waitTime time.Duration, callback func() error) error
}

// Driver is a factory function that creates a cache store.
type Driver func() Store

// Statistics holds cache performance metrics.
type Statistics struct {
	Hits    int64
	Misses  int64
	Writes  int64
	Deletes int64
}

// HitRatio returns the cache hit ratio (0.0 to 1.0).
func (s *Statistics) HitRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}
