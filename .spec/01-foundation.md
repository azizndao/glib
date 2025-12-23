# Phase 1: Foundation & Core Architecture

**Timeline**: Weeks 1-2  
**Priority**: Critical - Everything builds on this

## Overview

The foundation establishes the core patterns that the entire framework will build upon:
- Service Container for dependency management
- Service Providers for organized bootstrapping
- Enhanced Configuration system
- Application lifecycle management

## Components

### 1.1 Service Container

**Location**: `container/`

**Purpose**: Lightweight, type-safe dependency injection container for managing service instances and their dependencies.

#### Package Structure

```
container/
├── container.go          # Core container implementation
├── binding.go           # Binding types and structures
├── resolver.go          # Dependency resolution logic
├── provider.go          # Service provider interface
├── errors.go            # Container-specific errors
└── container_test.go    # Comprehensive tests
```

#### Core Types

```go
// Container manages service bindings and resolution
type Container struct {
    bindings   map[reflect.Type]*Binding
    instances  map[reflect.Type]interface{}
    aliases    map[string]reflect.Type
    mu         sync.RWMutex
}

// Binding represents a service binding
type Binding struct {
    Type      reflect.Type
    Factory   FactoryFunc
    Singleton bool
    Instance  interface{}
}

// FactoryFunc is a function that creates service instances
type FactoryFunc func(c *Container) (interface{}, error)
```

#### API Design

```go
// Create new container
container := container.New()

// Bind interface to implementation
container.Bind((*Database)(nil), func(c *Container) (interface{}, error) {
    cfg := c.MustResolve((*Config)(nil)).(Config)
    return database.Connect(cfg.Get("database.connection"))
})

// Singleton binding (single shared instance)
container.Singleton((*Cache)(nil), func(c *Container) (interface{}, error) {
    return cache.NewRedisCache(config)
})

// Instance binding (pre-created instance)
config := config.Load()
container.Instance((*Config)(nil), config)

// Resolution
db, err := container.Resolve((*Database)(nil))
// Type-safe helper
db := Resolve[*Database](container)

// Check if bound
if container.Bound((*Cache)(nil)) {
    // ...
}

// Aliases for convenience
container.Alias("db", (*Database)(nil))
db := container.ResolveAlias("db")
```

#### Contextual Binding

```go
// Different implementations based on consumer
container.When(&UserController{}).
    Needs((*Cache)(nil)).
    Give(func(c *Container) (interface{}, error) {
        return cache.NewRedisCache(config)
    })

container.When(&GuestController{}).
    Needs((*Cache)(nil)).
    Give(func(c *Container) (interface{}, error) {
        return cache.NewMemoryCache()
    })
```

#### Features

- **Type-safe binding**: Use interface types as keys
- **Factory functions**: Lazy instantiation
- **Singleton support**: Shared instances across application
- **Instance binding**: Pre-created objects
- **Contextual binding**: Different implementations per consumer
- **Aliases**: String-based resolution for convenience
- **Thread-safe**: Concurrent access protection
- **Error handling**: Clear error messages for missing bindings
- **Circular dependency detection**: Prevent infinite loops

#### Error Handling

```go
var (
    ErrBindingNotFound      = errors.New("binding not found")
    ErrCircularDependency   = errors.New("circular dependency detected")
    ErrInvalidType          = errors.New("invalid type for binding")
    ErrFactoryReturnedNil   = errors.New("factory returned nil")
)
```

#### Testing Support

```go
// Mock container for testing
func NewMockContainer() *Container {
    c := New()
    // Bind test implementations
    return c
}

// Swap binding for testing
container.Swap((*Database)(nil), mockDatabase)
```

---

### 1.2 Service Providers

**Location**: `foundation/`

**Purpose**: Organize application bootstrapping into logical, reusable components.

#### Package Structure

```
foundation/
├── application.go       # Main application struct
├── provider.go         # ServiceProvider interface
├── bootstrap.go        # Bootstrapping logic
├── kernel.go           # HTTP kernel
└── providers/          # Built-in providers
    ├── app_provider.go
    ├── database_provider.go
    ├── cache_provider.go
    ├── queue_provider.go
    └── auth_provider.go
```

#### Core Types

