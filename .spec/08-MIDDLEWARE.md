# 08. Middleware System

Specification and usage guide for Glib's middleware architecture.

---

## Table of Contents

1. [Overview](#overview)
2. [Native Glib Middleware](#native-glib-middleware)
3. [Standard HTTP/Chi Middleware](#standard-httpchi-middleware)
4. [Request & Response APIs](#request--response-apis)
5. [Targeting & Route Scoping](#targeting--route-scoping)
6. [Execution Ordering](#execution-ordering)
7. [Dependency Injection in Middleware](#dependency-injection-in-middleware)

---

## Overview

Glib provides a dual middleware system:

1. **Native Glib Middleware** (`func(glib.Request, glib.Next) glib.Response`):
    - Functional, type-safe request/response interceptor.
    - Pre-processing (validating tokens, adding context values) and post-processing (modifying response headers, auditing).
    - Immutable request model.
2. **Standard Chi/HTTP Middleware** (`func(http.Handler) http.Handler`):
    - Full compatibility with existing Go HTTP middleware ecosystems (e.g. `chi/middleware`, CORS handlers, metrics).

---

## Native Glib Middleware

### Signature

```go
func([dependencies...]) func(glib.Request, glib.Next) glib.Response
```

### Complete Example

```go
package middleware

import (
    "strings"

    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
    "my-app/services"
)

type ContextKey string

const UserIDKey ContextKey = "user_id"

// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        // 1. Pre-processing: extract and validate header
        authHeader := req.Header("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            return glib.Response{
                Err: errs.NewUnauthorized().WithMessage("authorization token required"),
            }
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            return glib.Response{Err: err}
        }

        // 2. Add claims to request context (immutable pattern)
        req = req.WithValue(UserIDKey, claims.UserID)

        // 3. Call next middleware or handler
        resp := next(req)

        // 4. Post-processing: set response headers
        resp.Header().Set("X-User-ID", claims.UserID.String())

        return resp
    }
}
```

---

## Standard HTTP/Chi Middleware

Traditional `net/http` middleware is also recognized when annotated with `// @Middleware`:

```go
// @Middleware name=request_logger target=all order=1
func RequestLogger() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            log.Printf("Incoming %s %s", r.Method, r.URL.Path)
            next.ServeHTTP(w, r)
        })
    }
}
```

Global router middleware can also be applied directly in `bootstrap.go`:

```go
app.Router.Use(middleware.RequestID)
app.Router.Use(middleware.RealIP)
app.Router.Use(middleware.Recoverer)
```

---

## Request & Response APIs

### `glib.Request`

The `glib.Request` struct provides an immutable API:

```go
// Reading Request Data
method  := req.Method()              // "GET", "POST", etc.
path    := req.Path()                // "/api/v1/posts"
fullURL := req.URL()                 // Complete URL string
ip      := req.RemoteAddr()          // Client IP address
hdr     := req.Header("X-Api-Key")   // Header value or ""
page    := req.Query("page")         // Query parameter or ""
tags    := req.QuerySlice("tag")     // Multiple query parameters []string
id      := req.PathValue("id")       // Chi URL path wildcard value
ctx     := req.Context()             // Underlying context.Context
val     := req.Value(key)            // Value from context

// Modifying Context (Returns a new Request copy)
req = req.WithContext(newCtx)
req = req.WithValue("user_id", userID)
req = req.WithValues(map[any]any{
    "user_id":  userID,
    "user_role": "admin",
})
```

### `glib.Response`

The `glib.Response` struct holds handler output:

```go
type Response struct {
    Payload    any         // Handler return value (for (T, error) handlers)
    Err        error       // Error to return (short-circuits success)
    HTTPStatus int         // Override status code if non-zero
}

// Manipulating headers
resp.Header().Set("X-Custom-Header", "value")
resp.WithHeader("X-Trace-ID", traceID)
```

---

## Targeting & Route Scoping

### Tag-Based Targeting

Middleware is automatically associated with controllers and routes based on tags:

- `target=all`: Applied to all routes in the application.
- `target=api`: Applied to any controller or route tagged with `tags=api`.
- `target=protected`: Applied to any controller or route tagged with `tags=protected`.

```go
// @Middleware name=ratelimit target=api order=5
// @Middleware name=auth target=protected order=10

// Controller tagged with 'api'
// @Controller path=/api/v1/posts tags=api
type Controller struct{}

// Route automatically gets 'ratelimit' from controller tag and 'auth' from route tag
// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreateRequest) (*Post, error)
```

### Route-Level Overrides (`with=...`)

Routes can explicitly override middleware behavior:

- Specify exact middleware chain:
    ```go
    // @Route method=GET path=/metrics with=admin_auth,audit
    ```
- Bypass all middleware:
    ```go
    // @Route method=GET path=/health with=none
    ```

---

## Execution Ordering

The `order` attribute determines middleware placement in the execution stack:

- Lower numbers execute earlier in the request lifecycle (outer middleware).
- Higher numbers execute closer to the handler (inner middleware).

```
Incoming Request
    │
    ▼
┌───────────────────────────────┐
│ 1. RateLimiter (order=5)      │
│   ┌───────────────────────────┤
│   │ 2. Auth (order=10)        │
│   │   ┌───────────────────────┤
│   │   │ 3. Handler Execution  │
│   │   └───────────────────────┤
│   │ 2. Auth Post-Processing   │
│   └───────────────────────────┤
│ 1. RateLimiter Post-Processing│
└───────────────────────────────┘
    │
    ▼
Outgoing Response
```

---

## Dependency Injection in Middleware

Middleware constructor functions receive dependencies automatically injected by type from registered `@Provider` and `@Config` instances:

```go
// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService, cfg *configs.Config) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        // Use jwtService and cfg directly
        return next(req)
    }
}
```
