// Package ratelimit provides middleware for rate limiting HTTP requests
package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/router"
	"github.com/azizndao/grouter/util"
)

// statusWriter wraps http.ResponseWriter to track the status code
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// DefaultConfig returns default configuration for rate limiting
func DefaultConfig() Config {
	return Config{
		Max:    100,
		Window: time.Minute,
		Store:  NewMemoryStore(),
		KeyGenerator: func(c *router.Ctx) string {
			return c.IP()
		},
		Handler: func(c *router.Ctx) error {
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

// LoadConfig loads rate limit Config from environment variables
// Environment variables:
//   - ENABLE_RATE_LIMIT (bool): enable/disable rate limiting
//   - RATE_LIMIT_MAX (int): max requests per window
//   - RATE_LIMIT_WINDOW (duration): window duration
//
// Returns nil if ENABLE_RATE_LIMIT=false, otherwise returns config
func LoadConfig() *Config {
	if !util.GetEnvBool("ENABLE_RATE_LIMIT", false) {
		return nil
	}

	cfg := DefaultConfig()
	cfg.Max = util.GetEnvInt("RATE_LIMIT_MAX", cfg.Max)
	cfg.Window = util.GetEnvDuration("RATE_LIMIT_WINDOW", cfg.Window)

	return &cfg
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
func RateLimit(config ...Config) router.Middleware {
	cfg := util.FirstOrDefault(config, DefaultConfig)

	// Use default store if none provided
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore()
	}

	// Set default header prefix if empty
	if cfg.HeaderPrefix == "" {
		cfg.HeaderPrefix = "X-RateLimit-"
	}

	return func(next router.Handler) router.Handler {
		return func(c *router.Ctx) error {
			ctx := c.Context()

			// Generate key for this client
			key := cfg.KeyGenerator(c)

			// Get current count to check limit (without incrementing yet if we need to skip based on result)
			count, ttl, err := cfg.Store.Get(ctx, key)
			if err != nil && err.Error() != "key not found" {
				// On storage error, allow the request but log the error
				// This prevents rate limiter failures from blocking all traffic
				c.Logger().Error(errors.Errorf("rate limiter storage error on Get %v", err), "key", key)
				return next(c)
			}

			// Check if limit already exceeded
			if count >= cfg.Max {
				// Set rate limit headers
				c.Set(cfg.HeaderPrefix+"Limit", strconv.Itoa(cfg.Max))
				c.Set(cfg.HeaderPrefix+"Remaining", "0")
				c.Set(cfg.HeaderPrefix+"Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
				c.Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))

				return cfg.Handler(c)
			}

			// Increment counter before execution to prevent race conditions
			count, ttl, err = cfg.Store.Increment(ctx, key, cfg.Window)
			if err != nil {
				// On storage error, allow the request but log the error
				// This prevents rate limiter failures from blocking all traffic
				c.Logger().Error(errors.Errorf("rate limiter storage error on Increment %v", err), "key", key)
				return next(c)
			}

			// Calculate remaining requests
			remaining := max(cfg.Max-count, 0)

			// Set rate limit headers
			c.Set(cfg.HeaderPrefix+"Limit", strconv.Itoa(cfg.Max))
			c.Set(cfg.HeaderPrefix+"Remaining", strconv.Itoa(remaining))
			c.Set(cfg.HeaderPrefix+"Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			// Wrap response writer to track status code if we need to skip based on success/failure
			var sw *statusWriter
			if cfg.SkipFailedRequests || cfg.SkipSuccessfulRequests {
				sw = &statusWriter{ResponseWriter: c.Response, statusCode: 0}
				c.Response = sw
			}

			// Execute handler
			err = next(c)

			// Check if we should skip this request based on success/failure
			if cfg.SkipFailedRequests || cfg.SkipSuccessfulRequests {
				status := sw.statusCode
				if status == 0 {
					status = http.StatusOK
				}

				isFailed := status >= 400
				isSuccessful := status >= 200 && status < 400

				shouldSkip := (cfg.SkipFailedRequests && isFailed) || (cfg.SkipSuccessfulRequests && isSuccessful)

				if shouldSkip {
					// Decrement the counter since we shouldn't have counted this request
					if decrementErr := cfg.Store.Decrement(ctx, key); decrementErr != nil {
						// Log error but don't fail the request
						c.Logger().Error(errors.Errorf("rate limiter storage error on Decrement %v", decrementErr), "key", key)
					}
				}
			}

			return err
		}
	}
}
