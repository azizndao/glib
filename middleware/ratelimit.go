package middleware

import (
	"sync"
	"time"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
)

// RateLimitConfig holds configuration for the RateLimit middleware
type RateLimitConfig struct {
	// Max is the maximum number of requests allowed in the time window
	Max int

	// Window is the time window for rate limiting
	Window time.Duration

	// KeyGenerator is a function that generates a unique key for each client
	// Default: uses IP address
	KeyGenerator func(*grouter.Ctx) string

	// Handler is called when rate limit is exceeded
	// Default: returns 429 Too Many Requests
	Handler grouter.Handler

	// SkipFailedRequests determines if failed requests should be counted
	// Default: false
	SkipFailedRequests bool

	// SkipSuccessfulRequests determines if successful requests should be counted
	// Default: false
	SkipSuccessfulRequests bool
}

// rateLimitEntry tracks request count and window start time for a client
type rateLimitEntry struct {
	count       int
	windowStart time.Time
	mu          sync.Mutex
}

// rateLimiter manages rate limiting state
type rateLimiter struct {
	config  RateLimitConfig
	clients map[string]*rateLimitEntry
	mu      sync.RWMutex
	cleanup *time.Ticker
}

// DefaultRateLimitConfig returns default configuration for rate limiting
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Max:    100,
		Window: time.Minute,
		KeyGenerator: func(c *grouter.Ctx) string {
			return c.IP()
		},
		Handler: func(c *grouter.Ctx) error {
			return errors.TooManyRequests("Too many requests, please try again later", nil)
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	}
}

// RateLimit creates a middleware that limits the number of requests per client.
// It uses a sliding window algorithm to track request counts.
//
// Example usage:
//
//	// Limit to 100 requests per minute per IP
//	router.Use(middleware.RateLimit())
//
//	// Custom configuration
//	router.Use(middleware.RateLimit(middleware.RateLimitConfig{
//	    Max:    50,
//	    Window: time.Minute,
//	    KeyGenerator: func(c *grouter.Ctx) string {
//	        // Rate limit by user ID if authenticated
//	        if userID := c.GetValue("userID"); userID != nil {
//	            return userID.(string)
//	        }
//	        return c.IP()
//	    },
//	}))
func RateLimit(config ...RateLimitConfig) grouter.Middleware {
	cfg := DefaultRateLimitConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	limiter := &rateLimiter{
		config:  cfg,
		clients: make(map[string]*rateLimitEntry),
	}

	// Start cleanup goroutine to remove old entries
	limiter.cleanup = time.NewTicker(cfg.Window)
	go limiter.cleanupRoutine()

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Generate key for this client
			key := cfg.KeyGenerator(c)

			// Check if client is rate limited
			if !limiter.allow(key) {
				// Set rate limit headers
				c.Set("X-RateLimit-Limit", string(rune(cfg.Max)))
				c.Set("X-RateLimit-Remaining", "0")
				c.Set("Retry-After", string(rune(int(cfg.Window.Seconds()))))

				return cfg.Handler(c)
			}

			// Execute handler
			err := next(c)

			// Update rate limit headers
			remaining := limiter.getRemaining(key)
			c.Set("X-RateLimit-Limit", string(rune(cfg.Max)))
			c.Set("X-RateLimit-Remaining", string(rune(remaining)))

			return err
		}
	}
}

// allow checks if a request from the given key should be allowed
func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	entry, exists := rl.clients[key]
	if !exists {
		entry = &rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		rl.clients[key] = entry
		rl.mu.Unlock()
		return true
	}
	rl.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if window has expired
	if now.Sub(entry.windowStart) > rl.config.Window {
		// Reset window
		entry.count = 1
		entry.windowStart = now
		return true
	}

	// Check if limit exceeded
	if entry.count >= rl.config.Max {
		return false
	}

	// Increment count
	entry.count++
	return true
}

// getRemaining returns the number of remaining requests for a key
func (rl *rateLimiter) getRemaining(key string) int {
	rl.mu.RLock()
	entry, exists := rl.clients[key]
	rl.mu.RUnlock()

	if !exists {
		return rl.config.Max
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	remaining := rl.config.Max - entry.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// cleanupRoutine periodically removes expired entries
func (rl *rateLimiter) cleanupRoutine() {
	for range rl.cleanup.C {
		now := time.Now()

		rl.mu.Lock()
		for key, entry := range rl.clients {
			entry.mu.Lock()
			if now.Sub(entry.windowStart) > rl.config.Window*2 {
				delete(rl.clients, key)
			}
			entry.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}
