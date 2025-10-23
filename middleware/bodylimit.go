package middleware

import (
	"github.com/azizndao/glib/util"
)

// Common size constants for convenience
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB

	// DefaultBodyLimit is the default maximum request body size (4MB)
	DefaultBodyLimit = 4 * MB
)

// BodyLimitConfig holds configuration for the BodyLimit middleware
type BodyLimitConfig struct {
	// MaxSize is the maximum allowed size of request body in bytes
	// Default: 4MB (DefaultBodyLimit)
	MaxSize int64
}

// DefaultBodyLimitConfig returns default configuration for body limit
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		MaxSize: int64(DefaultBodyLimit),
	}
}

// LoadBodyLimitConfig loads BodyLimitConfig from environment variables
// Environment variable: BODY_LIMIT (int64, bytes)
// Returns default config if BODY_LIMIT is not set
// Falls back to DefaultBodyLimitConfig if set but invalid
func LoadBodyLimitConfig() *BodyLimitConfig {
	size := util.GetEnvInt64("BODY_LIMIT", -1)
	if size == -1 {
		// Not set, return default
		cfg := DefaultBodyLimitConfig()
		return &cfg
	}

	if size <= 0 {
		// Invalid value, return default
		cfg := DefaultBodyLimitConfig()
		return &cfg
	}

	return &BodyLimitConfig{
		MaxSize: size,
	}
}
