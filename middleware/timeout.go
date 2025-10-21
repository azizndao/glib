package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/router"
	"github.com/azizndao/grouter/util"
)

const (
	// DefaultTimeout is the default timeout duration for requests
	DefaultTimeout = 30 * time.Second
)

// TimeoutConfig holds configuration for the Timeout middleware
type TimeoutConfig struct {
	// Timeout is the maximum duration for the request
	// Default: 30 seconds
	Timeout time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: DefaultTimeout,
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
// IMPORTANT: Handlers must respect context cancellation to prevent goroutine leaks.
// Always check ctx.Done() in long-running operations.
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
//	// Handler that respects context cancellation:
//	func handler(c *router.Ctx) error {
//	    select {
//	    case <-c.Context().Done():
//	        return c.Context().Err()
//	    case result := <-someLongOperation():
//	        return c.JSON(result)
//	    }
//	}
func Timeout(config ...TimeoutConfig) router.Middleware {
	cfg := util.FirstOrDefault(config, DefaultTimeoutConfig)

	return func(next router.Handler) router.Handler {
		return func(c *router.Ctx) error {
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
			// Buffered channels (size 1) prevent the goroutine from blocking on send
			// even if the select statement exits due to timeout
			done := make(chan error, 1)
			panicChan := make(chan any, 1)

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
					return errors.NewApi(http.StatusGatewayTimeout, "Gateway Timeout", nil)
				}

				// Headers already written - return error for logging/metrics
				return errors.NewApi(http.StatusGatewayTimeout, "Gateway Timeout (response already started)", nil)
			}
		}
	}
}
