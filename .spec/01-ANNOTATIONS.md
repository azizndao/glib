# Glib - Annotation Reference

Complete guide to all annotations supported by Glib code generator.

---

## Table of Contents

1. [@Controller](#controller) - Define HTTP controllers
2. [@Route](#route) - Define HTTP endpoints
3. [@Provider](#provider) - Define DI providers
4. [@Middleware](#middleware) - Define middleware
5. [Handler Patterns](#handler-patterns) - Supported method signatures

---

## @Controller

Marks a struct as an HTTP controller with base path and optional tags for middleware targeting.

### Syntax

```go
// @Controller path=<path-prefix> tags=<tag1,tag2>
type ControllerName struct {
    // Dependencies - auto-wired by type from providers
}
```

### Parameters

- **path** (required): Base path for all routes in this controller (must start with `/`)
- **tags** (optional): Comma-separated tags for middleware targeting (e.g., `tags=api,protected`)

### Auto-Wiring

All struct fields are automatically injected by type from registered providers. **No tags needed!**

### Examples

#### Simple Controller with Path

```go
// @Controller path=/api/v1/posts
type PostsController struct {
    PostService *services.PostService  // Auto-injected from provider
}

// Full path: GET /api/v1/posts
// @Route method=GET path=/
func (c *PostsController) Index(ctx context.Context) glib.Result[[]Post] {
    posts := c.PostService.GetPosts()
    return glib.OK(posts)
}

// Full path: GET /api/v1/posts/{id}
// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id int) glib.Result[*Post] {
    post := c.PostService.GetPost(id)
    return glib.OK(post)
}
```

#### Controller with Tags (for Middleware Targeting)

```go
// @Controller path=/api/v1/admin tags=protected,admin
type AdminController struct {
    UserService *services.UserService
    Logger      *services.Logger
}

// All routes in this controller can be targeted by middleware with target=protected or target=admin
// @Route method=GET path=/users
func (c *AdminController) ListUsers(ctx context.Context) glib.Result[[]User] {
    users := c.UserService.GetAll()
    return glib.OK(users)
}

// @Route method=DELETE path=/users/{id}
func (c *AdminController) DeleteUser(ctx context.Context, id uuid.UUID) glib.Result[any] {
    c.UserService.Delete(id)
    return glib.NoContent[any]()
}
```

#### Multiple Controllers with Different Tags

```go
// Public API - no special tags
// @Controller path=/api/v1/posts tags=api
type PostsController struct { /*...*/ }

// Protected API - requires authentication
// @Controller path=/api/v1/auth tags=protected
type AuthController struct { /*...*/ }

// Admin API - requires admin role
// @Controller path=/api/v1/admin tags=protected,admin
type AdminController struct { /*...*/ }
```

### Rules

1. **Path must start with `/`** (e.g., `/api`, `/api/v1/posts`)
2. **Tags are used for middleware targeting** (middleware with `target=api` will apply to routes tagged with `api`)
3. **All struct fields auto-injected by type** (no manual wiring)
4. **Controller can have zero or more routes**

---

## @Route

Defines an HTTP endpoint on a controller method.

### Syntax

```go
// @Route method=<METHOD> path=<path> tags=<tag1,tag2> with=<middleware>
func (c *ControllerName) MethodName(...) ... { }
```

### Parameters

- **method** (required): HTTP method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`)
- **path** (required): Path relative to controller prefix
- **tags** (optional): Tags for middleware targeting (in addition to controller tags)
- **with** (optional): Explicit middleware override:
  - `with=middleware1,middleware2` - Use only these middleware (overrides auto-targeting)
  - `with=none` - No middleware at all (not even controller-level)

### Path Parameters

Path parameters are extracted from the URL pattern:

```go
/{id}           // Captured as string by default
/{id}           // Converted to int
/{id}           // Converted to uuid.UUID
/{slug}         // String parameter
```

**Supported types:** `string`, `int`, `int64`, `uint64`, `float64`, `bool`, `uuid.UUID`

The generator automatically generates parsing code based on your method signature.

### Middleware Targeting

Middleware is applied automatically based on tags:

```go
// @Controller path=/api/posts tags=api
type PostsController struct {}

// Middleware with target=all applies to ALL routes
// Middleware with target=api applies to routes with tags=api
// @Route method=GET path=/
func (c *PostsController) Index(...) {}

// Additional protected tag - middleware with target=protected also applies
// @Route method=POST path=/ tags=protected
func (c *PostsController) Create(...) {}

// No middleware at all (even controller-level)
// @Route method=GET path=/health with=none
func (c *PostsController) Health(...) {}

// Explicit middleware override (only logger, ignores auto-targeting)
// @Route method=GET path=/export with=logger
func (c *PostsController) Export(...) {}
```

### Examples

#### Simple GET Endpoint

```go
// @Route method=GET path=/
func (c *PostsController) Index(ctx context.Context) glib.Result[[]Post] {
    posts := c.PostService.GetAll()
    return glib.OK(posts)
}
```

#### Route with Path Parameter

```go
// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id int) glib.Result[*Post] {
    post := c.PostService.GetPost(id)
    if post == nil {
        return glib.NotFound[*Post]("post not found")
    }
    return glib.OK(post)
}
```

#### Route with Multiple Path Parameters

```go
// @Route method=GET path=/{postID}/comments/{commentID}
func (c *Ctrl) GetComment(ctx context.Context, postID uuid.UUID, commentID uuid.UUID) glib.Result[*Comment]
// postID matched by name, commentID matched by name (order doesn't matter)

// @Route GET /{postID}/comments/{commentID}
func (c *Ctrl) Show(ctx context.Context, commentID uuid.UUID, postID uuid.UUID) (*Comment, error)
// ✅ Also valid - parameters matched by name, not position
```

#### POST with Request Body

```go
// @Route POST /
func (c *PostsController) Store(ctx context.Context, req CreatePostRequest) (*Post, error) {
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    c.DB.Create(post)
    return post, nil
}
```

#### Multiple Path Parameters

```go
// @Route GET /posts/{postID}/comments/{commentID}
func (c *CommentsController) Show(
    ctx context.Context,
    postID uuid.UUID,
    commentID uuid.UUID,
) (*Comment, error) {
    var comment Comment
    err := c.DB.Where("post_id = ? AND id = ?", postID, commentID).First(&comment).Error
    return &comment, err
}
```

#### Wildcard Route

```go
// @Route GET /static/{path...}
func (c *StaticController) Serve(ctx context.Context, path string, w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, filepath.Join("public", path))
}
```

### Rules

1. **Path is relative to controller prefix**
2. **Path parameters must have corresponding function parameters**
3. **Middleware names must be registered via `@Middleware`**
4. **One handler function per route**

---

## @Provider

Defines a factory function that creates dependencies for DI.

### Syntax

```go
// @Provider <scope>
func FunctionName(dependencies...) (ReturnType, error)
```

### Parameters

- **scope** (required): `singleton` or `transient`
  - `singleton`: Created once at app startup, shared across requests
  - `transient`: Created fresh every time it's injected

### Auto-Wiring

- Function parameters are auto-injected from other providers
- Return types (non-error) become injectable
- Multiple return values all become injectable
- Function name doesn't matter

### Examples

#### Database Provider

```go
// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.Name,
        cfg.Database.User,
        cfg.Database.Password,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    return db, nil
}
```

#### Cache Provider

```go
// @Provider singleton
func NewCache(cfg *Config) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Host, cfg.Cache.Port),
        Password: cfg.Cache.Password,
        DB:       0,
    })

    // Test connection
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to redis: %w", err)
    }

    return client, nil
}
```

#### Logger Provider (Transient)

```go
// @Provider transient
func NewLogger() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        AddSource: true,
    }))
}
```

#### Service with Dependencies

```go
// @Provider singleton
func NewPostService(db *gorm.DB, cache *redis.Client) *PostService {
    return &PostService{
        db:    db,
        cache: cache,
    }
}

