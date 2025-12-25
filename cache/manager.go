package cache

import (
	"fmt"
	"sync"
)

// Manager manages multiple cache stores with different drivers.
type Manager struct {
	stores       map[string]Store
	drivers      map[string]Driver
	defaultStore string
	mu           sync.RWMutex
}

// NewManager creates a new cache manager.
func NewManager() *Manager {
	return &Manager{
		stores:       make(map[string]Store),
		drivers:      make(map[string]Driver),
		defaultStore: "memory",
	}
}

// RegisterDriver registers a cache driver factory.
func (m *Manager) RegisterDriver(name string, driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[name] = driver
}

// Store returns a cache store by name. If the store doesn't exist,
// it will be created using the registered driver.
func (m *Manager) Store(name string) (Store, error) {
	if name == "" {
		name = m.defaultStore
	}

	// Try to get existing store
	m.mu.RLock()
	store, exists := m.stores[name]
	m.mu.RUnlock()

	if exists {
		return store, nil
	}

	// Create new store
	return m.resolve(name)
}

// resolve creates a new cache store.
func (m *Manager) resolve(name string) (Store, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if store, exists := m.stores[name]; exists {
		return store, nil
	}

	// Get driver
	driver, exists := m.drivers[name]
	if !exists {
		return nil, fmt.Errorf("cache driver not registered: %s", name)
	}

	// Create store
	store := driver()
	m.stores[name] = store

	return store, nil
}

// Default returns the default cache store.
func (m *Manager) Default() (Store, error) {
	return m.Store("")
}

// SetDefaultStore sets the default store name.
func (m *Manager) SetDefaultStore(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultStore = name
}

// Extend registers a custom store instance directly.
func (m *Manager) Extend(name string, store Store) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stores[name] = store
}
