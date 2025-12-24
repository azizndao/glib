package middleware

import (
	"time"
)

const (
	// DefaultTimeout is the default timeout duration for requests
	DefaultTimeout = 30 * time.Second
)

// TimeoutConfig holds configuration for the Timeout middleware
type TimeoutConfig struct {
	// Timeout is the maximum duration for the request
	// Default: 30 seconds
	Timeout time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: DefaultTimeout,
	}
}
