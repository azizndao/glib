package middleware

import (
	"glib/demo/services"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/errs"
)

// ContextKey type for context keys
type ContextKey string

const (
	UserIDKey   ContextKey = "user_id"
	UsernameKey ContextKey = "username"
	EmailKey    ContextKey = "email"
)

// Auth middleware
// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response {
	return func(req glib.Request, next glib.Next) glib.Response {
		authHeader := req.Header("Authorization")
		if authHeader == "" {
			return glib.Response{
				Err: errs.NewUnauthorized().Msg("Authorization header required").Err(),
			}
		}

		// Extract token from "Bearer <token>"
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if token == "" {
			return glib.Response{
				Err: errs.NewUnauthorized().
					Msg("Invalid authorization header format. Use: Bearer <token>").Err(),
			}
		}

		// Validate token
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			return glib.Response{Err: err}
		}

		// Add user info to request context using helper method
		req = req.WithValues(map[any]any{
			UserIDKey:   claims.UserID,
			UsernameKey: claims.Username,
			EmailKey:    claims.Email,
		})

		// Call next middleware/handler and add response header
		resp := next(req)
		resp.Header().Set("X-User-ID", claims.UserID.String())

		return resp
	}
}

// RateLimit middleware
// @Middleware name=ratelimit target=api order=5
func RateLimit() func(glib.Request, glib.Next) glib.Response {
	// Simple in-memory rate limiter (for demo purposes)
	requests := make(map[string][]time.Time)
	limit := 100 // requests per minute

	return func(req glib.Request, next glib.Next) glib.Response {
		ip := req.RemoteAddr()
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
			return glib.Response{
				Err: errs.B().
					Code(errs.ResourceExhausted).
					Msg("rate limit exceeded").Err(),
			}
		}

		// Record request
		requests[ip] = append(requests[ip], now)

		// Call next middleware/handler and add rate limit headers
		resp := next(req)
		resp.Header().Set("X-RateLimit-Limit", "100")
		resp.Header().Set("X-RateLimit-Remaining", string(rune(limit-len(requests[ip]))))

		return resp
	}
}
