// Package middleware provides common HTTP middleware implementations for grouter.
package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/azizndao/grouter/router"
	"github.com/azizndao/grouter/util"
)

// CORSConfig contains configuration for CORS middleware
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string // Headers that browsers are allowed to access
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns sensible default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "Accept", "Origin", "User-Agent", "DNT", "Cache-Control", "X-Mx-ReqToken", "Keep-Alive", "X-Requested-With", "If-Modified-Since"},
		MaxAge:         24 * time.Hour,
	}
}

// LoadCORSConfig loads CORSConfig from environment variables
// Environment variable: ENABLE_CORS (bool)
// Returns nil if ENABLE_CORS=false, otherwise returns default config
func LoadCORSConfig() *CORSConfig {
	if !util.GetEnvBool("ENABLE_CORS", true) {
		return nil
	}

	cfg := DefaultCORSConfig()
	return &cfg
}

// CORS middleware for handling Cross-Origin Resource Sharing
//
// Example usage:
//
//	// Use default configuration
//	router.Use(middleware.CORS())
//
//	// Custom configuration
//	router.Use(middleware.CORS(middleware.CORSConfig{
//	    AllowedOrigins: []string{"https://example.com"},
//	    AllowedMethods: []string{"GET", "POST"},
//	    AllowCredentials: true,
//	}))
func CORS(config ...CORSConfig) router.Middleware {
	cfg := util.FirstOrDefault(config, DefaultCORSConfig)

	return func(next router.Handler) router.Handler {
		return func(c *router.Ctx) error {
			origin := c.Get("Origin")

			// Set CORS headers
			if len(cfg.AllowedOrigins) > 0 {
				// Security: Don't allow wildcard origin with credentials
				if cfg.AllowCredentials {
					// When credentials are allowed, we must specify exact origin
					hasWildcard := slices.Contains(cfg.AllowedOrigins, "*")

					if hasWildcard && origin != "" {
						// Use the specific origin instead of wildcard
						c.Set("Access-Control-Allow-Origin", origin)
					} else {
						// Check if origin is in allowed list
						if slices.Contains(cfg.AllowedOrigins, origin) {
							c.Set("Access-Control-Allow-Origin", origin)
						}
					}
				} else {
					// Without credentials, wildcard is acceptable
					for _, allowedOrigin := range cfg.AllowedOrigins {
						if allowedOrigin == "*" || allowedOrigin == origin {
							c.Set("Access-Control-Allow-Origin", allowedOrigin)
							break
						}
					}
				}
			}

			if len(cfg.AllowedMethods) > 0 {
				c.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			}

			if len(cfg.AllowedHeaders) > 0 {
				c.Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			}

			if len(cfg.ExposedHeaders) > 0 {
				c.Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
			}

			if cfg.AllowCredentials {
				c.Set("Access-Control-Allow-Credentials", "true")
			}

			if cfg.MaxAge > 0 {
				c.Set("Access-Control-Max-Age", fmt.Sprintf("%d", int(cfg.MaxAge.Seconds())))
			}

			// Handle preflight requests (use 204 No Content as per spec)
			if c.Method() == http.MethodOptions {
				return c.Status(http.StatusNoContent).SendString("")
			}

			return next(c)
		}
	}
}
