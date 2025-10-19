package grouter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Common middleware implementations for the router

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS(options CORSOptions) Middleware {
	return func(next Handler) Handler {
		return func(c *Ctx) error {
			origin := c.Get("Origin")

			// Set CORS headers
			if len(options.AllowedOrigins) > 0 {
				for _, allowedOrigin := range options.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						c.Set("Access-Control-Allow-Origin", allowedOrigin)
						break
					}
				}
			}

			if len(options.AllowedMethods) > 0 {
				c.Set("Access-Control-Allow-Methods", strings.Join(options.AllowedMethods, ", "))
			}

			if len(options.AllowedHeaders) > 0 {
				c.Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
			}

			if options.AllowCredentials {
				c.Set("Access-Control-Allow-Credentials", "true")
			}

			if options.MaxAge > 0 {
				c.Set("Access-Control-Max-Age", fmt.Sprintf("%d", int(options.MaxAge.Seconds())))
			}

			// Handle preflight requests
			if c.Method() == http.MethodOptions {
				return c.Status(http.StatusOK).SendString("")
			}

			return next(c)
		}
	}
}

// CORSOptions contains configuration for CORS middleware
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSOptions returns sensible default CORS options
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "Accept", "Origin", "User-Agent", "DNT", "Cache-Control", "X-Mx-ReqToken", "Keep-Alive", "X-Requested-With", "If-Modified-Since"},
		MaxAge:         24 * time.Hour,
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
func Timeout(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(c *Ctx) error {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(c.Context(), timeout)
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
			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				// Handler completed before timeout
				c.Response = originalWriter
				return err
			case <-ctx.Done():
				// Timeout occurred - mark writer as timed out to prevent further writes
				mu.Lock()
				tw.timedOut = true
				alreadyWritten := tw.headerWritten
				mu.Unlock()

				// Only send timeout response if handler hasn't written anything yet
				c.Response = originalWriter
				if !alreadyWritten {
					return c.Status(http.StatusRequestTimeout).JSON(Error{
						Code: http.StatusRequestTimeout,
						Data: "Request Timeout",
					})
				}
				return nil
			}
		}
	}
}

// Recovery middleware with better error handling and optional callback
func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(c *Ctx) error {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()

					// Log the error with structured logging
					slog.Error("panic recovered",
						"error", err,
						"stack", string(stack),
					)

					// Return 500 error using Ctx methods
					c.
						Status(http.StatusInternalServerError).
						JSON(Error{
							Code: http.StatusInternalServerError,
							Data: "Internal Server Error",
						})
				}
			}()

			return next(c)
		}
	}
}
