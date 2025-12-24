// Package config provides configuration management with environment variable
// support, dot notation access, and configuration caching.
package config

import (
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config is the interface for configuration access.
type Config interface {
	// Get retrieves a configuration value by key (supports dot notation).
	Get(key string, defaultValue ...any) any

	// GetString retrieves a string configuration value.
	GetString(key string, defaultValue ...string) string

	// GetInt retrieves an integer configuration value.
	GetInt(key string, defaultValue ...int) int

	// GetInt64 retrieves an int64 configuration value.
	GetInt64(key string, defaultValue ...int64) int64

	// GetFloat retrieves a float64 configuration value.
	GetFloat(key string, defaultValue ...float64) float64

	// GetBool retrieves a boolean configuration value.
	GetBool(key string, defaultValue ...bool) bool

	// GetDuration retrieves a time.Duration configuration value.
	GetDuration(key string, defaultValue ...time.Duration) time.Duration

	// GetStringSlice retrieves a string slice configuration value.
	GetStringSlice(key string, defaultValue ...[]string) []string

	// Has checks if a configuration key exists.
	Has(key string) bool

	// Set sets a configuration value.
	Set(key string, value any)

	// All returns all configuration values.
	All() map[string]any

	// Env returns the current environment (e.g., "development", "production").
	Env(defaultValue ...string) string

	// IsDebug checks if debug mode is enabled.
	IsDebug() bool
}

// Repository implements the Config interface.
type Repository struct {
	items     map[string]any
	env       string
	mu        sync.RWMutex
	cached    bool   // true if config is loaded from cache
	cachePath string // path to cache file
}

// New creates a new configuration repository.
func New() *Repository {
	return &Repository{
		items: make(map[string]any),
		env:   getEnv("APP_ENV", "development"),
	}
}

// NewWithMap creates a new configuration repository with initial values.
func NewWithMap(items map[string]any) *Repository {
	return &Repository{
		items: items,
		env:   getEnv("APP_ENV", "development"),
	}
}

// LoadFromEnv loads configuration from environment variables.
// Environment variables should use uppercase and underscores (e.g., DATABASE_HOST).
func (r *Repository) LoadFromEnv(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Common configuration keys to load from environment
	envMappings := map[string]string{
		"app.name":  "APP_NAME",
		"app.env":   "APP_ENV",
		"app.debug": "APP_DEBUG",
		"app.url":   "APP_URL",
		"app.key":   "APP_KEY",

		"server.host":          "SERVER_HOST",
		"server.port":          "SERVER_PORT",
		"server.read_timeout":  "SERVER_READ_TIMEOUT",
		"server.write_timeout": "SERVER_WRITE_TIMEOUT",

		"database.driver":   "DB_DRIVER",
		"database.host":     "DB_HOST",
		"database.port":     "DB_PORT",
		"database.database": "DB_DATABASE",
		"database.username": "DB_USERNAME",
		"database.password": "DB_PASSWORD",

		"cache.driver": "CACHE_DRIVER",
		"cache.prefix": "CACHE_PREFIX",

		"queue.driver":     "QUEUE_DRIVER",
		"queue.connection": "QUEUE_CONNECTION",

		"log.level":   "LOG_LEVEL",
		"log.channel": "LOG_CHANNEL",
	}

	// Load from environment
	for key, envKey := range envMappings {
		if value := os.Getenv(envKey); value != "" {
			r.setNested(key, value)
		}
	}

	// Update env
	if env := os.Getenv("APP_ENV"); env != "" {
		r.env = env
	}
}

// Get retrieves a configuration value.
func (r *Repository) Get(key string, defaultValue ...any) any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value := r.getNested(key)
	if value != nil {
		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return nil
}

// GetString retrieves a string value.
func (r *Repository) GetString(key string, defaultValue ...string) string {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprintf("%v", value)
}

// GetInt retrieves an integer value.
func (r *Repository) GetInt(key string, defaultValue ...int) int {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

// GetInt64 retrieves an int64 value.
func (r *Repository) GetInt64(key string, defaultValue ...int64) int64 {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

// GetFloat retrieves a float64 value.
func (r *Repository) GetFloat(key string, defaultValue ...float64) float64 {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

// GetBool retrieves a boolean value.
func (r *Repository) GetBool(key string, defaultValue ...bool) bool {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(v)
		return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
	case int:
		return v != 0
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return false
}

// GetDuration retrieves a time.Duration value.
func (r *Repository) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := value.(type) {
	case time.Duration:
		return v
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	case int64:
		return time.Duration(v)
	case int:
		return time.Duration(v)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

// GetStringSlice retrieves a string slice value.
func (r *Repository) GetStringSlice(key string, defaultValue ...[]string) []string {
	value := r.Get(key)

	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}

	switch v := value.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		// Split by comma
		return strings.Split(v, ",")
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return nil
}

// Has checks if a key exists.
func (r *Repository) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.getNested(key) != nil
}

// Set sets a configuration value.
func (r *Repository) Set(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.setNested(key, value)
}

// All returns all configuration values.
func (r *Repository) All() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]any, len(r.items))
	maps.Copy(result, r.items)
	return result
}

// Env returns the current environment.
func (r *Repository) Env(defaultValue ...string) string {
	if r.env != "" {
		return r.env
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return "development"
}

// IsDebug checks if debug mode is enabled.
func (r *Repository) IsDebug() bool {
	return r.GetBool("app.debug", false)
}

// getNested retrieves a value using dot notation.
// Example: "database.connections.mysql.host"
func (r *Repository) getNested(key string) any {
	parts := strings.Split(key, ".")
	current := r.items

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil
		}

		// Last part - return the value
		if i == len(parts)-1 {
			return value
		}

		// Navigate deeper
		if nested, ok := value.(map[string]any); ok {
			current = nested
		} else {
			return nil
		}
	}

	return nil
}

// setNested sets a value using dot notation.
func (r *Repository) setNested(key string, value any) {
	parts := strings.Split(key, ".")
	current := r.items

	for i, part := range parts {
		// Last part - set the value
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		// Navigate or create nested map
		if nested, exists := current[part]; exists {
			if nestedMap, ok := nested.(map[string]any); ok {
				current = nestedMap
			} else {
				// Overwrite with new map
				newMap := make(map[string]any)
				current[part] = newMap
				current = newMap
			}
		} else {
			// Create new nested map
			newMap := make(map[string]any)
			current[part] = newMap
			current = newMap
		}
	}
}

// Merge merges another configuration into this one.
func (r *Repository) Merge(other map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, value := range other {
		r.setNested(key, value)
	}
}

// Helper function to get environment variable with default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
