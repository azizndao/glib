# Glib Common Module

[![Go Reference](https://pkg.go.dev/badge/github.com/azizndao/glib/common.svg)](https://pkg.go.dev/github.com/azizndao/glib/common)

Common utilities for the Glib framework. This module provides pure, standalone utilities with no framework dependencies, making them reusable in any Go project.

## Features

- **Error Handling** - Structured API errors with stack traces
- **Logging** - Enhanced slog with error handling and dev mode
- **Configuration** - Environment-based config with dot notation
- **Container** - Type-safe dependency injection container
- **Type Utilities** - Type conversion helpers
- **Environment Utilities** - Helper functions for reading environment variables

## Installation

```bash
go get github.com/azizndao/glib/common@latest
```

## Packages

### errors

Structured error handling with stack traces and API error responses.

#### Creating Errors

```go
import "github.com/azizndao/glib/common/errors"

// Standard error with stack trace
err := errors.New("something went wrong")
err := errors.Errorf("failed to process user %d", userID)

// API errors (HTTP-specific)
err := errors.BadRequest("Invalid input", nil)
err := errors.Unauthorized("Invalid credentials", nil)
err := errors.Forbidden("Access denied", nil)
err := errors.NotFound("User not found", nil)
err := errors.MethodNotAllowed("Method not allowed", nil)
err := errors.Conflict("Email already exists", nil)
err := errors.UnprocessableEntity("Validation failed", validationErrors)
err := errors.InternalServerError("Database error", err)

// Custom API error
err := errors.NewAPI(418, "I'm a teapot", data)
```

#### APIError Structure

```go
type APIError struct {
    Code    int    `json:"code"`    // HTTP status code
    Message string `json:"message"` // Human-readable message
    Data    any    `json:"data"`    // Additional error data
}
```

#### Error with Stack Traces

```go
// Create error with stack trace
err := errors.New("database connection failed")

// Stack trace is automatically collected
// Print stack trace for debugging
if e, ok := err.(*errors.Error); ok {
    fmt.Println(e.StackTrace())
}

// Wrap errors
err = errors.Errorf("failed to create user: %w", originalErr)
```

#### API Error Examples

```go
// Validation errors
validationErrors := map[string]string{
    "email": "Email is required",
    "password": "Password must be at least 8 characters",
}
return errors.UnprocessableEntity("Validation failed", validationErrors)

// Not found
user, err := findUser(id)
if err != nil {
    return errors.NotFound("User not found", err)
}

// Authorization
if !user.IsAdmin() {
    return errors.Forbidden("Admin access required", nil)
}

// Server error
if err := db.Save(&user); err != nil {
    return errors.InternalServerError("Failed to save user", err)
}
```

### slog

Enhanced structured logging with error handling and development mode.

#### Creating a Logger

```go
import "github.com/azizndao/glib/common/slog"

// From environment (respects IS_DEBUG env var)
logger := slog.Create()

// Custom handler
handler := slog.NewHandler(isDebug, os.Stdout)
logger := slog.New(handler)

// With attributes
logger = logger.With("service", "api", "version", "1.0")
```

#### Logging Levels

```go
// Standard levels
logger.Debug("Debug message", "key", "value")
logger.Info("Info message", "user_id", 123)
logger.Warn("Warning message", "count", 0)

// Error logging (special handling)
logger.Error(err, "context", "value")
logger.ErrorCtx(ctx, err, "request_id", reqID)

// With source location
logger.InfoWithSource(ctx, 0, "Custom source location")
```

#### Error Handling

The logger has special support for `*errors.Error`:

```go
// Automatically extracts stack trace
err := errors.New("database error")
logger.Error(err) // Logs with stack trace in debug mode

// API errors logged with context
apiErr := errors.BadRequest("Invalid input", nil)
logger.Error(apiErr) // Logs with HTTP code and message
```

#### Dev Mode vs Production

```bash
# Development mode (readable, colorized)
IS_DEBUG=true

# Production mode (JSON, structured)
IS_DEBUG=false
```

Dev mode output:
```
2024-01-15 10:30:45 INFO  Request started method=GET path=/users
```

Production mode output:
```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","msg":"Request started","method":"GET","path":"/users"}
```

### config

Environment-based configuration with dot notation and type-safe access.

#### Creating Config

```go
import "github.com/azizndao/glib/common/config"

// New empty config
cfg := config.New()

// Load from environment variables
cfg.LoadFromEnv("APP")

// With initial values
cfg := config.NewWithMap(map[string]any{
    "app.name": "MyApp",
    "database.host": "localhost",
})
```

#### Accessing Values

```go
// Dot notation access
name := cfg.GetString("app.name", "default")
port := cfg.GetInt("server.port", 8080)
debug := cfg.GetBool("app.debug", false)
timeout := cfg.GetDuration("server.timeout", 30*time.Second)

// Generic get
value := cfg.Get("database.host")

// Check existence
if cfg.Has("redis.host") {
    // Redis is configured
}

// Get all values
allConfig := cfg.All()
```

#### Environment Mapping

The config automatically maps environment variables:

```go
cfg.LoadFromEnv("APP")

// Maps these env vars:
// APP_NAME → app.name
// APP_ENV → app.env
// APP_DEBUG → app.debug
// APP_URL → app.url
// SERVER_HOST → server.host
// SERVER_PORT → server.port
// DATABASE_HOST → database.host
// DATABASE_PORT → database.port
// etc.
```

#### Setting Values

```go
// Set values programmatically
cfg.Set("app.name", "MyApp")
cfg.Set("features.enabled", []string{"auth", "api"})
```

#### Environment Helpers

```go
// Get environment
env := cfg.Env("production") // Returns APP_ENV or default

// Check debug mode
if cfg.IsDebug() {
    // Debug mode enabled
}
```

#### Configuration Cache

```go
// Save config to cache
if err := cfg.SaveCache("/tmp/config.cache"); err != nil {
    log.Fatal(err)
}

// Load from cache
cfg, err := config.LoadFromCache("/tmp/config.cache")
if err != nil {
    // Fall back to loading from environment
    cfg = config.New()
    cfg.LoadFromEnv("APP")
}

// Check if loaded from cache
if cfg.IsCached() {
    log.Println("Using cached configuration")
}
```

### container

Type-safe dependency injection container using Go generics.

#### Why Container is in Common

The DI container is a **pure utility** with no framework-specific logic. It can be used standalone in any Go project. The `foundation` module builds on this container to provide framework-specific patterns like `ServiceProvider`.

#### Creating a Container

```go
import "github.com/azizndao/glib/common/container"

c := container.New()
```

#### Binding Services

```go
// Factory binding (new instance each time)
container.Bind(c, func(c *container.Container) (*Database, error) {
    cfg, _ := container.Resolve[*Config](c)
    return NewDatabase(cfg)
})

// Singleton binding (single instance)
container.Singleton(c, func(c *container.Container) (*Config, error) {
    return LoadConfig()
})

// Bind existing instance
logger := slog.Create()
container.Instance(c, logger)
```

#### Resolving Services

```go
// Resolve service
db, err := container.Resolve[*Database](c)
if err != nil {
    log.Fatal(err)
}

// Must resolve (panics on error)
cfg := container.MustResolve[*Config](c)

// Check if bound
if container.Bound[*Logger](c) {
    logger := container.MustResolve[*Logger](c)
}
```

#### Complete Example

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/azizndao/glib/common/container"
    "github.com/azizndao/glib/common/slog"
)

type Config struct {
    Host string
    Port int
}

type Database struct {
    config *Config
    logger *slog.Logger
}

func NewDatabase(cfg *Config, logger *slog.Logger) *Database {
    return &Database{config: cfg, logger: logger}
}

func main() {
    c := container.New()
    
    // Bind config as singleton
    container.Singleton(c, func(c *container.Container) (*Config, error) {
        return &Config{Host: "localhost", Port: 5432}, nil
    })
    
    // Bind logger as singleton
    container.Singleton(c, func(c *container.Container) (*slog.Logger, error) {
        return slog.Create(), nil
    })
    
    // Bind database (depends on config and logger)
    container.Singleton(c, func(c *container.Container) (*Database, error) {
        cfg := container.MustResolve[*Config](c)
        logger := container.MustResolve[*slog.Logger](c)
        return NewDatabase(cfg, logger), nil
    })
    
    // Resolve database (dependencies resolved automatically)
    db, err := container.Resolve[*Database](c)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Database: %s:%d\n", db.config.Host, db.config.Port)
}
```

#### Advanced Features

```go
// Contextual bindings
container.When[*UserController](c).
    Needs[LoggerInterface]().
    Give(func(c *container.Container) (LoggerInterface, error) {
        return NewCustomLogger("user-controller"), nil
    })

// Tag bindings
container.Tag[*Service1](c, "services")
container.Tag[*Service2](c, "services")
services := container.Tagged[ServiceInterface](c, "services")

// Resolve all
repos := container.ResolveAll[RepositoryInterface](c)
```

### util

Environment variable helpers with type conversion.

```go
import "github.com/azizndao/glib/common/util"

// String values
host := util.GetEnv("HOST", "localhost")

// Integer values
port := util.GetEnvInt("PORT", 8080)
maxConns := util.GetEnvInt64("MAX_CONNECTIONS", 100)

// Boolean values
debug := util.GetEnvBool("DEBUG", false)
// Recognizes: true/false, 1/0, yes/no, on/off

// Duration values
timeout := util.GetEnvDuration("TIMEOUT", 30*time.Second)
// Accepts: 30s, 5m, 1h, etc.

// Float values
factor := util.GetEnvFloat("SCALE_FACTOR", 1.5)

// Check if env var exists
if util.HasEnv("API_KEY") {
    apiKey := util.GetEnv("API_KEY", "")
}
```

### typeutil

Type conversion utilities.

```go
import "github.com/azizndao/glib/common/typeutil"

// String conversions
i, err := typeutil.ToInt("123")
f, err := typeutil.ToFloat("3.14")
b, err := typeutil.ToBool("true")
d, err := typeutil.ToDuration("5m")

// With defaults
i := typeutil.ToIntDefault("invalid", 0)
f := typeutil.ToFloatDefault("invalid", 1.0)
```

## Module Structure

```
common/
├── errors/           # Error handling
│   ├── error.go      # Stack trace errors
│   └── api_error.go  # API/HTTP errors
├── slog/             # Enhanced logging
│   ├── slog.go       # Logger wrapper
│   └── handler.go    # Custom handlers
├── config/           # Configuration
│   ├── config.go     # Config repository
│   └── cache.go      # Config caching
├── container/        # DI container
│   └── container.go  # Type-safe container
├── util/             # Utilities
│   ├── env.go        # Environment helpers
│   └── default.go    # Default values
└── typeutil/         # Type conversion
    └── convert.go    # Conversion functions
```

## Design Philosophy

### Pure Utilities

All packages in `common` are **pure utilities** with:

- ✅ No dependencies on other glib modules
- ✅ No framework-specific logic
- ✅ Usable in any Go project
- ✅ Small, focused APIs

### Why Common Exists

The `common` module was extracted to:

1. **Reusability** - Use utilities in non-web projects
2. **Clear Dependencies** - Other modules depend on common, not vice versa
3. **Testing** - Test utilities independently
4. **Flexibility** - Mix and match with other frameworks

## Usage in Glib

Other glib modules depend on common:

```
common (this module)
  ↑
  ├── http         - Uses errors, slog, util
  ├── foundation   - Uses container, errors, slog
  ├── database     - Uses errors, slog
  └── validation   - Uses errors
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/azizndao/glib/common/config"
    "github.com/azizndao/glib/common/container"
    "github.com/azizndao/glib/common/errors"
    "github.com/azizndao/glib/common/slog"
    "github.com/azizndao/glib/common/util"
)

