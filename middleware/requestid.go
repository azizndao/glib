package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/azizndao/grouter/router"
	"github.com/azizndao/grouter/util"
)

const (
	// DefaultRequestIDContextKey is the default key used to store request ID in context
	DefaultRequestIDContextKey = "requestID"

	// DefaultRequestIDHeader is the default header name for request ID
	DefaultRequestIDHeader = "X-Request-ID"
)

// RequestIDConfig holds configuration for the RequestID middleware
type RequestIDConfig struct {
	// Header is the name of the header to use for request ID
	// Default: "X-Request-ID"
	Header string

	// Generator is a function that generates a unique request ID
	// Default: generates a random 16-byte hex string
	Generator func() string

	// ContextKey is the key used to store the request ID in context
	// Default: "requestID"
	ContextKey string
}

// DefaultRequestIDConfig returns default configuration for RequestID middleware
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		Header:     DefaultRequestIDHeader,
		Generator:  defaultRequestIDGenerator,
		ContextKey: DefaultRequestIDContextKey,
	}
}

// defaultRequestIDGenerator generates a random 16-byte hex string
func defaultRequestIDGenerator() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple counter-based ID if random fails
		return hex.EncodeToString([]byte("fallback"))
	}
	return hex.EncodeToString(b)
}

// LoadRequestIDConfig loads RequestIDConfig from environment variables
// Environment variable: ENABLE_REQUEST_ID (bool)
// Returns nil if ENABLE_REQUEST_ID=false, otherwise returns default config
func LoadRequestIDConfig() *RequestIDConfig {
	if !util.GetEnvBool("ENABLE_REQUEST_ID", true) {
		return nil
	}

	cfg := DefaultRequestIDConfig()
	return &cfg
}

// RequestID creates a middleware that adds a unique request ID to each request.
// The request ID is:
// 1. Read from the configured header if present in the request
// 2. Generated using the configured generator if not present
// 3. Added to the response header
// 4. Stored in the request context for use in handlers and other middleware
//
// Example usage:
//
//	router.Use(middleware.RequestID())
//
//	// Access in handler:
//	func handler(c *grouter.Ctx) error {
//	    requestID := c.GetValue("requestID").(string)
//	    log.Printf("Request ID: %s", requestID)
//	    return c.JSON(map[string]string{"request_id": requestID})
//	}
func RequestID(config ...RequestIDConfig) router.Middleware {
	cfg := util.FirstOrDefault(config, DefaultRequestIDConfig)

	return func(next router.Handler) router.Handler {
		return func(c *router.Ctx) error {
			// Check if request ID already exists in header
			requestID := c.Get(cfg.Header)
			if requestID == "" {
				// Generate new request ID
				requestID = cfg.Generator()
			}

			// Set request ID in response header
			c.Set(cfg.Header, requestID)

			// Store request ID in context
			c.Request = c.SetValue(cfg.ContextKey, requestID)

			return next(c)
		}
	}
}

// GetRequestID is a helper function to retrieve the request ID from context
// Returns empty string if not found
// Note: This uses the default context key. If you changed the ContextKey in config,
// use c.GetValue(yourContextKey) directly instead.
func GetRequestID(c *router.Ctx) string {
	if id := c.GetValue(DefaultRequestIDContextKey); id != nil {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
