package middleware

import (
	"context"
	"fmt"
	"glib/demo/services"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/pkg/middleware"
)

// ContextKey type for context keys
type ContextKey string

const (
	UserIDKey   ContextKey = "user_id"
	UsernameKey ContextKey = "username"
	EmailKey    ContextKey = "email"
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
func Auth(jwtService *services.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"unauthenticated","message":"Authorization header required"}}`))
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"unauthenticated","message":"Invalid authorization header format. Use: Bearer <token>"}}`))
				return
			}

			token := parts[1]

			// Validate token
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(fmt.Sprintf(`{"error":{"code":"unauthenticated","message":"Invalid token: %s"}}`, err.Error())))
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)

			// Continue with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
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
