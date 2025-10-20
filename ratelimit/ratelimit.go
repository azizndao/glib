// Package ratelimit provides middleware for rate limiting HTTP requests
package ratelimit

import (
	"strconv"
	"time"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/util"
)

// DefaultConfig returns default configuration for rate limiting
func DefaultConfig() Config {
	return Config{
		Max:    100,
		Window: time.Minute,
		Store:  NewMemoryStore(),
		KeyGenerator: func(c *grouter.Ctx) string {
			return c.IP()
		},
		Handler: func(c *grouter.Ctx) error {
			return errors.TooManyRequests("Too many requests, please try again later", nil)
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
		HeaderPrefix:           "X-RateLimit-",
	}
}

// DefaultRateLimitConfig is an alias for DefaultConfig (for backwards compatibility)
func DefaultRateLimitConfig() RateLimitConfig {
	return DefaultConfig()
}

// RateLimit creates a middleware that limits the number of requests per client.
// It uses a sliding window algorithm to track request counts.
//
// Example usage:
//
//	// Use default in-memory store
//	router.Use(ratelimit.RateLimit())
//
//	// Custom configuration with Redis
//	redisStore := ratelimit.NewRedisStore(redisClient, "ratelimit:")
//	router.Use(ratelimit.RateLimit(ratelimit.Config{
//	    Max:    50,
//	    Window: time.Minute,
//	    Store:  redisStore,
//	    KeyGenerator: func(c *grouter.Ctx) string {
//	        // Rate limit by user ID if authenticated
//	        if userID := c.GetValue("userID"); userID != nil {
//	            return userID.(string)
//	        }
//	        return c.IP()
//	    },
//	}))
func RateLimit(config ...Config) grouter.Middleware {
	cfg := util.FirstOrDefault(config, DefaultConfig)

	// Use default store if none provided
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore()
	}

	// Set default header prefix if empty
	if cfg.HeaderPrefix == "" {
		cfg.HeaderPrefix = "X-RateLimit-"
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			ctx := c.Context()

			// Generate key for this client
			key := cfg.KeyGenerator(c)

			// Increment counter and check limit
			count, ttl, err := cfg.Store.Increment(ctx, key, cfg.Window)
			if err != nil {
				// On storage error, allow the request but log the error
				// This prevents rate limiter failures from blocking all traffic
				return next(c)
			}

			// Check if limit exceeded
			if count > cfg.Max {
				// Set rate limit headers
				c.Set(cfg.HeaderPrefix+"Limit", strconv.Itoa(cfg.Max))
				c.Set(cfg.HeaderPrefix+"Remaining", "0")
				c.Set(cfg.HeaderPrefix+"Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
				c.Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))

				return cfg.Handler(c)
			}

			// Calculate remaining requests
			remaining := max(cfg.Max-count, 0)

			// Set rate limit headers
			c.Set(cfg.HeaderPrefix+"Limit", strconv.Itoa(cfg.Max))
			c.Set(cfg.HeaderPrefix+"Remaining", strconv.Itoa(remaining))
			c.Set(cfg.HeaderPrefix+"Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			// Execute handler
			return next(c)
		}
	}
}