```go
// ServiceProvider is the interface for service registration
type ServiceProvider interface {
    Register(app *Application)
    Boot(app *Application) error
}

// DeferrableProvider defers loading until needed
type DeferrableProvider interface {
    ServiceProvider
    Provides() []interface{}  // Services this provider offers
    IsDeferred() bool
}

// Application is the core application instance
type Application struct {
    container       *container.Container
    config          *config.Repository
    basePath        string
    providers       []ServiceProvider
    bootedProviders []ServiceProvider
    booted          bool
    mu              sync.RWMutex
}
```

#### Application API

```go
// Create new application
app := foundation.NewApplication("/path/to/app")

// Register providers
app.Register(&DatabaseProvider{})
app.Register(&CacheProvider{})

// Register multiple providers
app.RegisterProviders(
    &DatabaseProvider{},
    &CacheProvider{},
    &QueueProvider{},
)

// Boot application (runs all provider Boot methods)
if err := app.Boot(); err != nil {
    log.Fatal(err)
}

// Access container
container := app.Container()

// Access config
config := app.Config()

// Path helpers
dbPath := app.DatabasePath()
storagePath := app.StoragePath()
configPath := app.ConfigPath()
```

#### Service Provider Lifecycle

```go
type DatabaseProvider struct {
    app *Application
}

// Register: Register bindings in the container
// This runs BEFORE all providers are registered
// Do NOT resolve other services here
func (p *DatabaseProvider) Register(app *Application) {
    p.app = app
    
    app.Container().Singleton((*database.Manager)(nil), 
        func(c *container.Container) (interface{}, error) {
            cfg := app.Config()
            return database.NewManager(cfg)
        })
}

// Boot: Bootstrap the service
// This runs AFTER all providers are registered
// Safe to resolve services from container
func (p *DatabaseProvider) Boot(app *Application) error {
    // Run migrations if configured
    if app.Config().GetBool("database.auto_migrate", false) {
        db := app.Container().MustResolve((*database.Manager)(nil))
        return db.Migrate()
    }
    return nil
}
```

#### Deferred Providers

```go
type CacheProvider struct{}

func (p *CacheProvider) Register(app *Application) {
    app.Container().Singleton((*cache.Manager)(nil), 
        func(c *container.Container) (interface{}, error) {
            return cache.NewManager(app.Config())
        })
}

func (p *CacheProvider) Boot(app *Application) error {
    return nil
}

// Mark as deferred - only loads when cache.Manager is resolved
func (p *CacheProvider) Provides() []interface{} {
    return []interface{}{(*cache.Manager)(nil)}
}

func (p *CacheProvider) IsDeferred() bool {
    return true
}
```

#### Built-in Providers

**AppServiceProvider**
- Register basic application services
- Load configuration
- Set up error handler
- Register helpers

**DatabaseServiceProvider**
- Register database manager
- Set up connections
- Register migration system
- Register seeders

**CacheServiceProvider** (Deferred)
- Register cache manager
- Configure cache drivers
- Set up default cache

**QueueServiceProvider** (Deferred)
- Register queue manager
- Configure queue drivers
- Register job dispatcher
- Set up failed job handler

**AuthServiceProvider**
- Register authentication manager
- Configure guards
- Register user provider
- Set up password hasher

#### Application Lifecycle

```
1. Create Application
   ↓
2. Load Configuration
   ↓
3. Register Service Providers
   - Call Register() on each provider
   - Build dependency graph
   ↓
4. Boot Service Providers
   - Call Boot() on each provider in order
   - Resolve dependencies as needed
   ↓
5. Application Ready
   - Handle HTTP requests
   - Process queue jobs
   - Run scheduled tasks
```

---

### 1.3 Enhanced Configuration System

**Location**: `config/`

**Purpose**: Laravel-style cascading configuration with environment variables, caching, and type safety.

#### Current State

✅ Already have basic env loading in `util/env.go`  
⚠️ Need structured config files and dot notation access

#### Package Structure

```
config/
├── config.go           # Main config manager
├── repository.go       # Config storage with dot notation
├── loader.go          # Load from files + env
├── cache.go           # Config caching for production
├── env.go             # Environment variable helpers
└── config_test.go     # Tests

# User's project structure
config/
├── app.go             # Application config
├── database.go        # Database config
├── cache.go           # Cache config
├── queue.go           # Queue config
├── auth.go            # Auth config
└── cors.go            # CORS config
```

