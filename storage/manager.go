package storage

import (
	"fmt"
	"sync"
)

// Manager manages multiple storage disks with different drivers.
type Manager struct {
	disks       map[string]Storage
	drivers     map[string]Driver
	defaultDisk string
	mu          sync.RWMutex
}

// NewManager creates a new storage manager.
func NewManager() *Manager {
	return &Manager{
		disks:       make(map[string]Storage),
		drivers:     make(map[string]Driver),
		defaultDisk: "local",
	}
}

// RegisterDisk registers a storage disk factory.
func (m *Manager) RegisterDisk(name string, driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[name] = driver
}

// Disk returns a storage disk by name. If the disk doesn't exist,
// it will be created using the registered driver.
func (m *Manager) Disk(name string) (Storage, error) {
	if name == "" {
		name = m.defaultDisk
	}

	// Try to get existing disk
	m.mu.RLock()
	disk, exists := m.disks[name]
	m.mu.RUnlock()

	if exists {
		return disk, nil
	}

	// Create new disk
	return m.resolve(name)
}

// resolve creates a new storage disk.
func (m *Manager) resolve(name string) (Storage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if disk, exists := m.disks[name]; exists {
		return disk, nil
	}

	// Get driver
	driver, exists := m.drivers[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrDiskNotFound, name)
	}

	// Create disk
	disk := driver()
	m.disks[name] = disk

	return disk, nil
}

// Default returns the default storage disk.
func (m *Manager) Default() (Storage, error) {
	return m.Disk("")
}

// SetDefaultDisk sets the default disk name.
func (m *Manager) SetDefaultDisk(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDisk = name
}

// Extend registers a custom disk instance directly.
func (m *Manager) Extend(name string, disk Storage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disks[name] = disk
}
