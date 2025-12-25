package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
	"github.com/redis/go-redis/v9"
)

// Redis implements a Redis-backed cache store.
type Redis struct {
	client *redis.Client
	prefix string
}

// NewRedis creates a new Redis cache store.
func New(client *redis.Client, prefix string) *Redis {
	if prefix != "" && prefix[len(prefix)-1] != ':' {
		prefix = prefix + ":"
	}
	return &Redis{
		client: client,
		prefix: prefix,
	}
}

// prefixKey adds the prefix to a cache key.
func (r *Redis) prefixKey(key string) string {
	return r.prefix + key
}

// Get retrieves an item from cache.
func (r *Redis) Get(ctx context.Context, key string, dest any) error {
	data, err := r.client.Get(ctx, r.prefixKey(key)).Bytes()
	if err == redis.Nil {
		return cache.ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("redis get: %w", err)
	}

	return json.Unmarshal(data, dest)
}

// Put stores an item in cache with TTL.
func (r *Redis) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis marshal: %w", err)
	}

	return r.client.Set(ctx, r.prefixKey(key), data, ttl).Err()
}

// Forever stores an item without expiration.
func (r *Redis) Forever(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis marshal: %w", err)
	}

	return r.client.Set(ctx, r.prefixKey(key), data, 0).Err()
}

// Has checks if item exists in cache.
func (r *Redis) Has(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, r.prefixKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return count > 0, nil
}

// Missing checks if item doesn't exist.
func (r *Redis) Missing(ctx context.Context, key string) (bool, error) {
	has, err := r.Has(ctx, key)
	return !has, err
}

// Increment increments a numeric value.
func (r *Redis) Increment(ctx context.Context, key string, value int64) (int64, error) {
	result, err := r.client.IncrBy(ctx, r.prefixKey(key), value).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incrby: %w", err)
	}
	return result, nil
}

// Decrement decrements a numeric value.
func (r *Redis) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	result, err := r.client.DecrBy(ctx, r.prefixKey(key), value).Result()
	if err != nil {
		return 0, fmt.Errorf("redis decrby: %w", err)
	}
	return result, nil
}

// Forget removes an item from cache.
func (r *Redis) Forget(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefixKey(key)).Err()
}

// Flush clears all items from cache (flushes the entire database).
func (r *Redis) Flush(ctx context.Context) error {
	// If there's a prefix, only delete keys with that prefix
	if r.prefix != "" {
		pattern := r.prefix + "*"
		iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

		var keys []string
		for iter.Next(ctx) {
			keys = append(keys, iter.Val())
		}

		if err := iter.Err(); err != nil {
			return fmt.Errorf("redis scan: %w", err)
		}

		if len(keys) > 0 {
			return r.client.Del(ctx, keys...).Err()
		}
		return nil
	}

	// No prefix, flush entire database
	return r.client.FlushDB(ctx).Err()
}

// Remember gets value or stores callback result (cache-aside pattern).
func (r *Redis) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
	// Try to get from cache
	err := r.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}

	if err != cache.ErrCacheMiss {
		return err // Real error
	}

	// Cache miss - execute callback
	value, err := callback()
	if err != nil {
		return err
	}

	// Store in cache
	if err := r.Put(ctx, key, value, ttl); err != nil {
		return err
	}

	// Copy value to dest
	return internal.CopyValue(value, dest)
}

// RememberForever gets value or stores callback result forever.
func (r *Redis) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
	err := r.Get(ctx, key, dest)
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

	if err := r.Forever(ctx, key, value); err != nil {
		return err
	}

	return internal.CopyValue(value, dest)
}

// Pull retrieves and deletes an item.
func (r *Redis) Pull(ctx context.Context, key string, dest any) error {
	err := r.Get(ctx, key, dest)
	if err != nil {
		return err
	}

	return r.Forget(ctx, key)
}

// Add stores item only if it doesn't exist.
func (r *Redis) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("redis marshal: %w", err)
	}

	success, err := r.client.SetNX(ctx, r.prefixKey(key), data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}

	return success, nil
}

// Tags returns a tagged cache instance.
func (r *Redis) Tags(names ...string) cache.TaggedStore {
	return NewRedisTaggedCache(r, names)
}

// Lock creates a cache lock.
func (r *Redis) Lock(name string, ttl time.Duration) cache.Lock {
	return NewRedisLock(r.client, r.prefixKey("lock:"+name), ttl)
}