type PostService struct {
    db    *gorm.DB
    cache *redis.Client
}

func (s *PostService) GetPost(id uuid.UUID) (*Post, error) {
    // Check cache first
    key := fmt.Sprintf("post:%s", id)
    if val, err := s.cache.Get(context.Background(), key).Result(); err == nil {
        var post Post
        json.Unmarshal([]byte(val), &post)
        return &post, nil
    }

    // Fetch from database
    var post Post
    if err := s.db.First(&post, id).Error; err != nil {
        return nil, err
    }

    // Store in cache
    data, _ := json.Marshal(post)
    s.cache.Set(context.Background(), key, data, 10*time.Minute)

    return &post, nil
}
```

#### Multiple Return Values

```go
// @Provider singleton
func NewDatabaseConnections(cfg *Config) (*gorm.DB, *sql.DB, error) {
    gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, nil, err
    }

    sqlDB, err := gormDB.DB()
    if err != nil {
        return nil, nil, err
    }

    return gormDB, sqlDB, nil
}

// Now both *gorm.DB and *sql.DB are injectable!
```

### Rules

1. **Must return at least one non-error type**
2. **Last return value must be `error`**
3. **Parameters must be injectable types**
4. **No circular dependencies allowed** (detected at compile-time)
5. **Singleton scope is recommended for expensive resources**

---

## @Middleware

Defines reusable middleware that can be applied to routes automatically or explicitly.

### Syntax

```go
// @Middleware name=<name> target=<target> order=<order>
func FunctionName(dependencies...) func(http.Handler) http.Handler
// OR (new-style glib middleware)
func FunctionName(dependencies...) middleware.Middleware
```

### Parameters

- **name** (required): Middleware identifier (e.g., `name=auth`, `name=logger`)
- **target** (optional): Routes to apply this middleware to:
  - `target=all` - Apply to ALL routes (default if not specified)
  - `target=api` - Apply to routes/controllers with `tags=api`
  - `target=protected` - Apply to routes/controllers with `tags=protected`
  - `target=api,protected` - Apply to routes with either tag
- **order** (optional): Execution order (lower numbers run first, default=0)

### Auto-Targeting Rules

Middleware is automatically applied to routes based on tags:

1. **target=all**: Applied to every single route
2. **target=<tag>**: Applied to routes/controllers with matching tags
3. Routes can override with `with=none` to disable all middleware
4. Routes can override with `with=middleware1,middleware2` to use specific middleware only

### Execution Order

Middleware runs in this order:

1. Sort by `order` (ascending: 1, 2, 3, ...)
2. Within same order, sort by file path
3. Within same file, sort by line number

Example:

```go
// @Middleware name=logger target=all order=1
// @Middleware name=ratelimit target=api order=5
// @Middleware name=auth target=protected order=10
```

For a route with `tags=api,protected`:

- Order: logger (1) → ratelimit (5) → auth (10) → handler

### Two Middleware Styles

#### Old-Style: Standard Go Middleware

```go
// @Middleware name=logger target=all order=1
func Logger() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            log.Printf("[%s] %s - START", r.Method, r.URL.Path)

            next.ServeHTTP(w, r)

            log.Printf("[%s] %s - DONE (%v)", r.Method, r.URL.Path, time.Since(start))
        })
    }
}
```

#### New-Style: Glib Middleware

```go
// @Middleware name=ratelimit target=api order=5
func RateLimit() middleware.Middleware {
    requests := make(map[string][]time.Time)
    limit := 100 // requests per minute

    return func(req middleware.Request, next middleware.Next) glib.Result[any] {
        ip := req.HTTPRequest().RemoteAddr
        now := time.Now()

        // Clean old requests
        if reqs, ok := requests[ip]; ok {
            var recent []time.Time
            for _, t := range reqs {
                if now.Sub(t) < time.Minute {
                    recent = append(recent, t)
                }
            }
            requests[ip] = recent
        }

        // Check limit
        if len(requests[ip]) >= limit {
            return middleware.Error(
                fmt.Errorf("rate limit exceeded"),
                http.StatusTooManyRequests,
            )
        }

        // Record request
        requests[ip] = append(requests[ip], now)

        // Continue to next middleware/handler
        return next(req)
    }
}
```

### Examples

#### Authentication Middleware (Protected Routes Only)

```go
// @Middleware name=auth target=protected order=10
func Auth() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }

            // Token validation would go here
            next.ServeHTTP(w, r)
        })
    }
}
```

#### Global Logger (All Routes)

```go
// @Middleware name=logger target=all order=1
func Logger() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            log.Printf("→ %s %s", r.Method, r.URL.Path)
            next.ServeHTTP(w, r)
            log.Printf("← %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
        })
    }
}
```

#### Rate Limiter (API Routes Only)

```go
// @Middleware name=ratelimit target=api order=5
func RateLimit() middleware.Middleware {
    limiter := rate.NewLimiter(10, 20) // 10 req/sec, burst 20

    return func(req middleware.Request, next middleware.Next) glib.Result[any] {
        if !limiter.Allow() {
            return middleware.Error(
                fmt.Errorf("too many requests"),
                http.StatusTooManyRequests,
            )
        }
        return next(req)
    }
}
```

#### CORS (All Routes, Run First)

```go
// @Middleware name=cors target=all order=0
func CORS() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*")
            w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

