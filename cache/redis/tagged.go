package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
)

// RedisTaggedCache implements a Redis-backed tagged cache using Redis Sets.
type RedisTaggedCache struct {
	store *Redis
	tags  []string
}

// NewRedisTaggedCache creates a new tagged cache instance.
func NewRedisTaggedCache(store *Redis, tags []string) *RedisTaggedCache {
	return &RedisTaggedCache{
		store: store,
		tags:  tags,
	}
}

// taggedKey creates a key that includes tag information.
func (t *RedisTaggedCache) taggedKey(key string) string {
	return fmt.Sprintf("tag:%s:%s", strings.Join(t.tags, ":"), key)
}

// tagSetKey returns the Redis set key for a specific tag.
func (t *RedisTaggedCache) tagSetKey(tag string) string {
	return t.store.prefixKey(fmt.Sprintf("tag:%s:keys", tag))
}

// addKeyToTags adds a key to all tag sets.
func (t *RedisTaggedCache) addKeyToTags(ctx context.Context, key string) error {
	taggedKey := t.taggedKey(key)

	for _, tag := range t.tags {
		if err := t.store.client.SAdd(ctx, t.tagSetKey(tag), taggedKey).Err(); err != nil {
			return fmt.Errorf("redis sadd: %w", err)
		}
	}

	return nil
}

// Get retrieves an item from cache.
func (t *RedisTaggedCache) Get(ctx context.Context, key string, dest any) error {
	return t.store.Get(ctx, t.taggedKey(key), dest)
}

// Put stores an item in cache with TTL.
func (t *RedisTaggedCache) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := t.store.Put(ctx, t.taggedKey(key), value, ttl); err != nil {
		return err
	}
	return t.addKeyToTags(ctx, key)
}

// Forever stores an item without expiration.
func (t *RedisTaggedCache) Forever(ctx context.Context, key string, value any) error {
	if err := t.store.Forever(ctx, t.taggedKey(key), value); err != nil {
		return err
	}
	return t.addKeyToTags(ctx, key)
}

// Has checks if item exists in cache.
func (t *RedisTaggedCache) Has(ctx context.Context, key string) (bool, error) {
	return t.store.Has(ctx, t.taggedKey(key))
}

// Missing checks if item doesn't exist.
func (t *RedisTaggedCache) Missing(ctx context.Context, key string) (bool, error) {
	return t.store.Missing(ctx, t.taggedKey(key))
}

// Increment increments a numeric value.
func (t *RedisTaggedCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	if err := t.addKeyToTags(ctx, key); err != nil {
		return 0, err
	}
	return t.store.Increment(ctx, t.taggedKey(key), value)
}

// Decrement decrements a numeric value.
func (t *RedisTaggedCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	if err := t.addKeyToTags(ctx, key); err != nil {
		return 0, err
	}
	return t.store.Decrement(ctx, t.taggedKey(key), value)
}

// Forget removes an item from cache.
func (t *RedisTaggedCache) Forget(ctx context.Context, key string) error {
	taggedKey := t.taggedKey(key)

	// Remove from tag sets
	for _, tag := range t.tags {
		if err := t.store.client.SRem(ctx, t.tagSetKey(tag), taggedKey).Err(); err != nil {
			return fmt.Errorf("redis srem: %w", err)
		}
	}

	return t.store.Forget(ctx, taggedKey)
}

// Flush clears all items from cache (not tag-aware).
func (t *RedisTaggedCache) Flush(ctx context.Context) error {
	return t.store.Flush(ctx)
}

// FlushTags flushes all items with these tags.
func (t *RedisTaggedCache) FlushTags(ctx context.Context) error {
	for _, tag := range t.tags {
		setKey := t.tagSetKey(tag)

		// Get all keys for this tag
		keys, err := t.store.client.SMembers(ctx, setKey).Result()
		if err != nil {
			return fmt.Errorf("redis smembers: %w", err)
		}

		// Delete all tagged keys
		if len(keys) > 0 {
			// Add prefix to keys
			prefixedKeys := make([]string, len(keys))
			for i, key := range keys {
				prefixedKeys[i] = t.store.prefixKey(key)
			}

			if err := t.store.client.Del(ctx, prefixedKeys...).Err(); err != nil {
				return fmt.Errorf("redis del: %w", err)
			}
		}

		// Delete the tag set itself
		if err := t.store.client.Del(ctx, setKey).Err(); err != nil {
			return fmt.Errorf("redis del tag set: %w", err)
		}
	}

	return nil
}

// Remember gets value or stores callback result (cache-aside pattern).
func (t *RedisTaggedCache) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
	// Try to get from cache
	err := t.Get(ctx, key, dest)
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
	if err := t.Put(ctx, key, value, ttl); err != nil {
		return err
	}

	// Copy value to dest
	return internal.CopyValue(value, dest)
}

// RememberForever gets value or stores callback result forever.
func (t *RedisTaggedCache) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
	err := t.Get(ctx, key, dest)
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

	if err := t.Forever(ctx, key, value); err != nil {
		return err
	}

	return internal.CopyValue(value, dest)
}

// Pull retrieves and deletes an item.
func (t *RedisTaggedCache) Pull(ctx context.Context, key string, dest any) error {
	err := t.Get(ctx, key, dest)
	if err != nil {
		return err
	}

	return t.Forget(ctx, key)
}

// Add stores item only if it doesn't exist.
func (t *RedisTaggedCache) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	has, err := t.Has(ctx, key)
	if err != nil {
		return false, err
	}

	if has {
		return false, nil
	}

	return true, t.Put(ctx, key, value, ttl)
}

// Tags returns a new tagged cache instance with additional tags.
func (t *RedisTaggedCache) Tags(names ...string) cache.TaggedStore {
	allTags := append(t.tags, names...)
	return NewRedisTaggedCache(t.store, allTags)
}

// Lock creates a cache lock.
func (t *RedisTaggedCache) Lock(name string, ttl time.Duration) cache.Lock {
	return t.store.Lock(name, ttl)
}