#### Core Types

```go
// Repository stores configuration with nested key support
type Repository struct {
    items map[string]interface{}
    mu    sync.RWMutex
}

// Loader loads configuration from various sources
type Loader struct {
    configPath string
    envPath    string
}

// ConfigFunc returns configuration map
type ConfigFunc func() map[string]interface{}
```

#### API Design

```go
// Initialize config
config := config.New()

// Load from directory
loader := config.NewLoader("/path/to/config")
if err := loader.Load(config); err != nil {
    log.Fatal(err)
}

// Get value with dot notation
host := config.Get("database.connections.mysql.host")

// Type-safe getters
host := config.GetString("database.connections.mysql.host", "localhost")
port := config.GetInt("database.connections.mysql.port", 3306)
debug := config.GetBool("app.debug", false)
timeout := config.GetDuration("server.timeout", 30*time.Second)

// Check if exists
if config.Has("database.connections.postgres") {
    // ...
}

// Set value (typically for testing)
config.Set("app.debug", true)

// Get all config
all := config.All()

// Get nested config as map
dbConfig := config.GetStringMap("database.connections.mysql")
```

#### Configuration Files

**config/app.go**
```go
package config

import "github.com/azizndao/glib/util"

func App() map[string]interface{} {
    return map[string]interface{}{
        "name":  util.GetEnv("APP_NAME", "glib"),
        "env":   util.GetEnv("APP_ENV", "production"),
        "debug": util.GetEnvBool("APP_DEBUG", false),
        "url":   util.GetEnv("APP_URL", "http://localhost"),
        "timezone": util.GetEnv("APP_TIMEZONE", "UTC"),
        "locale": util.GetEnv("APP_LOCALE", "en"),
    }
}
```

**config/database.go**
```go
package config

import (
    "time"
    "github.com/azizndao/glib/util"
)

func Database() map[string]interface{} {
    return map[string]interface{}{
        "default": util.GetEnv("DB_CONNECTION", "mysql"),
        
        "connections": map[string]interface{}{
            "mysql": map[string]interface{}{
                "driver":   "mysql",
                "host":     util.GetEnv("DB_HOST", "localhost"),
                "port":     util.GetEnvInt("DB_PORT", 3306),
                "database": util.GetEnv("DB_DATABASE", "glib"),
                "username": util.GetEnv("DB_USERNAME", "root"),
                "password": util.GetEnv("DB_PASSWORD", ""),
                "charset":  util.GetEnv("DB_CHARSET", "utf8mb4"),
                "collation": util.GetEnv("DB_COLLATION", "utf8mb4_unicode_ci"),
                "prefix":   util.GetEnv("DB_PREFIX", ""),
                "timezone": util.GetEnv("DB_TIMEZONE", "UTC"),
                "pool": map[string]interface{}{
                    "max_open":     util.GetEnvInt("DB_MAX_OPEN", 100),
                    "max_idle":     util.GetEnvInt("DB_MAX_IDLE", 10),
                    "max_lifetime": util.GetEnvDuration("DB_MAX_LIFETIME", 1*time.Hour),
                },
            },
            
            "postgres": map[string]interface{}{
                "driver":   "postgres",
                "host":     util.GetEnv("DB_HOST", "localhost"),
                "port":     util.GetEnvInt("DB_PORT", 5432),
                "database": util.GetEnv("DB_DATABASE", "glib"),
                "username": util.GetEnv("DB_USERNAME", "postgres"),
                "password": util.GetEnv("DB_PASSWORD", ""),
                "sslmode":  util.GetEnv("DB_SSLMODE", "disable"),
                "timezone": util.GetEnv("DB_TIMEZONE", "UTC"),
                "pool": map[string]interface{}{
                    "max_open":     util.GetEnvInt("DB_MAX_OPEN", 100),
                    "max_idle":     util.GetEnvInt("DB_MAX_IDLE", 10),
                    "max_lifetime": util.GetEnvDuration("DB_MAX_LIFETIME", 1*time.Hour),
                },
            },
            
            "sqlite": map[string]interface{}{
                "driver":   "sqlite",
                "database": util.GetEnv("DB_DATABASE", "storage/database.sqlite"),
            },
        },
    }
}
```

#### Environment Variables

