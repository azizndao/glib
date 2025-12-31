package middleware

import (
	"context"
	"glib/demo/services"
	"log"
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
func Logger() middleware.Middleware {
	return func(request middleware.Request, next middleware.Next) glib.Result[any] {
		start := time.Now()
		log.Printf("[%s] %s - START", request.Method(), request.Path())

		result := next(request)

		log.Printf("[%s] %s - DONE (%v)", request.Method(), request.Path(), time.Since(start))

		return result
	}
}

// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) middleware.Middleware {
	return func(request middleware.Request, next middleware.Next) glib.Result[any] {
		authHeader := request.Header("Authorization")
		if authHeader == "" {
			return glib.Unauthorized[any]("Authorization header required")
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return glib.Unauthorized[any](
				"Invalid authorization header format. Use: Bearer <token>",
			)
		}

		token := parts[1]

		// Validate token
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			return glib.Fail[any](err)
		}

		// Add user info to context
		ctx := context.WithValue(request.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)

		// Continue with updated context
		return next(request.WithContext(ctx))
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
			return glib.TooManyRequests[any]("rate limit exceeded")
		}

		// Record request
		requests[ip] = append(requests[ip], now)

		// Continue to next middleware/handler
		return next(req)
	}
}
