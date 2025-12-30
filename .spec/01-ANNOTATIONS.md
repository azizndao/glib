# Glib 2.0 - Annotation Reference

Complete guide to all annotations supported by Glib 2.0 code generator.

---

## Table of Contents

1. [@Controller](#controller) - Define HTTP controllers
2. [@Route](#route) - Define HTTP endpoints
3. [@Provider](#provider) - Define DI providers
4. [@Middleware](#middleware) - Define middleware
5. [Request Struct Tags](#request-struct-tags) - Parse HTTP requests
6. [Config Schema](#config-schema) - Type-safe configuration

---

## @Controller

Marks a struct as an HTTP controller with optional prefix and middleware.

### Syntax

```go
// @Controller <path-prefix>
// @Middleware <middleware1>,<middleware2>,... (optional)
type ControllerName struct {
    // Dependencies - auto-wired by type
}
```

### Parameters

- **path-prefix** (required): Base path for all routes in this controller
- **@Middleware** (optional): Comma-separated middleware names applied to ALL routes

### Auto-Wiring

All struct fields are automatically injected by type from registered providers. **No tags needed!**

### Examples

#### Simple Controller (No Prefix, No Middleware)

```go
// @Controller /
type HomeController struct {}

// @Route GET /
func (c *HomeController) Index(ctx context.Context) (*HomeResponse, error) {
    return &HomeResponse{Message: "Welcome"}, nil
}
```

#### API Controller with Prefix

```go
// @Controller /api/v1/posts
type PostsController struct {
    DB *gorm.DB  // Auto-injected from provider
}

// Full path: GET /api/v1/posts
// @Route GET /
func (c *PostsController) Index(ctx context.Context) ([]Post, error) {
    var posts []Post
    c.DB.Find(&posts)
    return posts, nil
}

// Full path: GET /api/v1/posts/{id}
// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    var post Post
    c.DB.First(&post, id)
    return &post, nil
}
```

#### Controller with Prefix and Middleware

```go
// @Controller /api/v1/admin
// @Middleware auth,admin
type AdminController struct {
    DB     *gorm.DB
    Logger *slog.Logger
}

// Middleware: [auth, admin] (from controller)
// @Route GET /users
func (c *AdminController) ListUsers(ctx context.Context) ([]User, error) {
    var users []User
    c.DB.Find(&users)
    return users, nil
}

// Middleware: [auth, admin, ratelimit] (controller + route)
// @Route DELETE /users/{id}
// @Middleware ratelimit
func (c *AdminController) DeleteUser(ctx context.Context, id uuid.UUID) error {
    return c.DB.Delete(&User{}, id).Error
}
```

### Rules

1. **Prefix must start with `/`** (e.g., `/api`, `/api/v1/posts`)
2. **Middleware names must match `@Middleware` declarations**
3. **All struct fields auto-injected** (no manual wiring)
4. **Controller can have zero or more routes**

---

## @Route

Defines an HTTP endpoint on a controller method.

### Syntax

```go
// @Route <METHOD> <path>
// @Middleware <middleware1>,<middleware2>,... (optional)
func (c *ControllerName) MethodName(...) ... { }
```

### Parameters

- **METHOD** (required): HTTP method (GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD)
- **path** (required): Path relative to controller prefix
- **@Middleware** (optional): Additional middleware (appends to controller middleware)

### Path Parameters

Use `:name` or `:name:type` for path parameters:

```go
/{id}                  // String parameter (default)
/{id}             // Integer parameter
/{id}            // UUID parameter
/:slug:string        // Explicit string
/{path...}               // Wildcard (matches rest of path)
```

**Supported types:** `string`, `int`, `int64`, `uuid`

### Middleware Behavior

By default, route middleware **appends** to controller middleware:

```go
// @Controller /api/posts
// @Middleware auth
type PostsController struct {}

// Middleware chain: [auth]
// @Route GET /
func (c *PostsController) Index(...) {}

// Middleware chain: [auth, ratelimit]
// @Route POST /
// @Middleware ratelimit
func (c *PostsController) Store(...) {}

// Middleware chain: [] (cleared)
// @Route GET /public
// @Middleware none
func (c *PostsController) Public(...) {}
```

Use `@Middleware none` to clear all controller middleware.

### Examples

#### Simple GET Endpoint

```go
// @Route GET /
func (c *PostsController) Index(ctx context.Context) ([]Post, error) {
    var posts []Post
    c.DB.Find(&posts)
    return posts, nil
}
```

#### Route with Path Parameter

```go
// @Route GET /{postID}/comments/{commentID}
func (c *Ctrl) Show(ctx context.Context, postID uuid.UUID, commentID uuid.UUID) (*Comment, error)
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

Defines reusable middleware that can be applied to routes.

### Syntax

```go
// @Middleware <name>
func FunctionName() func(http.Handler) http.Handler
```

### Parameters

- **name** (required): Middleware identifier used in `@Controller` and `@Route`

### Signature

Must return `func(http.Handler) http.Handler` - standard Go middleware pattern.

### Examples

#### Authentication Middleware

```go
// @Middleware auth
func AuthMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
                return
            }
            
            // Validate token
            claims, err := validateJWT(token)
            if err != nil {
                http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
                return
            }
            
            // Add user to context
            ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### Rate Limiting Middleware

```go
// @Middleware ratelimit
func RateLimitMiddleware() func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(10, 20) // 10 req/sec, burst 20
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, `{"error":"Too many requests"}`, http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

#### CORS Middleware

```go
// @Middleware cors
func CORSMiddleware() func(http.Handler) http.Handler {
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

#### Logging Middleware with Dependencies

```go
// @Middleware logger
func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap response writer to capture status
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

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

#### Admin Authorization Middleware

```go
// @Middleware admin
func AdminMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Context().Value("user_id")
            if userID == nil {
                http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
                return
            }
            
            // Check if user is admin (query database, check claims, etc.)
            isAdmin := checkIfUserIsAdmin(userID)
            if !isAdmin {
                http.Error(w, `{"error":"Forbidden - Admin access required"}`, http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Rules

1. **Must return standard Go middleware signature**
2. **Can have dependencies injected** (like controllers)
3. **Name must be unique across application**
4. **Middleware applied in order specified**

---

## Request Struct Tags

Define how fields are parsed from HTTP requests.

### Supported Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `param:"name"` | URL path parameter | `ID uuid.UUID \`param:"id"\`` |
| `query:"name"` | Query string parameter | `Page int \`query:"page"\`` |
| `header:"Name"` | HTTP header | `Auth string \`header:"Authorization"\`` |
| `json:"name"` | Request body field | `Title string \`json:"title"\`` |
| `default:"value"` | Default value | `Page int \`query:"page" default:"1"\`` |
| `validate:"rules"` | Validation rules | `Email string \`validate:"required,email"\`` |

### Tag Precedence

1. `param` - highest priority
2. `header`
3. `query`
4. `json` - default for fields without other tags

### Examples

#### Simple Request (Query Parameters)

```go
type ListPostsRequest struct {
    Page     int    `query:"page" default:"1" validate:"min=1"`
    PageSize int    `query:"page_size" default:"20" validate:"min=1,max=100"`
    Sort     string `query:"sort" default:"created_at" validate:"oneof=created_at title"`
    Order    string `query:"order" default:"desc" validate:"oneof=asc desc"`
    Search   string `query:"search"`
}

// Usage: GET /posts?page=2&page_size=50&sort=title&order=asc&search=golang
```

#### Create Request (Body + Headers)

```go
type CreatePostRequest struct {
    // Headers
    Authorization string    `header:"Authorization" validate:"required"`
    ContentType   string    `header:"Content-Type" validate:"required"`
    
    // Body (JSON)
    Title    string   `json:"title" validate:"required,min=3,max=200"`
    Content  string   `json:"content" validate:"required,min=10"`
    Tags     []string `json:"tags" validate:"dive,min=2"`
    Draft    bool     `json:"draft" default:"false"`
}

// Usage: POST /posts
// Headers: Authorization: Bearer xxx, Content-Type: application/json
// Body: {"title":"My Post","content":"Post content...","tags":["go","web"]}
```

#### Update Request (Path + Query + Body)

```go
type UpdatePostRequest struct {
    // Path parameter (matched from route /{id})
    ID uuid.UUID `param:"id" validate:"required"`
    
    // Query parameter
    Reason string `query:"reason"`
    
    // Body
    Title   string `json:"title" validate:"required,min=3"`
    Content string `json:"content" validate:"required,min=10"`
}

// Usage: PUT /posts/123e4567-e89b-12d3-a456-426614174000?reason=typo
// Body: {"title":"Updated Title","content":"Updated content..."}
```

#### Complex Request (All Sources)

```go
type ComplexRequest struct {
    // Path parameters
    PostID    uuid.UUID `param:"postId" validate:"required"`
    CommentID uuid.UUID `param:"commentId" validate:"required"`
    
    // Query parameters
    Include string `query:"include" default:"author"`
    Format  string `query:"format" default:"json" validate:"oneof=json xml"`
    
    // Headers
    Authorization string `header:"Authorization" validate:"required"`
    IfMatch       string `header:"If-Match"`
    UserAgent     string `header:"User-Agent"`
    
    // Body
    Content  string   `json:"content" validate:"required,min=1,max=5000"`
    Mentions []string `json:"mentions" validate:"dive,uuid"`
}
```

### Validation Rules

Uses [go-playground/validator](https://github.com/go-playground/validator). Common rules:

- `required` - Field must be present
- `email` - Valid email format
- `min=n` - Minimum value/length
- `max=n` - Maximum value/length
- `oneof=a b c` - Must be one of values
- `uuid` - Valid UUID
- `url` - Valid URL
- `dive` - Validate slice/array elements

**Validation happens before handler is called. Returns 400 Bad Request if validation fails.**

---

## Config Schema

Define type-safe configuration by creating a `Config` struct.

### Location

Can be **anywhere** in your project - scanner auto-discovers `type Config struct`.

Recommended location: `config/config.go`

### Supported Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `env:"NAME"` | Environment variable | `Port int \`env:"PORT"\`` |
| `default:"value"` | Default value | `Port int \`env:"PORT" default:"8080"\`` |
| `required:"true"` | Must be set | `DBHost string \`env:"DB_HOST" required:"true"\`` |
| `validate:"rules"` | Validation rules | `Env string \`env:"ENV" validate:"oneof=dev prod"\`` |

### Example

```go
package config

type Config struct {
    App struct {
        Name        string `env:"APP_NAME" default:"myapp"`
        Environment string `env:"APP_ENV" default:"development" validate:"oneof=development staging production"`
        Port        int    `env:"APP_PORT" default:"8080" validate:"min=1,max=65535"`
        Host        string `env:"APP_HOST" default:"localhost"`
        Debug       bool   `env:"APP_DEBUG" default:"false"`
    }
    
    Database struct {
        Driver   string `env:"DB_DRIVER" default:"postgres" validate:"oneof=postgres mysql sqlite"`
        Host     string `env:"DB_HOST" required:"true"`
        Port     int    `env:"DB_PORT" default:"5432"`
        Name     string `env:"DB_NAME" required:"true"`
        User     string `env:"DB_USER" required:"true"`
        Password string `env:"DB_PASSWORD" required:"true"`
        SSLMode  string `env:"DB_SSL_MODE" default:"disable"`
        MaxConns int    `env:"DB_MAX_CONNS" default:"10" validate:"min=1,max=100"`
    }
    
    Cache struct {
        Driver   string `env:"CACHE_DRIVER" default:"redis" validate:"oneof=redis memory"`
        Host     string `env:"CACHE_HOST" default:"localhost"`
        Port     int    `env:"CACHE_PORT" default:"6379"`
        Password string `env:"CACHE_PASSWORD"`
        DB       int    `env:"CACHE_DB" default:"0"`
    }
    
    Storage struct {
        Driver    string `env:"STORAGE_DRIVER" default:"local" validate:"oneof=local s3"`
        LocalPath string `env:"STORAGE_LOCAL_PATH" default:"./storage"`
        S3Bucket  string `env:"STORAGE_S3_BUCKET"`
        S3Region  string `env:"STORAGE_S3_REGION"`
    }
    
    JWT struct {
        Secret     string        `env:"JWT_SECRET" required:"true"`
        Expiration time.Duration `env:"JWT_EXPIRATION" default:"24h"`
    }
}
```

### Generated Code

Code generator creates `generated/config_gen.go`:

```go
// LoadConfig loads and validates application configuration
func LoadConfig() (*config.Config, error) {
    // Load .env file if exists (ignore error if not found)
    _ = godotenv.Load()
    
    cfg := &config.Config{}
    
    // Parse environment variables
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    // Validate configuration
    validate := validator.New()
    if err := validate.Struct(cfg); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return cfg, nil
}
```

### Usage in Providers

```go
// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error) {
    // Type-safe access!
    dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
        cfg.Database.Host,    // Type: string
        cfg.Database.Port,    // Type: int
        cfg.Database.Name,    // Type: string
        cfg.Database.User,    // Type: string
        cfg.Database.Password // Type: string
    )
    return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
```

### .env File Example

```env
# App
APP_NAME=myblog
APP_ENV=development
APP_PORT=8080
APP_HOST=0.0.0.0

# Database
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myblog
DB_USER=postgres
DB_PASSWORD=secret
DB_SSL_MODE=disable

# Cache
CACHE_DRIVER=redis
CACHE_HOST=localhost
CACHE_PORT=6379

# Storage
STORAGE_DRIVER=local
STORAGE_LOCAL_PATH=./storage

# JWT
JWT_SECRET=my-super-secret-key
JWT_EXPIRATION=24h
```

---

## Summary

| Annotation | Purpose | Scope |
|------------|---------|-------|
| `@Controller` | Define HTTP controller | Struct |
| `@Route` | Define HTTP endpoint | Method |
| `@Provider` | Define DI provider | Function |
| `@Middleware` | Define middleware | Function |
| Request Tags | Parse HTTP request | Struct Field |
| Config Tags | Load configuration | Struct Field |

All annotations are **discovered automatically** - no manual registration needed!
