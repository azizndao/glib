package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_RegisterDriver(t *testing.T) {
	manager := cache.NewManager()

	// Register memory driver
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	// Get store
	store, err := manager.Store("memory")
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestManager_DefaultStore(t *testing.T) {
	manager := cache.NewManager()

	// Register memory driver
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	// Get default store
	store, err := manager.Default()
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestManager_SetDefaultStore(t *testing.T) {
	manager := cache.NewManager()

	// Register drivers
	manager.RegisterDriver("memory1", func() cache.Store {
		return memory.New()
	})
	manager.RegisterDriver("memory2", func() cache.Store {
		return memory.New()
	})

	// Set default
	manager.SetDefaultStore("memory2")

	// Default should be memory2
	store, err := manager.Default()
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestManager_UnregisteredDriver(t *testing.T) {
	manager := cache.NewManager()

	// Try to get unregistered driver
	_, err := manager.Store("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestManager_StoreReuse(t *testing.T) {
	manager := cache.NewManager()

	callCount := 0
	manager.RegisterDriver("memory", func() cache.Store {
		callCount++
		return memory.New()
	})

	// Get store twice
	store1, err := manager.Store("memory")
	require.NoError(t, err)
	store2, err := manager.Store("memory")
	require.NoError(t, err)

	// Should be same instance
	assert.Equal(t, store1, store2)
	// Driver should only be called once
	assert.Equal(t, 1, callCount)
}

func TestManager_Extend(t *testing.T) {
	manager := cache.NewManager()

	// Create custom store
	customStore := memory.New()
	manager.Extend("custom", customStore)

	// Get it
	store, err := manager.Store("custom")
	require.NoError(t, err)
	assert.Equal(t, customStore, store)
}

func TestManager_MultipleStores(t *testing.T) {
	ctx := context.Background()
	manager := cache.NewManager()

	// Register multiple drivers
	manager.RegisterDriver("store1", func() cache.Store {
		return memory.New()
	})
	manager.RegisterDriver("store2", func() cache.Store {
		return memory.New()
	})

	// Get both stores
	store1, err := manager.Store("store1")
	require.NoError(t, err)
	store2, err := manager.Store("store2")
	require.NoError(t, err)

	// Put value in store1
	err = store1.Put(ctx, "key", "value1", 1*time.Minute)
	require.NoError(t, err)

	// Stores are independent
	var result string
	err = store2.Get(ctx, "key", &result)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	// store1 has the value
	err = store1.Get(ctx, "key", &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)
}
