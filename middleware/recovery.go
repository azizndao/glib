// Package middleware provides common middleware implementations for the grouter package.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/util"
)

// RecoveryConfig holds configuration for the Recovery middleware
type RecoveryConfig struct {
	// EnableStackTrace determines if stack traces should be logged
	// Default: true (disable in production for performance)
	EnableStackTrace bool

	// PanicHandler is an optional custom handler for panics
	// If nil, default behavior is to return 500 error
	PanicHandler func(*grouter.Ctx, any)
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		EnableStackTrace: true,
		PanicHandler:     nil,
	}
}

// Recovery middleware for panic recovery
//
// Example usage:
//
//	// Use default configuration
//	router.Use(middleware.Recovery())
//
//	// Custom configuration
//	router.Use(middleware.Recovery(middleware.RecoveryConfig{
//	    EnableStackTrace: false,
//	    PanicHandler: func(c *grouter.Ctx, err any) {
//	        log.Printf("Panic: %v", err)
//	        c.Status(500).JSON(map[string]string{"error": "internal server error"})
//	    },
//	}))
func Recovery(config ...RecoveryConfig) grouter.Middleware {
	cfg := util.FirstOrDefault(config, DefaultRecoveryConfig)
	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) (err error) {
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
						panicErr = fmt.Errorf("%s", x)
					case error:
						panicErr = x
					default:
						panicErr = fmt.Errorf("%v", x)
					}

					// Get request ID if available
					requestID := GetRequestID(c)

					// Build log attributes
					attrs := []any{
						"error", panicErr,
						"method", c.Method(),
						"path", c.Path(),
						"remote_addr", c.IP(),
					}

					if requestID != "" {
						attrs = append(attrs, "request_id", requestID)
					}

					// Add stack trace if enabled
					if cfg.EnableStackTrace {
						attrs = append(attrs, "stack", string(debug.Stack()))
					}

					// Log the panic
					slog.Error("panic recovered", attrs...)

					// Check if headers were already written
					if rw, ok := c.Response.(interface{ HeadersWritten() bool }); ok && rw.HeadersWritten() {
						// Can't send error response, headers already sent
						// Just log and set error for middleware to see
						err = fmt.Errorf("panic after headers sent: %w", panicErr)
						return
					}

					// Call custom panic handler if provided
					if cfg.PanicHandler != nil {
						cfg.PanicHandler(c, rvr)
						return
					}

					// Default: return 500 error
					err = errors.New(http.StatusInternalServerError, "Internal Server Error", panicErr)
				}
			}()

			return next(c)
		}
	}
}
