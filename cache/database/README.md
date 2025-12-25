# Glib Cache - Database Driver

Persistent database-backed cache driver for the Glib framework using GORM.

## Features

- **Persistent storage** using any GORM-supported database
- **TTL support** with automatic expiration via queries
- **Distributed locks** with database row-level locking
- **Thread-safe** operations across multiple processes
- **ACID compliance** for reliable distributed caching
- **Automatic migrations** for cache tables

## Installation

```bash
go get github.com/azizndao/glib/cache/database
```

## Usage

```go
package main

import (
    "context"
    "time"

    "github.com/azizndao/glib/cache"
    "github.com/azizndao/glib/cache/database"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    // Create GORM database connection
    db, _ := gorm.Open(sqlite.Open("cache.db"), &gorm.Config{})

    // Create cache manager
    manager := cache.NewManager()

    // Register database driver
    manager.RegisterDriver("database", func() cache.Store {
        store, _ := database.New(db, "cache_", 24*time.Hour)
        return store
    })

    // Get cache instance
    store, _ := manager.Store("database")

    // Basic operations
    ctx := context.Background()
    store.Put(ctx, "key", "value", 5*time.Minute)

    var result string
    store.Get(ctx, "key", &result)

    // Distributed locks (using database transactions)
    lock := store.Lock("process-job", 30*time.Second)
    lock.Get(ctx, func() error {
        // Critical section - protected across all processes
        return nil
    })
}
```

## Configuration

```go
// New(db, tablePrefix, defaultTTL)
store, err := database.New(db, "cache_", 24*time.Hour)
```

Parameters:

- `db`: GORM database instance
- `tablePrefix`: Prefix for cache tables (e.g., "cache\_" creates "cache_entries" and "cache_locks")
- `defaultTTL`: Default time-to-live for cached items

## Database Schema

The driver creates two tables:

### cache_entries

```sql
CREATE TABLE cache_entries (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    expiration DATETIME NOT NULL,
    INDEX idx_expiration (expiration)
);
```

### cache_locks

```sql
CREATE TABLE cache_locks (
    name VARCHAR(255) PRIMARY KEY,
    owner VARCHAR(255) NOT NULL,
    expiration DATETIME NOT NULL,
    INDEX idx_owner (owner),
    INDEX idx_expiration (expiration)
);
```

## Supported Databases

Works with any GORM-supported database:

- PostgreSQL
- MySQL/MariaDB
- SQLite
- SQL Server
- And more...

## Cleanup

The database driver does **not** automatically delete expired entries (unlike memory/redis drivers). You should set up a periodic cleanup job:

```go
// Run this periodically (e.g., via cron)
db.Where("expiration < ?", time.Now()).Delete(&database.CacheEntry{})
db.Where("expiration < ?", time.Now()).Delete(&database.CacheLock{})
```

## Performance Considerations

- Use indexed columns (expiration, owner) for better query performance
- Consider partitioning tables for very large datasets
- The database driver is slower than memory/redis but provides persistence
- Best for: Configuration caching, session storage, distributed coordination

## License

MIT License - see LICENSE file for details
