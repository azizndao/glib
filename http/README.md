# Glib HTTP Server

[![Go Reference](https://pkg.go.dev/badge/github.com/azizndao/glib.svg)](https://pkg.go.dev/github.com/azizndao/glib)

A fast, elegant HTTP server for Go with a Laravel-inspired API. Built on [Chi router](https://github.com/go-chi/chi) with an intuitive context-based request/response abstraction.

## Features

- **Simple & Elegant API** - Laravel-inspired context (`Ctx`) with chainable methods
- **Powerful Routing** - Built on Chi router with full pattern matching support
- **Auto Configuration** - Environment-based setup with sensible defaults
- **Middleware Stack** - Built-in middleware for logging, compression, CORS, rate limiting, and more
- **Error Handling** - Structured error responses with proper HTTP status codes
- **Request Validation** - Pluggable validator interface with i18n support
- **Graceful Shutdown** - Production-ready lifecycle management
- **Type-Safe Handlers** - Error-returning handlers for cleaner code

## Installation

```bash
go get github.com/azizndao/glib@latest
```

### Optional Dependencies

For validation support:
```bash
go get github.com/azizndao/glib/validation@latest
```

For common utilities (errors, logging, config):
```bash
go get github.com/azizndao/glib/common@latest
```

## Quick Start

### Basic Server

```go
package main

import (
    "github.com/azizndao/glib"
)

func main() {
    // Create server with auto-configuration from environment
    server := glib.New(glib.Config{})
    
    // Get router to register routes
    r := server.Router()
    
    // Register routes
    r.Get("/", func(c *glib.Ctx) error {
        return c.JSON(map[string]string{
            "message": "Hello, Glib!",
        })
    })
    
    r.Get("/users/{id}", func(c *glib.Ctx) error {
        id := c.PathValue("id")
        return c.JSON(map[string]string{
            "user_id": id,
        })
    })
    
    // Start with graceful shutdown
    if err := server.ListenWithGracefulShutdown(); err != nil {
        server.Logger().Error(err)
    }
}
```

### With Validation

```go
package main

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/validation"
    "github.com/go-playground/locales/en"
    ent "github.com/go-playground/validator/v10/translations/en"
)

type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
}

func main() {
    // Create validator with i18n support
    validator := validation.New(validation.Config{
        DefaultLocale:     "en",
        UseJSONFieldNames: true,
        Locales: []validation.LocaleConfig{
            validation.Locale(en.New(), ent.RegisterDefaultTranslations),
        },
    })
    
    // Create server with validator
    server := glib.New(glib.Config{
        Validator: validator,
    })
    
    r := server.Router()
    
    r.Post("/users", func(c *glib.Ctx) error {
        var req CreateUserRequest
        
        // Parse and validate in one call
        // Uses Accept-Language header for error messages
        if err := c.ValidateBody(&req); err != nil {
            return err // Returns structured 400 error
        }
        
        // Business logic here...
        
        return c.Created(req)
    })
    
    server.ListenWithGracefulShutdown()
}
```

## Configuration

Glib uses environment variables for configuration. Create a `.env` file or set these in your environment:

### Server Configuration

```bash
# Server
HOST=localhost
PORT=8080
READ_TIMEOUT=10s
WRITE_TIMEOUT=10s
IDLE_TIMEOUT=120s
SHUTDOWN_TIMEOUT=30s

# Logging
LOG_LEVEL=info              # debug, info, warn, error
LOG_FORMAT=json             # json, text
IS_DEBUG=false              # Enable debug mode

# Middleware (all default to true)
ENABLE_REAL_IP=true
ENABLE_REQUEST_ID=true
ENABLE_LOGGER=true
ENABLE_RECOVERY=true
ENABLE_COMPRESSION=true
ENABLE_BODY_LIMIT=true
ENABLE_CORS=true

# Body Limit
MAX_BODY_SIZE=10485760      # 10MB in bytes

# Compression
COMPRESSION_LEVEL=5         # 1-9 (5 is default)

# CORS
CORS_ALLOWED_ORIGINS=*
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type,X-CSRF-Token
CORS_EXPOSED_HEADERS=Link
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=300

# Rate Limiting (optional)
ENABLE_RATE_LIMIT=false
RATE_LIMIT_MAX=100          # Max requests
RATE_LIMIT_WINDOW=60s       # Per time window
```

## Core API Reference

### Server

The main `Server` provides lifecycle management and configuration.

```go
// Create server
server := glib.New(glib.Config{
    Validator: validator, // Optional, uses NoOpValidator if nil
})

// Get router for route registration
router := server.Router()

// Get logger
logger := server.Logger()

// Start server (blocks until shutdown)
server.Listen()

// Start with TLS
server.ListenTLS("cert.pem", "key.pem")

// Recommended: Start with graceful shutdown (handles SIGINT/SIGTERM)
server.ListenWithGracefulShutdown()

// Manual shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

### Router

The `Router` interface provides routing and middleware management (powered by Chi).

#### HTTP Methods

```go
r.Get("/path", handler)       // GET
r.Post("/path", handler)      // POST
r.Put("/path", handler)       // PUT
r.Patch("/path", handler)     // PATCH
r.Delete("/path", handler)    // DELETE
r.Options("/path", handler)   // OPTIONS
r.Head("/path", handler)      // HEAD
r.Connect("/path", handler)   // CONNECT
r.Trace("/path", handler)     // TRACE

// Match all methods
r.HandleFunc("/path", handler)
```

#### Route Parameters

```go
// Path parameters (Chi style)
r.Get("/users/{id}", func(c *glib.Ctx) error {
    id := c.PathValue("id")
    return c.JSON(map[string]string{"user_id": id})
})

// Regex patterns
r.Get("/posts/{slug:[a-z-]+}", handler)

// Optional parameters
r.Get("/files/{*filepath}", handler)
```

#### Route Groups

```go
// Group with shared prefix
r.Route("/api/v1", func(api glib.Router) {
    api.Get("/users", listUsers)
    api.Post("/users", createUser)
    
    // Nested groups
    api.Route("/users/{id}", func(user glib.Router) {
        user.Get("/", getUser)
        user.Put("/", updateUser)
        user.Delete("/", deleteUser)
    })
})

// Group with shared middleware
r.Group(func(protected glib.Router) {
    protected.Use(authMiddleware)
    
    protected.Get("/profile", getProfile)
    protected.Post("/logout", logout)
})
```

#### Middleware

```go
// Global middleware (applied to all routes)
r.Use(loggingMiddleware, authMiddleware)

// Route-specific middleware
r.With(adminMiddleware).Get("/admin", adminHandler)

// Use Chi middleware directly
import chimiddleware "github.com/go-chi/chi/v5/middleware"
r.UseHTTP(chimiddleware.Timeout(60 * time.Second))
```

#### Custom Handlers

```go
// Mount external http.Handler
r.Mount("/debug", middleware.Profiler())

// Custom 404
r.NotFound(func(c *glib.Ctx) error {
    return errors.NotFound("Page not found", nil)
})

// Custom 405
r.MethodNotAllowed(func(c *glib.Ctx) error {
    return errors.MethodNotAllowed("Method not allowed", nil)
})
```

### Context (`Ctx`)

The `Ctx` provides easy access to request data and response helpers.

#### Request Data

```go
// Path parameters
id := c.PathValue("id")
userID, err := c.PathInt("userID")
price, err := c.PathFloat("price")

// Query parameters
search := c.Query("q")
page := c.QueryIntDefault("page", 1)
limit := c.QueryIntDefault("limit", 20)
active := c.QueryBool("active")
tags := c.QueryArray("tags") // ?tags=go&tags=web

// Request body
var req CreateUserRequest
err := c.ParseBody(&req)      // Parse JSON

// Parse and validate
err := c.ValidateBody(&req)   // Parse + validate

// Generic helper
req, err := glib.ValidateBody[CreateUserRequest](c)

// Raw body
body, err := c.Body()         // []byte (cached)

// Form data
name := c.FormValue("name")
file, header, err := c.FormFile("upload")

// Headers
contentType := c.ContentType()
auth := c.Authorization()
token := c.BearerToken()      // Extracts from "Bearer <token>"
userAgent := c.UserAgent()
custom := c.Get("X-Custom-Header")

// Cookies
cookie, err := c.GetCookie("session")
value := c.GetCookieDefault("theme", "light")

// Request info
method := c.Method()          // GET, POST, etc.
path := c.Path()              // /users/123
ip := c.IP()                  // Client IP (respects X-Forwarded-For)
scheme := c.Scheme()          // http or https
host := c.Host()              // example.com
baseURL := c.BaseURL()        // https://example.com
url := c.URL()                // *url.URL

// Context values
c.SetValue("user", user)
user := c.GetValue("user")

// Logger
c.Logger().Info("Processing request", "user_id", userID)
```

#### Response Helpers

```go
// JSON
return c.JSON(data)
return c.Status(201).JSON(data)

// Status shortcuts
return c.Created(data)        // 201
return c.Accepted(data)       // 202
return c.NoContent()          // 204

// Text
return c.SendString("Hello")

// HTML
return c.HTML([]byte("<h1>Hello</h1>"))

// XML
return c.XML(data)

// Files
return c.File("./public/index.html")
return c.SendFile("./file.pdf", true) // With download header
return c.Download("./report.pdf", "monthly-report.pdf")

// Redirects
return c.Redirect(302, "/login")

// Streaming
return c.Stream(func(w io.Writer) error {
    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "chunk %d\n", i)
        time.Sleep(time.Second)
    }
    return nil
})

// Server-Sent Events
return c.SSE("message", `{"data": "hello"}`)

// Headers
c.Set("X-Custom", "value")
c.SetHeaders(map[string]string{
    "X-API-Version": "1.0",
    "X-Rate-Limit": "100",
})

// Cookies
c.SetCookie(&http.Cookie{
    Name:     "session",
    Value:    "abc123",
    MaxAge:   3600,
    HttpOnly: true,
    Secure:   true,
})
c.ClearCookie("session")

// Status checks
if c.IsSuccess() { }        // 2xx
if c.IsClientError() { }    // 4xx
if c.IsServerError() { }    // 5xx

// Content negotiation
if c.AcceptsJSON() { }
if c.AcceptsHTML() { }
if c.Accepts("application/xml") { }

// Request ID (if middleware enabled)
requestID := c.GetRequestID()
c.SetRequestID("custom-id")
```

## Error Handling

Glib uses structured errors from `common/errors` package.

### Creating Errors

```go
import "github.com/azizndao/glib/common/errors"

// Predefined error types
return errors.BadRequest("Invalid input", err)
return errors.Unauthorized("Invalid credentials", nil)
return errors.Forbidden("Access denied", nil)
return errors.NotFound("User not found", nil)
return errors.MethodNotAllowed("Method not allowed", nil)
return errors.Conflict("Email already exists", nil)
return errors.UnprocessableEntity("Validation failed", validationErrors)
return errors.InternalServerError("Database error", err)

// Custom error
return errors.NewAPI(418, "I'm a teapot", nil)

// With structured data
return errors.UnprocessableEntity("Validation failed", map[string]string{
    "email": "Email is required",
    "password": "Password must be at least 8 characters",
})
```

### Error Response Format

All errors are automatically converted to JSON:

```json
{
  "code": 400,
  "message": "Invalid input",
  "data": "detailed error information"
}
```

### Handler Error Handling

```go
r.Get("/users/{id}", func(c *glib.Ctx) error {
    id := c.PathValue("id")
    
    user, err := findUser(id)
    if err != nil {
        return errors.NotFound("User not found", err)
    }
    
    return c.JSON(user)
})
// Router automatically handles error and sends JSON response
```

## Middleware

### Built-in Middleware

Glib includes a configurable middleware stack (see [Configuration](#configuration)):

- **RealIP** - Extract real client IP from proxy headers
- **RequestID** - Generate unique request IDs
- **Logger** - Request/response logging (structured or debug)
- **Recoverer** - Panic recovery
- **Compress** - GZIP/Deflate compression
- **BodyLimit** - Request body size limiting
- **RateLimit** - Rate limiting by IP
- **CORS** - Cross-origin resource sharing

### Custom Middleware

```go
func loggingMiddleware(next glib.HandleFunc) glib.HandleFunc {
    return func(c *glib.Ctx) error {
        start := time.Now()
        
        c.Logger().Info("Request started",
            "method", c.Method(),
            "path", c.Path(),
        )
        
        err := next(c)
        
        c.Logger().Info("Request completed",
            "duration", time.Since(start),
        )
        
        return err
    }
}

// Apply globally
r.Use(loggingMiddleware)

// Apply to specific routes
r.With(loggingMiddleware).Get("/users", listUsers)
```

### Middleware with Configuration

```go
func authMiddleware(requiredRole string) glib.Middleware {
    return func(next glib.HandleFunc) glib.HandleFunc {
        return func(c *glib.Ctx) error {
            token := c.BearerToken()
            if token == "" {
                return errors.Unauthorized("Missing token", nil)
            }
            
            // Verify token and check role
            user, err := verifyToken(token)
            if err != nil {
                return errors.Unauthorized("Invalid token", err)
            }
            
            if user.Role != requiredRole {
                return errors.Forbidden("Insufficient permissions", nil)
            }
            
            c.SetValue("user", user)
            return next(c)
        }
    }
}

// Usage
r.With(authMiddleware("admin")).Get("/admin", adminHandler)
```

## Validation

Glib supports pluggable validators through the `Validator` interface. The official `validation` module provides i18n support.

### Setup

```go
import (
    "github.com/azizndao/glib/validation"
    "github.com/go-playground/locales/en"
    "github.com/go-playground/locales/es"
    "github.com/go-playground/locales/fr"
    ent "github.com/go-playground/validator/v10/translations/en"
    est "github.com/go-playground/validator/v10/translations/es"
    frt "github.com/go-playground/validator/v10/translations/fr"
)

validator := validation.New(validation.Config{
    DefaultLocale:     "en",
    UseJSONFieldNames: true,
    Locales: []validation.LocaleConfig{
        validation.Locale(en.New(), ent.RegisterDefaultTranslations),
        validation.Locale(es.New(), est.RegisterDefaultTranslations),
        validation.Locale(fr.New(), frt.RegisterDefaultTranslations),
    },
})

server := glib.New(glib.Config{
    Validator: validator,
})
```

### Usage

```go
type CreateProductRequest struct {
    Name        string  `json:"name" validate:"required,min=3,max=100"`
    Description string  `json:"description" validate:"required,min=10"`
    Price       float64 `json:"price" validate:"required,gt=0"`
    Stock       int     `json:"stock" validate:"required,gte=0"`
    SKU         string  `json:"sku" validate:"required,alphanum"`
}

r.Post("/products", func(c *glib.Ctx) error {
    var req CreateProductRequest
    
    // Validates using Accept-Language header
    if err := c.ValidateBody(&req); err != nil {
        return err // Returns 400 with localized errors
    }
    
    // req is valid, proceed with business logic
    return c.Created(req)
})
```

### Custom Validator

Implement the `Validator` interface:

```go
type CustomValidator struct {
    // Your validation logic
}

func (v CustomValidator) Validate(s interface{}, locale string) error {
    // Implement validation
    return nil
}

// Use it
server := glib.New(glib.Config{
    Validator: CustomValidator{},
})
```

## Complete Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/common/errors"
    "github.com/azizndao/glib/validation"
    "github.com/go-playground/locales/en"
    ent "github.com/go-playground/validator/v10/translations/en"
)

// Models
type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
}

// Middleware
func authMiddleware(next glib.HandleFunc) glib.HandleFunc {
    return func(c *glib.Ctx) error {
        token := c.BearerToken()
        if token == "" {
            return errors.Unauthorized("Missing token", nil)
        }
        
        // Verify token (simplified)
        if token != "valid-token" {
            return errors.Unauthorized("Invalid token", nil)
        }
        
        c.SetValue("user_id", "123")
        return next(c)
    }
}

// Handlers
func listUsers(c *glib.Ctx) error {
    page := c.QueryIntDefault("page", 1)
    limit := c.QueryIntDefault("limit", 20)
    
    users := []User{
        {ID: "1", Email: "user@example.com", Name: "John Doe", CreatedAt: time.Now()},
    }
    
    return c.JSON(map[string]interface{}{
        "users": users,
        "page":  page,
        "limit": limit,
    })
}

func createUser(c *glib.Ctx) error {
    var req CreateUserRequest
    
    if err := c.ValidateBody(&req); err != nil {
        return err
    }
    
    user := User{
        ID:        "123",
        Email:     req.Email,
        Name:      req.Name,
        CreatedAt: time.Now(),
    }
    
    return c.Created(user)
}

func getUser(c *glib.Ctx) error {
    id := c.PathValue("id")
    
    user := User{
        ID:        id,
        Email:     "user@example.com",
        Name:      "John Doe",
        CreatedAt: time.Now(),
    }
    
    return c.JSON(user)
}

func updateUser(c *glib.Ctx) error {
    id := c.PathValue("id")
    
    var req CreateUserRequest
    if err := c.ValidateBody(&req); err != nil {
        return err
    }
    
    user := User{
        ID:        id,
        Email:     req.Email,
        Name:      req.Name,
        CreatedAt: time.Now(),
    }
    
    return c.JSON(user)
}

func deleteUser(c *glib.Ctx) error {
    id := c.PathValue("id")
    
    c.Logger().Info("Deleting user", "id", id)
    
    return c.NoContent()
}

func main() {
    // Setup validator
    validator := validation.New(validation.Config{
        DefaultLocale:     "en",
        UseJSONFieldNames: true,
        Locales: []validation.LocaleConfig{
            validation.Locale(en.New(), ent.RegisterDefaultTranslations),
        },
    })
    
    // Create server
    server := glib.New(glib.Config{
        Validator: validator,
    })
    
    r := server.Router()
    
    // Public routes
    r.Get("/health", func(c *glib.Ctx) error {
        return c.JSON(map[string]string{"status": "ok"})
    })
    
    // API routes
    r.Route("/api/v1", func(api glib.Router) {
        // Public endpoints
        api.Get("/users", listUsers)
        api.Get("/users/{id}", getUser)
        
        // Protected endpoints
        api.Group(func(protected glib.Router) {
            protected.Use(authMiddleware)
            
            protected.Post("/users", createUser)
            protected.Put("/users/{id}", updateUser)
            protected.Delete("/users/{id}", deleteUser)
        })
    })
    
    // Custom 404
    r.NotFound(func(c *glib.Ctx) error {
        return errors.NotFound("Endpoint not found", nil)
    })
    
    // Start server
    server.Logger().Info("Starting server on " + server.Address())
    
    if err := server.ListenWithGracefulShutdown(); err != nil {
        server.Logger().Error(err)
    }
}
```

## Testing

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    
    "github.com/azizndao/glib"
)

