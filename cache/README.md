# Glib Cache Module

A flexible, Laravel-inspired caching system for Go with support for multiple drivers, cache tags, distributed locks, and the cache-aside (remember) pattern.

## Features

- **Modular Drivers**: Zero-dependency core with optional driver modules
- **Multiple Backends**: Memory, Redis, and Database drivers available
- **Cache Tags**: Group related cache items and invalidate them together
- **Distributed Locks**: Coordinate access to shared resources across processes
- **Remember Pattern**: Automatic cache-aside implementation
- **Type-Safe**: Generic-based operations with compile-time type safety
- **Thread-Safe**: All operations use proper synchronization
- **Context-First**: All methods accept `context.Context` for cancellation

## Installation

### Core Module (Required)

```bash
go get github.com/azizndao/glib/cache
```

### Driver Modules (Choose what you need)

```bash
# Memory driver (zero dependencies)
go get github.com/azizndao/glib/cache/memory

# Redis driver
go get github.com/azizndao/glib/cache/redis

# Database driver (GORM)
go get github.com/azizndao/glib/cache/database
```

**Why separate modules?** The modular architecture means you only import what you need. No Redis or GORM dependencies unless you actually use those drivers!

## Quick Start

### Memory Driver (Simplest)

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/azizndao/glib/cache"
    "github.com/azizndao/glib/cache/memory"
)

func main() {
    // Create cache manager
    manager := cache.NewManager()
    
    // Register memory driver
    manager.RegisterDriver("memory", func() cache.Store {
        return memory.New()
    })
    
    // Get default cache
    store, _ := manager.Default()
    ctx := context.Background()
    
    // Store a value
    store.Put(ctx, "user:123", "John Doe", 5*time.Minute)
    
    // Retrieve a value
    var name string
    store.Get(ctx, "user:123", &name)
    fmt.Println(name) // Output: John Doe
}
```

### Redis Driver (Production)

```go
import (
    "github.com/azizndao/glib/cache"
    "github.com/azizndao/glib/cache/redis"
    goredis "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := goredis.NewClient(&goredis.Options{
        Addr: "localhost:6379",
    })

    // Create cache manager
    manager := cache.NewManager()
    
    // Register redis driver with prefix
    manager.RegisterDriver("redis", func() cache.Store {
        return redis.New(client, "myapp")
    })
    
    store, _ := manager.Store("redis")
    // Now use distributed cache across multiple servers!
}
```

### Database Driver (Persistent)

```go
import (
    "github.com/azizndao/glib/cache"
    "github.com/azizndao/glib/cache/database"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    // Connect to database
    db, _ := gorm.Open(sqlite.Open("cache.db"), &gorm.Config{})

    // Create cache manager
    manager := cache.NewManager()
    
    // Register database driver
    manager.RegisterDriver("database", func() cache.Store {
        store, _ := database.New(db, "cache_", 24*time.Hour)
        return store
    })
    
    store, _ := manager.Store("database")
    // Cache persists across restarts!
}
```

## Core Operations

### Basic Operations

```go
ctx := context.Background()

// Put with TTL
store.Put(ctx, "key", "value", 5*time.Minute)

// Store forever (no expiration)
store.Forever(ctx, "key", "value")

// Get value
var result string
err := store.Get(ctx, "key", &result)
if err == cache.ErrCacheMiss {
    // Key not found
}

// Check existence
exists, _ := store.Has(ctx, "key")
missing, _ := store.Missing(ctx, "key")

// Delete
store.Forget(ctx, "key")

// Clear all
store.Flush(ctx)
```

### Atomic Operations

```go
// Increment counter
count, _ := store.Increment(ctx, "visits", 1)

// Decrement counter
count, _ := store.Decrement(ctx, "visits", 1)

// Add only if doesn't exist
added, _ := store.Add(ctx, "key", "value", 1*time.Hour)
if added {
    // Successfully added
}

// Pull (get and delete)
var value string
store.Pull(ctx, "token", &value)
```

### Cache-Aside Pattern (Remember)

The `Remember` method implements the cache-aside pattern automatically:

```go
var user User

// Get from cache or execute callback and store result
err := store.Remember(ctx, "user:123", &user, 1*time.Hour, func() (any, error) {
    // This only executes on cache miss
    return db.FindUser(123)
})

// RememberForever stores without expiration
store.RememberForever(ctx, "config", &config, func() (any, error) {
    return loadConfig()
})
```

## Cache Tags

Group related cache items and invalidate them together:

```go
// Create tagged cache
usersCache := store.Tags("users")
postsCache := store.Tags("posts")

// Store with tags
usersCache.Put(ctx, "user:1", user1, 1*time.Hour)
usersCache.Put(ctx, "user:2", user2, 1*time.Hour)
postsCache.Put(ctx, "post:1", post1, 1*time.Hour)

// Flush all items with "users" tag
usersCache.FlushTags(ctx)

// Multiple tags
cache := store.Tags("users", "premium")
cache.Put(ctx, "premium:1", premiumUser, 1*time.Hour)

