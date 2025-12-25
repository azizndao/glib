# Cache API Example

This example demonstrates how to use the Glib cache module with HTTP handlers to cache expensive database queries.

## Features Demonstrated

- **Cache-Aside Pattern**: Automatic caching with `Remember()`
- **Cache Invalidation**: Manual cache clearing
- **Statistics Monitoring**: Real-time cache performance metrics
- **HTTP Integration**: Using cache in REST API handlers

## Running the Example

```bash
go run main.go
```

The server will start on <http://localhost:8080>

## API Endpoints

### GET /users?id={id}

Fetch a user by ID (with caching).

```bash
# First request (slow - database query)
curl http://localhost:8080/users?id=1

# Second request (fast - from cache)
curl http://localhost:8080/users?id=1
```

### POST /users/invalidate?id={id}

Invalidate cached user data.

```bash
curl -X POST http://localhost:8080/users/invalidate?id=1
```

### GET /cache/stats

View cache statistics.

```bash
curl http://localhost:8080/cache/stats
```

Response:

```json
{
    "hits": 5,
    "misses": 3,
    "writes": 3,
    "deletes": 1,
    "hit_ratio": 0.625,
    "size": 2
}
```

## Testing the Cache

### Test 1: Cache Hit

```bash
# First request - cache miss (slow)
time curl http://localhost:8080/users?id=1

# Second request - cache hit (fast)
time curl http://localhost:8080/users?id=1

# Check statistics
curl http://localhost:8080/cache/stats
```

### Test 2: Cache Invalidation

```bash
# Fetch user (cache miss)
curl http://localhost:8080/users?id=1

# Invalidate cache
curl -X POST http://localhost:8080/users/invalidate?id=1

# Fetch again (cache miss again)
curl http://localhost:8080/users?id=1

# Check statistics
curl http://localhost:8080/cache/stats
```

### Test 3: Multiple Users

```bash
# Fetch different users
curl http://localhost:8080/users?id=1
curl http://localhost:8080/users?id=2
curl http://localhost:8080/users?id=3

# All should be cached now
curl http://localhost:8080/users?id=1
curl http://localhost:8080/users?id=2
curl http://localhost:8080/users?id=3

# Check statistics
curl http://localhost:8080/cache/stats
```

## Code Walkthrough

### 1. Cache Setup

```go
cacheManager := cache.NewManager()
cacheManager.RegisterDriver("memory", func() cache.Store {
    return drivers.NewMemory()
})
```

### 2. Service Layer with Caching

```go
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    cacheKey := fmt.Sprintf("user:%d", id)

    var user User
    err := s.cache.Remember(ctx, cacheKey, &user, 5*time.Minute, func() (any, error) {
        log.Printf("Cache miss - fetching user %d from database", id)
        return s.repo.Find(id)
    })

    return &user, err
}
```

The `Remember` method:

- Checks cache first
- On miss, executes the callback (database query)
- Automatically stores the result in cache
- Returns the value

### 3. Cache Invalidation

```go
func (s *UserService) InvalidateUser(ctx context.Context, id int) error {
    cacheKey := fmt.Sprintf("user:%d", id)
    return s.cache.Forget(ctx, cacheKey)
}
```

### 4. Monitoring

```go
if memStore, ok := store.(*drivers.Memory); ok {
    stats := memStore.GetStats()
    // Access: stats.Hits, stats.Misses, stats.HitRatio()
}
```

## Performance Impact

Without cache:

- Each request: ~100ms (database query time)
- 10 requests: ~1000ms

With cache:

- First request: ~100ms (database + cache write)
- Subsequent requests: <1ms (memory read)
- 10 requests (1 miss + 9 hits): ~109ms

**Speedup: ~91% reduction in response time!**

## Production Considerations

### 1. Cache Key Strategy

```go
// Good: Namespace by entity type
cacheKey := fmt.Sprintf("user:%d", id)

// Better: Include version for easy invalidation
cacheKey := fmt.Sprintf("user:v1:%d", id)
```

### 2. TTL Selection

```go
// Short-lived: Session data
cache.Put(ctx, key, value, 15*time.Minute)

// Medium-lived: User profiles
cache.Put(ctx, key, value, 1*time.Hour)

// Long-lived: Configuration
cache.Put(ctx, key, value, 24*time.Hour)
```

### 3. Error Handling

```go
err := cache.Remember(ctx, key, &value, ttl, callback)
if err != nil {
    // Cache errors shouldn't break your app
    // Fall back to direct database query
    return repo.Find(id)
}
```

### 4. Monitoring

```go
// Log cache statistics periodically
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        stats := store.GetStats()
        log.Printf("Cache: hits=%d misses=%d ratio=%.2f",
            stats.Hits, stats.Misses, stats.HitRatio())
    }
}()
```

## Next Steps

- Try using **Redis driver** for distributed caching
- Implement **cache tags** for grouped invalidation
- Add **cache warming** on startup
- Use **locks** for preventing cache stampede
- Monitor cache with **Prometheus metrics**

## See Also

- [Cache Documentation](../../cache/README.md)
- [Database ORM](../../database/README.md)
- [HTTP Router](../../http/README.md)
