# Middleware System

## Overview

Glib supports two middleware signatures:

1. **Glib-style** (Recommended): `func(glib.Request, glib.Next) glib.Response`
   - Clean, functional API inspired by Encore.dev
   - Supports both pre-processing and post-processing
   - Immutable request pattern with context helpers
   - Type-safe response handling

2. **Standard** (Backward Compatible): `func(http.Handler) http.Handler`
   - Traditional Go HTTP middleware pattern
   - Full compatibility with existing middleware ecosystems
   - Direct access to `http.ResponseWriter` and `*http.Request`

## Glib-Style Middleware (Recommended)

### Signature

```go
func(glib.Request, glib.Next) glib.Response
```

Where:
- `glib.Request` - Immutable request wrapper with helper methods
- `glib.Next` - Function to call the next middleware/handler: `func(glib.Request) glib.Response`
- `glib.Response` - Response struct with payload, error, status, and headers

### Basic Example

```go
// @Middleware name=auth target=protected order=10
func Auth(jwtService *JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        // Pre-processing: Validate token
        token := req.Header("Authorization")
        if token == "" {
            return glib.Response{
                Err: errs.NewUnauthorized().Msg("missing token").Err(),
            }
        }
        
        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            return glib.Response{Err: err}
        }
        
        // Add values to context using helper
        req = req.WithValues(map[any]any{
            UserIDKey: claims.UserID,
            RoleKey:   claims.Role,
        })
        
        // Call next middleware/handler
        resp := next(req)
        
        // Post-processing: Add response headers
        resp.Header().Set("X-User-ID", claims.UserID.String())
        resp.Header().Set("X-Request-ID", uuid.New().String())
        
        return resp
    }
}
```

### Request API

The `glib.Request` type provides a clean, immutable API:

#### Reading Request Data

```go
// HTTP basics
req.Method()              // "GET", "POST", etc.
req.Path()                // "/api/users"
req.URL()                 // Full URL string
req.RemoteAddr()          // Client IP

// Headers
req.Header("Authorization")      // Single header value
req.Header("Content-Type")       // Returns "" if not present

// Query parameters
req.Query("page")                // First value, "" if not present
req.QuerySlice("tags")           // All values, empty slice if not present

// Path parameters (Go 1.22+)
req.PathValue("id")              // From route pattern like "/users/{id}"

// Context values
req.Context()                    // Full context
req.Value(UserIDKey)             // Get value from context
```

#### Modifying Context (Immutable Pattern)

All modification methods return a **new** `Request` instance:

```go
// Single value
req = req.WithValue(UserIDKey, claims.UserID)

// Multiple values (more efficient)
req = req.WithValues(map[any]any{
    UserIDKey: claims.UserID,
    RoleKey:   claims.Role,
    TenantKey: claims.TenantID,
})

// Custom context
ctx := context.WithTimeout(req.Context(), 5*time.Second)
req = req.WithContext(ctx)
```

#### Advanced: Access Underlying Request

```go
// Get *http.Request with updated context
httpReq := req.HTTPRequest()

// Get original *http.Request (rarely needed)
originalReq := req.RawHTTPRequest()
```

### Response API

The `glib.Response` struct provides flexible response handling:

```go
type Response struct {
    Payload    any           // Response data (will be JSON-encoded)
    Err        error         // Error to return (uses glib error handling)
    HTTPStatus int           // Override status code (0 = auto-detect)
    headers    http.Header   // Response headers (use Header() to access)
}
```

#### Creating Responses

```go
// Error response
return glib.Response{
    Err: errs.NewUnauthorized().Msg("invalid token").Err(),
}

// Success response with payload
return glib.Response{
    Payload: user,
}

// Custom status code
return glib.Response{
    Payload:    result,
    HTTPStatus: http.StatusCreated,
}

// With headers
resp := next(req)
resp.Header().Set("X-Custom-Header", "value")
resp.Header().Add("Set-Cookie", cookieValue)
return resp
```

#### Modifying Headers

```go
resp := next(req)

// Set/overwrite header
resp.Header().Set("X-User-ID", userID)

// Add header (allows duplicates)
resp.Header().Add("Set-Cookie", cookie1)
resp.Header().Add("Set-Cookie", cookie2)

// Delete header
resp.Header().Del("X-Sensitive-Data")

return resp
```

