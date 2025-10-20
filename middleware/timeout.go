package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
)

// TimeoutConfig holds configuration for the Timeout middleware
type TimeoutConfig struct {
	// Timeout is the maximum duration for the request
	// Default: 30 seconds
	Timeout time.Duration

	// ErrorHandler is called when timeout occurs
	// Default: returns 504 Gateway Timeout
	ErrorHandler grouter.Handler
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: 30 * time.Second,
		ErrorHandler: func(c *grouter.Ctx) error {
			return errors.New(http.StatusGatewayTimeout, "Gateway Timeout", nil)
		},
	}
}

// timeoutWriter wraps http.ResponseWriter to prevent writes after timeout
type timeoutWriter struct {
	http.ResponseWriter
	mu            *sync.Mutex
	timedOut      bool
	headerWritten bool
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.headerWritten {
		return
	}
	tw.headerWritten = true
	tw.ResponseWriter.WriteHeader(code)
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if !tw.headerWritten {
		tw.headerWritten = true
	}
	return tw.ResponseWriter.Write(b)
}

// Timeout middleware for request timeout handling
//
// Example usage:
//
//	// Use default configuration (30 seconds)
//	router.Use(middleware.Timeout())
//
//	// Custom timeout duration
//	router.Use(middleware.Timeout(middleware.TimeoutConfig{
//	    Timeout: 10 * time.Second,
//	}))
//
//	// Custom timeout with error handler
//	router.Use(middleware.Timeout(middleware.TimeoutConfig{
//	    Timeout: 5 * time.Second,
//	    ErrorHandler: func(c *grouter.Ctx) error {
//	        return c.Status(504).JSON(map[string]string{"error": "request timeout"})
//	    },
//	}))
func Timeout(config ...TimeoutConfig) grouter.Middleware {
	cfg := DefaultTimeoutConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(c.Context(), cfg.Timeout)
			defer cancel()

			// Create a mutex for synchronizing response writes
			mu := &sync.Mutex{}

			// Wrap the response writer to prevent writes after timeout
			tw := &timeoutWriter{
				ResponseWriter: c.Response,
				mu:             mu,
				timedOut:       false,
				headerWritten:  false,
			}

			// Replace the response writer and request context
			originalWriter := c.Response
			c.Response = tw
			c.Request = c.Request.WithContext(ctx)

			// Execute handler with timeout
			done := make(chan error, 1)
			panicChan := make(chan interface{}, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						panicChan <- r
					}
				}()
				done <- next(c)
			}()

			select {
			case err := <-done:
				// Handler completed before timeout
				c.Response = originalWriter
				return err

			case p := <-panicChan:
				// Panic in handler
				c.Response = originalWriter
				panic(p)

			case <-ctx.Done():
				// Timeout occurred - mark writer as timed out to prevent further writes
				mu.Lock()
				tw.timedOut = true
				alreadyWritten := tw.headerWritten
				mu.Unlock()

				// Restore original writer
				c.Response = originalWriter

				// Log timeout event
				slog.Warn("request timeout",
					"path", c.Path(),
					"method", c.Method(),
					"timeout", cfg.Timeout,
					"headers_written", alreadyWritten,
				)

				// Only send timeout response if handler hasn't written anything yet
				if !alreadyWritten {
					return cfg.ErrorHandler(c)
				}

				// Headers already written - return error for logging/metrics
				return errors.New(http.StatusGatewayTimeout, "Gateway Timeout (response already started)", nil)
			}
		}
	}
}
