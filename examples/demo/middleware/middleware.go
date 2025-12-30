package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/pkg/middleware"
)

// @Middleware name=logger target=all order=1
func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			log.Printf("[%s] %s - START", r.Method, r.URL.Path)

			next.ServeHTTP(w, r)

			log.Printf("[%s] %s - DONE (%v)", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

// @Middleware name=auth target=protected order=10
func Auth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Token validation would go here
			next.ServeHTTP(w, r)
		})
	}
}

// @Middleware name=ratelimit target=api order=5
// RateLimit is a new-style middleware that demonstrates the glib.Middleware signature
func RateLimit() middleware.Middleware {
	// Simple in-memory rate limiter (for demo purposes)
	requests := make(map[string][]time.Time)
	limit := 100 // requests per minute

	return func(req middleware.Request, next middleware.Next) glib.Result[any] {
		ip := req.HTTPRequest().RemoteAddr
		now := time.Now()

		// Clean old requests
		if reqs, ok := requests[ip]; ok {
			var recent []time.Time
			for _, t := range reqs {
				if now.Sub(t) < time.Minute {
					recent = append(recent, t)
				}
			}
			requests[ip] = recent
		}

		// Check limit
		if len(requests[ip]) >= limit {
			return middleware.Error(
				fmt.Errorf("rate limit exceeded"),
				http.StatusTooManyRequests,
			)
		}

		// Record request
		requests[ip] = append(requests[ip], now)

		// Continue to next middleware/handler
		return next(req)
	}
}
