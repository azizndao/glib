package grouter

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Common middleware implementations for the router

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS(options CORSOptions) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set CORS headers
			if len(options.AllowedOrigins) > 0 {
				for _, allowedOrigin := range options.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
						break
					}
				}
			}

			if len(options.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(options.AllowedMethods, ", "))
			}

			if len(options.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
			}

			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if options.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", int(options.MaxAge.Seconds())))
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
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

// Timeout middleware for request timeout handling
func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request Timeout")
	}
}

// Recovery middleware with better error handling and optional callback
func Recovery(callback func(err any, stack []byte)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()

					// Call callback if provided
					if callback != nil {
						callback(err, stack)
					}

					// Log the error
					slog.Error("PANIC: %v\n%s\n", err, stack)

					// Return 500 error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Chain combines multiple middleware into a single middleware
func Chain(middleware ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		return next
	}
}
