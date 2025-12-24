package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/azizndao/glib/common/config"
)

func TestRepository_New(t *testing.T) {
	repo := config.New()
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestRepository_GetString(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"name": "glib",
	})

	value := repo.GetString("name")
	if value != "glib" {
		t.Errorf("expected 'glib', got '%s'", value)
	}

	// Test with default
	value = repo.GetString("missing", "default")
	if value != "default" {
		t.Errorf("expected 'default', got '%s'", value)
	}
}

func TestRepository_GetInt(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"port":      8080,
		"portStr":   "9000",
		"portFloat": 7000.0,
	})

	// Test int
	value := repo.GetInt("port")
	if value != 8080 {
		t.Errorf("expected 8080, got %d", value)
	}

	// Test string conversion
	value = repo.GetInt("portStr")
	if value != 9000 {
		t.Errorf("expected 9000, got %d", value)
	}

	// Test float conversion
	value = repo.GetInt("portFloat")
	if value != 7000 {
		t.Errorf("expected 7000, got %d", value)
	}

	// Test default
	value = repo.GetInt("missing", 3000)
	if value != 3000 {
		t.Errorf("expected 3000, got %d", value)
	}
}

func TestRepository_GetBool(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"debug":    true,
		"strTrue":  "true",
		"strYes":   "yes",
		"str1":     "1",
		"strOn":    "on",
		"intTrue":  1,
		"intFalse": 0,
	})

	tests := []struct {
		key      string
		expected bool
	}{
		{"debug", true},
		{"strTrue", true},
		{"strYes", true},
		{"str1", true},
		{"strOn", true},
		{"intTrue", true},
		{"intFalse", false},
		{"missing", false},
	}

	for _, tt := range tests {
		value := repo.GetBool(tt.key)
		if value != tt.expected {
			t.Errorf("key %s: expected %v, got %v", tt.key, tt.expected, value)
		}
	}
}

func TestRepository_GetDuration(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"timeout":    30 * time.Second,
		"timeoutStr": "1m30s",
	})

	// Test duration
	value := repo.GetDuration("timeout")
	if value != 30*time.Second {
		t.Errorf("expected 30s, got %v", value)
	}

	// Test string parsing
	value = repo.GetDuration("timeoutStr")
	if value != 90*time.Second {
		t.Errorf("expected 90s, got %v", value)
	}

	// Test default
	value = repo.GetDuration("missing", 5*time.Second)
	if value != 5*time.Second {
		t.Errorf("expected 5s, got %v", value)
	}
}

func TestRepository_GetStringSlice(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"hosts":    []string{"localhost", "example.com"},
		"hostsStr": "host1,host2,host3",
		"hostsAny": []any{"a", "b", "c"},
	})

	// Test []string
	value := repo.GetStringSlice("hosts")
	if len(value) != 2 || value[0] != "localhost" {
		t.Errorf("unexpected value: %v", value)
	}

	// Test comma-separated string
	value = repo.GetStringSlice("hostsStr")
	if len(value) != 3 || value[0] != "host1" {
		t.Errorf("unexpected value: %v", value)
	}

	// Test []any
	value = repo.GetStringSlice("hostsAny")
	if len(value) != 3 || value[0] != "a" {
		t.Errorf("unexpected value: %v", value)
	}
}

func TestRepository_DotNotation(t *testing.T) {
	repo := config.New()

	// Set nested values
	repo.Set("database.host", "localhost")
	repo.Set("database.port", 5432)
	repo.Set("database.connections.mysql.driver", "mysql")

	// Get nested values
	host := repo.GetString("database.host")
	if host != "localhost" {
		t.Errorf("expected 'localhost', got '%s'", host)
	}

	port := repo.GetInt("database.port")
	if port != 5432 {
		t.Errorf("expected 5432, got %d", port)
	}

	driver := repo.GetString("database.connections.mysql.driver")
	if driver != "mysql" {
		t.Errorf("expected 'mysql', got '%s'", driver)
	}
}

func TestRepository_Has(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"name": "glib",
	})

	if !repo.Has("name") {
		t.Error("expected Has('name') to return true")
	}

	if repo.Has("missing") {
		t.Error("expected Has('missing') to return false")
	}
}