### Common Patterns

#### 1. Authentication

```go
func Auth(jwtService *JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        token := req.Header("Authorization")
        if len(token) > 7 && token[:7] == "Bearer " {
            token = token[7:]
        }
        
        claims, err := jwtService.Validate(token)
        if err != nil {
            return glib.Response{Err: err}
        }
        
        req = req.WithValue(UserKey, claims.UserID)
        return next(req)
    }
}
```

#### 2. Rate Limiting

```go
func RateLimit(limiter *RateLimiter) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        key := req.RemoteAddr()
        
        allowed, remaining := limiter.Allow(key)
        if !allowed {
            return glib.Response{
                Err: errs.B().Code(errs.ResourceExhausted).
                    Msg("rate limit exceeded").Err(),
            }
        }
        
        resp := next(req)
        resp.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
        return resp
    }
}
```

#### 3. Request ID / Tracing

```go
func RequestID() func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        requestID := req.Header("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        req = req.WithValue(RequestIDKey, requestID)
        
        resp := next(req)
        resp.Header().Set("X-Request-ID", requestID)
        return resp
    }
}
```

#### 4. Timeout

```go
func Timeout(duration time.Duration) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        ctx, cancel := context.WithTimeout(req.Context(), duration)
        defer cancel()
        
        req = req.WithContext(ctx)
        return next(req)
    }
}
```

#### 5. CORS

```go
func CORS(origins []string) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        origin := req.Header("Origin")
        
        // Check if origin is allowed
        allowed := false
        for _, o := range origins {
            if o == "*" || o == origin {
                allowed = true
                break
            }
        }
        
        if !allowed {
            return glib.Response{
                Err: errs.NewForbidden().Msg("origin not allowed").Err(),
            }
        }
        
        resp := next(req)
        resp.Header().Set("Access-Control-Allow-Origin", origin)
        resp.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
        resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        return resp
    }
}
```

## Standard Middleware (Backward Compatible)

### Signature

```go
func(http.Handler) http.Handler
```

### Example

```go
// @Middleware name=logger target=all order=1
func Logger() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Pre-processing
            log.Printf("Request: %s %s", r.Method, r.URL.Path)
            
            // Call next
            next.ServeHTTP(w, r)
            
            // Post-processing (limited - response already written)
            log.Printf("Duration: %v", time.Since(start))
        })
    }
}
```

### Limitations

Standard middleware has limited post-processing capabilities because:
- Response is already written by the time control returns
- Cannot modify response payload or status code after handler executes
- Cannot access structured response data

**Recommendation:** Use glib-style middleware for any scenario requiring post-processing.

## Middleware Registration

### Annotation

```go
// @Middleware name=<name> target=<target> order=<order>
```

Parameters:
- `name` (required): Unique identifier for the middleware
- `target` (optional): Target group (default: "all")
- `order` (optional): Execution order, lower runs first (default: 100)

### Example

```go
// @Middleware name=auth target=protected order=10
func Auth(jwtService *JWTService) func(glib.Request, glib.Next) glib.Response {
    // ...
}

// @Middleware name=cors target=api order=5
func CORS() func(glib.Request, glib.Next) glib.Response {
    // ...
}
```

## Dependency Injection

Middleware factories can request dependencies just like providers:

```go
// @Middleware name=auth
func Auth(
    jwtService *JWTService,       // Singleton dependency
    logger *Logger,                // Transient dependency
    config *Config,                // Config dependency
) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        // Use dependencies
        logger.Info("Auth middleware called")
        // ...
    }
}
```

## Code Generation

The generator automatically:

1. **Detects signature type** by analyzing return type
2. **Wraps glib middleware** into standard `http.Handler` middleware
3. **Injects dependencies** into middleware factory functions
4. **Stores in `MiddlewareContainer`** for use in route registration

### Generated Code Example

