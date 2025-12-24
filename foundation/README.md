# Glib Foundation Module

[![Go Reference](https://pkg.go.dev/badge/github.com/azizndao/glib/foundation.svg)](https://pkg.go.dev/github.com/azizndao/glib/foundation)

The foundation module provides the dependency injection framework and application lifecycle management for Glib. It implements the ServiceProvider pattern inspired by Laravel, enabling modular, testable applications.

## Features

- **Application Container** - Central application instance managing lifecycle
- **ServiceProvider Pattern** - Modular service registration and bootstrapping
- **Lifecycle Management** - Register → Boot → Shutdown phases
- **Dependency Injection** - Built on `common/container` with type safety
- **Configuration Integration** - Seamless config management
- **Graceful Shutdown** - Signal handling and cleanup
- **Environment Support** - Development, staging, production modes

## Installation

```bash
go get github.com/azizndao/glib/foundation@latest
```

## Quick Start

### Basic Application

```go
package main

import (
    "log"
    
    "github.com/azizndao/glib/foundation"
)

func main() {
    // Create application
    app := foundation.New(".")
    
    // Register service providers
    app.Register(&MyServiceProvider{})
    
    // Bootstrap and run
    if err := app.Bootstrap(); err != nil {
        log.Fatal(err)
    }
    
    // Your application logic here
    log.Println("App is running!")
}
```

### With Graceful Shutdown

```go
func main() {
    app := foundation.New(".")
    app.Register(&MyServiceProvider{})
    
    // Run with automatic graceful shutdown
    if err := app.Run(func() error {
        // Start your services here
        return startServer()
    }); err != nil {
        log.Fatal(err)
    }
}
```

## Core Concepts

### Application

The `Application` is the central container that manages:

- Service container (DI)
- Configuration
- Service providers
- Application lifecycle
- Environment settings

```go
// Create application
app := foundation.New("/path/to/project")

// Access components
container := app.Container()
config := app.Config()
logger := app.Logger()
basePath := app.BasePath()

// Environment checks
if app.IsProduction() { }
if app.IsDevelopment() { }
if app.IsTesting() { }
if app.IsDebug() { }
```

### ServiceProvider

ServiceProviders are the central place to register and bootstrap services.

#### ServiceProvider Interface

```go
type ServiceProvider interface {
    // Register registers services in the container
    // Called first for all providers
    Register(app *Application) error
    
    // Boot performs initialization after all providers registered
    // Called after all Register methods
    Boot(app *Application) error
}
```

#### Creating a ServiceProvider

```go
package mypkg

import (
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/common/container"
)

type MyServiceProvider struct {
    foundation.BaseServiceProvider
}

func (p *MyServiceProvider) Register(app *foundation.Application) error {
    // Bind services in container
    return container.Singleton(app.Container(), func(c *container.Container) (*MyService, error) {
        cfg := container.MustResolve[config.Config](c)
        return NewMyService(cfg), nil
    })
}

func (p *MyServiceProvider) Boot(app *foundation.Application) error {
    // Initialize service (called after all providers registered)
    svc := container.MustResolve[*MyService](app.Container())
    return svc.Initialize()
}
```

#### Using BaseServiceProvider

The `BaseServiceProvider` provides default implementations, so you only override what you need:

```go
type SimpleProvider struct {
    foundation.BaseServiceProvider
}

// Only override Register, Boot is no-op from base
func (p *SimpleProvider) Register(app *foundation.Application) error {
    return container.Instance(app.Container(), &MyConfig{})
}
```

## Application Lifecycle

### 1. Creation

```go
app := foundation.New(".")
```

Creates application with:
- Empty service container
- Default logger
- Base path set
- Environment from `APP_ENV`

### 2. Registration

```go
app.Register(&DatabaseProvider{})
app.Register(&CacheProvider{})
app.Register(&MailProvider{})
```

Registers service providers (doesn't call `Register()` yet).

### 3. Bootstrap

```go
if err := app.Bootstrap(); err != nil {
    log.Fatal(err)
}
```

Bootstrap performs:
1. Load configuration
2. Call `Register()` on all providers
3. Call `Boot()` on all providers

### 4. Run

```go
app.Run(func() error {
    // Your application logic
    return server.Listen()
})
```

Runs application with:
- Graceful shutdown handling (SIGINT, SIGTERM)
- Automatic cleanup
- Error handling

### 5. Shutdown

```go
// Automatic with Run(), or manual:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
app.Shutdown(ctx)
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/common/config"
    "github.com/azizndao/glib/common/container"
    "github.com/azizndao/glib/common/slog"
)

// Service definition
type Database struct {
    config *config.Repository
    logger *slog.Logger
}

func NewDatabase(cfg *config.Repository, logger *slog.Logger) *Database {
    return &Database{config: cfg, logger: logger}
}

func (db *Database) Connect() error {
    host := db.config.GetString("database.host", "localhost")
    db.logger.Info("Connecting to database", "host", host)
    return nil
}

// Service provider
type DatabaseServiceProvider struct {
    foundation.BaseServiceProvider
}

func (p *DatabaseServiceProvider) Register(app *foundation.Application) error {
    return container.Singleton(app.Container(), func(c *container.Container) (*Database, error) {
        cfg := container.MustResolve[*config.Repository](c)
        logger := container.MustResolve[*slog.Logger](c)
        return NewDatabase(cfg, logger), nil
    })
}

func (p *DatabaseServiceProvider) Boot(app *foundation.Application) error {
    db := container.MustResolve[*Database](app.Container())
    return db.Connect()
}

// Application
func main() {
    // Create application
    app := foundation.New(".")
    
    // Register providers
    app.Register(&DatabaseServiceProvider{})
    
    // Bootstrap
    if err := app.Bootstrap(); err != nil {
        log.Fatal(err)
    }
    
    // Resolve and use services
    db := container.MustResolve[*Database](app.Container())
    
    fmt.Println("Application started!")
    
    // Your application logic here...
}
```

## Advanced Usage

### Deferred Providers

Deferred providers are only loaded when their services are requested:

```go
type ExpensiveServiceProvider struct {
    foundation.BaseServiceProvider
}

func (p *ExpensiveServiceProvider) Provides() []string {
    return []string{"expensive.service"}
}

func (p *ExpensiveServiceProvider) Register(app *foundation.Application) error {
    // Only called when expensive.service is requested
    return container.Singleton(app.Container(), func(c *container.Container) (*ExpensiveService, error) {
        return NewExpensiveService(), nil
    })
}
```

### Multiple Providers

```go
// Register multiple providers at once
app.Register(
    &ConfigServiceProvider{},
    &LogServiceProvider{},
    &DatabaseServiceProvider{},
    &CacheServiceProvider{},
    &QueueServiceProvider{},
    &MailServiceProvider{},
)
```

### Environment-Specific Providers

```go
app.Register(&BaseServiceProvider{})

if app.IsProduction() {
    app.Register(&ProductionServiceProvider{})
} else {
    app.Register(&DevelopmentServiceProvider{})
}
```

### Provider Dependencies

Providers can depend on services from other providers:

```go
type CacheServiceProvider struct {
    foundation.BaseServiceProvider
}

func (p *CacheServiceProvider) Register(app *foundation.Application) error {
    return container.Singleton(app.Container(), func(c *container.Container) (*Cache, error) {
        // Depends on config from ConfigServiceProvider
        cfg := container.MustResolve[*config.Repository](c)
        return NewCache(cfg), nil
    })
}

type SessionServiceProvider struct {
    foundation.BaseServiceProvider
}

func (p *SessionServiceProvider) Register(app *foundation.Application) error {
    return container.Singleton(app.Container(), func(c *container.Container) (*Session, error) {
        // Depends on cache from CacheServiceProvider
        cache := container.MustResolve[*Cache](c)
        return NewSession(cache), nil
    })
}
```

Order of registration doesn't matter - dependencies are resolved automatically during `Boot()`.

### Configuration

```go
// Set custom config
cfg := config.New()
cfg.LoadFromEnv("APP")
app.SetConfig(cfg)

// Or bootstrap will create one automatically
app.Bootstrap()

// Access config
cfg := app.Config()
port := cfg.GetInt("server.port", 8080)
```

### Logging

```go
// Set custom logger
logger := slog.Create()
app.SetLogger(logger)

// Use logger
app.Logger().Info("Application started")
```

### Container Access

```go
// Get container
c := app.Container()

// Register services directly
container.Singleton(c, func(c *container.Container) (*MyService, error) {
    return NewMyService(), nil
})

// Resolve services
svc, err := container.Resolve[*MyService](c)
```

## Why Foundation vs Container?

You might wonder why `foundation` and `container` are separate modules.

### container (in common/)

- **Pure utility** - Generic DI container
- **No framework logic** - Can be used standalone
- **Type-safe** - Uses Go generics
- **Reusable** - Use in any Go project

```go
// Container is a pure utility
c := container.New()
container.Singleton(c, func(c *container.Container) (*DB, error) {
    return NewDB(), nil
})
db := container.MustResolve[*DB](c)
```

### foundation (this module)

- **Framework-specific** - Application lifecycle
- **ServiceProvider pattern** - Modular service registration
- **Boot phases** - Register → Boot → Shutdown
- **Configuration** - Environment, config, logging
- **Graceful shutdown** - Signal handling

```go
// Foundation adds framework patterns
app := foundation.New(".")
app.Register(&DatabaseProvider{}) // ServiceProvider pattern
app.Bootstrap()                    // Lifecycle management
app.Run(start)                     // Graceful shutdown
```

**Analogy:**
- `container` = illuminate/container (Laravel)
- `foundation` = illuminate/foundation (Laravel)

## Integration with Other Modules

### With HTTP Module

```go
import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/foundation"
)

type HTTPServiceProvider struct {
    foundation.BaseServiceProvider
}

func (p *HTTPServiceProvider) Register(app *foundation.Application) error {
    return container.Singleton(app.Container(), func(c *container.Container) (*glib.Server, error) {
        validator := container.MustResolve[*validation.Validator](c)
        return glib.New(glib.Config{Validator: validator}), nil
    })
}

func main() {
    app := foundation.New(".")
    app.Register(&ValidationServiceProvider{})
    app.Register(&HTTPServiceProvider{})
    
    app.Run(func() error {
        server := container.MustResolve[*glib.Server](app.Container())
        return server.ListenWithGracefulShutdown()
    })
}
```

### With Database Module

```go
import (
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/database"
)

func main() {
    app := foundation.New(".")
    
    // Database provider registers database manager
    app.Register(&database.DatabaseServiceProvider{})
    
    app.Bootstrap()
    
    // Resolve database
    db := container.MustResolve[*gorm.DB](app.Container())
}
```

## Testing

### Testing with Application

```go
package myapp_test

import (
    "testing"
    
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/common/container"
)

func TestApplication(t *testing.T) {
    app := foundation.New(".")
    app.Register(&MyServiceProvider{})
    
    if err := app.Bootstrap(); err != nil {
        t.Fatal(err)
    }
    
    // Test service resolution
    svc, err := container.Resolve[*MyService](app.Container())
    if err != nil {
        t.Fatal(err)
    }
    
    if svc == nil {
        t.Fatal("Expected service")
    }
}
```

### Testing ServiceProviders

```go
func TestMyServiceProvider(t *testing.T) {
    app := foundation.New(".")
    provider := &MyServiceProvider{}
    
    // Test registration
    if err := provider.Register(app); err != nil {
        t.Fatal(err)
    }
    
    // Test service is bound
    if !container.Bound[*MyService](app.Container()) {
        t.Fatal("Expected service to be bound")
    }
}
```

### Mock Providers for Testing

```go
type MockDatabaseProvider struct {
    foundation.BaseServiceProvider
}

func (p *MockDatabaseProvider) Register(app *foundation.Application) error {
    // Use in-memory database for tests
    return container.Instance(app.Container(), NewMockDB())
}

func TestWithMockDB(t *testing.T) {
    app := foundation.New(".")
    app.Register(&MockDatabaseProvider{})
    app.Bootstrap()
    
    // Tests use mock database
}
```

## Environment Variables

Foundation respects these environment variables:

```bash
# Environment
APP_ENV=production          # development, staging, production, testing

# Shutdown
APP_SHUTDOWN_TIMEOUT=30s    # Graceful shutdown timeout

# See common/config for full list of config env vars
```

## Architecture

```
Foundation Module
├── Application         - Central container & lifecycle
├── ServiceProvider     - Service registration interface
├── ProviderRepository  - Provider management
└── BaseServiceProvider - Convenience base class

Dependencies:
├── common/container    - DI container (pure utility)
├── common/config       - Configuration management
└── common/slog         - Structured logging
```

## Design Decisions

### Why ServiceProvider Pattern?

1. **Modularity** - Each provider encapsulates service setup
2. **Testability** - Easy to mock providers
3. **Clarity** - Clear separation of concerns
4. **Laravel-inspired** - Familiar to PHP developers
5. **Boot phase** - Initialize services after all registered

### Why Separate from Container?

1. **Pure utility** - Container can be used standalone
2. **Framework patterns** - Foundation adds app lifecycle
3. **Flexibility** - Use container without foundation
4. **Reusability** - Container in any Go project

## Best Practices

### 1. One Provider Per Service

```go
// Good: Focused provider
type DatabaseServiceProvider struct {}

// Bad: God provider
type AllServicesProvider struct {}
```

### 2. Use Boot for Initialization

```go
func (p *Provider) Register(app *foundation.Application) error {
    // Only bind to container
    return container.Singleton(app.Container(), ...)
}

func (p *Provider) Boot(app *foundation.Application) error {
    // Initialize after all services registered
    svc := container.MustResolve[*Service](app.Container())
    return svc.Connect()
}
```

### 3. Declare Dependencies Explicitly

```go
func (p *Provider) Register(app *foundation.Application) error {
    return container.Singleton(app.Container(), func(c *container.Container) (*Service, error) {
        // Explicit dependency resolution
        cfg := container.MustResolve[*Config](c)
        logger := container.MustResolve[*slog.Logger](c)
        return NewService(cfg, logger), nil
    })
}
```

### 4. Use Environment Checks

```go
func (p *Provider) Boot(app *foundation.Application) error {
    if app.IsProduction() {
        // Production-only initialization
    }
    
    if app.IsDebug() {
        // Debug logging
    }
    
    return nil
}
```

## Related Modules

- **[common/container](../common#container)** - Type-safe DI container
- **[common/config](../common#config)** - Configuration management
- **[database](../database)** - Database provider example
- **[http](../http)** - HTTP server integration

## Examples

See [example/foundation](../example/foundation) for a complete working example.

## Contributing

Contributions are welcome! Please ensure:

1. ✅ ServiceProvider interface compatibility
2. ✅ Lifecycle phases respected
3. ✅ Tests included
4. ✅ Documentation updated

## License

This module is part of the Glib framework. See the main repository for license information.

## Roadmap

- [ ] Provider auto-discovery
- [ ] Provider dependency graph visualization
- [ ] Hot reload support for development
- [ ] Provider lifecycle hooks (starting, stopping, etc.)
- [ ] Performance monitoring and profiling