type App struct {
    config *config.Repository
    logger *slog.Logger
}

func main() {
    // Create container
    c := container.New()
    
    // Bind config
    container.Singleton(c, func(c *container.Container) (*config.Repository, error) {
        cfg := config.New()
        cfg.LoadFromEnv("APP")
        return cfg, nil
    })
    
    // Bind logger
    container.Singleton(c, func(c *container.Container) (*slog.Logger, error) {
        return slog.Create(), nil
    })
    
    // Bind app
    container.Singleton(c, func(c *container.Container) (*App, error) {
        cfg := container.MustResolve[*config.Repository](c)
        logger := container.MustResolve[*slog.Logger](c)
        return &App{config: cfg, logger: logger}, nil
    })
    
    // Resolve and run
    app := container.MustResolve[*App](c)
    if err := app.Run(); err != nil {
        app.logger.Error(err)
    }
}

func (a *App) Run() error {
    // Use config
    port := a.config.GetInt("server.port", 8080)
    
    a.logger.Info("Starting application",
        "port", port,
        "env", a.config.Env(),
    )
    
    // Simulate error
    if a.config.GetBool("simulate.error", false) {
        return errors.New("simulated error for testing")
    }
    
    return nil
}
```

## Testing

```go
package mypackage_test

import (
    "testing"
    
    "github.com/azizndao/glib/common/config"
    "github.com/azizndao/glib/common/container"
    "github.com/azizndao/glib/common/errors"
)

