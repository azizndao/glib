// Package middleware provides common HTTP middleware implementations for grouter.
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/azizndao/grouter"
)

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS(options CORSOptions) grouter.Middleware {
	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
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
