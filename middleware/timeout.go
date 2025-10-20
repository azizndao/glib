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
func Timeout(timeout time.Duration) grouter.Middleware {
	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
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
					"timeout", timeout,
					"headers_written", alreadyWritten,
				)

				// Only send timeout response if handler hasn't written anything yet
				if !alreadyWritten {
					return errors.New(http.StatusGatewayTimeout, "Gateway Timeout", nil)
				}

				// Headers already written - return error for logging/metrics
				return errors.New(http.StatusGatewayTimeout, "Gateway Timeout (response already started)", nil)
			}
		}
	}
}
