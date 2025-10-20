package middleware

import (
	"fmt"
	"io"
	"net/http"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/util"
)

// BodyLimitConfig holds configuration for the BodyLimit middleware
type BodyLimitConfig struct {
	// MaxSize is the maximum allowed size of request body in bytes
	// Default: 4MB (4 * 1024 * 1024)
	MaxSize int64

	// SkipFunc is a function that determines if body size check should be skipped
	// Default: nil (check all requests)
	SkipFunc func(*grouter.Ctx) bool

	// ErrorHandler is called when body size exceeds limit
	// Default: returns 413 Request Entity Too Large
	ErrorHandler grouter.Handler
}

// DefaultBodyLimitConfig returns default configuration for body limit
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		MaxSize:  4 * 1024 * 1024, // 4MB
		SkipFunc: nil,
		ErrorHandler: func(c *grouter.Ctx) error {
			return errors.RequestEntityTooLarge(
				fmt.Sprintf("Request body too large. Maximum size is %d bytes", 4*1024*1024),
				nil,
			)
		},
	}
}

// BodyLimit creates a middleware that limits the maximum size of request bodies.
// This helps prevent DoS attacks and excessive memory usage.
//
// Example usage:
//
//	// Use default limit (4MB)
//	router.Use(middleware.BodyLimit())
//
//	// Custom configuration
//	router.Use(middleware.BodyLimit(middleware.BodyLimitConfig{
//	    MaxSize: 10 * 1024 * 1024, // 10MB
//	    SkipFunc: func(c *grouter.Ctx) bool {
//	        // Skip limit for file upload endpoints
//	        return strings.HasPrefix(c.Path(), "/upload")
//	    },
//	}))
//
//	// Using helper function
//	router.Use(middleware.BodyLimitWithSize(10 * 1024 * 1024)) // 10MB
func BodyLimit(config ...BodyLimitConfig) grouter.Middleware {
	cfg := util.FirstOrDefault(config, DefaultBodyLimitConfig)

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Skip if skip function returns true
			if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
				return next(c)
			}

			// Only check for methods that may have a body
			method := c.Method()
			if method != http.MethodPost && method != http.MethodPut &&
				method != http.MethodPatch && method != http.MethodDelete {
				return next(c)
			}

			// Wrap the request body with a limited reader
			c.Request.Body = http.MaxBytesReader(c.Response, c.Request.Body, cfg.MaxSize)

			// Execute handler
			err := next(c)

			// Check if error is due to body size limit
			if err != nil {
				if err.Error() == "http: request body too large" {
					return cfg.ErrorHandler(c)
				}
			}

			return err
		}
	}
}

// Common size constants for convenience
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

// limitedReader wraps io.ReadCloser to enforce size limit
type limitedReader struct {
	reader    io.ReadCloser
	remaining int64
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.remaining <= 0 {
		return 0, io.EOF
	}

	if int64(len(p)) > lr.remaining {
		p = p[:lr.remaining]
	}

	n, err = lr.reader.Read(p)
	lr.remaining -= int64(n)
	return n, err
}

func (lr *limitedReader) Close() error {
	return lr.reader.Close()
}