// Flush all premium users
cache.FlushTags(ctx)
```

## Distributed Locks

Coordinate access to shared resources across processes:

```go
// Create lock with 30 second TTL
lock := store.Lock("process-job", 30*time.Second)

// Execute code while holding lock
err := lock.Get(ctx, func() error {
    // Critical section - only one process executes this
    return processJob()
})

if err == cache.ErrLockNotAcquired {
    // Another process holds the lock
}

// Wait to acquire lock (with timeout)
err = lock.Block(ctx, 5*time.Second, func() error {
    // This will wait up to 5 seconds to acquire the lock
    return processJob()
})

if err == cache.ErrLockTimeout {
    // Couldn't acquire lock within timeout
}
```

### Manual Lock Management

```go
lock := store.Lock("resource", 1*time.Minute)

// Acquire
acquired, _ := lock.Acquire(ctx)
if acquired {
    defer lock.Release(ctx)
    // Do work...
}

// Force release (regardless of ownership)
lock.ForceRelease(ctx)

// Get owner identifier
owner := lock.Owner()
```

## Multiple Stores

Manage multiple cache stores with different configurations:

```go
manager := cache.NewManager()

// Register different drivers
manager.RegisterDriver("memory", func() cache.Store {
    return memory.New()
})

manager.RegisterDriver("redis", func() cache.Store {
    return redis.New(redisClient, "myapp")
})

// Use specific stores
memCache, _ := manager.Store("memory")
redisCache, _ := manager.Store("redis")

// Set default store
manager.SetDefaultStore("redis")
defaultCache, _ := manager.Default()

// Extend with custom store
customStore := MyCustomStore{}
manager.Extend("custom", customStore)
```

## Available Drivers

The cache module supports three production-ready drivers as separate modules:

| Driver   | Use Case | Performance | Persistence | Distributed | Dependencies |
|----------|----------|-------------|-------------|-------------|--------------|
| **Memory**   | Single process, dev/test | ⚡⚡⚡ Fastest | ❌ No | ❌ No | ✅ None |
| **Redis**    | Multi-process, production | ⚡⚡ Fast | ✅ Yes | ✅ Yes | go-redis/v9 |
| **Database** | Small apps, no Redis | ⚡ Moderate | ✅ Yes | ✅ Yes | GORM |

### Memory Driver

**Module:** `github.com/azizndao/glib/cache/memory`

In-memory cache for single-process applications:

```go
import "github.com/azizndao/glib/cache/memory"

store := memory.New()
```

**Features:**
- Thread-safe operations
- Automatic cleanup of expired items
- TTL support
- Statistics tracking
- Perfect for development and testing

**See:** [Memory Driver Documentation](./memory/README.md)

### Redis Driver

**Module:** `github.com/azizndao/glib/cache/redis`

Distributed cache backed by Redis:

```go
import (
    "github.com/azizndao/glib/cache/redis"
    goredis "github.com/redis/go-redis/v9"
)

client := goredis.NewClient(&goredis.Options{
    Addr: "localhost:6379",
})

// With key prefix
store := redis.New(client, "myapp")

// Without prefix
store := redis.New(client, "")
```

**Features:**
- Distributed caching across multiple processes/servers
- Redis-backed distributed locks (SETNX + Lua scripts)
- Tag support using Redis Sets
- Atomic operations with native Redis INCR/DECR
- Persistent cache that survives restarts

**See:** [Redis Driver Documentation](./redis/README.md)

### Database Driver

**Module:** `github.com/azizndao/glib/cache/database`

Persistent cache using database tables:

```go
import (
    "github.com/azizndao/glib/cache/database"
    "gorm.io/gorm"
)

// tablePrefix, defaultTTL
store, err := database.New(db, "cache_", 24*time.Hour)
```

**Features:**
- ACID-compliant persistent storage
- Database-backed distributed locks (row-level locking)
- Works with PostgreSQL, MySQL, SQLite, etc.
- Simple setup, no additional infrastructure

**Note:** Requires periodic cleanup of expired entries (unlike Memory/Redis which auto-expire).

**See:** [Database Driver Documentation](./database/README.md)

## Driver Comparison

### When to use Memory Driver

✅ Single-process applications  
✅ Development and testing  
✅ Session storage for non-critical data  
✅ Temporary caching where persistence isn't needed  

❌ Multi-server deployments  
❌ Data that must survive restarts  

### When to use Redis Driver

✅ Multi-server/distributed applications  
✅ Production environments  
✅ High-traffic applications  
✅ Need for distributed locks  
✅ Cache sharing between microservices  

❌ Simple single-server apps (overkill)  
❌ When Redis infrastructure isn't available  

### When to use Database Driver

✅ Small to medium traffic applications  
✅ When Redis is not available  
✅ Need ACID guarantees for cache operations  
✅ Cache must survive all failures  
✅ Development/testing environments  

❌ High-traffic applications (slower than Redis)  
❌ Frequently accessed hot data (use Redis instead)  

## Error Handling

The cache module defines standard errors:

```go
err := store.Get(ctx, "key", &value)

