package util

import (
	"os"
	"strconv"
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
