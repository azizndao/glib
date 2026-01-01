package utils

import (
	"net/url"
	"os"
	"strings"
	"time"
)

// GetEnv retrieves an environment variable.
// Returns the value and a boolean indicating if the variable exists.
// Uses os.LookupEnv to properly distinguish between unset and empty values.
func GetEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// GetEnvOr retrieves an environment variable or returns a fallback.
// Returns the actual value if the variable exists (even if empty),
// or the fallback if the variable is not set.
func GetEnvOr(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}

// GetEnvSlice retrieves a comma-separated environment variable as a string slice.
// Trims whitespace from each element and filters out empty strings.
// Returns the fallback slice if the variable doesn't exist.
func GetEnvSlice(key, fallback string) []string {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = fallback
	}
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetEnvInt retrieves an environment variable as an integer.
// Returns the fallback value if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvInt(key string, fallback int) (int, error) {
	v, ok := GetEnv(key)
	if !ok {
		return fallback, nil
	}
	return ParseInt(v, key)
}

// GetEnvInt64 retrieves an environment variable as an int64.
// Returns the fallback value if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvInt64(key string, fallback int64) (int64, error) {
	v, ok := GetEnv(key)
	if !ok {
		return fallback, nil
	}
	return ParseInt64(v, key)
}

// GetEnvUint64 retrieves an environment variable as a uint64.
// Returns the fallback value if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvUint64(key string, fallback uint64) (uint64, error) {
	v, ok := GetEnv(key)
	if !ok {
		return fallback, nil
	}
	return ParseUint64(v, key)
}

// GetEnvFloat64 retrieves an environment variable as a float64.
// Returns the fallback value if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvFloat64(key string, fallback float64) (float64, error) {
	v, ok := GetEnv(key)
	if !ok {
		return fallback, nil
	}
	return ParseFloat64(v, key)
}

// GetEnvBool retrieves an environment variable as a boolean.
// Returns the fallback value if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvBool(key string, fallback bool) (bool, error) {
	v, ok := GetEnv(key)
	if !ok {
		return fallback, nil
	}
	return ParseBool(v, key)
}

// GetEnvDuration retrieves an environment variable as a time.Duration.
// Returns the fallback value if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := GetEnv(key)
	if !ok {
		return fallback, nil
	}
	return ParseDuration(v, key)
}

// GetEnvURL retrieves an environment variable as a *url.URL.
// Returns nil if the variable doesn't exist.
// Returns an error if parsing fails.
func GetEnvURL(key string) (*url.URL, error) {
	v, ok := GetEnv(key)
	if !ok {
		return nil, nil
	}
	return ParseURL(v, key)
}
