// Package middleware provides common middleware implementations for the grouter package.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/router"
	"github.com/azizndao/grouter/util"
)

// LoadRecoveryConfig loads recovery middleware enabled state from environment variables
// Environment variables:
//   - ENABLE_RECOVERY (bool): enable/disable recovery middleware (default: true)
//
// Returns true if recovery middleware should be enabled, false otherwise
func LoadRecoveryConfig() bool {
	return util.GetEnvBool("ENABLE_RECOVERY", true)
}

// Recovery middleware for panic recovery
// Stack traces are always included in panic logs for debugging
//
// Example usage:
//
//	router.Use(middleware.Recovery())
func Recovery() router.Middleware {
	return func(next router.Handler) router.Handler {
		return func(c *router.Ctx) (err error) {
			defer func() {
				if rvr := recover(); rvr != nil {
					// Don't recover http.ErrAbortHandler - let it propagate
					// This allows the server to abort the connection gracefully
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}

					// Convert panic to error
					var panicErr error
					switch x := rvr.(type) {
					case string:
						panicErr = errors.Errorf("%s", x)
					case error:
						panicErr = x
					default:
						panicErr = errors.Errorf("%v", x)
					}

					// Get request ID if available
					requestID := GetRequestID(c)

					// Build log attributes
					attrs := []any{
						"method", c.Method(),
						"path", c.Path(),
						"remote_addr", c.IP(),
					}

					if requestID != "" {
						attrs = append(attrs, "request_id", requestID)
					}

					// Always include stack trace for debugging
					attrs = append(attrs, "stack", string(debug.Stack()))

					// Log the panic using the context logger
					c.Logger().Error(panicErr, attrs...)

					// Check if headers were already written
					if rw, ok := c.Response.(interface{ HeadersWritten() bool }); ok && rw.HeadersWritten() {
						// Can't send error response, headers already sent
						// Just log and set error for middleware to see
						err = errors.Errorf("panic after headers sent: %w", panicErr)
						return
					}

					// Return 500 error
					err = errors.NewApi(http.StatusInternalServerError, "Internal Server Error", panicErr)
				}
			}()

			return next(c)
		}
	}
}