#### Middleware with Dependencies

```go
// @Middleware name=logger target=all order=1
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
            next.ServeHTTP(wrapped, r)

            logger.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", wrapped.statusCode,
                "duration", time.Since(start),
            )
        })
    }
}
```

### Complete Example: Middleware Targeting

```go
// middleware/middleware.go
package middleware

// Runs first for ALL routes
// @Middleware name=logger target=all order=1
func Logger() func(http.Handler) http.Handler { /*...*/ }

// Runs for API routes only
// @Middleware name=ratelimit target=api order=5
func RateLimit() middleware.Middleware { /*...*/ }

// Runs for protected routes only
// @Middleware name=auth target=protected order=10
func Auth() func(http.Handler) http.Handler { /*...*/ }

// controllers/post/controller.go
// All routes inherit tags=api
// @Controller path=/api/v1/posts tags=api
type PostsController struct {}

// Middleware: [logger, ratelimit] (target=all + target=api)
// @Route method=GET path=/
func (c *PostsController) Index(...) {}

// Middleware: [logger, ratelimit, auth] (adds tags=protected)
// @Route method=POST path=/ tags=protected
func (c *PostsController) Create(...) {}

// Middleware: [] (explicitly disabled)
// @Route method=GET path=/health with=none
func (c *PostsController) Health(...) {}

// Middleware: [logger] (explicit override, only logger runs)
// @Route method=GET path=/export with=logger
func (c *PostsController) Export(...) {}
```

