package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
	"github.com/redis/go-redis/v9"
)

// RedisLock implements distributed locking with Redis using SETNX and Lua scripts.
type RedisLock struct {
	client *redis.Client
	name   string
	owner  string
	ttl    time.Duration
}

// NewRedisLock creates a new Redis-based distributed lock.
func NewRedisLock(client *redis.Client, name string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		client: client,
		name:   name,
		owner:  internal.GenerateOwnerID(),
		ttl:    ttl,
	}
}

// Acquire attempts to acquire the lock.
func (l *RedisLock) Acquire(ctx context.Context) (bool, error) {
	success, err := l.client.SetNX(ctx, l.name, l.owner, l.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis lock acquire: %w", err)
	}
	return success, nil
}

// Release releases the lock only if owned by this instance.
func (l *RedisLock) Release(ctx context.Context) error {
	// Lua script to atomically check owner and delete
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	err := l.client.Eval(ctx, script, []string{l.name}, l.owner).Err()
	if err != nil {
		return fmt.Errorf("redis lock release: %w", err)
	}
	return nil
}

// ForceRelease forcefully releases the lock regardless of owner.
func (l *RedisLock) ForceRelease(ctx context.Context) error {
	err := l.client.Del(ctx, l.name).Err()
	if err != nil {
		return fmt.Errorf("redis lock force release: %w", err)
	}
	return nil
}

// Owner returns the lock owner identifier.
func (l *RedisLock) Owner() string {
	return l.owner
}

// Get executes callback while holding the lock.
func (l *RedisLock) Get(ctx context.Context, callback func() error) error {
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
func (l *RedisLock) Block(ctx context.Context, waitTime time.Duration, callback func() error) error {
	deadline := time.Now().Add(waitTime)

	for {
		acquired, err := l.Acquire(ctx)
		if err != nil {
			return err
		}

		if acquired {
			defer l.Release(ctx)
			return callback()
		}

		if time.Now().After(deadline) {
			return cache.ErrLockTimeout
		}

		// Wait a bit before retrying
		select {
		case <-time.After(100 * time.Millisecond):
			// Continue loop
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

