package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/memory"
	"github.com/azizndao/glib/common/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceProvider_Register(t *testing.T) {
	c := container.New()
	provider := cache.NewServiceProvider()

	// Register the provider
	err := provider.Register(c)
	require.NoError(t, err)

	// Resolve the manager
	manager, err := container.Resolve[*cache.Manager](c)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestServiceProvider_Boot(t *testing.T) {
	c := container.New()
	provider := cache.NewServiceProvider()

	// Register and boot
	err := provider.Register(c)
	require.NoError(t, err)

	err = provider.Boot(c)
	require.NoError(t, err)

	// Resolve the manager
	manager, err := container.Resolve[*cache.Manager](c)
	require.NoError(t, err)

	// Manually register memory driver (Boot no longer does this)
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})
	manager.SetDefaultStore("memory")

	// Verify default store is configured
	store, err := manager.Default()
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Test that the store works
	ctx := context.Background()
	err = store.Put(ctx, "test-key", "test-value", 1*time.Minute)
	require.NoError(t, err)

	var value string
	err = store.Get(ctx, "test-key", &value)
	require.NoError(t, err)
	assert.Equal(t, "test-value", value)
}

func TestServiceProvider_Singleton(t *testing.T) {
	c := container.New()
	provider := cache.NewServiceProvider()

	// Register
	err := provider.Register(c)
	require.NoError(t, err)

	// Resolve twice
	manager1, err := container.Resolve[*cache.Manager](c)
	require.NoError(t, err)

	manager2, err := container.Resolve[*cache.Manager](c)
	require.NoError(t, err)

	// Should be the same instance
	assert.Same(t, manager1, manager2)
}

func TestServiceProvider_Provides(t *testing.T) {
	provider := cache.NewServiceProvider()

	provides := provider.Provides()
	assert.Contains(t, provides, "*cache.Manager")
}
