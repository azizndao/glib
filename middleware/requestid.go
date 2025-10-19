package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/azizndao/grouter"
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
		Header:     "X-Request-ID",
		Generator:  defaultRequestIDGenerator,
		ContextKey: "requestID",
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
func RequestID(config ...RequestIDConfig) grouter.Middleware {
	cfg := DefaultRequestIDConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
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
func GetRequestID(c *grouter.Ctx) string {
	if id := c.GetValue("requestID"); id != nil {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
