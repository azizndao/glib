package util

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// GetEnv returns the environment variable value or the default if not set
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns the environment variable value as int or the default if not set or invalid
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvInt64 returns the environment variable value as int64 or the default if not set or invalid
func GetEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvBool returns the environment variable value as bool or the default if not set or invalid
// Accepts: true/false, 1/0, yes/no, on/off (case insensitive)
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch value {
		case "true", "1", "yes", "on", "True", "TRUE", "YES", "ON":
			return true
		case "false", "0", "no", "off", "False", "FALSE", "NO", "OFF":
			return false
		}
	}
	return defaultValue
}

// GetEnvDuration returns the environment variable value as time.Duration or the default if not set or invalid
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvStringSlice returns the environment variable value as a slice of strings or the default if not set
// Values should be comma-separated. Whitespace around each value is trimmed.
// Example: "value1,value2,value3" or "value1, value2, value3"
func GetEnvStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Split by comma and trim whitespace
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	// Return default if no valid values found
	if len(result) == 0 {
		return defaultValue
	}

	return result
}

// GetEnvLogFormat returns the environment variable value as a log format string or the default if not set or invalid
// Accepts: default, combined, short, tiny (case insensitive)
func GetEnvLogFormat(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Normalize to lowercase for comparison
	normalized := strings.ToLower(strings.TrimSpace(value))

	// Validate against known formats
	switch normalized {
	case "default", "combined", "short", "tiny":
		return normalized
	default:
		return defaultValue
	}
}
