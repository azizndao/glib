package database

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/azizndao/glib/config"
)

// Manager manages multiple database connections.
type Manager struct {
	connections map[string]*Connection
	config      config.Config
	logger      *slog.Logger
	defaultConn string
	mu          sync.RWMutex
}

// NewManager creates a new database manager.
func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		connections: make(map[string]*Connection),
		config:      cfg,
		logger:      logger,
		defaultConn: cfg.GetString("database.default", "mysql"),
	}
}

// Connection returns a named database connection.
// If the connection doesn't exist, it will be created.
func (m *Manager) Connection(name string) (*Connection, error) {
	m.mu.RLock()
	conn, exists := m.connections[name]
	m.mu.RUnlock()

	if exists {
		return conn, nil
	}

	// Create new connection
	return m.connect(name)
}

// DB returns the default database connection.
func (m *Manager) DB() (*Connection, error) {
	return m.Connection(m.defaultConn)
}

// AddConnection adds a new connection with the given configuration.
func (m *Manager) AddConnection(name string, cfg ConnectionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if connection already exists
	if _, exists := m.connections[name]; exists {
		return fmt.Errorf("connection %s already exists", name)
	}

	// Create GORM connection
	gormDB, err := m.openConnection(cfg)
	if err != nil {
		return fmt.Errorf("failed to open connection %s: %w", name, err)
	}

	// Configure connection pool
	if err := m.configurePool(gormDB, cfg.Pool); err != nil {
		return fmt.Errorf("failed to configure pool for %s: %w", name, err)
	}

	// Create connection wrapper
	conn := NewConnection(name, cfg, gormDB)
	m.connections[name] = conn

	m.logger.Info("Database connection established",
		"name", name,
		"driver", cfg.Driver,
		"host", cfg.Host,
		"database", cfg.Database,
	)

	return nil
}

// connect creates a new connection based on configuration.
func (m *Manager) connect(name string) (*Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check it wasn't created while waiting for lock
	if conn, exists := m.connections[name]; exists {
		return conn, nil
	}

	// Load connection config from configuration
	cfg, err := m.loadConnectionConfig(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load config for %s: %w", name, err)
	}

	// Create GORM connection
	gormDB, err := m.openConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection %s: %w", name, err)
	}

	// Configure connection pool
	if err := m.configurePool(gormDB, cfg.Pool); err != nil {
		return nil, fmt.Errorf("failed to configure pool for %s: %w", name, err)
	}

	// Create connection wrapper
	conn := NewConnection(name, cfg, gormDB)
	m.connections[name] = conn

	m.logger.Info("Database connection established",
		"name", name,
		"driver", cfg.Driver,
		"host", cfg.Host,
		"database", cfg.Database,
	)

	return conn, nil
}

// openConnection opens a GORM database connection based on driver.
func (m *Manager) openConnection(cfg ConnectionConfig) (*gorm.DB, error) {
	// Create custom logger
	gormLogger := NewGormLogger(m.logger, 200*time.Millisecond)

	// GORM config
	gormConfig := &gorm.Config{
		Logger: gormLogger,
	}

	// Open connection based on driver
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.DSN())
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.DSN())
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN())
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}

	// Open connection
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// configurePool configures the connection pool.
func (m *Manager) configurePool(db *gorm.DB, pool PoolConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Set defaults if not specified
	if pool.MaxOpen == 0 {
		pool.MaxOpen = 100
	}
	if pool.MaxIdle == 0 {
		pool.MaxIdle = 10
	}
	if pool.MaxLifetime == 0 {
		pool.MaxLifetime = time.Hour
	}

	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	sqlDB.SetConnMaxLifetime(pool.MaxLifetime)

	return nil
}

// loadConnectionConfig loads connection configuration from config.
func (m *Manager) loadConnectionConfig(name string) (ConnectionConfig, error) {
	prefix := fmt.Sprintf("database.connections.%s", name)

	cfg := ConnectionConfig{
		Driver:    m.config.GetString(prefix+".driver", "mysql"),
		Host:      m.config.GetString(prefix+".host", "localhost"),
		Port:      m.config.GetInt(prefix+".port", 3306),
		Database:  m.config.GetString(prefix+".database", ""),
		Username:  m.config.GetString(prefix+".username", "root"),
		Password:  m.config.GetString(prefix+".password", ""),
		Charset:   m.config.GetString(prefix+".charset", "utf8mb4"),
		Collation: m.config.GetString(prefix+".collation", "utf8mb4_unicode_ci"),
		Prefix:    m.config.GetString(prefix+".prefix", ""),
		Timezone:  m.config.GetString(prefix+".timezone", "UTC"),
		SSLMode:   m.config.GetString(prefix+".sslmode", "disable"),
		Pool: PoolConfig{
			MaxOpen:     m.config.GetInt(prefix+".pool.max_open", 100),
			MaxIdle:     m.config.GetInt(prefix+".pool.max_idle", 10),
			MaxLifetime: m.config.GetDuration(prefix+".pool.max_lifetime", time.Hour),
		},
	}

	if cfg.Database == "" {
		return cfg, fmt.Errorf("database name is required for connection %s", name)
	}

	return cfg, nil
}

// Close closes all database connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, conn := range m.connections {
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection %s: %w", name, err))
		} else {
			m.logger.Info("Database connection closed", "name", name)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// Ping checks all connections.
func (m *Manager) Ping() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, conn := range m.connections {
		if err := conn.Ping(); err != nil {
			return fmt.Errorf("connection %s failed ping: %w", name, err)
		}
	}

	return nil
}

// Connections returns all connection names.
func (m *Manager) Connections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}
	return names
}

// HasConnection checks if a connection exists.
func (m *Manager) HasConnection(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connections[name]
	return exists
}
