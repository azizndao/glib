# Glib Cache - Memory Driver

High-performance in-memory cache driver for the Glib framework.

## Features

- **Fast in-memory storage** with automatic cleanup
- **TTL support** with automatic expiration
- **Cache tags** for grouped invalidation
- **Thread-safe** operations
- **Lock support** for coordinating concurrent access
- **Statistics** tracking (hits, misses, writes, deletes)

## Installation

```bash
go get github.com/azizndao/glib/cache/memory
```

## Usage

```go
package main

import (
    "context"
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

    // Get cache instance
    store, _ := manager.Store("memory")

    // Basic operations
    ctx := context.Background()
    store.Put(ctx, "key", "value", 5*time.Minute)

    var result string
    store.Get(ctx, "key", &result)

    // Cache tags
    tagged := store.Tags("users", "posts")
    tagged.Put(ctx, "feed:123", data, time.Hour)
    tagged.FlushTags(ctx) // Invalidate all tagged items

    // Locks
    lock := store.Lock("process-job", 30*time.Second)
    lock.Get(ctx, func() error {
        // Critical section
        return nil
    })
}
```

## Configuration

The memory driver requires no configuration and is ready to use immediately:

```go
store := memory.New()
```

## Performance

The memory driver is optimized for:

- Fast read/write operations (O(1) complexity)
- Automatic cleanup of expired items
- Minimal memory overhead
- Thread-safe concurrent access

## Statistics

Track cache performance:

```go
stats := store.(*memory.Memory).GetStats()
fmt.Printf("Hit ratio: %.2f%%\n", stats.HitRatio() * 100)
```

## License

MIT License - see LICENSE file for details
