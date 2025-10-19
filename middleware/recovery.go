// Package middleware provides common middleware implementations for the grouter package.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
)

// Recovery middleware with better error handling and optional callback
func Recovery() grouter.Middleware {
	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
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
						JSON(errors.New(http.StatusInternalServerError, "Internal Server Error", nil))
				}
			}()

			return next(c)
		}
	}
}
