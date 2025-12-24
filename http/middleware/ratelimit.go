package middleware

import (
	"time"

	"github.com/azizndao/glib/common/util"
)

// RateLimitConfig holds configuration for the RateLimit middleware
type RateLimitConfig struct {
	// Max is the maximum number of requests allowed in the time window
	Max int

	// Window is the time window for rate limiting
	Window time.Duration
}

// DefaultRateLimitConfig returns default configuration for rate limiting
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
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
func LoadRateLimitConfig() *RateLimitConfig {
	if !util.GetEnvBool("ENABLE_RATE_LIMIT", false) {
		return nil
	}

	cfg := DefaultRateLimitConfig()
	cfg.Max = util.GetEnvInt("RATE_LIMIT_MAX", cfg.Max)
	cfg.Window = util.GetEnvDuration("RATE_LIMIT_WINDOW", cfg.Window)

	return &cfg
}
