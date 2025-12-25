# Glib Cache - Redis Driver

Distributed Redis-backed cache driver for the Glib framework with support for tags and distributed locks.

## Features

- **Distributed caching** with Redis
- **TTL support** with automatic expiration
- **Cache tags** using Redis Sets for grouped invalidation
- **Distributed locks** using Redis SETNX and Lua scripts
- **Thread-safe** operations across multiple processes
- **Atomic operations** for increment/decrement

## Installation

```bash
go get github.com/azizndao/glib/cache/redis
```

## Usage

```go
package main

import (
    "context"
    "time"

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

    // Register redis driver
    manager.RegisterDriver("redis", func() cache.Store {
        return redis.New(client, "myapp")
    })

    // Get cache instance
    store, _ := manager.Store("redis")

    // Basic operations
    ctx := context.Background()
    store.Put(ctx, "key", "value", 5*time.Minute)

    var result string
    store.Get(ctx, "key", &result)

    // Cache tags (uses Redis Sets)
    tagged := store.Tags("users", "posts")
    tagged.Put(ctx, "feed:123", data, time.Hour)
    tagged.FlushTags(ctx) // Invalidate all tagged items

    // Distributed locks
    lock := store.Lock("process-job", 30*time.Second)
    lock.Get(ctx, func() error {
        // Critical section
        return nil
    })
}
```

## Configuration

```go
// Create with prefix (all keys will be prefixed)
store := redis.New(client, "myapp")

// Without prefix
store := redis.New(client, "")
```

## Tags Implementation

The Redis driver uses Redis Sets to track tagged cache entries:

- Each tag gets a Set: `tag:{tagname}` containing all keys
- When flushing tags, all keys in the tag sets are deleted
- More efficient than scanning for pattern matches

## Distributed Locks

Locks use:

- `SETNX` for atomic lock acquisition
- `PEXPIRE` for TTL
- Lua scripts for atomic release
- UUID-based lock ownership

## Requirements

- Redis 2.6.12+ (for Lua script support)
- Go 1.23+

## License

MIT License - see LICENSE file for details
