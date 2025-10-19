# GRouter

A minimal HTTP router for Go that leverages Go 1.22+ enhanced routing features while providing a clean, intuitive interface for building web applications.

## Philosophy

**GRouter is an opinionated framework.** It was created for my personal approach to building Go APIs and reflects specific design decisions that I find valuable:

- **Ctx-based middleware**: Uses `*Ctx` instead of `http.Handler` for cleaner composition and richer APIs
- **Builder/fluent pattern**: Chainable method calls for elegant request handling
- **Integrated validation**: Built-in `go-playground/validator` with i18n support out of the box
- **Structured errors**: Proper HTTP status codes with consistent JSON error responses
- **Rich context helpers**: 30+ utility methods to minimize boilerplate

This framework prioritizes developer experience and clean code over flexibility. If you prefer a more minimalist or standard library approach, this may not be the right fit.

## Features

- **Clean API**: Intuitive routing interface with fluent/builder pattern
- **Enhanced HTTP routing**: Built on Go 1.22+ `net/http` improvements
- **Request validation**: Integrated `go-playground/validator` with struct tags
- **i18n support**: Multi-language validation error messages (auto-detect from `Accept-Language`)
- **Colorful logging**: Beautiful, configurable request logging with ANSI colors
- **Error handling**: Graceful error handling with structured error responses
- **Middleware support**: Ctx-based middleware with built-in Logger, Recovery, CORS, Timeout, RequestID, RateLimit, Compress, BodyLimit
- **Route groups**: Organize routes with prefixes and group-specific middleware
- **Request tracking**: Built-in request ID generation and tracking
- **Rate limiting**: Configurable rate limiting per IP or custom key
- **Compression**: Automatic gzip compression for responses
- **Security**: Body size limits, CORS, secure cookie handling
- **Rich context helpers**: 30+ utility methods for requests, responses, validation, cookies
- **Type safety**: Full type safety with Go's type system
- **Production ready**: Battle-tested with comprehensive error handling

## Installation

```bash
go get github.com/azizndao/grouter
```

## Quick Start

```go
package main

import (
    "fmt"
    "log/slog"
    "net/http"

    "github.com/azizndao/grouter"
    "github.com/azizndao/grouter/middleware"
)

func main() {
    router := grouter.NewRouter()

    // Add middleware
    router.Use(middleware.Logger(), middleware.Recovery())

    // Define routes
    router.Get("/hello", func(c *grouter.Ctx) error {
        return c.Status(200).JSON(map[string]string{"message": "Hello World"})
    })

    router.Get("/hello/{name}", func(c *grouter.Ctx) error {
        return c.Status(200).JSON(map[string]string{
            "message": fmt.Sprintf("Hello %s", c.PathValue("name")),
            "query":   c.Query("q"),
        })
    })

    slog.Default().Info("Server started")
    http.ListenAndServe(":8080", router.Handler())
}
```

## API Reference

### Router Creation

```go
// Create router with default options
router := grouter.NewRouter()

// Create router with custom options
router := grouter.NewRouterWithOptions(grouter.RouterOptions{
    AutoOPTIONS:           true,
    AutoHEAD:              true, 
    TrailingSlashRedirect: true,
    EnableLogging:         true,
})
```

### HTTP Methods

```go
router.Get("/path", handler)
router.Post("/path", handler)
router.Put("/path", handler)
router.Patch("/path", handler)
router.Delete("/path", handler)
router.Option("/path", handler)
router.Head("/path", handler)

// Generic method handler
router.Handle("METHOD", "/path", handler)

// Direct http.Handler registration
router.Route("/prefix", httpHandler)
```

### Route Groups

```go
// Create a group with prefix
api := router.Group("/api")
api.Get("/users", getUsersHandler)
api.Post("/users", createUserHandler)

// Groups with middleware
admin := router.Group("/admin", authMiddleware, adminMiddleware)
admin.Get("/dashboard", dashboardHandler)
```

### Context Methods

The `Ctx` type uses a builder/fluent pattern where setter methods return `*Ctx`, allowing you to chain method calls:

```go
// Chain multiple operations together
return c.Status(201).
    Set("X-Custom-Header", "value").
    Set("Location", "/users/123").
    JSON(user)
```

