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
        return c.JSON(200, map[string]string{"message": "Hello World"})
    })

    router.Get("/hello/{name}", func(c *grouter.Ctx) error {
        return c.JSON(200, map[string]string{
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

#### Request Data

```go
func handler(c *grouter.Ctx) error {
    // Path parameters (Go 1.22+ routing)
    id := c.PathValue("id")
    
    // Query parameters
    search := c.Query("search")
    limit, err := c.QueryInt("limit")
    
    // Headers
    auth := c.Get("Authorization")
    
    // Request info
    method := c.Method()
    path := c.Path()
    ip := c.IP()
    
    // Parse JSON body
    var user User
    if err := c.BodyParser(&user); err != nil {
        return err
    }
    
    return nil
}
```

#### Response Helpers

```go
func handler(c *grouter.Ctx) error {
    // JSON response
    return c.JSON(200, map[string]string{"status": "ok"})
    
    // Text response
    return c.SendString(200, "Hello World")
    
    // HTML response
    return c.HTML(200, []byte("<h1>Hello World</h1>"))
    
    // File response
    return c.File(200, "/path/to/file.pdf")
    
    // Redirect
    return c.Redirect(302, "/new-location")
    
    // Set status and headers (note: Status() must be called before response methods)
    c.Status(201).Set("Location", "/users/123")
    return c.JSON(201, user)
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

```go
func customMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Before request
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        // After request
        duration := time.Since(start)
        log.Printf("Request took %v", duration)
    })
}

router.Use(customMiddleware)
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
    
    return c.JSON(200, user)
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

```go
func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := grouter.NewCtx(w, r)
        r = ctx.SetValue("user", user)
        next.ServeHTTP(w, r)
    })
}

func handler(c *grouter.Ctx) error {
    user := c.GetValue("user")
    return c.JSON(user)
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
