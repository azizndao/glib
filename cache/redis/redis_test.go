package redis

import (
	"context"
	"testing"
	"time"

	
	"github.com/azizndao/glib/cache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRedisTest creates a test Redis client connected to localhost.
// Skip tests if Redis is not available.
func setupRedisTest(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use DB 15 for tests
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up before tests
	client.FlushDB(ctx)

	t.Cleanup(func() {
		client.FlushDB(ctx)
		client.Close()
	})

	return client
}

func TestRedis_PutAndGet(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	// Store a string
	err := store.Put(ctx, "name", "John Doe", 1*time.Minute)
	require.NoError(t, err)

	// Retrieve it
	var name string
	err = store.Get(ctx, "name", &name)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", name)
}

func TestRedis_GetNonExistent(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	var value string
	err := store.Get(ctx, "nonexistent", &value)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestRedis_PutComplex(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	type User struct {
		ID    int
		Name  string
		Email string
	}

	user := User{ID: 1, Name: "John", Email: "john@example.com"}
	err := store.Put(ctx, "user:1", user, 1*time.Minute)
	require.NoError(t, err)

	var retrieved User
	err = store.Get(ctx, "user:1", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, user, retrieved)
}

func TestRedis_Forever(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	err := store.Forever(ctx, "permanent", "forever")
	require.NoError(t, err)

	// Check TTL is -1 (no expiration)
	ttl := client.TTL(ctx, "test:permanent").Val()
	assert.Equal(t, time.Duration(-1), ttl)

	var value string
	err = store.Get(ctx, "permanent", &value)
	require.NoError(t, err)
	assert.Equal(t, "forever", value)
}

func TestRedis_HasAndMissing(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	store.Put(ctx, "exists", "value", 1*time.Minute)

	has, err := store.Has(ctx, "exists")
	require.NoError(t, err)
	assert.True(t, has)

	missing, err := store.Missing(ctx, "exists")
	require.NoError(t, err)
	assert.False(t, missing)

	has, err = store.Has(ctx, "nothere")
	require.NoError(t, err)
	assert.False(t, has)

	missing, err = store.Missing(ctx, "nothere")
	require.NoError(t, err)
	assert.True(t, missing)
}

func TestRedis_Increment(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	count, err := store.Increment(ctx, "counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = store.Increment(ctx, "counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
}

func TestRedis_Decrement(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	store.Increment(ctx, "counter", 10)

	count, err := store.Decrement(ctx, "counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(9), count)

	count, err = store.Decrement(ctx, "counter", 3)
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
}

func TestRedis_Forget(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	store.Put(ctx, "temp", "value", 1*time.Minute)

	has, _ := store.Has(ctx, "temp")
	assert.True(t, has)

	err := store.Forget(ctx, "temp")
	require.NoError(t, err)

	has, _ = store.Has(ctx, "temp")
	assert.False(t, has)
}

func TestRedis_Flush(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	store.Put(ctx, "key1", "value1", 1*time.Minute)
	store.Put(ctx, "key2", "value2", 1*time.Minute)
	store.Put(ctx, "key3", "value3", 1*time.Minute)

	err := store.Flush(ctx)
	require.NoError(t, err)

	has, _ := store.Has(ctx, "key1")
	assert.False(t, has)
	has, _ = store.Has(ctx, "key2")
	assert.False(t, has)
	has, _ = store.Has(ctx, "key3")
	assert.False(t, has)
}

func TestRedis_Remember(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	callCount := 0
	callback := func() (any, error) {
		callCount++
		return "computed", nil
	}

	var result string
	err := store.Remember(ctx, "computed", &result, 1*time.Minute, callback)
	require.NoError(t, err)
	assert.Equal(t, "computed", result)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	err = store.Remember(ctx, "computed", &result, 1*time.Minute, callback)
	require.NoError(t, err)
	assert.Equal(t, "computed", result)
	assert.Equal(t, 1, callCount, "callback should not be called again")
}

func TestRedis_RememberForever(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	callCount := 0
	callback := func() (any, error) {
		callCount++
		return "forever", nil
	}

	var result string
	err := store.RememberForever(ctx, "forever", &result, callback)
	require.NoError(t, err)
	assert.Equal(t, "forever", result)
	assert.Equal(t, 1, callCount)

	// Check no expiration
	ttl := client.TTL(ctx, "test:forever").Val()
	assert.Equal(t, time.Duration(-1), ttl)
}

func TestRedis_Pull(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	store.Put(ctx, "temp", "value", 1*time.Minute)

	var value string
	err := store.Pull(ctx, "temp", &value)
	require.NoError(t, err)
	assert.Equal(t, "value", value)

	// Should be deleted after pull
	has, _ := store.Has(ctx, "temp")
	assert.False(t, has)
}

func TestRedis_Add(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	// First add should succeed
	added, err := store.Add(ctx, "unique", "first", 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, added)

	// Second add should fail (key exists)
	added, err = store.Add(ctx, "unique", "second", 1*time.Minute)
	require.NoError(t, err)
	assert.False(t, added)

	// Value should still be "first"
	var value string
	store.Get(ctx, "unique", &value)
	assert.Equal(t, "first", value)
}

func TestRedis_Tags_Basic(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	tagged := store.Tags("users", "posts")

	err := tagged.Put(ctx, "item1", "value1", 1*time.Minute)
	require.NoError(t, err)

	var value string
	err = tagged.Get(ctx, "item1", &value)
	require.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestRedis_Tags_FlushTags(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	// Create items with different tags
	userCache := store.Tags("users")
	postCache := store.Tags("posts")
	bothCache := store.Tags("users", "posts")

	userCache.Put(ctx, "user1", "John", 1*time.Minute)
	postCache.Put(ctx, "post1", "Hello", 1*time.Minute)
	bothCache.Put(ctx, "both1", "Data", 1*time.Minute)

	// Flush users tag
	err := userCache.FlushTags(ctx)
	require.NoError(t, err)

	// User items should be gone
	var value string
	err = userCache.Get(ctx, "user1", &value)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	err = bothCache.Get(ctx, "both1", &value)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	// Post items should still exist
	err = postCache.Get(ctx, "post1", &value)
	require.NoError(t, err)
	assert.Equal(t, "Hello", value)
}

func TestRedis_Lock_BasicAcquireRelease(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	lock := store.Lock("resource", 10*time.Second)

	acquired, err := lock.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Try to acquire again (should fail)
	acquired, err = lock.Acquire(ctx)
	require.NoError(t, err)
	assert.False(t, acquired)

	// Release
	err = lock.Release(ctx)
	require.NoError(t, err)

	// Should be able to acquire again
	acquired, err = lock.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestRedis_Lock_Get(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	lock := store.Lock("resource", 10*time.Second)

	executed := false
	err := lock.Get(ctx, func() error {
		executed = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)

	// Lock should be released after callback
	acquired, _ := lock.Acquire(ctx)
	assert.True(t, acquired)
}

func TestRedis_Lock_Block(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	lock1 := store.Lock("resource", 1*time.Second)
	lock2 := store.Lock("resource", 1*time.Second)

	// Acquire first lock
	acquired, err := lock1.Acquire(ctx)
	require.NoError(t, err)
	require.True(t, acquired)

	// Release after 500ms
	go func() {
		time.Sleep(500 * time.Millisecond)
		lock1.Release(ctx)
	}()

	// Second lock should wait and then acquire
	executed := false
	err = lock2.Block(ctx, 2*time.Second, func() error {
		executed = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestRedis_Lock_BlockTimeout(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	lock1 := store.Lock("resource", 10*time.Second)
	lock2 := store.Lock("resource", 10*time.Second)

	// Acquire first lock
	lock1.Acquire(ctx)

	// Second lock should timeout
	err := lock2.Block(ctx, 300*time.Millisecond, func() error {
		return nil
	})

	assert.ErrorIs(t, err, cache.ErrLockTimeout)
}

func TestRedis_Lock_ForceRelease(t *testing.T) {
	client := setupRedisTest(t)
	store := New(client, "test")
	ctx := context.Background()

	lock1 := store.Lock("resource", 10*time.Second)
	lock2 := store.Lock("resource", 10*time.Second)

	lock1.Acquire(ctx)

	// Force release with different lock instance
	err := lock2.ForceRelease(ctx)
	require.NoError(t, err)

	// Should be able to acquire now
	acquired, err := lock2.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestRedis_PrefixIsolation(t *testing.T) {
	client := setupRedisTest(t)
	store1 := New(client, "app1")
	store2 := New(client, "app2")
	ctx := context.Background()

	store1.Put(ctx, "key", "value1", 1*time.Minute)
	store2.Put(ctx, "key", "value2", 1*time.Minute)

	var value1, value2 string
	store1.Get(ctx, "key", &value1)
	store2.Get(ctx, "key", &value2)

	assert.Equal(t, "value1", value1)
	assert.Equal(t, "value2", value2)
}