#### Request Data

```go
func handler(c *grouter.Ctx) error {
    // Path parameters (Go 1.22+ routing)
    id := c.PathValue("id")

    // Query parameters
    search := c.Query("search")
    page := c.QueryDefault("page", "1")
    limit, err := c.QueryInt("limit")
    price, err := c.QueryFloat("price")
    active := c.QueryBool("active")
    tags := c.QueryAll("tag") // Get all values for repeated param

    // Headers
    auth := c.Get("Authorization")
    authAlt := c.Authorization()      // Convenience method
    contentType := c.ContentType()    // Convenience method
    allHeaders := c.GetHeaders()      // Get all headers

    // Request info
    method := c.Method()
    path := c.Path()
    ip := c.IP()
    userAgent := c.UserAgent()
    baseURL := c.BaseURL()            // e.g. "https://example.com"
    scheme := c.Scheme()              // "http" or "https"
    host := c.Host()                  // "example.com"
    isSecure := c.IsSecure()          // true if HTTPS
    acceptsJSON := c.AcceptsJSON()    // Check Accept header
    acceptsHTML := c.AcceptsHTML()    // Check Accept header

    // Parse JSON body
    var user User
    if err := c.ParseBody(&user); err != nil {
        return err
    }

    // Or get raw body
    bodyBytes, err := c.Body()

    // Form data
    email := c.FormValue("email")
    file, header, err := c.FormFile("avatar")

    // Cookies
    sessionCookie, err := c.GetCookie("session")

    return nil
}
```

#### Response Helpers

```go
func handler(c *grouter.Ctx) error {
    // JSON response - chain Status() with JSON()
    return c.Status(200).JSON(map[string]string{"status": "ok"})

    // Text response - chain Status() with SendString()
    return c.Status(200).SendString("Hello World")

    // HTML response - chain Status() with HTML()
    return c.Status(200).HTML([]byte("<h1>Hello World</h1>"))

    // File response
    return c.File("/path/to/file.pdf")

    // Redirect
    return c.Redirect(302, "/new-location")

    // Chain multiple setters before response
    return c.Status(201).
        Set("Location", "/users/123").
        Set("X-Custom-Header", "value").
        JSON(user)

    // Set multiple headers at once using SetHeaders
    return c.SetHeaders(map[string]string{
        "X-Custom-Header": "value",
        "X-Request-ID":    "12345",
    }).Status(200).JSON(data)

    // Cookie management with chaining
    return c.SetCookie(&http.Cookie{
        Name:  "session",
        Value: "token123",
    }).Status(200).JSON(map[string]string{"message": "Cookie set"})

    // Clear cookie
    c.ClearCookie("old-session")
    return c.Status(200).JSON(map[string]string{"message": "Cookie cleared"})
}
```

### Middleware

All middleware in GRouter now uses the `*Ctx` interface, providing a cleaner and more powerful API.

**Middleware signature:** `func(grouter.Handler) grouter.Handler` where `Handler` is `func(*Ctx) error`

#### Built-in Middleware