func TestRepository_Set(t *testing.T) {
	repo := config.New()

	repo.Set("key", "value")

	if !repo.Has("key") {
		t.Error("expected key to exist")
	}

	value := repo.GetString("key")
	if value != "value" {
		t.Errorf("expected 'value', got '%s'", value)
	}
}

func TestRepository_All(t *testing.T) {
	initial := map[string]any{
		"name": "glib",
		"port": 8080,
	}

	repo := config.NewWithMap(initial)

	all := repo.All()

	if len(all) != 2 {
		t.Errorf("expected 2 items, got %d", len(all))
	}

	if all["name"] != "glib" {
		t.Errorf("expected name='glib', got %v", all["name"])
	}

	// Modify returned map shouldn't affect internal state
	all["name"] = "modified"

	value := repo.GetString("name")
	if value != "glib" {
		t.Error("external modification affected internal state")
	}
}

func TestRepository_Env(t *testing.T) {
	// Set environment variable
	os.Setenv("APP_ENV", "testing")
	defer os.Unsetenv("APP_ENV")

	repo := config.New()

	env := repo.Env()
	if env != "testing" {
		t.Errorf("expected 'testing', got '%s'", env)
	}
}

func TestRepository_IsDebug(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"app": map[string]any{
			"debug": true,
		},
	})

	if !repo.IsDebug() {
		t.Error("expected IsDebug() to return true")
	}

	repo.Set("app.debug", false)

	if repo.IsDebug() {
		t.Error("expected IsDebug() to return false")
	}
}

func TestRepository_LoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("APP_NAME", "TestApp")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
	}()

	repo := config.New()
	repo.LoadFromEnv("")

	// Check loaded values
	appName := repo.GetString("app.name")
	if appName != "TestApp" {
		t.Errorf("expected 'TestApp', got '%s'", appName)
	}

	dbHost := repo.GetString("database.host")
	if dbHost != "localhost" {
		t.Errorf("expected 'localhost', got '%s'", dbHost)
	}

	dbPort := repo.GetString("database.port")
	if dbPort != "5432" {
		t.Errorf("expected '5432', got '%s'", dbPort)
	}
}

func TestRepository_Merge(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"name": "glib",
		"port": 8080,
	})

	repo.Merge(map[string]any{
		"port":  9000, // Override
		"debug": true, // New
	})

	if repo.GetInt("port") != 9000 {
		t.Error("expected port to be overridden")
	}

	if !repo.GetBool("debug") {
		t.Error("expected debug to be added")
	}

	if repo.GetString("name") != "glib" {
		t.Error("expected name to remain unchanged")
	}
}

func TestRepository_ThreadSafety(t *testing.T) {
	repo := config.New()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 100; i++ {
		go func(n int) {
			repo.Set("key", n)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		go func() {
			_ = repo.GetInt("key")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 200; i++ {
		<-done
	}
}

func TestRepository_NestedMapCreation(t *testing.T) {
	repo := config.New()

	// Set deeply nested value
	repo.Set("a.b.c.d.e", "value")

	// Verify structure was created
	value := repo.GetString("a.b.c.d.e")
	if value != "value" {
		t.Errorf("expected 'value', got '%s'", value)
	}
}

func TestRepository_TypeConversions(t *testing.T) {
	repo := config.NewWithMap(map[string]any{
		"intValue":   42,
		"floatValue": 3.14,
		"boolValue":  true,
		"strValue":   "hello",
	})

	// Int to string
	strVal := repo.GetString("intValue")
	if strVal != "42" {
		t.Errorf("expected '42', got '%s'", strVal)
	}

	// Float to int
	intVal := repo.GetInt("floatValue")
	if intVal != 3 {
		t.Errorf("expected 3, got %d", intVal)
	}

	// Bool to string
	strBool := repo.GetString("boolValue")
	if strBool != "true" {
		t.Errorf("expected 'true', got '%s'", strBool)
	}
}

// Benchmarks
func BenchmarkRepository_GetString(b *testing.B) {
	repo := config.NewWithMap(map[string]any{
		"name": "glib",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.GetString("name")
	}
}

func BenchmarkRepository_GetString_Nested(b *testing.B) {
	repo := config.New()
	repo.Set("database.connections.mysql.host", "localhost")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.GetString("database.connections.mysql.host")
	}
}

func BenchmarkRepository_Set(b *testing.B) {
	repo := config.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.Set("key", "value")
	}
}
