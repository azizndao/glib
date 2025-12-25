package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
)

// TaggedMemory implements a tagged cache store for the memory driver.
type TaggedMemory struct {
	store *Memory
	tags  []string
}

// newTaggedMemory creates a new tagged memory store.
func newTaggedMemory(store *Memory, tags []string) *TaggedMemory {
	return &TaggedMemory{
		store: store,
		tags:  tags,
	}
}

// tagKey returns the tag set key.
func (t *TaggedMemory) tagKey() string {
	return "tag:" + strings.Join(t.tags, "|")
}

// taggedKey returns a cache key prefixed with the tag namespace.
func (t *TaggedMemory) taggedKey(key string) string {
	return fmt.Sprintf("tagged:%s:%s", t.tagKey(), key)
}

// Get retrieves an item from the tagged cache.
func (t *TaggedMemory) Get(ctx context.Context, key string, dest any) error {
	return t.store.Get(ctx, t.taggedKey(key), dest)
}

// Put stores an item in the tagged cache.
func (t *TaggedMemory) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
	// Store the key in the tag set
	t.addKeyToTagSet(ctx, key)
	return t.store.Put(ctx, t.taggedKey(key), value, ttl)
}

// Forever stores an item in the tagged cache without expiration.
func (t *TaggedMemory) Forever(ctx context.Context, key string, value any) error {
	t.addKeyToTagSet(ctx, key)
	return t.store.Forever(ctx, t.taggedKey(key), value)
}

// Has checks if an item exists in the tagged cache.
func (t *TaggedMemory) Has(ctx context.Context, key string) (bool, error) {
	return t.store.Has(ctx, t.taggedKey(key))
}

// Missing checks if an item doesn't exist in the tagged cache.
func (t *TaggedMemory) Missing(ctx context.Context, key string) (bool, error) {
	return t.store.Missing(ctx, t.taggedKey(key))
}

// Increment increments a numeric value in the tagged cache.
func (t *TaggedMemory) Increment(ctx context.Context, key string, value int64) (int64, error) {
	t.addKeyToTagSet(ctx, key)
	return t.store.Increment(ctx, t.taggedKey(key), value)
}

// Decrement decrements a numeric value in the tagged cache.
func (t *TaggedMemory) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	t.addKeyToTagSet(ctx, key)
	return t.store.Decrement(ctx, t.taggedKey(key), value)
}

// Forget removes an item from the tagged cache.
func (t *TaggedMemory) Forget(ctx context.Context, key string) error {
	t.removeKeyFromTagSet(ctx, key)
	return t.store.Forget(ctx, t.taggedKey(key))
}

// Flush clears all items from the tagged cache (same as FlushTags).
func (t *TaggedMemory) Flush(ctx context.Context) error {
	return t.FlushTags(ctx)
}

// Remember implements the cache-aside pattern for tagged cache.
func (t *TaggedMemory) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
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

	if err := t.Put(ctx, key, value, ttl); err != nil {
		return err
	}

	return internal.CopyValue(value, dest)
}

// RememberForever is like Remember but stores forever.
func (t *TaggedMemory) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
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

// Pull retrieves and deletes an item from the tagged cache.
func (t *TaggedMemory) Pull(ctx context.Context, key string, dest any) error {
	err := t.Get(ctx, key, dest)
	if err != nil {
		return err
	}

	_ = t.Forget(ctx, key)
	return nil
}

// Add stores an item only if it doesn't exist in the tagged cache.
func (t *TaggedMemory) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	has, err := t.Has(ctx, key)
	if err != nil {
		return false, err
	}

	if has {
		return false, nil
	}

	return true, t.Put(ctx, key, value, ttl)
}

// Tags returns a nested tagged cache instance.
func (t *TaggedMemory) Tags(names ...string) cache.TaggedStore {
	// Merge current tags with new tags
	allTags := append(t.tags, names...)
	return newTaggedMemory(t.store, allTags)
}

// Lock creates a cache lock (in-memory implementation).
func (t *TaggedMemory) Lock(name string, ttl time.Duration) cache.Lock {
	// Use the store's lock implementation with tagged key
	return t.store.Lock(t.taggedKey(name), ttl)
}

// FlushTags flushes all items with these tags.
func (t *TaggedMemory) FlushTags(ctx context.Context) error {
	// Get all keys associated with these tags
	keys := t.getKeysFromTagSet(ctx)

	// Delete all tagged keys
	for _, key := range keys {
		_ = t.store.Forget(ctx, t.taggedKey(key))
	}

	// Delete the tag set itself
	_ = t.store.Forget(ctx, t.tagKey())

	return nil
}

// addKeyToTagSet adds a key to the tag set.
func (t *TaggedMemory) addKeyToTagSet(ctx context.Context, key string) {
	var keys []string
	_ = t.store.Get(ctx, t.tagKey(), &keys)

	// Add key if not already present
	found := false
	for _, k := range keys {
		if k == key {
			found = true
			break
		}
	}

	if !found {
		keys = append(keys, key)
		_ = t.store.Forever(ctx, t.tagKey(), keys)
	}
}

// removeKeyFromTagSet removes a key from the tag set.
func (t *TaggedMemory) removeKeyFromTagSet(ctx context.Context, key string) {
	var keys []string
	if err := t.store.Get(ctx, t.tagKey(), &keys); err != nil {
		return
	}

	// Remove key
	newKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}

	if len(newKeys) > 0 {
		_ = t.store.Forever(ctx, t.tagKey(), newKeys)
	} else {
		_ = t.store.Forget(ctx, t.tagKey())
	}
}

// getKeysFromTagSet returns all keys associated with the tag set.
func (t *TaggedMemory) getKeysFromTagSet(ctx context.Context) []string {
	var keys []string
	_ = t.store.Get(ctx, t.tagKey(), &keys)
	return keys
}
