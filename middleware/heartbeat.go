package middleware

import (
	"net/http"
	"strings"

	"github.com/azizndao/grouter"
)

// Heartbeat creates a middleware that responds to health check requests.
// It intercepts requests to the specified endpoint and returns a 200 OK response
// without executing the rest of the middleware chain or route handlers.
//
// This is useful for:
// - Load balancer health checks
// - Kubernetes liveness/readiness probes
// - Uptime monitoring services
//
// Example usage:
//
//	// Respond to GET /ping with 200 OK
//	router.Use(middleware.Heartbeat("/ping"))
//
//	// Respond to GET /health with custom response
//	router.Use(middleware.HeartbeatWithResponse("/health", "OK"))
func Heartbeat(endpoint string) grouter.Middleware {
	return HeartbeatWithResponse(endpoint, ".")
}

// HeartbeatWithResponse creates a heartbeat middleware with a custom response body
func HeartbeatWithResponse(endpoint string, response string) grouter.Middleware {
	// Normalize endpoint
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	responseBytes := []byte(response)

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Only respond to GET or HEAD requests at the specified endpoint
			if (c.Method() == http.MethodGet || c.Method() == http.MethodHead) &&
				c.Path() == endpoint {

				// Set headers
				c.Set("Content-Type", "text/plain")
				c.Response.WriteHeader(http.StatusOK)

				// Write response body (skip for HEAD requests)
				if c.Method() == http.MethodGet {
					c.Response.Write(responseBytes)
				}

				// Don't call next handler
				return nil
			}

			return next(c)
		}
	}
}