switch err {
case cache.ErrCacheMiss:
    // Key not found
case cache.ErrLockNotAcquired:
    // Could not acquire lock
case cache.ErrLockTimeout:
    // Lock acquisition timed out
default:
    // Other error
}
```

## Best Practices

### 1. Use Context

Always pass context for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

store.Get(ctx, "key", &value)
```

### 2. Choose Appropriate TTL

```go
// Short-lived data
store.Put(ctx, "session", session, 15*time.Minute)

// Medium-lived data
store.Put(ctx, "user", user, 1*time.Hour)

// Long-lived data
store.Put(ctx, "config", config, 24*time.Hour)

// Permanent data
store.Forever(ctx, "app_name", "MyApp")
```

### 3. Use Remember Pattern

Instead of manual cache management:

```go
// ❌ Bad
var data Data
err := store.Get(ctx, "key", &data)
if err == cache.ErrCacheMiss {
    data, err = loadData()
    store.Put(ctx, "key", data, 1*time.Hour)
}

// ✅ Good
var data Data
store.Remember(ctx, "key", &data, 1*time.Hour, loadData)
```

### 4. Use Tags for Related Data

```go
// When user is updated, invalidate all related caches
userCache := store.Tags("user", fmt.Sprintf("user:%d", userID))
userCache.Put(ctx, "profile", profile, 1*time.Hour)
userCache.Put(ctx, "settings", settings, 1*time.Hour)

// Later, flush all user data
userCache.FlushTags(ctx)
```

### 5. Use Locks for Critical Sections

```go
lock := store.Lock("update-counter", 30*time.Second)
lock.Get(ctx, func() error {
    // Read-modify-write operation
    var count int64
    store.Get(ctx, "counter", &count)
    count++
    return store.Put(ctx, "counter", count, 1*time.Hour)
})
```

## Architecture

```
github.com/azizndao/glib/cache/
├── cache/                      # Core module (zero deps)
│   ├── store.go               # Interfaces (Store, TaggedStore, Lock)
│   ├── manager.go             # Cache manager
│   ├── provider.go            # Service provider
│   ├── base_store.go          # Base implementations
│   ├── go.mod                 # No driver dependencies!
│   └── internal/
│       └── helpers.go         # Shared utilities
│
├── memory/                     # Memory driver module
│   ├── memory.go              # In-memory store
│   ├── lock.go                # In-memory locks
│   ├── tagged.go              # Tagged memory cache
│   ├── go.mod                 # Depends on: cache only
│   └── README.md
│
├── redis/                      # Redis driver module
│   ├── redis.go               # Redis store
│   ├── lock.go                # Distributed locks
│   ├── tagged.go              # Redis Sets for tags
│   ├── go.mod                 # Depends on: cache + go-redis
│   └── README.md
│
└── database/                   # Database driver module
    ├── database.go            # GORM-backed store
    ├── lock.go                # DB-backed locks
    ├── migrations/            # SQL migrations
    ├── go.mod                 # Depends on: cache + gorm
    └── README.md
```

## Testing

Run tests for specific modules:

```bash
# Core module tests
cd cache && go test ./...

# Memory driver tests
cd cache/memory && go test ./...

# Redis driver tests (requires Redis)
cd cache/redis && go test ./...

# Database driver tests
cd cache/database && go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

## Migration from v0.0.x

If you were using the old monolithic cache module, update your imports:

**Before:**
```go
import "github.com/azizndao/glib/cache/drivers"

manager.RegisterDriver("memory", func() cache.Store {
    return drivers.NewMemory()
})
```

**After:**
```go
import "github.com/azizndao/glib/cache/memory"

manager.RegisterDriver("memory", func() cache.Store {
    return memory.New()
})
```

**Changes:**
- `cache/drivers` → `cache/memory`, `cache/redis`, `cache/database`
- `drivers.NewMemory()` → `memory.New()`
- `drivers.NewRedis()` → `redis.New()`
- `drivers.NewDatabase()` → `database.New()`

## Performance Considerations

### Memory Driver
- **Reads**: O(1) with RWMutex
- **Writes**: O(1) with Mutex
- **Cleanup**: O(n) every minute (background goroutine)
- **Memory**: All data in RAM

### Redis Driver
- **Reads**: O(1) Redis GET
- **Writes**: O(1) Redis SET
- **Tags**: O(n) for flush operations
- **Network**: Latency depends on Redis connection

### Database Driver
- **Reads**: O(1) with indexed queries
- **Writes**: O(1) with primary key updates
- **Cleanup**: Manual cleanup recommended via cron
- **Locks**: Row-level locking overhead

## Contributing

Contributions are welcome! Please:

1. Write tests for new features
2. Follow existing code style
3. Update documentation
4. Add examples for new features

## License

Part of the Glib framework. See main repository for license.

## See Also

- [Memory Driver Documentation](./memory/README.md)
- [Redis Driver Documentation](./redis/README.md)
- [Database Driver Documentation](./database/README.md)
- [Database ORM](../database/README.md)
- [HTTP Router](../http/README.md)
- [Foundation](../foundation/README.md)
