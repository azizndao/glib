package middleware

import (
	"time"

	"github.com/azizndao/glib/util"
)

// Config holds configuration for the RateLimit middleware
type Config struct {
	// Max is the maximum number of requests allowed in the time window
	Max int

	// Window is the time window for rate limiting
	Window time.Duration
}

// DefaultConfig returns default configuration for rate limiting
func DefaultConfig() Config {
	return Config{
		Max:    100,
		Window: time.Minute,
	}
}

// LoadRateLimitConfig loads rate limit Config from environment variables
// Environment variables:
//   - ENABLE_RATE_LIMIT (bool): enable/disable rate limiting
//   - RATE_LIMIT_MAX (int): max requests per window
//   - RATE_LIMIT_WINDOW (duration): window duration
//
// Returns nil if ENABLE_RATE_LIMIT=false, otherwise returns config
func LoadRateLimitConfig() *Config {
	if !util.GetEnvBool("ENABLE_RATE_LIMIT", false) {
		return nil
	}

	cfg := DefaultConfig()
	cfg.Max = util.GetEnvInt("RATE_LIMIT_MAX", cfg.Max)
	cfg.Window = util.GetEnvDuration("RATE_LIMIT_WINDOW", cfg.Window)

	return &cfg
}
