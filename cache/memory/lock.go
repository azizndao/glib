package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/azizndao/glib/cache"
)

// MemoryLock implements an in-memory lock (single-process only).
type MemoryLock struct {
	store    *Memory
	name     string
	ttl      time.Duration
	owner    string
	mu       sync.Mutex
	acquired bool
	lockKey  string
}

// newMemoryLock creates a new memory lock.
func newMemoryLock(store *Memory, name string, ttl time.Duration) *MemoryLock {
	return &MemoryLock{
		store:   store,
		name:    name,
		ttl:     ttl,
		owner:   fmt.Sprintf("lock:%d", time.Now().UnixNano()),
		lockKey: fmt.Sprintf("lock:%s", name),
	}
}

// Acquire attempts to acquire the lock.
func (l *MemoryLock) Acquire(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Try to add the lock (only succeeds if it doesn't exist)
	success, err := l.store.Add(ctx, l.lockKey, l.owner, l.ttl)
	if err != nil {
		return false, err
	}

	if success {
		l.acquired = true
		return true, nil
	}

	return false, nil
}

// Release releases the lock.
func (l *MemoryLock) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.acquired {
		return cache.ErrLockNotAcquired
	}

	// Only release if we own the lock
	var currentOwner string
	err := l.store.Get(ctx, l.lockKey, &currentOwner)
	if err != nil {
		return err
	}

	if currentOwner != l.owner {
		return fmt.Errorf("cannot release lock owned by another process")
	}

	err = l.store.Forget(ctx, l.lockKey)
	if err == nil {
		l.acquired = false
	}

	return err
}

// ForceRelease forcefully releases the lock regardless of ownership.
func (l *MemoryLock) ForceRelease(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.store.Forget(ctx, l.lockKey)
	if err == nil {
		l.acquired = false
	}

	return err
}

// Owner returns the lock owner identifier.
func (l *MemoryLock) Owner() string {
	return l.owner
}

// Get executes callback while holding lock.
// It acquires the lock, executes the callback, then releases the lock.
func (l *MemoryLock) Get(ctx context.Context, callback func() error) error {
	acquired, err := l.Acquire(ctx)
	if err != nil {
		return err
	}

	if !acquired {
		return cache.ErrLockNotAcquired
	}

	defer l.Release(ctx)

	return callback()
}

// Block waits to acquire lock then executes callback.
// It retries acquiring the lock until waitTime expires.
func (l *MemoryLock) Block(ctx context.Context, waitTime time.Duration, callback func() error) error {
	deadline := time.Now().Add(waitTime)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Try to acquire
		acquired, err := l.Acquire(ctx)
		if err != nil {
			return err
		}

		if acquired {
			defer l.Release(ctx)
			return callback()
		}

		// Check if we've exceeded wait time
		if time.Now().After(deadline) {
			return cache.ErrLockTimeout
		}

		// Wait before retrying
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
