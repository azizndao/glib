// Package database provides database connection management and GORM integration.
package database

import (
	"fmt"
	"time"
)

// ConnectionConfig defines database connection settings.
type ConnectionConfig struct {
	Driver    string     `json:"driver"`    // mysql, postgres, sqlite
	Host      string     `json:"host"`      // Database host
	Port      int        `json:"port"`      // Database port
	Database  string     `json:"database"`  // Database name
	Username  string     `json:"username"`  // Database username
	Password  string     `json:"password"`  // Database password
	Charset   string     `json:"charset"`   // Character set (MySQL)
	Collation string     `json:"collation"` // Collation (MySQL)
	Prefix    string     `json:"prefix"`    // Table prefix
	Timezone  string     `json:"timezone"`  // Timezone
	SSLMode   string     `json:"sslmode"`   // SSL mode (PostgreSQL)
	Pool      PoolConfig `json:"pool"`      // Connection pool settings
}

// PoolConfig defines connection pool settings.
type PoolConfig struct {
	MaxOpen     int           `json:"max_open"`     // Maximum number of open connections
	MaxIdle     int           `json:"max_idle"`     // Maximum number of idle connections
	MaxLifetime time.Duration `json:"max_lifetime"` // Maximum lifetime of a connection
}

// DSN returns the Data Source Name for the connection.
func (c *ConnectionConfig) DSN() string {
	switch c.Driver {
	case "mysql":
		return c.mysqlDSN()
	case "postgres", "postgresql":
		return c.postgresDSN()
	case "sqlite":
		return c.sqliteDSN()
	default:
		return ""
	}
}

// mysqlDSN builds a MySQL DSN string.
func (c *ConnectionConfig) mysqlDSN() string {
	// Format: username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
	dsn := c.Username
	if c.Password != "" {
		dsn += ":" + c.Password
	}
	dsn += "@tcp(" + c.Host
	if c.Port > 0 {
		dsn += fmt.Sprintf(":%d", c.Port)
	}
	dsn += ")/" + c.Database

	// Add parameters
	params := "?"
	if c.Charset != "" {
		params += "charset=" + c.Charset + "&"
	}
	params += "parseTime=True&"
	if c.Timezone != "" {
		params += "loc=" + c.Timezone + "&"
	}

	// Remove trailing & or ?
	if len(params) > 1 {
		dsn += params[:len(params)-1]
	}

	return dsn
}

// postgresDSN builds a PostgreSQL DSN string.
func (c *ConnectionConfig) postgresDSN() string {
	// Format: host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai
	dsn := "host=" + c.Host
	if c.Port > 0 {
		dsn += fmt.Sprintf(" port=%d", c.Port)
	}
	dsn += " user=" + c.Username
	if c.Password != "" {
		dsn += " password=" + c.Password
	}
	dsn += " dbname=" + c.Database
	if c.SSLMode != "" {
		dsn += " sslmode=" + c.SSLMode
	}
	if c.Timezone != "" {
		dsn += " TimeZone=" + c.Timezone
	}
	return dsn
}

// sqliteDSN builds a SQLite DSN string.
func (c *ConnectionConfig) sqliteDSN() string {
	// Format: /path/to/database.db
	return c.Database
}