**.env**
```bash
# Application
APP_NAME=glib
APP_ENV=local
APP_DEBUG=true
APP_URL=http://localhost:8080

# Database
DB_CONNECTION=mysql
DB_HOST=localhost
DB_PORT=3306
DB_DATABASE=glib
DB_USERNAME=root
DB_PASSWORD=secret
DB_MAX_OPEN=100
DB_MAX_IDLE=10
DB_MAX_LIFETIME=1h

# Cache
CACHE_DRIVER=redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Queue
QUEUE_CONNECTION=redis
QUEUE_RETRY_AFTER=90
```

#### Configuration Caching

For production performance, cache all config into a single file:

```go
// Cache configuration
if err := config.Cache("/path/to/cache/config.json"); err != nil {
    log.Fatal(err)
}

// Load from cache (much faster)
if err := config.LoadCache("/path/to/cache/config.json"); err != nil {
    // Fall back to loading from files
    loader.Load(config)
}

// Check if cached
if config.IsCached() {
    // Using cached config
}
```

**CLI Commands:**
```bash
glib config:cache   # Cache configuration
glib config:clear   # Clear config cache
```

#### Features

- **Dot notation**: Access nested values easily
- **Type safety**: Type-safe getter methods
- **Environment cascade**: Config files reference env vars with defaults
- **Caching**: Single file cache for production
- **Immutability**: Config is read-only after loading (except in tests)
- **Validation**: Validate required config on boot
- **Hot reload**: Reload config without restart (development)

---

### 1.4 Application Bootstrap

**Location**: `foundation/bootstrap.go`

**Purpose**: Orchestrate the complete application startup sequence.

#### Bootstrap Sequence

```go
// Bootstrap the application
func Bootstrap(basePath string) (*Application, error) {
    // 1. Create application instance
    app := NewApplication(basePath)
    
    // 2. Load environment variables
    if err := loadEnvironment(app); err != nil {
        return nil, err
    }
    
    // 3. Load configuration
    if err := loadConfiguration(app); err != nil {
        return nil, err
    }
    
    // 4. Register service providers
    registerProviders(app)
    
    // 5. Boot providers
    if err := app.Boot(); err != nil {
        return nil, err
    }
    
    // 6. Load routes
    if err := loadRoutes(app); err != nil {
        return nil, err
    }
    
    return app, nil
}

func loadEnvironment(app *Application) error {
    envPath := app.BasePath() + "/.env"
    return godotenv.Load(envPath)
}

func loadConfiguration(app *Application) error {
    loader := config.NewLoader(app.ConfigPath())
    return loader.Load(app.Config())
}

func registerProviders(app *Application) {
    app.RegisterProviders(
        &AppServiceProvider{},
        &DatabaseServiceProvider{},
        &CacheServiceProvider{},
        &QueueServiceProvider{},
        &AuthServiceProvider{},
    )
}

func loadRoutes(app *Application) error {
    router := app.Container().MustResolve((*Router)(nil)).(Router)
    
    // Load API routes
    apiRoutes := routes.API()
    apiRoutes(router.Group("/api"))
    
    // Load web routes
    webRoutes := routes.Web()
    webRoutes(router)
    
    return nil
}
```

#### HTTP Kernel

```go
// Kernel handles HTTP request lifecycle
type Kernel struct {
    app        *Application
    router     Router
    middleware []Middleware
}

// Handle an incoming HTTP request
func (k *Kernel) Handle(w http.ResponseWriter, r *http.Request) {
    // Create request context
    ctx := glib.NewCtx(w, r, k.app.Logger(), k.app.Validator())
    
    // Run through middleware pipeline
    handler := k.router.Handler()
    for i := len(k.middleware) - 1; i >= 0; i-- {
        handler = k.middleware[i](handler)
    }
    
    // Execute
    handler.ServeHTTP(w, r)
}

// Terminate performs any final actions
func (k *Kernel) Terminate(w http.ResponseWriter, r *http.Request) {
    // Cleanup, flush logs, etc.
}
```

---

## Integration with Existing Code

### Integrating with Current glib.Server

