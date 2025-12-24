package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Connection wraps a GORM database connection.
type Connection struct {
	db     *gorm.DB
	name   string
	driver string
	config ConnectionConfig
}

// NewConnection creates a new database connection.
func NewConnection(name string, config ConnectionConfig, gormDB *gorm.DB) *Connection {
	return &Connection{
		db:     gormDB,
		name:   name,
		driver: config.Driver,
		config: config,
	}
}

// DB returns the underlying GORM instance.
func (c *Connection) DB() *gorm.DB {
	return c.db
}

// Name returns the connection name.
func (c *Connection) Name() string {
	return c.name
}

// Driver returns the driver name.
func (c *Connection) Driver() string {
	return c.driver
}

// Close closes the database connection.
func (c *Connection) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// Ping checks if the database connection is alive.
func (c *Connection) Ping() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.Ping()
}

// Stats returns database statistics.
func (c *Connection) Stats() sql.DBStats {
	sqlDB, _ := c.db.DB()
	if sqlDB == nil {
		return sql.DBStats{}
	}
	return sqlDB.Stats()
}

// Table specifies the table to query.
func (c *Connection) Table(name string) *gorm.DB {
	return c.db.Table(name)
}

// Model specifies the model to query.
func (c *Connection) Model(value any) *gorm.DB {
	return c.db.Model(value)
}

// Select specifies fields to select.
func (c *Connection) Select(query any, args ...any) *gorm.DB {
	return c.db.Select(query, args...)
}

// Where adds a WHERE clause.
func (c *Connection) Where(query any, args ...any) *gorm.DB {
	return c.db.Where(query, args...)
}

// Or adds an OR clause.
func (c *Connection) Or(query any, args ...any) *gorm.DB {
	return c.db.Or(query, args...)
}

// Not adds a NOT clause.
func (c *Connection) Not(query any, args ...any) *gorm.DB {
	return c.db.Not(query, args...)
}

// Order specifies the order.
func (c *Connection) Order(value any) *gorm.DB {
	return c.db.Order(value)
}

// Limit specifies the number of records to return.
func (c *Connection) Limit(limit int) *gorm.DB {
	return c.db.Limit(limit)
}

// Offset specifies the number of records to skip.
func (c *Connection) Offset(offset int) *gorm.DB {
	return c.db.Offset(offset)
}

// Group specifies the GROUP BY clause.
func (c *Connection) Group(name string) *gorm.DB {
	return c.db.Group(name)
}

// Having adds a HAVING clause.
func (c *Connection) Having(query any, args ...any) *gorm.DB {
	return c.db.Having(query, args...)
}

// Join specifies a JOIN clause.
func (c *Connection) Joins(query string, args ...any) *gorm.DB {
	return c.db.Joins(query, args...)
}

// Preload preloads associations.
func (c *Connection) Preload(query string, args ...any) *gorm.DB {
	return c.db.Preload(query, args...)
}

// Create inserts a new record.
func (c *Connection) Create(value any) *gorm.DB {
	return c.db.Create(value)
}

// Save updates a record or creates it if it doesn't exist.
func (c *Connection) Save(value any) *gorm.DB {
	return c.db.Save(value)
}

// Update updates a single column.
func (c *Connection) Update(column string, value any) *gorm.DB {
	return c.db.Update(column, value)
}

// Updates updates multiple columns.
func (c *Connection) Updates(values any) *gorm.DB {
	return c.db.Updates(values)
}

// Delete deletes records.
func (c *Connection) Delete(value any, conds ...any) *gorm.DB {
	return c.db.Delete(value, conds...)
}

// Find finds records.
func (c *Connection) Find(dest any, conds ...any) *gorm.DB {
	return c.db.Find(dest, conds...)
}

// First finds the first record ordered by primary key.
func (c *Connection) First(dest any, conds ...any) *gorm.DB {
	return c.db.First(dest, conds...)
}

// Last finds the last record ordered by primary key.
func (c *Connection) Last(dest any, conds ...any) *gorm.DB {
	return c.db.Last(dest, conds...)
}

// Take finds the first record without ordering.
func (c *Connection) Take(dest any, conds ...any) *gorm.DB {
	return c.db.Take(dest, conds...)
}

// Count returns the number of records.
func (c *Connection) Count(count *int64) *gorm.DB {
	return c.db.Count(count)
}

// Pluck queries a single column and scans into a slice.
func (c *Connection) Pluck(column string, dest any) *gorm.DB {
	return c.db.Pluck(column, dest)
}

// Scan scans result into a struct.
func (c *Connection) Scan(dest any) *gorm.DB {
	return c.db.Scan(dest)
}

// Raw executes raw SQL.
func (c *Connection) Raw(sql string, values ...any) *gorm.DB {
	return c.db.Raw(sql, values...)
}

// Exec executes raw SQL that doesn't return rows.
func (c *Connection) Exec(sql string, values ...any) *gorm.DB {
	return c.db.Exec(sql, values...)
}

// Transaction executes a function within a transaction.
func (c *Connection) Transaction(fc func(tx *Connection) error, opts ...*sql.TxOptions) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		txConn := &Connection{
			db:     tx,
			name:   c.name,
			driver: c.driver,
			config: c.config,
		}
		return fc(txConn)
	}, opts...)
}

// Begin starts a transaction.
func (c *Connection) Begin(opts ...*sql.TxOptions) *Connection {
	return &Connection{
		db:     c.db.Begin(opts...),
		name:   c.name,
		driver: c.driver,
		config: c.config,
	}
}

// Commit commits a transaction.
func (c *Connection) Commit() *gorm.DB {
	return c.db.Commit()
}

// Rollback rolls back a transaction.
func (c *Connection) Rollback() *gorm.DB {
	return c.db.Rollback()
}

// SavePoint creates a savepoint.
func (c *Connection) SavePoint(name string) *gorm.DB {
	return c.db.SavePoint(name)
}

// RollbackTo rolls back to a savepoint.
func (c *Connection) RollbackTo(name string) *gorm.DB {
	return c.db.RollbackTo(name)
}

// WithContext returns a new connection with the given context.
func (c *Connection) WithContext(ctx context.Context) *Connection {
	return &Connection{
		db:     c.db.WithContext(ctx),
		name:   c.name,
		driver: c.driver,
		config: c.config,
	}
}

// AutoMigrate runs auto migration for the given models.
func (c *Connection) AutoMigrate(dst ...any) error {
	return c.db.AutoMigrate(dst...)
}

// Migrator returns the migrator interface.
func (c *Connection) Migrator() gorm.Migrator {
	return c.db.Migrator()
}

// SetMaxIdleConns sets the maximum number of idle connections.
func (c *Connection) SetMaxIdleConns(n int) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(n)
	return nil
}

// SetMaxOpenConns sets the maximum number of open connections.
func (c *Connection) SetMaxOpenConns(n int) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(n)
	return nil
}

// SetConnMaxLifetime sets the maximum lifetime of a connection.
func (c *Connection) SetConnMaxLifetime(d time.Duration) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetConnMaxLifetime(d)
	return nil
}
