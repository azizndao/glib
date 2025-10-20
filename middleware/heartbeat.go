package middleware

import (
	"net/http"
	"strings"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/util"
)

// HeartbeatConfig holds configuration for the Heartbeat middleware
type HeartbeatConfig struct {
	// Endpoint is the path to respond to
	// Default: "/ping"
	Endpoint string

	// Response is the response body to send
	// Default: "."
	Response string
}

// DefaultHeartbeatConfig returns default heartbeat configuration
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Endpoint: "/ping",
		Response: ".",
	}
}

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
//	// Use default configuration (GET /ping returns ".")
//	router.Use(middleware.Heartbeat())
//
//	// Custom endpoint
//	router.Use(middleware.Heartbeat(middleware.HeartbeatConfig{
//	    Endpoint: "/health",
//	}))
//
//	// Custom endpoint and response
//	router.Use(middleware.Heartbeat(middleware.HeartbeatConfig{
//	    Endpoint: "/health",
//	    Response: "OK",
//	}))
func Heartbeat(config ...HeartbeatConfig) grouter.Middleware {
	cfg := util.FirstOrDefault(config, DefaultHeartbeatConfig)

	// Normalize endpoint
	if !strings.HasPrefix(cfg.Endpoint, "/") {
		cfg.Endpoint = "/" + cfg.Endpoint
	}

	responseBytes := []byte(cfg.Response)

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Only respond to GET or HEAD requests at the specified endpoint
			if (c.Method() == http.MethodGet || c.Method() == http.MethodHead) &&
				c.Path() == cfg.Endpoint {

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
