package middleware

import (
	"github.com/azizndao/glib/ratelimit"
	"github.com/azizndao/glib/router"
	"github.com/azizndao/glib/validation"
)

// StackConfig holds configuration for building the middleware stack
type StackConfig struct {
	Locales []validation.LocaleConfig
	Store   ratelimit.Store // Optional: Custom store for rate limiting
}

// Stack builds a middleware stack from environment variables.
// Middleware are loaded and applied in this specific order:
//  1. RealIP - Extract real client IP from proxy headers
//  2. RequestID - Generate unique request IDs
//  3. Recovery - Panic recovery (prevents crashes)
//  4. Logger - Request/response logging
//  5. Compress - GZIP/Deflate compression
//  6. BodyLimit - Request body size limiting
//  7. RateLimit - Rate limiting (if configured)
//  8. CORS - Cross-origin resource sharing
//  9. Validation - Request validation with i18n (if locales provided)
//
// Each middleware can be disabled via its corresponding ENABLE_* environment variable.
// Pass StackConfig with validation locales and optional custom store.
func Stack(config StackConfig) []router.Middleware {
	middlewares := make([]router.Middleware, 0)

	// Order matters! These middleware are applied in the order specified

	// RealIP should be early to extract correct client IP
	if realIPCfg := LoadRealIPConfig(); realIPCfg != nil {
		middlewares = append(middlewares, RealIP(*realIPCfg))
	}

	// RequestID early for logging
	if requestIDCfg := LoadRequestIDConfig(); requestIDCfg != nil {
		middlewares = append(middlewares, RequestID(*requestIDCfg))
	}

	// Recovery should be early to catch panics from other middleware
	if LoadRecoveryConfig() {
		middlewares = append(middlewares, Recovery())
	}

	// Logger after recovery and request ID
	if loggerCfg := LoadLoggerConfig(); loggerCfg != nil {
		middlewares = append(middlewares, Logger(*loggerCfg))
	}

	// Compression
	if compressCfg := LoadCompressConfig(); compressCfg != nil {
		middlewares = append(middlewares, Compress(*compressCfg))
	}

	// Body limit
	if bodyLimitCfg := LoadBodyLimitConfig(); bodyLimitCfg != nil {
		middlewares = append(middlewares, BodyLimit(*bodyLimitCfg))
	}

	// Rate limiting (if enabled via env)
	if rateLimitCfg := ratelimit.LoadConfig(); rateLimitCfg != nil {
		// Use custom store if provided
		if config.Store != nil {
			rateLimitCfg.Store = config.Store
		}
		middlewares = append(middlewares, ratelimit.RateLimit(*rateLimitCfg))
	}

	// CORS
	if corsCfg := LoadCORSConfig(); corsCfg != nil {
		middlewares = append(middlewares, CORS(*corsCfg))
	}

	// Validation (if locales provided)
	if len(config.Locales) > 0 {
		middlewares = append(middlewares, validation.Middleware(config.Locales...))
	}

	return middlewares
}