```go
import (
    "github.com/azizndao/grouter/middleware"
    "github.com/azizndao/grouter/validation"
    "github.com/go-playground/locales/fr"
    "github.com/go-playground/locales/es"
    fr_translations "github.com/go-playground/validator/v10/translations/fr"
    es_translations "github.com/go-playground/validator/v10/translations/es"
)

// Request ID middleware - generates unique request IDs
router.Use(middleware.RequestID())

// Access request ID in handlers
func handler(c *grouter.Ctx) error {
    requestID := middleware.GetRequestID(c)
    return c.JSON(map[string]string{"request_id": requestID})
}

// Logger middleware - colorful request logging
router.Use(middleware.Logger())

// Logger with custom format
router.Use(middleware.LoggerTiny())    // Minimal format
router.Use(middleware.LoggerShort())   // Short format
router.Use(middleware.LoggerCombined()) // Combined format with user agent

// Recovery middleware - panic recovery
router.Use(middleware.Recovery())

// Compression middleware - gzip compression
router.Use(middleware.Compress())

// Compression with custom config
router.Use(middleware.Compress(middleware.CompressConfig{
    Level:     gzip.BestCompression,
    MinLength: 2048, // Only compress responses > 2KB
}))

// Body size limit middleware - prevent DoS attacks
router.Use(middleware.BodyLimit5MB())  // 5MB limit
router.Use(middleware.BodyLimit10MB()) // 10MB limit
router.Use(middleware.BodyLimitWithSize(20 * 1024 * 1024)) // 20MB

// Rate limiting middleware - prevent abuse
router.Use(middleware.RateLimit()) // 100 requests/minute by default

// Rate limiting with custom config
router.Use(middleware.RateLimit(middleware.RateLimitConfig{
    Max:    50,
    Window: time.Minute,
    KeyGenerator: func(c *grouter.Ctx) string {
        // Rate limit by user ID if authenticated
        if userID := c.GetValue("userID"); userID != nil {
            return userID.(string)
        }
        return c.IP()
    },
}))

// CORS middleware - cross-origin resource sharing
router.Use(middleware.CORS(middleware.DefaultCORSOptions()))

// CORS with custom options
router.Use(middleware.CORS(middleware.CORSOptions{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           24 * time.Hour,
}))

// Timeout middleware - request timeout handling
router.Use(middleware.Timeout(30 * time.Second))

// Validator middleware - request validation with i18n
router.Use(validation.Middleware(
    validation.Locale(fr.New(), fr_translations.RegisterDefaultTranslations),
    validation.Locale(es.New(), es_translations.RegisterDefaultTranslations),
))

// Combine multiple middleware - recommended production setup
router.Use(
    middleware.RequestID(),              // Request tracking
    middleware.Recovery(),               // Panic recovery
    middleware.Logger(),                 // Request logging
    middleware.Compress(),               // Response compression
    middleware.BodyLimit5MB(),           // Body size limit
    middleware.RateLimit(),              // Rate limiting
    middleware.CORS(middleware.DefaultCORSOptions()), // CORS
    validation.Middleware(...),          // Validation
)
```

#### Custom Middleware

Middleware works directly with the `*Ctx` interface for cleaner composition:

```go
// Basic middleware template
func customMiddleware(next grouter.Handler) grouter.Handler {
    return func(c *grouter.Ctx) error {
        // Before request - access to full Ctx API
        start := time.Now()
        userID := c.Get("X-User-ID")

        // Execute next handler
        err := next(c)

        // After request
        duration := time.Since(start)
        log.Printf("User %s - Request took %v", userID, duration)

        return err
    }
}

router.Use(customMiddleware)

// Authentication middleware example
func authMiddleware(next grouter.Handler) grouter.Handler {
    return func(c *grouter.Ctx) error {
        token := c.Authorization()
        if token == "" {
            return c.Status(401).JSON(map[string]string{"error": "Unauthorized"})
        }

        // Validate token and set user in context
        user, err := validateToken(token)
        if err != nil {
            return c.Status(401).JSON(map[string]string{"error": "Invalid token"})
        }

        c.Request = c.SetValue("user", user)
        return next(c)
    }
}

// Rate limiting middleware example
func rateLimiter(requestsPerMinute int) grouter.Middleware {
    // Setup rate limiter
    limiter := rate.NewLimiter(rate.Limit(requestsPerMinute), requestsPerMinute)

    return func(next grouter.Handler) grouter.Handler {
        return func(c *grouter.Ctx) error {
            if !limiter.Allow() {
                return c.Status(429).JSON(map[string]string{
                    "error": "Too many requests",
                })
            }
            return next(c)
        }
    }
}
```

### Error Handling

GRouter handlers return errors that are automatically logged:

```go
import "github.com/azizndao/grouter/errors"

func handler(c *grouter.Ctx) error {
    user, err := findUser(id)
    if err != nil {
        return errors.ErrorNotFound("User not found", err)
    }

    if !user.IsActive {
        return errors.ErrorForbidden("User is inactive", nil)
    }

    return c.Status(200).JSON(user)
}
```

GRouter provides structured error handling with built-in error types that return appropriate HTTP status codes and JSON responses:

