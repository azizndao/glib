package database

import (
	"context"
	"testing"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupDatabaseTest creates a test database connection.
func setupDatabaseTest(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate tables
	err = db.AutoMigrate(&CacheEntry{}, &CacheLock{})
	require.NoError(t, err)

	cleanup := func() {
		// Drop tables
		db.Migrator().DropTable(&CacheEntry{}, &CacheLock{})
	}

	return db, cleanup
}

func TestDatabase_PutAndGet(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_GetNonExistent(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	var value string
	err := store.Get(ctx, "nonexistent", &value)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestDatabase_PutComplex(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Forever(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	err := store.Forever(ctx, "permanent", "forever")
	require.NoError(t, err)

	var value string
	err = store.Get(ctx, "permanent", &value)
	require.NoError(t, err)
	assert.Equal(t, "forever", value)
}

func TestDatabase_Expiration(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	// Store with very short TTL
	err := store.Put(ctx, "temp", "value", 100*time.Millisecond)
	require.NoError(t, err)

	// Should exist initially
	has, err := store.Has(ctx, "temp")
	require.NoError(t, err)
	assert.True(t, has)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	var value string
	err = store.Get(ctx, "temp", &value)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)
}

func TestDatabase_HasAndMissing(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Increment(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	count, err := store.Increment(ctx, "counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = store.Increment(ctx, "counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
}

func TestDatabase_Decrement(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	store.Increment(ctx, "counter", 10)

	count, err := store.Decrement(ctx, "counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(9), count)

	count, err = store.Decrement(ctx, "counter", 3)
	require.NoError(t, err)
	assert.Equal(t, int64(6), count)
}

func TestDatabase_Forget(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	store.Put(ctx, "temp", "value", 1*time.Minute)

	has, _ := store.Has(ctx, "temp")
	assert.True(t, has)

	err := store.Forget(ctx, "temp")
	require.NoError(t, err)

	has, _ = store.Has(ctx, "temp")
	assert.False(t, has)
}

func TestDatabase_Flush(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Remember(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_RememberForever(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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
}

func TestDatabase_Pull(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Add(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_CleanupExpired(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	// Add some expired and non-expired items
	store.Put(ctx, "expired1", "value", 1*time.Millisecond)
	store.Put(ctx, "expired2", "value", 1*time.Millisecond)
	store.Put(ctx, "active", "value", 1*time.Hour)

	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	err := store.CleanupExpired(ctx)
	require.NoError(t, err)

	// Verify expired items are removed
	var count int64
	db.Model(&CacheEntry{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// Active item should still exist
	has, _ := store.Has(ctx, "active")
	assert.True(t, has)
}

func TestDatabase_Lock_BasicAcquireRelease(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Lock_Get(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Lock_Block(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Lock_BlockTimeout(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Lock_ForceRelease(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
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

func TestDatabase_Lock_ExpiredLockReacquisition(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	// Create lock with short TTL
	lock1 := store.Lock("resource", 100*time.Millisecond)
	lock2 := store.Lock("resource", 10*time.Second)

	// Acquire first lock
	acquired, err := lock1.Acquire(ctx)
	require.NoError(t, err)
	require.True(t, acquired)

	// Wait for lock to expire
	time.Sleep(200 * time.Millisecond)

	// Second lock should be able to acquire expired lock
	acquired, err = lock2.Acquire(ctx)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestDatabase_PrefixIsolation(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store1 := New(db, "app1")
	store2 := New(db, "app2")
	ctx := context.Background()

	store1.Put(ctx, "key", "value1", 1*time.Minute)
	store2.Put(ctx, "key", "value2", 1*time.Minute)

	var value1, value2 string
	store1.Get(ctx, "key", &value1)
	store2.Get(ctx, "key", &value2)

	assert.Equal(t, "value1", value1)
	assert.Equal(t, "value2", value2)
}

func TestDatabase_UpdateExisting(t *testing.T) {
	db, cleanup := setupDatabaseTest(t)
	defer cleanup()

	store := New(db, "test")
	ctx := context.Background()

	// Put initial value
	err := store.Put(ctx, "key", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Update with new value
	err = store.Put(ctx, "key", "value2", 1*time.Minute)
	require.NoError(t, err)

	// Should have the new value
	var value string
	err = store.Get(ctx, "key", &value)
	require.NoError(t, err)
	assert.Equal(t, "value2", value)

	// Should only have one entry
	var count int64
	db.Model(&CacheEntry{}).Where("key LIKE ?", "test:%").Count(&count)
	assert.Equal(t, int64(1), count)
}