func TestCreateUser(t *testing.T) {
    server := glib.New(glib.Config{})
    r := server.Router()
    
    r.Post("/users", createUser)
    
    body := strings.NewReader(`{"email":"test@example.com","password":"password123","name":"Test"}`)
    req := httptest.NewRequest("POST", "/users", body)
    req.Header.Set("Content-Type", "application/json")
    
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    
    if w.Code != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", w.Code)
    }
}
```

## Architecture

Glib HTTP server is built on these components:

- **Chi Router** - Fast, lightweight HTTP router
- **Context Abstraction** - Simplified request/response handling
- **Middleware Stack** - Composable request processing
- **Error System** - Structured error responses
- **Validator Interface** - Pluggable validation

### Design Decisions

1. **Context-Based** - All handlers receive `*Ctx` for consistent API
2. **Error Returns** - Handlers return errors for cleaner code
3. **Chi Foundation** - Leverages battle-tested router
4. **Environment Config** - 12-factor app methodology
5. **Minimal Core** - Optional validation/database modules

## Module Dependencies

```
http (this module)
├── common/errors     - Error handling
├── common/slog       - Structured logging
├── common/util       - Environment helpers
└── validation        - Optional request validation
```

## Related Modules

- **[common](../common)** - Common utilities (errors, logging, config)
- **[validation](../validation)** - Request validation with i18n
- **[foundation](../foundation)** - Dependency injection framework
- **[database](../database)** - Database manager and ORM

## Examples

See [example/](../example/) directory for more examples:

- **[basic](../example/basic)** - Simple server setup
- **[comprehensive](../example/comprehensive)** - Full-featured application
- **[sub_routing](../example/sub_routing)** - Advanced routing patterns

## Contributing

Contributions are welcome! Please see the main [repository](https://github.com/azizndao/glib) for contribution guidelines.

## License

This module is part of the Glib framework. See the main repository for license information.

## Roadmap

See the main [README](../README.md) for the v2.0.0+ roadmap.