```go
import "github.com/azizndao/grouter/errors"

// Available error helpers:
errors.ErrorBadRequest(data, internal)           // 400
errors.ErrorUnauthorized(data, internal)         // 401
errors.ErrorForbidden(data, internal)            // 403
errors.ErrorNotFound(data, internal)             // 404
errors.ErrorConflict(data, internal)             // 409
errors.ErrorGone(data, internal)                 // 410
errors.ErrorUnprocessableEntity(data, internal)  // 422
errors.ErrorInternalServerError(data, internal)  // 500

// Standard errors are automatically converted to 500 responses
return fmt.Errorf("something went wrong") // Returns 500 with {"Code": 500, "Data": "Server Error"}
```

### Validation

GRouter provides powerful request validation with multi-language support using `go-playground/validator`.

#### Setup Validator Middleware

```go
import (
    "github.com/azizndao/grouter"
    "github.com/azizndao/grouter/validation"
    "github.com/go-playground/locales/fr"
    "github.com/go-playground/locales/es"
    fr_translations "github.com/go-playground/validator/v10/translations/fr"
    es_translations "github.com/go-playground/validator/v10/translations/es"
)

// Add validator middleware with multiple languages
router.Use(validation.Middleware(
    validation.Locale(fr.New(), fr_translations.RegisterDefaultTranslations),
    validation.Locale(es.New(), es_translations.RegisterDefaultTranslations),
))
```

#### Using Validation

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
    Age      int    `json:"age" validate:"required,gte=18"`
}

func createUser(c *grouter.Ctx) error {
    var req CreateUserRequest

    // Parse and validate in one call
    // Validation errors returned in user's language from Accept-Language header
    if err := c.ValidateBody(&req); err != nil {
        return err
    }

    // req is now validated
    return c.Status(201).JSON(map[string]string{"message": "User created"})
}
```

#### Validation Responses

Validation errors are automatically returned in the user's preferred language:

**English** (`Accept-Language: en`):
```json
{
  "code": 422,
  "data": {
    "email": "email must be a valid email address",
    "password": "password must be at least 8 characters in length",
    "age": "age must be 18 or greater"
  }
}
```

**French** (`Accept-Language: fr`):
```json
{
  "code": 422,
  "data": {
    "email": "email doit être une adresse email valide",
    "password": "password doit faire au moins 8 caractères",
    "age": "age doit être 18 ou plus"
  }
}
```

**Spanish** (`Accept-Language: es`):
```json
{
  "code": 422,
  "data": {
    "email": "email debe ser una dirección de correo electrónico válida",
    "password": "password debe tener al menos 8 caracteres",
    "age": "age debe ser 18 o más"
  }
}
```

#### Validation Tags

Supports all standard validator tags:
- `required` - Field is required
- `email` - Valid email address
- `min=n` - Minimum length/value
- `max=n` - Maximum length/value
- `gte=n`, `lte=n`, `gt=n`, `lt=n` - Numeric comparisons
- `oneof=red green blue` - Value must be one of the specified options
- `url`, `uri`, `uuid` - Format validation
- And many more from [go-playground/validator](https://github.com/go-playground/validator)

### Logging

GRouter includes colorful request logging:

```go
import "github.com/azizndao/grouter/middleware"

// Use default logger configuration
router.Use(middleware.Logger())

// Or use predefined logger formats
router.Use(middleware.LoggerTiny())      // Minimal format
router.Use(middleware.LoggerShort())     // Short format
router.Use(middleware.LoggerCombined())  // Combined format with user agent
```

## Advanced Usage

### Route Information

```go
// Get all registered routes
routes := router.Routes()
for _, route := range routes {
    fmt.Printf("%s %s -> %s\n", route.Method, route.Pattern, route.Group)
}
```

### Context Values

Store and retrieve values in the request context:

```go
func authMiddleware(next grouter.Handler) grouter.Handler {
    return func(c *grouter.Ctx) error {
        // Authenticate and set user in context
        user := authenticateUser(c)
        c.Request = c.SetValue("user", user)
        return next(c)
    }
}

func handler(c *grouter.Ctx) error {
    // Retrieve user from context
    user := c.GetValue("user")
    return c.Status(200).JSON(user)
}

// Access underlying context
ctx := c.Context()
```

## Requirements

- Go 1.22 or later (for enhanced routing features)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