```go
// Current code in glib.go
func New(config Config) *Server {
    // OLD WAY: Direct instantiation
    
    // NEW WAY: Use application foundation
    app := foundation.NewApplication(".")
    
    // Bootstrap application
    if err := foundation.Bootstrap(app); err != nil {
        log.Fatal(err)
    }
    
    // Create router from container
    router := app.Container().MustResolve((*Router)(nil)).(Router)
    
    // Build server
    server := &Server{
        app:     app,
        router:  router,
        logger:  app.Logger(),
        // ...
    }
    
    return server
}
```

### Migration Path

**Phase 1.1**: Implement container and providers without breaking existing API  
**Phase 1.2**: Gradually migrate existing initialization to use container  
**Phase 1.3**: Update documentation with new patterns  
**Phase 1.4**: Keep backward compatibility for at least one major version

---

## Testing Strategy

### Container Tests

```go
func TestContainerBinding(t *testing.T) {
    c := container.New()
    
    c.Bind((*Database)(nil), func(c *Container) (interface{}, error) {
        return &MockDatabase{}, nil
    })
    
    db, err := c.Resolve((*Database)(nil))
    assert.NoError(t, err)
    assert.NotNil(t, db)
}

func TestContainerSingleton(t *testing.T) {
    c := container.New()
    
    c.Singleton((*Cache)(nil), func(c *Container) (interface{}, error) {
        return &RedisCache{}, nil
    })
    
    cache1, _ := c.Resolve((*Cache)(nil))
    cache2, _ := c.Resolve((*Cache)(nil))
    
    // Same instance
    assert.Equal(t, cache1, cache2)
}
```

### Provider Tests

```go
func TestDatabaseProviderRegistration(t *testing.T) {
    app := foundation.NewApplication(".")
    provider := &DatabaseProvider{}
    
    provider.Register(app)
    
    assert.True(t, app.Container().Bound((*database.Manager)(nil)))
}

func TestDeferredProvider(t *testing.T) {
    app := foundation.NewApplication(".")
    provider := &CacheProvider{}
    
    // Should be deferred
    assert.True(t, provider.IsDeferred())
    
    // Should provide cache manager
    provides := provider.Provides()
    assert.Contains(t, provides, (*cache.Manager)(nil))
}
```

### Configuration Tests

```go
func TestConfigDotNotation(t *testing.T) {
    config := config.New()
    config.Set("database.connections.mysql.host", "localhost")
    
    host := config.GetString("database.connections.mysql.host")
    assert.Equal(t, "localhost", host)
}

func TestConfigEnvironmentFallback(t *testing.T) {
    os.Setenv("DB_HOST", "production.db")
    
    config := loadDatabaseConfig()
    host := config.GetString("database.connections.mysql.host")
    
    assert.Equal(t, "production.db", host)
}
```

---

## Performance Considerations

### Container Performance

- **Singleton caching**: Once resolved, singletons are cached
- **Lock-free reads**: Use sync.RWMutex for concurrent access
- **Lazy loading**: Deferred providers only load when needed
- **No reflection in hot path**: Reflection only during binding

### Configuration Performance

- **Cache in production**: Single JSON file load
- **Memory efficient**: Store as nested maps
- **Fast lookups**: O(1) for simple keys, O(n) for dot notation
- **No environment variable access**: Pre-loaded at startup

---

## Documentation Requirements

### User Documentation

- **Getting Started**: How to use the container
- **Service Providers**: Creating custom providers
- **Configuration**: Setting up config files
- **Best Practices**: When to use singleton vs factory

### API Documentation

- **Godoc comments**: Every public function
- **Examples**: Code examples in documentation
- **Type definitions**: Clear interface definitions

### Migration Guides

- **From raw Go**: How glib improves structure
- **From other frameworks**: Comparison with Echo, Gin, Fiber
- **From Laravel**: Translating Laravel concepts to Go

---

## Success Metrics

### Phase 1 Complete When:

- ✅ Service container can manage dependencies
- ✅ Service providers organize bootstrapping
- ✅ Configuration supports dot notation and env vars
- ✅ Application lifecycle is clear and testable
- ✅ All tests pass with >90% coverage
- ✅ Documentation is complete
- ✅ Example application uses new patterns
- ✅ Performance benchmarks meet targets

---

## Next Phase Preview

Once foundation is complete, Phase 2 will build the database layer on top of:
- Container provides database connections
- Configuration defines database settings
- Service provider bootstraps database manager

The foundation makes all subsequent phases cleaner and more maintainable.