```go
func (c *App) initMiddleware() error {
    // Glib-style middleware gets wrapped
    c.middleware.AuthMiddleware = wrapGlibMiddleware(
        middleware.Auth(c.JWTService),
    )
    
    // Standard middleware used directly
    c.middleware.LoggerMiddleware = middleware.Logger()
    
    return nil
}

func wrapGlibMiddleware(mw func(glib.Request, glib.Next) glib.Response) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            req := glib.NewRequest(r)
            
            // Track if handler wrote response
            capture := &responseCapture{ResponseWriter: w}
            
            nextFn := func(req glib.Request) glib.Response {
                next.ServeHTTP(capture, req.HTTPRequest())
                return glib.Response{}
            }
            
            resp := mw(req, nextFn)
            
            // Write response if not already written
            if !capture.handlerWrote {
                // Write headers, status, payload or error
            }
        })
    }
}
```

## CLI Commands

### Create Middleware

```bash
glib make middleware auth
```

Generates `middleware/auth.go` with glib-style signature:

```go
package middleware

import "github.com/azizndao/glib"

// @Middleware auth
func Auth() func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        // TODO: Pre-processing logic
        
        resp := next(req)
        
        // TODO: Post-processing logic
        
        return resp
    }
}
```

The file includes a commented alternative showing the standard signature.

## Migration Guide

### From Standard to Glib-Style

**Before (Standard):**
```go
func Auth(jwtService *JWTService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                glib.WriteError(w, errs.NewUnauthorized().Msg("missing token").Err())
                return
            }
            
            claims, err := jwtService.ValidateToken(token)
            if err != nil {
                glib.WriteError(w, err)
                return
            }
            
            ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**After (Glib-style):**
```go
func Auth(jwtService *JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        token := req.Header("Authorization")
        if token == "" {
            return glib.Response{
                Err: errs.NewUnauthorized().Msg("missing token").Err(),
            }
        }
        
        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            return glib.Response{Err: err}
        }
        
        req = req.WithValue(UserIDKey, claims.UserID)
        
        resp := next(req)
        resp.Header().Set("X-User-ID", claims.UserID.String())
        return resp
    }
}
```

### Benefits of Migration

1. **Cleaner code**: No need for `http.ResponseWriter` manipulation
2. **Type safety**: Structured `Response` type instead of raw HTTP writes
3. **Post-processing**: Full control over response after handler executes
4. **Immutability**: Clear request modification pattern
5. **Helper methods**: `WithValues()`, `WithValue()` simplify context updates

## Best Practices

1. **Use glib-style for new middleware** - Better API and more capabilities
2. **Keep middleware focused** - One responsibility per middleware
3. **Use immutable pattern** - Always reassign: `req = req.WithValue(...)`
4. **Leverage helpers** - Use `WithValues()` for multiple context values
5. **Order matters** - Lower order numbers run first (auth before rate limiting)
6. **Early returns** - Return error responses immediately, don't call `next()`
7. **Post-process selectively** - Only call `next()` and modify response when needed
8. **Test independently** - Middleware should be testable without full app setup

## Testing

### Unit Test Example

```go
func TestAuthMiddleware(t *testing.T) {
    jwtService := &mockJWTService{}
    mw := Auth(jwtService)
    
    t.Run("missing token", func(t *testing.T) {
        req := glib.NewRequest(httptest.NewRequest("GET", "/", nil))
        
        resp := mw(req, func(req glib.Request) glib.Response {
            t.Fatal("next should not be called")
            return glib.Response{}
        })
        
        assert.NotNil(t, resp.Err)
        assert.Contains(t, resp.Err.Error(), "missing token")
    })
    
    t.Run("valid token", func(t *testing.T) {
        httpReq := httptest.NewRequest("GET", "/", nil)
        httpReq.Header.Set("Authorization", "Bearer valid-token")
        req := glib.NewRequest(httpReq)
        
        jwtService.claims = &Claims{UserID: "123"}
        
        nextCalled := false
        resp := mw(req, func(req glib.Request) glib.Response {
            nextCalled = true
            assert.Equal(t, "123", req.Value(UserIDKey))
            return glib.Response{Payload: "success"}
        })
        
        assert.True(t, nextCalled)
        assert.Nil(t, resp.Err)
        assert.Equal(t, "success", resp.Payload)
    })
}
```

## See Also

- [02-HANDLERS.md](./02-HANDLERS.md) - Handler request/response patterns
- [04-ERROR-HANDLING.md](./04-ERROR-HANDLING.md) - Error response format
- [03-CODE-GENERATION.md](./03-CODE-GENERATION.md) - How middleware is generated
