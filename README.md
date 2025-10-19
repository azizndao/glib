# GRouter

A minimal HTTP router for Go that leverages Go 1.22+ enhanced routing features while providing a clean, intuitive interface for building web applications.

## Features

- **Clean API**: Intuitive routing interface
- **Enhanced HTTP routing**: Built on Go 1.22+ `net/http` improvements
- **Colorful logging**: Beautiful, configurable request logging with ANSI colors
- **Error handling**: Graceful error handling with automatic logging
- **Middleware support**: Composable middleware chain with built-in middleware
- **Route groups**: Organize routes with prefixes and group-specific middleware
- **Context helpers**: Convenient methods for request/response handling
- **Type safety**: Full type safety with Go's type system
- **Zero dependencies**: Uses only Go standard library

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
)

func main() {
    router := grouter.NewRouter()

    // Add middleware
    router.Use(grouter.Logger(), grouter.Recovery(func(err any, stack []byte) {
        fmt.Printf("PANIC: %v\n%s\n", err, stack)
    }))

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
    if err := c.BodyParser(&user); err != nil {
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

#### Built-in Middleware

```go
// Logger middleware
router.Use(grouter.Logger())

// Recovery middleware
router.Use(grouter.Recovery(func(err any, stack []byte) {
    log.Printf("PANIC: %v\n%s", err, stack)
}))

// Multiple middleware
router.Use(
    grouter.Logger(),
    grouter.Recovery(func(err any, stack []byte) {
        log.Printf("PANIC: %v\n%s", err, stack)
    }),
)
```

#### Custom Middleware

Middleware now works with the `*Ctx` interface, providing cleaner and more powerful middleware composition:

```go
// Middleware signature: func(grouter.Handler) grouter.Handler
// where Handler is: func(*Ctx) error

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

// Example: Authentication middleware
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
```

### Error Handling

GRouter handlers return errors that are automatically logged:

```go
func handler(c *grouter.Ctx) error {
    user, err := findUser(id)
    if err != nil {
        return grouter.ErrorNotFound("User not found", err)
    }

    if !user.IsActive {
        return grouter.ErrorForbidden("User is inactive", nil)
    }

    return c.Status(200).JSON(user)
}
```

GRouter provides structured error handling with built-in error types that return appropriate HTTP status codes and JSON responses:

```go
// Available error helpers:
grouter.ErrorBadRequest(data, internal)           // 400
grouter.ErrorUnauthorized(data, internal)         // 401
grouter.ErrorForbidden(data, internal)            // 403
grouter.ErrorNotFound(data, internal)             // 404
grouter.ErrorConflict(data, internal)             // 409
grouter.ErrorGone(data, internal)                 // 410
grouter.ErrorUnprocessableEntity(data, internal)  // 422
grouter.ErrorInternalServerError(data, internal)  // 500

// Standard errors are automatically converted to 500 responses
return fmt.Errorf("something went wrong") // Returns 500 with {"Code": 500, "Data": "Server Error"}
```

### Logging

GRouter includes colorful request logging:

```go
// Use default logger configuration
router.Use(grouter.Logger())

// Logger is automatically enabled when EnableLogging is true in RouterOptions
router := grouter.NewRouterWithOptions(grouter.RouterOptions{
    EnableLogging: true,
})
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
