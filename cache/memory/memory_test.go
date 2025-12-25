package memory

import (
	"context"
	"testing"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory_PutAndGet(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Put a string value
	err := store.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Get the value back
	var result string
	err = store.Get(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)
}

func TestMemory_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	var result string
	err := store.Get(ctx, "nonexistent", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Expiration(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Put with short TTL
	err := store.Put(ctx, "key1", "value1", 100*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	var result string
	err = store.Get(ctx, "key1", &result)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	err = store.Get(ctx, "key1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Forever(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	err := store.Forever(ctx, "key1", "value1")
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Should still exist
	var result string
	err = store.Get(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)
}

func TestMemory_Has(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Initially shouldn't exist
	has, err := store.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)

	// Put value
	err = store.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Now should exist
	has, err = store.Has(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestMemory_Missing(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Initially should be missing
	missing, err := store.Missing(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, missing)

	// Put value
	err = store.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Now shouldn't be missing
	missing, err = store.Missing(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, missing)
}

func TestMemory_Increment(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Increment non-existent key
	val, err := store.Increment(ctx, "counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Increment again
	val, err = store.Increment(ctx, "counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), val)

	// Increment by negative (decrement)
	val, err = store.Increment(ctx, "counter", -2)
	require.NoError(t, err)
	assert.Equal(t, int64(4), val)
}

func TestMemory_Decrement(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Set initial value
	_, err := store.Increment(ctx, "counter", 10)
	require.NoError(t, err)

	// Decrement
	val, err := store.Decrement(ctx, "counter", 3)
	require.NoError(t, err)
	assert.Equal(t, int64(7), val)
}

func TestMemory_Forget(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Put value
	err := store.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Delete it
	err = store.Forget(ctx, "key1")
	require.NoError(t, err)

	// Should not exist
	var result string
	err = store.Get(ctx, "key1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Flush(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Put multiple values
	err := store.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)
	err = store.Put(ctx, "key2", "value2", 1*time.Minute)
	require.NoError(t, err)

	// Flush all
	err = store.Flush(ctx)
	require.NoError(t, err)

	// Both should be gone
	var result string
	err = store.Get(ctx, "key1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
	err = store.Get(ctx, "key2", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Remember(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	callCount := 0
	callback := func() (any, error) {
		callCount++
		return "computed", nil
	}

	// First call should execute callback
	var result string
	err := store.Remember(ctx, "key1", &result, 1*time.Minute, callback)
	require.NoError(t, err)
	assert.Equal(t, "computed", result)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	err = store.Remember(ctx, "key1", &result, 1*time.Minute, callback)
	require.NoError(t, err)
	assert.Equal(t, "computed", result)
	assert.Equal(t, 1, callCount) // Callback not called again
}

func TestMemory_RememberForever(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	callCount := 0
	callback := func() (any, error) {
		callCount++
		return "computed", nil
	}

	// First call should execute callback
	var result string
	err := store.RememberForever(ctx, "key1", &result, callback)
	require.NoError(t, err)
	assert.Equal(t, "computed", result)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	err = store.RememberForever(ctx, "key1", &result, callback)
	require.NoError(t, err)
	assert.Equal(t, "computed", result)
	assert.Equal(t, 1, callCount)
}

func TestMemory_Pull(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Put value
	err := store.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Pull it
	var result string
	err = store.Pull(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)

	// Should be gone now
	err = store.Get(ctx, "key1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Add(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Add should succeed for non-existent key
	added, err := store.Add(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, added)

	// Add should fail for existing key
	added, err = store.Add(ctx, "key1", "value2", 1*time.Minute)
	require.NoError(t, err)
	assert.False(t, added)

	// Value should still be original
	var result string
	err = store.Get(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)
}

func TestMemory_Cleanup(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Put value with short TTL
	err := store.Put(ctx, "key1", "value1", 50*time.Millisecond)
	require.NoError(t, err)

	// Trigger cleanup manually
	time.Sleep(100 * time.Millisecond)
	store.removeExpired()

	// Item should be removed
	has, err := store.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestMemory_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Test concurrent reads and writes
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := "key"
				_ = store.Put(ctx, key, n, 1*time.Minute)
				var result int
				_ = store.Get(ctx, key, &result)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMemory_Tags_BasicOperations(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Create tagged store
	tagged := store.Tags("users", "posts")

	// Put value in tagged cache
	err := tagged.Put(ctx, "key1", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Get from tagged cache
	var result string
	err = tagged.Get(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)

	// Regular store shouldn't have this key (different namespace)
	err = store.Get(ctx, "key1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Tags_FlushTags(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Create tagged stores with SAME tags
	usersCache1 := store.Tags("users")
	usersCache2 := store.Tags("users")
	postsCache := store.Tags("posts")

	// Store values in users cache
	_ = usersCache1.Put(ctx, "user1", "Alice", 1*time.Hour)
	_ = usersCache1.Put(ctx, "user2", "Bob", 1*time.Hour)

	// Store value in posts cache
	_ = postsCache.Put(ctx, "post1", "Post A", 1*time.Hour)

	// Regular cache (no tags)
	_ = store.Put(ctx, "regular", "Regular Value", 1*time.Hour)

	// Verify data exists
	var result string
	err := usersCache2.Get(ctx, "user1", &result)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)

	// Flush users tag
	err = usersCache1.FlushTags(ctx)
	require.NoError(t, err)

	// Users cache should be flushed
	err = usersCache2.Get(ctx, "user1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	err = usersCache2.Get(ctx, "user2", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	// Posts cache should still have data
	err = postsCache.Get(ctx, "post1", &result)
	require.NoError(t, err)
	assert.Equal(t, "Post A", result)

	// Regular cache should still have data
	err = store.Get(ctx, "regular", &result)
	require.NoError(t, err)
	assert.Equal(t, "Regular Value", result)
}

func TestMemory_Tags_NestedTags(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Create tagged store
	tagged := store.Tags("level1")
	nestedTagged := tagged.Tags("level2")

	// Store in nested tagged cache
	err := nestedTagged.Put(ctx, "key1", "nested value", 1*time.Minute)
	require.NoError(t, err)

	// Get from nested tagged cache
	var result string
	err = nestedTagged.Get(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, "nested value", result)

	// First level shouldn't have this key
	err = tagged.Get(ctx, "key1", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestMemory_Tags_Increment(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Use same tag to ensure we're in the same namespace
	tagged := store.Tags("counters")

	// Increment in tagged cache
	count, err := tagged.Increment(ctx, "views", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = tagged.Increment(ctx, "views", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)

	// Flush and verify it's gone
	_ = tagged.FlushTags(ctx)

	// After flush, incrementing should start fresh
	count, err = tagged.Increment(ctx, "views", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "After flush, counter should restart at 1")
}

func TestMemory_Lock_BasicAcquireRelease(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock := store.Lock("test-lock", 1*time.Minute)

	// Acquire lock
	acquired, err := lock.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Release lock
	err = lock.Release(ctx)
	require.NoError(t, err)
}

func TestMemory_Lock_CannotAcquireTwice(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock1 := store.Lock("test-lock", 1*time.Minute)
	lock2 := store.Lock("test-lock", 1*time.Minute)

	// First lock succeeds
	acquired1, err := lock1.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired1)

	// Second lock fails (already held)
	acquired2, err := lock2.Acquire(ctx)
	require.NoError(t, err)
	assert.False(t, acquired2)

	// Release first lock
	_ = lock1.Release(ctx)

	// Now second lock can acquire
	acquired3, err := lock2.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired3)
}

func TestMemory_Lock_Get(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock := store.Lock("test-lock", 1*time.Minute)

	executed := false
	err := lock.Get(ctx, func() error {
		executed = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestMemory_Lock_GetWithError(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock1 := store.Lock("test-lock", 1*time.Minute)
	lock2 := store.Lock("test-lock", 1*time.Minute)

	// First lock acquires
	_, _ = lock1.Acquire(ctx)
	defer lock1.Release(ctx)

	// Second lock fails
	executed := false
	err := lock2.Get(ctx, func() error {
		executed = true
		return nil
	})

	assert.ErrorIs(t, err, cache.ErrLockNotAcquired)
	assert.False(t, executed)
}

func TestMemory_Lock_Block(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock1 := store.Lock("test-lock", 1*time.Second)
	lock2 := store.Lock("test-lock", 1*time.Second)

	// First lock acquires
	_, _ = lock1.Acquire(ctx)

	// Start goroutine that releases lock after 200ms
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = lock1.Release(ctx)
	}()

	// Second lock blocks and waits
	executed := false
	err := lock2.Block(ctx, 2*time.Second, func() error {
		executed = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestMemory_Lock_BlockTimeout(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock1 := store.Lock("test-lock", 1*time.Minute)
	lock2 := store.Lock("test-lock", 1*time.Minute)

	// First lock acquires and never releases
	_, _ = lock1.Acquire(ctx)
	defer lock1.Release(ctx)

	// Second lock blocks but times out
	executed := false
	err := lock2.Block(ctx, 300*time.Millisecond, func() error {
		executed = true
		return nil
	})

	assert.ErrorIs(t, err, cache.ErrLockTimeout)
	assert.False(t, executed)
}

func TestMemory_Lock_ForceRelease(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	lock1 := store.Lock("test-lock", 1*time.Minute)
	lock2 := store.Lock("test-lock", 1*time.Minute)

	// First lock acquires
	_, _ = lock1.Acquire(ctx)

	// Force release from second lock
	err := lock2.ForceRelease(ctx)
	require.NoError(t, err)

	// Now second lock can acquire
	acquired, err := lock2.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestMemory_Statistics_HitsAndMisses(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Reset stats to start clean
	store.ResetStats()

	// Store a value
	_ = store.Put(ctx, "key1", "value1", 1*time.Minute)

	// Get (hit)
	var result string
	_ = store.Get(ctx, "key1", &result)

	// Get non-existent (miss)
	_ = store.Get(ctx, "key2", &result)

	// Get again (hit)
	_ = store.Get(ctx, "key1", &result)

	stats := store.GetStats()
	assert.Equal(t, int64(2), stats.Hits, "should have 2 hits")
	assert.Equal(t, int64(1), stats.Misses, "should have 1 miss")
	assert.Equal(t, int64(1), stats.Writes, "should have 1 write")
	assert.Equal(t, float64(2.0/3.0), stats.HitRatio())
}

func TestMemory_Statistics_Writes(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	store.ResetStats()

	_ = store.Put(ctx, "key1", "value1", 1*time.Minute)
	_ = store.Put(ctx, "key2", "value2", 1*time.Minute)
	_ = store.Forever(ctx, "key3", "value3")

	stats := store.GetStats()
	assert.Equal(t, int64(3), stats.Writes, "should have 3 writes")
}

func TestMemory_Statistics_Deletes(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	store.ResetStats()

	_ = store.Put(ctx, "key1", "value1", 1*time.Minute)
	_ = store.Put(ctx, "key2", "value2", 1*time.Minute)
	_ = store.Forget(ctx, "key1")

	stats := store.GetStats()
	assert.Equal(t, int64(1), stats.Deletes, "should have 1 delete")
}

func TestMemory_Statistics_Size(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	assert.Equal(t, int64(0), store.Size())

	_ = store.Put(ctx, "key1", "value1", 1*time.Minute)
	assert.Equal(t, int64(1), store.Size())

	_ = store.Put(ctx, "key2", "value2", 1*time.Minute)
	assert.Equal(t, int64(2), store.Size())

	_ = store.Forget(ctx, "key1")
	assert.Equal(t, int64(1), store.Size())

	_ = store.Flush(ctx)
	assert.Equal(t, int64(0), store.Size())
}

func TestMemory_Statistics_ResetStats(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	// Generate some activity
	_ = store.Put(ctx, "key1", "value1", 1*time.Minute)
	var result string
	_ = store.Get(ctx, "key1", &result)
	_ = store.Get(ctx, "key2", &result)

	// Verify stats exist
	stats := store.GetStats()
	assert.Greater(t, stats.Hits, int64(0))
	assert.Greater(t, stats.Misses, int64(0))

	// Reset stats
	store.ResetStats()

	// Verify stats are reset
	stats = store.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Writes)
	assert.Equal(t, int64(0), stats.Deletes)
}

func TestMemory_Statistics_Increment(t *testing.T) {
	ctx := context.Background()
	store := New()
	defer store.Stop()

	store.ResetStats()

	// Increment counts as a write
	_, _ = store.Increment(ctx, "counter", 1)
	_, _ = store.Increment(ctx, "counter", 5)

	stats := store.GetStats()
	assert.Equal(t, int64(2), stats.Writes, "increments should count as writes")
}