### Rules

1. **Name must be unique** across all middleware
2. **Target determines auto-application** based on route/controller tags
3. **Order determines execution sequence** (lower = earlier)
4. **Dependencies auto-injected** from providers
5. **Two signatures supported**: Standard Go middleware or glib.Middleware
6. **Generator detects signature automatically** and wraps appropriately

---

## Handler Patterns

Glib supports two main handler patterns based on your method signature.

### Pattern 1: Result[T] Handlers (Recommended)

Type-safe handlers that return `glib.Result[T]` for automatic error handling and response serialization.

```go
// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]Post] {
    posts := c.Service.GetAll()
    return glib.OK(posts)  // Auto-serialized to JSON
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*Post] {
    post := c.Service.GetPost(id)
    if post == nil {
        return glib.NotFound[*Post]("post not found")
    }
    return glib.OK(post)
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreateRequest) glib.Result[*Post] {
    post := c.Service.Create(req)
    return glib.Created(post)  // Returns 201 Created
}

// @Route method=DELETE path=/{id}
func (c *Controller) Delete(ctx context.Context, id int) glib.Result[any] {
    c.Service.Delete(id)
    return glib.NoContent[any]()  // Returns 204 No Content
}
```

**Generated code handles:**

- Path parameter parsing (with type conversion)
- Request body parsing (JSON decode)
- Response serialization (JSON encode)
- Error handling (via `Result.Write(w)`)
- Middleware wrapping

### Pattern 2: Raw HTTP Handlers

Direct access to `http.ResponseWriter` and `*http.Request` for custom response handling.

```go
// @Route method=GET path=/export
func (c *Controller) Export(w http.ResponseWriter, r *http.Request) {
    // Custom response - CSV export
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=posts.csv")
    w.WriteHeader(http.StatusOK)

    fmt.Fprintln(w, "id,title,created_at")
    // Write CSV data...
}

// @Route method=GET path=/stream
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    // Server-Sent Events
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    for {
        select {
        case <-r.Context().Done():
            return
        case <-time.After(1 * time.Second):
            fmt.Fprintf(w, "data: %s\n\n", time.Now().Format(time.RFC3339))
            flusher.Flush()
        }
    }
}

// @Route method=GET path=/health with=none
func (c *Controller) Health(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

**Use raw HTTP handlers for:**

- File downloads/uploads
- Streaming responses (SSE, WebSockets)
- Custom content types (CSV, XML, binary)
- Fine-grained response control
- Health check endpoints

### Comparison

| Feature         | Result[T] Pattern   | Raw HTTP Pattern     |
| --------------- | ------------------- | -------------------- |
| Type safety     | ✅ Full type safety | ❌ Manual handling   |
| Auto JSON       | ✅ Automatic        | ❌ Manual            |
| Error handling  | ✅ Built-in         | ❌ Manual            |
| Path params     | ✅ Auto-parsed      | ❌ Manual extraction |
| Request body    | ✅ Auto-parsed      | ❌ Manual decode     |
| Middleware      | ✅ Supported        | ✅ Supported         |
| Streaming       | ❌ Not suitable     | ✅ Full control      |
| Custom headers  | ❌ Limited          | ✅ Full control      |
| Code generation | More code           | Minimal code         |

### Choosing a Pattern

**Use Result[T] for:**

- Standard CRUD operations
- JSON APIs
- Type-safe error handling
- Less boilerplate code

**Use Raw HTTP for:**

- File operations
- Streaming responses
- Custom content types
- Performance-critical paths
- Health checks

---

## Summary

### Annotation Quick Reference

```go
// Controllers
// @Controller path=</path> tags=<tag1,tag2>
```