func TestErrorHandling(t *testing.T) {
    err := errors.BadRequest("test error", nil)
    
    apiErr, ok := err.(*errors.APIError)
    if !ok {
        t.Fatal("Expected APIError")
    }
    
    if apiErr.Code != 400 {
        t.Errorf("Expected code 400, got %d", apiErr.Code)
    }
}

func TestContainer(t *testing.T) {
    c := container.New()
    
    // Bind test config
    container.Instance(c, &config.Repository{})
    
    // Resolve
    cfg, err := container.Resolve[*config.Repository](c)
    if err != nil {
        t.Fatal(err)
    }
    
    if cfg == nil {
        t.Fatal("Expected config")
    }
}
```

## Environment Variables

Common module respects these environment variables:

```bash
# Logging
IS_DEBUG=false          # Enable debug mode
LOG_LEVEL=info          # Log level (debug, info, warn, error)
LOG_FORMAT=json         # Log format (json, text)

# Application
APP_ENV=production      # Environment (development, staging, production)
APP_NAME=MyApp          # Application name
APP_DEBUG=false         # Debug mode
APP_URL=http://localhost:8080  # Application URL

# See config package for full list of supported env vars
```

## Related Modules

- **[http](../http)** - HTTP server (uses common/errors, common/slog)
- **[foundation](../foundation)** - DI framework (uses common/container)
- **[database](../database)** - Database manager (uses common/errors)
- **[validation](../validation)** - Request validation (uses common/errors)

## Contributing

Contributions are welcome! Please ensure:

1. ✅ No dependencies on other glib modules
2. ✅ Utilities are framework-agnostic
3. ✅ Tests included
4. ✅ Documentation updated

## License

This module is part of the Glib framework. See the main repository for license information.

## Roadmap

- [ ] Additional error types
- [ ] Configuration validation
- [ ] Container performance optimizations
- [ ] More type conversion utilities
