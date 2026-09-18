# 01. Annotations Reference

Complete guide to all annotations and struct tags supported by the Glib code generation engine.

---

## Table of Contents

1. [@Controller](#controller) - Define HTTP controllers
2. [@Route](#route) - Define HTTP endpoints
3. [@Provider](#provider) - Define DI providers
4. [@Middleware](#middleware) - Define HTTP middleware
5. [@Config](#config) - Define environment configurations
6. [Struct Tags Reference](#struct-tags-reference) - Request binding, validation, and response metadata

---

## @Controller

Marks a struct as an HTTP controller with a base route prefix and optional tags for middleware auto-targeting.

### Syntax

```go
// @Controller path=<path-prefix> [tags=<tag1,tag2,...>]
type ControllerName struct {
    // Dependencies - auto-wired by type from registered providers
}
```

### Attributes

- **`path`** (required): Base URL path for all routes in this controller (e.g. `/api/v1/posts`).
- **`tags`** (optional): Comma-separated list of tags used for middleware auto-targeting (e.g. `tags=api,protected`).

### Auto-Wiring

All struct fields in a controller are automatically resolved and injected from registered `@Provider` functions matching their type.

### Example

```go
package posts

import (
    "context"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "my-app/models"
    "my-app/services"
)

// @Controller path=/api/v1/posts tags=api,protected
type Controller struct {
    DB          *gorm.DB                 // Injected singleton provider
    PostService *services.PostService    // Injected singleton provider
    Logger      *services.Logger         // Injected transient provider
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Post, error) {
    return c.PostService.GetByID(id)
}
```

---

## @Route

Marks a method on a controller struct as an HTTP endpoint handler.

### Syntax

```go
// @Route method=<METHOD> path=<path> [tags=<tag1,tag2,...>] [with=<mw1,mw2,...|none>]
func (c *Controller) MethodName(ctx context.Context, [params...]) (T, error)
```

### Attributes

- **`method`** (required): HTTP method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`).
- **`path`** (required): Path relative to the controller's base path (e.g. `/`, `/{id}`, `/export`).
- **`tags`** (optional): Comma-separated tags that inherit and extend controller-level tags for middleware auto-targeting.
- **`with`** (optional):
    - Comma-separated list of middleware names to apply explicitly: `with=auth,ratelimit`
    - `with=none` to bypass all tag-targeted middleware for this specific route.

### Example

```go
// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) ([]models.Post, error) {
    return c.PostService.GetAll()
}

// Route with custom tags
// @Route method=POST path=/ tags=admin
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) (*models.Post, error) {
    return c.PostService.Create(req)
}

// Route with explicit middleware override
// @Route method=GET path=/export with=auth,audit
func (c *Controller) Export(ctx context.Context) ([]models.Post, error) {
    return c.PostService.GetAll()
}

// Route bypassing all middleware
// @Route method=GET path=/health with=none
func (c *Controller) Health(ctx context.Context) (map[string]string, error) {
    return map[string]string{"status": "ok"}, nil
}
```

---

## @Provider

Marks a constructor function as a dependency injection provider.

### Syntax

```go
// @Provider <lifecycle>
func NewService([dependencies...]) (ProvidedType, error)
// or
func NewService([dependencies...]) ProvidedType
```

### Lifecycles

- **`singleton`**: Instantiated once when `InitContainer(ctx)` is called. The single instance is shared across the entire application.
- **`transient`**: A provider factory is generated; injected controllers receive a new instance per controller initialization.

### Rules

1. Functions must return `(T, error)` or `(T)`.
2. Parameters are resolved by type from other registered `@Provider` or `@Config` types.
3. Dependencies must form a Directed Acyclic Graph (DAG). Circular dependencies produce compile-time generation errors.

### Example

```go
package services

import (
    "fmt"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "my-app/configs"
)

// @Provider singleton
func NewDatabase(cfg *configs.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
        cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
        cfg.Database.User, cfg.Database.Password)
    return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// @Provider singleton
func NewPostService(db *gorm.DB) *PostService {
    return &PostService{db: db}
}

// @Provider transient
func NewLogger() *Logger {
    return &Logger{}
}
```

---

## @Middleware

Marks a middleware constructor function.

### Syntax

```go
// @Middleware name=<name> [target=<target>] [order=<order>]
func NewMiddleware([dependencies...]) func(glib.Request, glib.Next) glib.Response
// or standard signature:
func NewMiddleware([dependencies...]) func(http.Handler) http.Handler
```

### Attributes

- **`name`** (required): Unique identifier for the middleware (referenced in `with=<name>`).
- **`target`** (optional, default `all`): Target tag determining which routes this middleware applies to (`all`, `protected`, `api`, etc.).
- **`order`** (optional, default `100`): Integer priority for execution order (lower numbers execute first in the chain).

### Signatures

1. **Native Glib Middleware** (Recommended):
    ```go
    func(glib.Request, glib.Next) glib.Response
    ```
2. **Standard HTTP/Chi Middleware**:
    ```go
    func(http.Handler) http.Handler
    ```

### Example

```go
package middleware

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
    "my-app/services"
)

// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        token := req.Header("Authorization")
        if token == "" {
            return glib.Response{
                Err: errs.NewUnauthorized().WithMessage("missing token"),
            }
        }

        claims, err := jwtService.Validate(token)
        if err != nil {
            return glib.Response{Err: err}
        }

        req = req.WithValue("user_id", claims.UserID)
        return next(req)
    }
}
```

---

## @Config

Marks a configuration struct to be automatically loaded from environment variables on startup.

### Syntax

```go
// @Config
type Config struct {
    // Fields with env and default tags
}
```

### Struct Tags for Config

- **`env:"VAR_NAME"`**: Environment variable name to bind to this field.
- **`default:"value"`**: Default value if the environment variable is not set.

### Example

```go
package configs

import "fmt"

// @Config
type Config struct {
    Server struct {
        Host string `env:"APP_HOST" default:"0.0.0.0"`
        Port int    `env:"APP_PORT" default:"8080"`
    }
    Database struct {
        Host     string `env:"DB_HOST" default:"localhost"`
        Port     int    `env:"DB_PORT" default:"5432"`
        User     string `env:"DB_USER" default:"postgres"`
        Password string `env:"DB_PASSWORD" default:"secret"`
        Name     string `env:"DB_NAME" default:"app_db"`
    }
}

func (c *Config) Addr() string {
    return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
```

---

## Struct Tags Reference

### 1. Request Struct Tags

| Tag              | Purpose                                                    | Example                                                   |
| :--------------- | :--------------------------------------------------------- | :-------------------------------------------------------- |
| `json:"..."`     | Binds field from JSON request body                         | `Title string \`json:"title"\``                           |
| `query:"..."`    | Binds field from URL query string                          | `Page int \`query:"page"\``                               |
| `header:"..."`   | Binds field from incoming HTTP request header              | `APIKey string \`header:"X-API-Key"\``                    |
| `validate:"..."` | Defines validation rules via `go-playground/validator/v10` | `Email string \`json:"email" validate:"required,email"\`` |

```go
type ListPostsRequest struct {
    Page    int    `query:"page" validate:"required,min=1"`
    PerPage int    `query:"per_page" validate:"required,min=1,max=100"`
    Search  string `query:"search" validate:"omitempty,max=50"`
    APIKey  string `header:"X-API-Key" validate:"required"`
}
```

### 2. Response Struct Tags (Metadata)

| Tag                       | Purpose                                                 | Example                                   |
| :------------------------ | :------------------------------------------------------ | :---------------------------------------- |
| `response:"httpstatus"`   | Sets the HTTP response status code (must be `int` type) | `Status int \`response:"httpstatus"\``    |
| `header:"Name"`           | Sets an HTTP response header                            | `Location string \`header:"Location"\``   |
| `header:"Name,omitempty"` | Sets an HTTP response header only if value is non-empty | `ETag string \`header:"ETag,omitempty"\`` |

```go
type CreatePostResponse struct {
    StatusCode int       `json:"-" response:"httpstatus"`
    Location   string    `json:"-" header:"Location"`
    ETag       string    `json:"-" header:"ETag,omitempty"`

    ID         uuid.UUID `json:"id"`
    Title      string    `json:"title"`
}
```
