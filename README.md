# glib

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.25-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**glib is an opinionated HTTP framework for Go** that provides a modern, production-ready foundation for building REST APIs with minimal boilerplate.

## Philosophy

glib was created with specific design principles that prioritize developer experience and code clarity:

- **Ctx-based middleware**: Uses `*Ctx` instead of `http.Handler` for cleaner composition and richer APIs
- **Builder/fluent pattern**: Chainable method calls for elegant request handling
- **Integrated validation**: Built-in `go-playground/validator` with i18n support out of the box
- **Structured errors**: Proper HTTP status codes with consistent JSON error responses
- **Integrated logger**: Access logger from anywhere via `c.Logger()` for consistent, centralized logging
- **Rich context helpers**: 30+ utility methods to minimize boilerplate
- **Environment-driven**: All configuration via environment variables for 12-factor app compliance

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Environment Configuration](#environment-configuration)
- [API Reference](#api-reference)
  - [Server Creation](#server-creation)
  - [Server Methods](#server-methods)
  - [Router Methods](#router-methods)
  - [Context Methods](#context-methods)
  - [Middleware](#middleware)
  - [Error Handling](#error-handling)
  - [Validation](#validation)
  - [Logging](#logging)
- [Advanced Usage](#advanced-usage)
- [Requirements](#requirements)
- [License](#license)

## Features

- **Clean API**: Intuitive routing interface with fluent/builder pattern
- **Enhanced HTTP routing**: Built on Go 1.22+ `net/http` improvements with enhanced pattern matching
- **Request validation**: Integrated `go-playground/validator` with struct tags
- **i18n support**: Multi-language validation error messages (auto-detect from `Accept-Language`)
- **Integrated logging**: Access logger via `c.Logger()` in any handler/middleware for consistent logging
- **Colorful dev logging**: Beautiful, configurable request logging with ANSI colors for development
- **Structured production logging**: JSON logging with slog for production environments
- **Error handling**: Graceful error handling with structured error responses
- **Middleware support**: Ctx-based middleware with built-in implementations:
  - **RealIP**: Extract real client IP from proxy headers
  - **RequestID**: Generate unique request IDs for tracing
  - **Recovery**: Panic recovery with stack traces
  - **Logger**: Request/response logging (dev and production modes)
  - **Compress**: Automatic gzip/deflate compression
  - **BodyLimit**: Request body size limits (prevent DoS)
  - **RateLimit**: Configurable rate limiting per IP or custom key
  - **CORS**: Cross-origin resource sharing
  - **Timeout**: Request timeout handling
  - **Validation**: Optional validation middleware (enabled when locales configured)
- **Route groups**: Organize routes with prefixes and group-specific middleware
- **Request tracking**: Built-in request ID generation and tracking
- **Compression**: Automatic gzip compression for responses
- **Security**: Body size limits, CORS, secure cookie handling
- **Rich context helpers**: 30+ utility methods for requests, responses, validation, cookies
- **Type safety**: Full type safety with Go's type system
- **Graceful shutdown**: Automatic signal handling (SIGINT/SIGTERM) with configurable timeout
- **Production ready**: Battle-tested with comprehensive error handling

## Installation

```bash
go get github.com/azizndao/glib
```

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/azizndao/glib"
    "github.com/azizndao/glib/router"
)

func main() {
    // Create server - all configuration loaded from environment variables
    // See Environment Configuration section below for available options
    server := glib.New(glib.Config{})

    // Get the router to register routes
    r := server.Router()

    // Define routes
    r.Get("/hello", func(c *router.Ctx) error {
        return c.JSON(map[string]string{"message": "Hello World"})
    })

    r.Get("/hello/{name}", func(c *router.Ctx) error {
        return c.JSON(map[string]string{
            "message": fmt.Sprintf("Hello %s", c.PathValue("name")),
            "query":   c.Query("q"),
        })
    })

    // Start server with automatic graceful shutdown on SIGINT/SIGTERM
    server.Logger().Info("Starting server", "address", server.Address())
    if err := server.ListenWithGracefulShutdown(); err != nil {
        server.Logger().Error(err)
    }
}
```

Run the server:

```bash
# Set environment variables (or create .env file)
export HOST=localhost
export PORT=8080
export IS_DEBUG=true

# Run the server
go run main.go
```

## Environment Configuration

glib is fully configurable via environment variables. Create a `.env` file in your project root:

```env
# Server Configuration
IS_DEBUG=false              # Debug mode (enables colored DevMode logging)

# Server settings
HOST=localhost              # Server host
PORT=8080                   # Server port

# Timeouts (Go duration format: 10s, 1m, 1h30m)
READ_TIMEOUT=10s            # Maximum duration for reading request
WRITE_TIMEOUT=10s           # Maximum duration for writing response
IDLE_TIMEOUT=120s           # Maximum idle time between requests
SHUTDOWN_TIMEOUT=30s        # Maximum time to wait for graceful shutdown

# Middleware enable/disable (true/false, 1/0, yes/no, on/off)
ENABLE_REAL_IP=true         # Extract real client IP from proxy headers
ENABLE_REQUEST_ID=true      # Generate unique request IDs
ENABLE_RECOVERY=true        # Panic recovery with stack traces
ENABLE_LOGGER=true          # Request/response logging
ENABLE_COMPRESS=true        # Gzip/deflate compression
ENABLE_CORS=true            # CORS support
ENABLE_RATE_LIMIT=true      # Rate limiting

# CORS Configuration
CORS_ALLOWED_ORIGINS=*                              # Comma-separated origins
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type,Accept
CORS_EXPOSED_HEADERS=                               # Optional: headers browsers can access
CORS_ALLOW_CREDENTIALS=false                        # Allow cookies/credentials
CORS_MAX_AGE=24h                                    # Preflight cache duration

# Body limit (in bytes, e.g., 4194304 = 4MB)
BODY_LIMIT=5242880          # 5MB

# Rate limiting
RATE_LIMIT_MAX=100          # Maximum requests per window
RATE_LIMIT_WINDOW=1m        # Time window for rate limiting

# Logger configuration (format options only apply when IS_DEBUG=true)
LOGGER_FORMAT=default       # Options: default, combined, short, tiny
LOGGER_TIME_FORMAT=15:04:05 # Go time layout
```

Copy `.env.example` from the repository to get started.

## API Reference

### Server Creation

```go
import (
    "github.com/azizndao/glib"
    "github.com/go-playground/locales/fr"
    "github.com/go-playground/locales/es"
    frt "github.com/go-playground/validator/v10/translations/fr"
    est "github.com/go-playground/validator/v10/translations/es"
)

// Create server with default configuration (loads from environment variables)
server := glib.New(glib.Config{})

// Create server with validation locales for i18n error messages
server := glib.New(glib.Config{
    Locales: []glib.LocaleConfig{
        glib.Locale(fr.New(), frt.RegisterDefaultTranslations),
        glib.Locale(es.New(), est.RegisterDefaultTranslations),
    },
})

// Access the router
r := server.Router()

// Access the logger
logger := server.Logger()

// Get server address
addr := server.Address() // Returns "host:port"
```

### Server Methods

```go
// Start HTTP server with graceful shutdown (recommended)
err := server.ListenWithGracefulShutdown()

// Start HTTP server without graceful shutdown
err := server.Listen()

// Start HTTPS server with graceful shutdown
err := server.ListenTLSWithGracefulShutdown(certFile, keyFile)

// Start HTTPS server without graceful shutdown
err := server.ListenTLS(certFile, keyFile)

// Manually shutdown the server
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err := server.Shutdown(ctx)

// Register custom rate limit stores for cleanup
server.RegisterStore(redisStore)
```

### Router Methods

#### HTTP Methods

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

#### Route Groups

```go
// Create a group with prefix
api := router.Group("/api")
api.Get("/users", getUsersHandler)
api.Post("/users", createUserHandler)

// Groups with middleware
admin := router.Group("/admin", authMiddleware, adminMiddleware)
admin.Get("/dashboard", dashboardHandler)

// Nested groups
v1 := router.Group("/api/v1")
users := v1.Group("/users")
users.Get("/", listUsers)
users.Get("/{id}", getUser)
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
func handler(c *router.Ctx) error {
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

    // Logger - access the application logger
    c.Logger().Info("Processing request", "user_id", id)
    c.Logger().Error(err, "operation", "parse_body")

    return nil
}
```

#### Response Helpers

```go
func handler(c *router.Ctx) error {
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

All middleware in glib uses the `*Ctx` interface, providing a cleaner and more powerful API.

**Middleware signature:** `func(router.Handler) router.Handler` where `Handler` is `func(*Ctx) error`

When using `glib.New()`, middleware are **automatically loaded and configured from environment variables**. You can disable individual middleware by setting their corresponding `ENABLE_*` environment variable to `false`.

#### Built-in Middleware

```go
import (
    "github.com/azizndao/glib/middleware"
    "github.com/azizndao/glib/ratelimit"
    "github.com/azizndao/glib/router"
)

// With glib.New(), middleware are automatically loaded from environment variables
// No manual router.Use() calls needed unless you want custom configuration

// Request ID middleware - generates unique request IDs (auto-enabled with ENABLE_REQUEST_ID=true)
// Access request ID in handlers
func handler(c *router.Ctx) error {
    requestID := middleware.GetRequestID(c)
    return c.JSON(map[string]string{"request_id": requestID})
}

// === Manual Middleware Configuration Examples ===
// These are only needed if you're NOT using glib.New() or need custom config

// Logger middleware (auto-enabled with ENABLE_LOGGER=true)
// Configuration loaded from environment variables:
//   - IS_DEBUG: determines logging mode (false=JSON/structured, true=colorful console)
//   - LOGGER_FORMAT: default, combined, short, tiny (only for IS_DEBUG=true)
//   - LOGGER_TIME_FORMAT: Go time layout string (only for IS_DEBUG=true)
r.Use(middleware.Logger())

// Logger with custom programmatic format (overrides env vars)
r.Use(middleware.Logger(middleware.LoggerConfig{
    Format: middleware.LogFormatTiny, // Minimal format
}))

// Structured logging with slog (recommended for production)
r.Use(middleware.Logger(middleware.LoggerConfig{
    UseStructuredLogging: true,
    Logger:               slog.Default(),
    LogLevel:             slog.LevelInfo,
}))

// Recovery middleware - panic recovery (auto-enabled with ENABLE_RECOVERY=true)
// Stack traces are always included in panic logs for debugging
r.Use(middleware.Recovery())

// Compression middleware - gzip/deflate compression (auto-enabled with ENABLE_COMPRESS=true)
r.Use(middleware.Compress())

// Compression with custom level
r.Use(middleware.Compress(middleware.CompressConfig{
    Level: gzip.BestCompression,
}))

// Body size limit middleware - prevent DoS attacks (configured via BODY_LIMIT env var)
r.Use(middleware.BodyLimit())

// Body limit with custom size
r.Use(middleware.BodyLimit(middleware.BodyLimitConfig{
    MaxSize: 10 * middleware.MB, // 10MB
}))

// Rate limiting middleware - prevent abuse (auto-enabled with ENABLE_RATE_LIMIT=true)
r.Use(ratelimit.RateLimit())

// Rate limiting with custom config
r.Use(ratelimit.RateLimit(ratelimit.Config{
    Max:    50,
    Window: time.Minute,
    KeyGenerator: func(c *router.Ctx) string {
        // Rate limit by user ID if authenticated
        if userID := c.GetValue("userID"); userID != nil {
            return userID.(string)
        }
        return c.IP()
    },
}))

// CORS middleware - cross-origin resource sharing (auto-enabled with ENABLE_CORS=true)
// Configuration loaded from environment variables:
//   - CORS_ALLOWED_ORIGINS: comma-separated list (default: "*")
//   - CORS_ALLOWED_METHODS: comma-separated list (default: "GET,POST,PUT,PATCH,DELETE,OPTIONS")
//   - CORS_ALLOWED_HEADERS: comma-separated list
//   - CORS_EXPOSED_HEADERS: comma-separated list (optional)
//   - CORS_ALLOW_CREDENTIALS: boolean (default: false)
//   - CORS_MAX_AGE: duration (default: "24h")
r.Use(middleware.CORS())

// CORS with custom programmatic config (overrides env vars)
r.Use(middleware.CORS(middleware.CORSConfig{
    AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    ExposedHeaders:   []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           24 * time.Hour,
}))

// Timeout middleware - request timeout handling
r.Use(middleware.Timeout())

// Timeout with custom duration
r.Use(middleware.Timeout(middleware.TimeoutConfig{
    Timeout: 10 * time.Second,
}))

// RealIP middleware - extract real client IP from proxy headers (auto-enabled with ENABLE_REAL_IP=true)
r.Use(middleware.RealIP()) // Trusts common private networks by default

// RealIP with custom trusted proxies
r.Use(middleware.RealIP(middleware.RealIPConfig{
    TrustedProxies: []string{"10.0.0.0/8"}, // Only trust this network
    Headers:        []string{"CF-Connecting-IP", "X-Forwarded-For"},
}))
```

#### Custom Middleware

Middleware works directly with the `*router.Ctx` interface for cleaner composition:

```go
import "github.com/azizndao/glib/router"

// Basic middleware template
func customMiddleware(next router.Handler) router.Handler {
    return func(c *router.Ctx) error {
        // Before request - access to full Ctx API including logger
        start := time.Now()
        userID := c.Get("X-User-ID")

        c.Logger().Debug("Request started", "user_id", userID, "path", c.Path())

        // Execute next handler
        err := next(c)

        // After request - log with context
        duration := time.Since(start)
        c.Logger().Info("Request completed",
            "user_id", userID,
            "duration", duration,
            "status", c.statusCode,
        )

        return err
    }
}

// Apply custom middleware
r := server.Router()
r.Use(customMiddleware)

// Authentication middleware example
func authMiddleware(next router.Handler) router.Handler {
    return func(c *router.Ctx) error {
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
```

### Error Handling

glib provides structured error handling with built-in error types that return appropriate HTTP status codes and JSON responses:

```go
import "github.com/azizndao/glib/errors"

func handler(c *router.Ctx) error {
    user, err := findUser(id)
    if err != nil {
        return errors.NotFound("User not found", err)
    }

    if !user.IsActive {
        return errors.Forbidden("User is inactive", nil)
    }

    return c.Status(200).JSON(user)
}
```

#### Available Error Helpers

```go
import "github.com/azizndao/glib/errors"

// Available error helpers:
errors.BadRequest(data, internal)           // 400
errors.Unauthorized(data, internal)         // 401
errors.Forbidden(data, internal)            // 403
errors.NotFound(data, internal)             // 404
errors.Conflict(data, internal)             // 409
errors.Gone(data, internal)                 // 410
errors.UnprocessableEntity(data, internal)  // 422
errors.InternalServerError(data, internal)  // 500

// Standard errors are automatically converted to 500 responses
return fmt.Errorf("something went wrong") // Returns 500 with {"code": 500, "data": "Server Error"}
```

### Validation

glib provides powerful request validation with multi-language support using `go-playground/validator`.

#### Setup Validation

```go
import (
    "github.com/azizndao/glib"
    "github.com/go-playground/locales/fr"
    "github.com/go-playground/locales/es"
    frt "github.com/go-playground/validator/v10/translations/fr"
    est "github.com/go-playground/validator/v10/translations/es"
)

// Create server with validation locales
server := glib.New(glib.Config{
    Locales: []glib.LocaleConfig{
        glib.Locale(fr.New(), frt.RegisterDefaultTranslations),
        glib.Locale(es.New(), est.RegisterDefaultTranslations),
    },
})
```

#### Using Validation

```go
import "github.com/azizndao/glib/router"

type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
    Age      int    `json:"age" validate:"required,gte=18"`
}

func createUser(c *router.Ctx) error {
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

glib includes comprehensive logging with support for both development (colored) and production (JSON) modes:

#### Application Logging

Access the logger from any route handler or middleware using `c.Logger()`:

```go
func handler(c *router.Ctx) error {
    // Log informational messages
    c.Logger().Info("Processing user request",
        "user_id", c.PathValue("id"),
        "action", "update",
    )

    // Log errors with structured data
    if err := doSomething(); err != nil {
        c.Logger().Error(err,
            "operation", "database_query",
            "user_id", c.PathValue("id"),
        )
        return err
    }

    // Log warnings
    c.Logger().Warn("Deprecated API called", "endpoint", c.Path())

    // Log debug information
    c.Logger().Debug("Cache hit", "key", cacheKey)

    return c.JSON(map[string]string{"status": "ok"})
}
```

#### Request Logging

The logger middleware automatically logs all requests and responses:

```go
// Available log formats (for IS_DEBUG=true mode):
// - LogFormatDefault: Standard format with all details (default)
// - LogFormatTiny: Minimal format (timestamp, method, status, duration)
// - LogFormatShort: Short format (timestamp, method, path, status, duration)
// - LogFormatCombined: Combined format with user agent

// Custom format via environment variables:
// LOGGER_FORMAT=tiny
// LOGGER_TIME_FORMAT=15:04:05

// Or programmatic configuration:
r.Use(middleware.Logger(middleware.LoggerConfig{
    Format: middleware.LogFormatTiny,
}))
```

The logger respects the `IS_DEBUG` environment variable:
- `IS_DEBUG=true`: Colorful console logging with DevMode handler
- `IS_DEBUG=false`: Structured JSON logging for production

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
import "github.com/azizndao/glib/router"

func authMiddleware(next router.Handler) router.Handler {
    return func(c *router.Ctx) error {
        // Authenticate and set user in context
        user := authenticateUser(c)
        c.Request = c.SetValue("user", user)
        return next(c)
    }
}

func handler(c *router.Ctx) error {
    // Retrieve user from context
    user := c.GetValue("user")
    return c.Status(200).JSON(user)
}

// Access underlying context
ctx := c.Context()
```

### Rate Limiting with Redis

```go
import (
    "github.com/azizndao/glib/ratelimit"
    "github.com/redis/go-redis/v9"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create Redis store
redisStore := ratelimit.NewRedisStore(redisClient)

// Register store for cleanup on shutdown
server.RegisterStore(redisStore)

// Use Redis store for rate limiting
r.Use(ratelimit.RateLimit(ratelimit.Config{
    Max:    100,
    Window: time.Minute,
    Store:  redisStore,
}))
```

## Requirements

- Go 1.25+ (as specified in go.mod)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

Built with:
- [go-playground/validator](https://github.com/go-playground/validator) - Struct and field validation
- [samber/lo](https://github.com/samber/lo) - Functional utilities
- [joho/godotenv](https://github.com/joho/godotenv) - Environment variable loading
- Go standard library's enhanced `net/http` (Go 1.22+)
