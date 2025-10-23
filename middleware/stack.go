package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/httprate"
)

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
func Stack(logger *slog.Logger) chi.Middlewares {
	middlewares := make([]func(http.Handler) http.Handler, 0)

	// Order matters! These middleware are applied in the order specified

	// RealIP should be early to extract correct client IP
	if util.GetEnvBool("ENABLE_REAL_IP", true) {
		middlewares = append(middlewares, middleware.RealIP)
	}

	// RequestID early for logging
	if util.GetEnvBool("ENABLE_REQUEST_ID", true) {
		middlewares = append(middlewares, middleware.RequestID)
	}

	// Logger after recovery and request ID
	if util.GetEnvBool("ENABLE_LOGGER", true) {
		if util.GetEnvBool("IS_DEBUG", false) {
			middlewares = append(middlewares, middleware.Logger)
		} else {
			middlewares = append(middlewares, httplog.RequestLogger(logger, &httplog.Options{}))
		}
	}

	// Recovery should be early to catch panics from other middleware
	if util.GetEnvBool("ENABLE_RECOVERY", true) {
		middlewares = append(middlewares, middleware.Recoverer)
	}

	// Compression
	if compressCfg := LoadCompressConfig(); compressCfg != nil {
		middlewares = append(middlewares, middleware.Compress(compressCfg.Level))
	}

	// Body limit
	if bodyLimitCfg := LoadBodyLimitConfig(); bodyLimitCfg != nil {
		middlewares = append(middlewares, middleware.RequestSize(bodyLimitCfg.MaxSize))
	}

	// Rate limiting (if enabled via env)
	if rateLimitCfg := LoadRateLimitConfig(); rateLimitCfg != nil {
		middlewares = append(middlewares, httprate.Limit(
			rateLimitCfg.Max,
			rateLimitCfg.Window,
			httprate.WithKeyByRealIP(),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				err := errors.NewApi(http.StatusTooManyRequests, "Rate-limited", nil)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(err)
			}),
		))
	}

	// CORS
	if corsCfg := LoadCORSOptions(); corsCfg != nil {
		middlewares = append(middlewares, cors.Handler(*corsCfg))
	}
	return middlewares
}
