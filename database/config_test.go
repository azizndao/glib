package database_test

import (
	"testing"

	"github.com/azizndao/glib/database"
)

func TestConnectionConfig_DSN_MySQL(t *testing.T) {
	cfg := database.ConnectionConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "secret",
		Charset:  "utf8mb4",
		Timezone: "UTC",
	}

	dsn := cfg.DSN()
	expected := "root:secret@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=UTC"

	if dsn != expected {
		t.Errorf("MySQL DSN mismatch.\nExpected: %s\nGot: %s", expected, dsn)
	}
}

func TestConnectionConfig_DSN_PostgreSQL(t *testing.T) {
	cfg := database.ConnectionConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "postgres",
		Password: "secret",
		SSLMode:  "disable",
		Timezone: "UTC",
	}

	dsn := cfg.DSN()
	expected := "host=localhost port=5432 user=postgres password=secret dbname=testdb sslmode=disable TimeZone=UTC"

	if dsn != expected {
		t.Errorf("PostgreSQL DSN mismatch.\nExpected: %s\nGot: %s", expected, dsn)
	}
}

func TestConnectionConfig_DSN_SQLite(t *testing.T) {
	cfg := database.ConnectionConfig{
		Driver:   "sqlite",
		Database: "/tmp/test.db",
	}

	dsn := cfg.DSN()
	expected := "/tmp/test.db"

	if dsn != expected {
		t.Errorf("SQLite DSN mismatch.\nExpected: %s\nGot: %s", expected, dsn)
	}
}

func TestConnectionConfig_DSN_UnsupportedDriver(t *testing.T) {
	cfg := database.ConnectionConfig{
		Driver:   "unsupported",
		Database: "testdb",
	}

	dsn := cfg.DSN()
	if dsn != "" {
		t.Errorf("Expected empty DSN for unsupported driver, got: %s", dsn)
	}
}

func TestPoolConfig_Defaults(t *testing.T) {
	pool := database.PoolConfig{}

	if pool.MaxOpen != 0 {
		t.Errorf("Expected MaxOpen to be 0 by default, got %d", pool.MaxOpen)
	}

	if pool.MaxIdle != 0 {
		t.Errorf("Expected MaxIdle to be 0 by default, got %d", pool.MaxIdle)
	}

	if pool.MaxLifetime != 0 {
		t.Errorf("Expected MaxLifetime to be 0 by default, got %v", pool.MaxLifetime)
	}
}
