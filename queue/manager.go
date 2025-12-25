package queue

import (
	"fmt"
	"sync"
)

// Manager manages multiple queue connections with different drivers.
type Manager struct {
	connections map[string]Queue
	drivers     map[string]Driver
	configs     map[string]Config
	defaultConn string
	registry    *JobRegistry
	mu          sync.RWMutex
}

// NewManager creates a new queue manager.
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]Queue),
		drivers:     make(map[string]Driver),
		configs:     make(map[string]Config),
		defaultConn: "default",
		registry:    globalRegistry,
	}
}

// RegisterDriver registers a queue driver.
func (m *Manager) RegisterDriver(name string, driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[name] = driver
}

// RegisterConfig registers a queue connection configuration.
func (m *Manager) RegisterConfig(name string, config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[name] = config
}

// Connection returns a queue connection by name.
// If the connection doesn't exist, it will be created using the registered driver.
func (m *Manager) Connection(name string) (Queue, error) {
	if name == "" {
		name = m.defaultConn
	}

	// Try to get existing connection
	m.mu.RLock()
	conn, exists := m.connections[name]
	m.mu.RUnlock()

	if exists {
		return conn, nil
	}

	// Create new connection
	return m.resolve(name)
}

// resolve creates a new queue connection.
func (m *Manager) resolve(name string) (Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if conn, exists := m.connections[name]; exists {
		return conn, nil
	}

	// Get config
	config, exists := m.configs[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrQueueNotFound, name)
	}

	// Get driver
	driver, exists := m.drivers[config.Driver]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrDriverNotRegistered, config.Driver)
	}

	// Create connection
	conn, err := driver(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue connection %s: %w", name, err)
	}

	m.connections[name] = conn

	return conn, nil
}

// Default returns the default queue connection.
func (m *Manager) Default() (Queue, error) {
	return m.Connection("")
}

// SetDefaultConnection sets the default connection name.
func (m *Manager) SetDefaultConnection(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultConn = name
}

// Close closes all queue connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, conn := range m.connections {
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// RegisterJob registers a job type with the global registry.
func (m *Manager) RegisterJob(job Job) {
	m.registry.Register(job)
}

// GetRegistry returns the job registry.
func (m *Manager) GetRegistry() *JobRegistry {
	return m.registry
}

// Push pushes a job to the default queue connection.
func (m *Manager) Push(job Job, opts *Options) (string, error) {
	queue, err := m.Default()
	if err != nil {
		return "", err
	}
	return queue.Push(nil, job, opts)
}

// Extend adds queue manager methods to the Manager
// This allows for a more Laravel-like API

// OnConnection sets the connection for the next operation
type ConnectionScope struct {
	manager *Manager
	conn    string
}

// OnConnection returns a connection-scoped manager.
func (m *Manager) OnConnection(name string) *ConnectionScope {
	return &ConnectionScope{
		manager: m,
		conn:    name,
	}
}

// Push pushes a job to the specified connection.
func (cs *ConnectionScope) Push(job Job, opts *Options) (string, error) {
	queue, err := cs.manager.Connection(cs.conn)
	if err != nil {
		return "", err
	}

	if opts == nil {
		opts = DefaultOptions()
	}
	opts.Connection = cs.conn

	return queue.Push(nil, job, opts)
}
