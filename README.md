# GRouter

A minimal HTTP router for Go that leverages Go 1.22+ enhanced routing features while providing a clean, intuitive interface for building web applications.

## Features

- **Clean API**: Intuitive routing interface
- **Enhanced HTTP routing**: Built on Go 1.22+ `net/http` improvements
- **Colorful logging**: Beautiful, configurable request logging with ANSI colors
- **Error handling**: Built-in error types and graceful error handling
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
    router.GET("/hello", func(c *grouter.Ctx) error {
        return c.JSON(map[string]string{"message": "Hello World"})
    })

    router.GET("/hello/{name}", func(c *grouter.Ctx) error {
        return c.JSON(map[string]string{
            "message": fmt.Sprintf("Hello %s", c.PathValue("name")),
            "query":   c.Query("q"),
        })
    })

    slog.Info("Server started on :8080")
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
router.GET("/path", handler)
router.POST("/path", handler)
router.PUT("/path", handler)
router.PATCH("/path", handler)
router.DELETE("/path", handler)
router.OPTIONS("/path", handler)
router.HEAD("/path", handler)

// Generic method handler
router.Handle("METHOD", "/path", handler)
```

### Route Groups

```go
// Create a group with prefix
api := router.Group("/api")
api.GET("/users", getUsersHandler)
api.POST("/users", createUserHandler)

// Groups with middleware
admin := router.Group("/admin", authMiddleware, adminMiddleware)
admin.GET("/dashboard", dashboardHandler)
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
    return c.JSON(map[string]string{"status": "ok"})
    
    // Text response
    return c.SendString("Hello World")
    
    // Set status and headers
    return c.Status(201).Set("Location", "/users/123").JSON(user)
}
```

### Middleware

#### Built-in Middleware

```go
// Logger with different formats
router.Use(grouter.Logger())              // Default format
router.Use(grouter.LoggerTiny())          // Minimal format
router.Use(grouter.LoggerShort())         // Short format
router.Use(grouter.LoggerCombined())      // Apache combined format

// Recovery middleware
router.Use(grouter.Recovery(func(err any, stack []byte) {
    log.Printf("PANIC: %v\n%s", err, stack)
}))

// CORS middleware
router.Use(grouter.CORS(grouter.DefaultCORSOptions()))

// Timeout middleware
router.Use(grouter.Timeout(30 * time.Second))

// Chain multiple middleware
router.Use(grouter.Chain(
    grouter.Logger(),
    grouter.Recovery(nil),
    grouter.CORS(grouter.DefaultCORSOptions()),
))
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

GRouter provides structured error handling with built-in error types:

```go
func handler(c *grouter.Ctx) error {
    user, err := findUser(id)
    if err != nil {
        return grouter.ErrorNotFound("User not found", err)
    }
    
    if !user.IsActive {
        return grouter.ErrorForbidden("User is inactive", nil)
    }
    
    return c.JSON(user)
}

// Available error helpers:
// ErrorBadRequest(data, internal)
// ErrorUnauthorized(data, internal) 
// ErrorForbidden(data, internal)
// ErrorNotFound(data, internal)
// ErrorMethodNotAllowed(data, internal)
// ErrorInternalServerError(data, internal)
// ErrorServiceUnavailable(data, internal)
// ErrorGatewayTimeout(data, internal)
```

### Logging Formats

GRouter includes beautiful colored logging with multiple formats:

- **Default**: Structured format with method, path, status, duration, and size
- **Tiny**: Minimal format with just timestamp, method, status, and duration
- **Short**: Includes method, path, status, and duration
- **Combined**: Apache combined log format with user agent

```go
// Configure custom logger
config := grouter.LoggerConfig{
    Format:     grouter.LogFormatDefault,
    TimeFormat: "15:04:05",
    Output:     os.Stdout,
    Skip: func(r *http.Request) bool {
        return strings.HasPrefix(r.URL.Path, "/health")
    },
}
router.Use(grouter.Logger(config))
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
```

## Requirements

- Go 1.22 or later (for enhanced routing features)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
