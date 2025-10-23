// Package middleware provides common HTTP middleware implementations for glib.
package middleware

import (
	"net/http"

	"github.com/azizndao/glib/util"
	"github.com/go-chi/cors"
)

// DefaultCORSOptions returns sensible default CORS configuration
func DefaultCORSOptions() cors.Options {
	return cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}
}

// LoadCORSOptions loads CORSConfig from environment variables
// Environment variables:
//   - ENABLE_CORS (bool): enable/disable CORS middleware (default: true)
//   - CORS_ALLOWED_ORIGINS (string): comma-separated list of allowed origins (default: "*")
//     Example: "https://example.com,https://app.example.com"
//   - CORS_ALLOWED_METHODS (string): comma-separated list of allowed HTTP methods
//     Example: "GET,POST,PUT,DELETE"
//   - CORS_ALLOWED_HEADERS (string): comma-separated list of allowed headers
//     Example: "Authorization,Content-Type"
//   - CORS_EXPOSED_HEADERS (string): comma-separated list of headers browsers can access
//   - CORS_ALLOW_CREDENTIALS (bool): whether to allow credentials (default: false)
//   - CORS_MAX_AGE (duration): how long preflight requests can be cached
//     Example: "24h", "3600s" (default: 24h)
//
// Returns nil if ENABLE_CORS=false, otherwise returns config with values from env or defaults
func LoadCORSOptions() *cors.Options {
	if !util.GetEnvBool("ENABLE_CORS", true) {
		return nil
	}

	options := DefaultCORSOptions()

	// Load configuration from environment variables
	options.Debug = util.GetEnvBool("IS_DEBUG", options.Debug)
	options.AllowedOrigins = util.GetEnvStringSlice("CORS_ALLOWED_ORIGINS", options.AllowedOrigins)
	options.AllowedMethods = util.GetEnvStringSlice("CORS_ALLOWED_METHODS", options.AllowedMethods)
	options.AllowedHeaders = util.GetEnvStringSlice("CORS_ALLOWED_HEADERS", options.AllowedHeaders)
	options.ExposedHeaders = util.GetEnvStringSlice("CORS_EXPOSED_HEADERS", options.ExposedHeaders)
	options.AllowCredentials = util.GetEnvBool("CORS_ALLOW_CREDENTIALS", options.AllowCredentials)
	options.OptionsPassthrough = util.GetEnvBool("CORS_OPTIONS_PASSTHROUGH", options.OptionsPassthrough)
	maxAge := int(util.GetEnvDuration("CORS_MAX_AGE", 0).Seconds())
	if maxAge > 0 {
		options.MaxAge = maxAge
	}

	return &options
}
