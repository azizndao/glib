# Glib

**Code-generation-first web framework for Go**

Glib uses annotation-based code generation to eliminate web boilerplate in Go. Annotate your controllers, routes, dependency providers, and middleware with simple comments, and Glib generates type-safe HTTP routing, dependency injection, parameter binding, request validation, and error serialization.

---

## Features

- **Annotation-Driven**: Define controllers, routes, middleware, and dependency injection providers with clean Go comments.
- **Code Generation First**: Fast AST-based scanner compiles type-safe routing, DI container, and parameter parsers with no runtime reflection.
- **Idiomatic Go Handlers**: Use natural Go handler signatures (`(T, error)`, `error`, or raw `(http.ResponseWriter, *http.Request)`).
- **Built-in Dependency Injection**: Supports `singleton` and `transient` lifecycles with automatic topological dependency resolution.
- **Automatic Request Binding & Validation**: Automatically binds path parameters, query parameters (`query:`), headers (`header:`), and JSON bodies (`json:`) with `validate:` tags (`go-playground/validator/v10`).
- **Response Metadata Support**: Control response HTTP status codes and response headers via struct tags (`response:"httpstatus"`, `header:"..."`).
- **First-Class Middleware**: Type-safe native middleware (`func(glib.Request, glib.Next) glib.Response`) or standard Chi/HTTP middleware with tag-based auto-targeting and explicit overrides (`with=...`).
- **Structured Error Handling**: Encore-style structured error codes with automatic HTTP status mapping, builder helpers (`errs.B()`), and Goyave-style nested validation error details.
- **Built-in Internationalization (i18n)**: Generate type-safe translators for errors, success messages, and validation messages from TOML locale files.
- **Configuration Management**: Declare app configs using `@Config` structs with `env:` and `default:` tags.
- **Native Hot Reload**: `glib dev` provides instant file watching, incremental code regeneration, and server restart.
- **Developer CLI**: Complete toolchain to scaffold components (`glib make`), validate annotations (`glib validate`), and generate code (`glib generate`).

---

## Quick Start

### Installation

```bash
go install github.com/azizndao/glib/cmd/glib@latest
```

### Create a New Project

```bash
# Initialize a new project with example health controller
glib init my-app --example
cd my-app

# Generate boilerplate for controllers, providers, or middleware
glib make controller posts
glib make provider database

# Generate all code
glib generate

# Run the server
go run .
```

### Development Mode with Hot Reload

```bash
glib dev
```

`glib dev` will:

1. Scan project annotations and generate code incrementally.
2. Build and start your server.
3. Watch for `.go` file modifications with debouncing.
4. Auto-regenerate code and restart the server on changes.

---

## End-to-End Example

### 1. Define a Controller

Controllers are structs annotated with `// @Controller`. Dependencies are automatically injected by type. Handlers use standard Go return signatures `(T, error)` or `error`.

```go
// controllers/posts/controller.go
package posts

import (
    "context"
    "errors"

    "uuid"
    "gorm.io/gorm"

    "github.com/azizndao/glib/errs"
    "my-app/models"
    "my-app/services"
)

// @Controller path=/api/v1/posts tags=api
type Controller struct {
    DB     *gorm.DB         // Auto-injected singleton provider
    Logger *services.Logger // Auto-injected transient provider
}

// Request & query models with validation tags
type ListPostsQuery struct {
    Page    int    `query:"page" validate:"required,min=1"`
    PerPage int    `query:"per_page" validate:"required,min=1,max=100"`
    Search  string `query:"q" validate:"omitempty,max=50"`
}

type CreatePostRequest struct {
    Title   string    `json:"title" validate:"required,min=3,max=200"`
    Content string    `json:"content" validate:"required,min=10"`
    Tags    []string  `json:"tags" validate:"omitempty,dive,min=2"`
    AuthorID uuid.UUID `json:"author_id" validate:"required,uuid4"`
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context, query ListPostsQuery) ([]models.Post, error) {
    var posts []models.Post
    db := c.DB.Limit(query.PerPage).Offset((query.Page - 1) * query.PerPage)
    if query.Search != "" {
        db = db.Where("title LIKE ?", "%"+query.Search+"%")
    }
    if err := db.Find(&posts).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to fetch posts").Cause(err).Err()
    }
    return posts, nil
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Post, error) {
    var post models.Post
    if err := c.DB.First(&post, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errs.NewNotFound().WithMessage("post not found")
        }
        return nil, errs.B().Code(errs.Internal).Msg("database error").Cause(err).Err()
    }
    return &post, nil
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) (*models.Post, error) {
    post := &models.Post{
        ID:       uuid.New(),
        Title:    req.Title,
        Content:  req.Content,
        AuthorID: req.AuthorID,
    }

    if err := c.DB.Create(post).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to create post").Cause(err).Err()
    }

    return post, nil // POST automatically responds with HTTP 201 Created
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error {
    res := c.DB.Delete(&models.Post{}, "id = ?", id)
    if res.Error != nil {
        return errs.B().Code(errs.Internal).Msg("failed to delete post").Cause(res.Error).Err()
    }
    if res.RowsAffected == 0 {
        return errs.NewNotFound().WithMessage("post not found")
    }
    return nil // DELETE error-only handler automatically responds with HTTP 204 No Content
}
```

### 2. Define Providers (Dependency Injection)

Providers are constructor functions annotated with `// @Provider <lifecycle>`. Glib supports `singleton` and `transient` lifecycles.

```go
// services/database.go
package services

import (
    "fmt"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "my-app/configs"
)

// @Provider singleton
func NewDatabase(cfg *configs.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.Name,
        cfg.Database.User,
        cfg.Database.Password,
    )
    return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
```

```go
// services/logger.go
package services

import (
    "log/slog"
    "os"
)

type Logger struct {
    *slog.Logger
}

// @Provider transient
func NewLogger() *Logger {
    return &Logger{
        Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
    }
}
```

### 3. Define Middleware

Glib supports native middleware (`func(glib.Request, glib.Next) glib.Response`) as well as standard `func(http.Handler) http.Handler` signatures.

```go
// middleware/auth.go
package middleware

import (
    "strings"

    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
    "my-app/services"
)

// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        authHeader := req.Header("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            return glib.Response{
                Err: errs.NewUnauthorized().WithMessage("authorization token required"),
            }
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            return glib.Response{Err: err}
        }

        // Attach user info to request context
        req = req.WithValue("user_id", claims.UserID)

        // Proceed down middleware chain
        resp := next(req)

        // Post-processing: attach response header
        resp.Header().Set("X-User-ID", claims.UserID.String())
        return resp
    }
}
```

### 4. Define Configuration

Declare application configurations using `@Config` structs with `env:` and `default:` tags.

```go
// configs/config.go
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
        Name     string `env:"DB_NAME" default:"app_db"`
        User     string `env:"DB_USER" default:"postgres"`
        Password string `env:"DB_PASSWORD" default:"secret"`
    }
}

func (c *Config) Addr() string {
    return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
```

### 5. Application Bootstrap (`main.go` & `bootstrap.go`)

Glib generates an `InitContainer(ctx)` function that initializes all providers, controllers, middleware, and the routing tree (`*generated.App`).

```go
// bootstrap.go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5/middleware"
    "my-app/generated"
)

func Bootstrap(ctx context.Context) (*http.Server, error) {
    // 1. Initialize DI container & router
    app, err := generated.InitContainer(ctx)
    if err != nil {
        return nil, err
    }

    // 2. Add global Chi middleware
    app.Router.Use(middleware.RequestID)
    app.Router.Use(middleware.RealIP)
    app.Router.Use(middleware.Logger)
    app.Router.Use(middleware.Recoverer)

    // 3. Register generated routes
    if err := app.RegisterRoutes(); err != nil {
        return nil, err
    }

    return &http.Server{
        Addr:         app.Config.Addr(),
        Handler:      app.Router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
    }, nil
}
```

```go
// main.go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    ctx := context.Background()
    server, err := Bootstrap(ctx)
    if err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }

    go func() {
        log.Printf("🚀 Server running on %s", server.Addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = server.Shutdown(shutdownCtx)
}
```

---

## Annotations Reference

### `@Controller`

Marks a struct as an HTTP controller.

```go
// @Controller path=/api/v1/posts tags=api,protected
type PostsController struct {
    DB *gorm.DB // Injected from provider
}
```

- `path`: Base route prefix for all handlers in this controller (e.g. `/api/v1/posts`).
- `tags`: Comma-separated tags used for middleware auto-targeting (e.g. `api,protected`).

### `@Route`

Marks a controller method as an HTTP route handler.

```go
// @Route method=GET path=/{id} tags=protected with=auth,ratelimit
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error)
```

- `method`: HTTP method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`).
- `path`: Route path relative to the controller's prefix (e.g. `/`, `/{id}`, `/search`).
- `tags`: Optional comma-separated tags for middleware targeting.
- `with`: Explicit middleware list (e.g. `with=auth,ratelimit`) or `with=none` to bypass targeted middleware.

### `@Provider`

Marks a constructor function as a dependency provider for the DI container.

```go
// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error)

// @Provider transient
func NewLogger() *Logger
```

- `singleton`: Instantiated once during `InitContainer` and reused across all requests and controllers.
- `transient`: A factory is generated; injected controllers receive a fresh instance per controller initialization.
- Return types can be `(T, error)` or `T`.

### `@Middleware`

Marks a middleware constructor function.

```go
// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response
```

- `name`: Unique name of the middleware.
- `target`: Tag to target automatically (`all`, `protected`, `api`, etc.).
- `order`: Integer execution priority (lower executes earlier, default `100`).

### `@Config`

Marks a configuration struct to be automatically loaded from environment variables during startup.

```go
// @Config
type Config struct {
    Port int    `env:"PORT" default:"8080"`
    Host string `env:"HOST" default:"localhost"`
}
```

---

## Handler Patterns

Glib supports 3 handler signatures:

### 1. Data & Error Pattern: `(T, error)`

The standard pattern for endpoints that return JSON data.

```go
func (c *Controller) Handler(ctx context.Context, [params...]) (T, error)
```

- **Default Status Code**:
    - `POST` handlers return `201 Created`
    - `GET`, `PUT`, `PATCH` handlers return `200 OK`
- When `error` is non-nil, Glib serializes the structured error with the matching HTTP status code.

### 2. Error Only Pattern: `error`

Ideal for operations that return no content on success (e.g., `DELETE` or empty `PUT`/`POST`).

```go
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error
```

- **Default Status Code**:
    - `DELETE` handlers return `204 No Content`
    - When returning `nil`, an empty HTTP response is sent with no body.

### 3. Raw HTTP Pattern: `(http.ResponseWriter, *http.Request)`

For low-level control over streaming, Server-Sent Events (SSE), WebSockets, file downloads, or custom protocols.

```go
// @Route method=GET path=/events
func (c *Controller) Events(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher, _ := w.(http.Flusher)
    for {
        select {
        case <-r.Context().Done():
            return
        case msg := <-c.EventChan:
            fmt.Fprintf(w, "data: %s\n\n", msg)
            flusher.Flush()
        }
    }
}
```

---

## Request Binding & Validation

Glib analyzes your handler parameters and automatically generates binding and validation code.

### Path Parameters

Parameters typed as primitives (`int`, `string`, `int64`, etc.) or `uuid.UUID` with names matching common path identifiers (`id`, `uuid`, `key`, `slug`, `*Id`, `*Key`) are automatically extracted from URL path wildcards:

```go
// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*Post, error)
```

### Query & Header Structs

Define a struct with `query:` and `header:` tags to parse query strings and request headers:

```go
type SearchFilter struct {
    Query     string `query:"q" validate:"required,min=2"`
    Limit     int    `query:"limit" validate:"omitempty,min=1,max=100"`
    ClientVer string `header:"X-Client-Version"`
}

// @Route method=GET path=/search
func (c *Controller) Search(ctx context.Context, filter SearchFilter) ([]Post, error)
```

### JSON Request Bodies

Structs with `json:` tags (or without query/header tags) are parsed from the JSON request body:

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,min=3,max=30"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
}

// @Route method=POST path=/users
func (c *Controller) Create(ctx context.Context, req CreateUserRequest) (*User, error)
```

### Conditional Validation (`validator.Validable`)

Implement `validator.Validable` to conditionally enable or skip validation:

```go
func (r CreateUserRequest) Validate() bool {
    return true // Return false to bypass validation tags
}
```

### Validation Error Format

Validation errors produce structured Goyave-style responses with HTTP status `400 Bad Request`:

```json
{
    "error": {
        "code": "invalid_argument",
        "message": "Validation failed",
        "details": {
            "body": {
                "email": {
                    "errors": ["email must be a valid email address"]
                },
                "username": {
                    "errors": ["username must be at least 3 characters"]
                }
            }
        }
    }
}
```

---

## Response Metadata (Headers & Status Codes)

Customize response HTTP status codes and headers by tagging fields on your response struct:

```go
type CreatePostResponse struct {
    StatusCode int       `json:"-" response:"httpstatus"`  // Custom HTTP status
    Location   string    `json:"-" header:"Location"`      // Set Location header
    ETag       string    `json:"-" header:"ETag,omitempty"` // Set ETag header if non-empty

    ID         uuid.UUID `json:"id"`
    Title      string    `json:"title"`
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) (*CreatePostResponse, error) {
    post := c.service.Create(req)
    return &CreatePostResponse{
        StatusCode: 201,
        Location:   "/api/v1/posts/" + post.ID.String(),
        ETag:       `"v1-hash"`,
        ID:         post.ID,
        Title:      post.Title,
    }, nil
}
```

---

## Error Handling

Glib features structured errors via `github.com/azizndao/glib/errs` inspired by Encore.dev.

### Error Codes & HTTP Status Mapping

| Error Code                | HTTP Status                 | Description                                   |
| :------------------------ | :-------------------------- | :-------------------------------------------- |
| `errs.InvalidArgument`    | `400 Bad Request`           | Client specified invalid argument             |
| `errs.FailedPrecondition` | `400 Bad Request`           | System not in state required for execution    |
| `errs.OutOfRange`         | `400 Bad Request`           | Operation out of valid range                  |
| `errs.Unauthenticated`    | `401 Unauthorized`          | Request lacks valid credentials               |
| `errs.PermissionDenied`   | `403 Forbidden`             | Caller lacks permission                       |
| `errs.NotFound`           | `404 Not Found`             | Requested resource not found                  |
| `errs.AlreadyExists`      | `409 Conflict`              | Resource already exists                       |
| `errs.Aborted`            | `409 Conflict`              | Operation aborted due to concurrency conflict |
| `errs.ResourceExhausted`  | `429 Too Many Requests`     | Rate limit or quota exceeded                  |
| `errs.Canceled`           | `499 Client Closed Request` | Operation canceled by caller                  |
| `errs.Internal`           | `500 Internal Server Error` | Unexpected internal server error              |
| `errs.Unknown`            | `500 Internal Server Error` | Unknown error                                 |
| `errs.DataLoss`           | `500 Internal Server Error` | Unrecoverable data loss                       |
| `errs.Unimplemented`      | `501 Not Implemented`       | Operation not implemented                     |
| `errs.Unavailable`        | `503 Service Unavailable`   | Service temporarily unavailable               |
| `errs.DeadlineExceeded`   | `504 Gateway Timeout`       | Operation deadline expired                    |

### Error Construction Helpers

```go
// 1. Shorthand helper constructors
err := errs.NewNotFound().WithMessage("user not found")
err := errs.NewUnauthorized().WithMessage("invalid token")
err := errs.NewBadRequest().WithMessage("malformed request")
err := errs.NewForbidden().WithMessage("access denied")
err := errs.NewConflict().WithMessage("email already registered")
err := errs.NewInternal().WithMessage("something went wrong")

// 2. Fluent Builder Pattern
err := errs.B().
    Code(errs.PermissionDenied).
    Msg("user does not have admin permissions").
    Cause(underlyingErr).
    Err()

// 3. Wrapping existing errors
err := errs.Wrap(sqlErr, "database query failed")
err := errs.WrapCode(sqlErr, errs.NotFound, "record not found")
```

---

## Internationalization (i18n)

Glib includes built-in code generation for typed translation of errors, success responses, and validation messages.

### 1. Define Locale Files (`locales/en.toml`, `locales/fr.toml`)

```toml
# locales/en.toml
[errors.posts]
not_found = "Post with ID '{id}' was not found"

[success]
post_created = "Post '{title}' created successfully"
```

### 2. Configure i18n in `.config.toml`

```toml
[i18n]
enabled = true
locales_dir = "locales"
default_locale = "en"
supported_locales = ["en", "fr"]
detect_from = ["header", "query"]
query_param = "lang"
```

### 3. Use in Controllers & Services

```go
// @Controller path=/api/v1/posts
type Controller struct {
    I18n *i18n.Translator // Auto-injected
}

func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    post, err := c.service.Find(id)
    if err != nil {
        msg := c.I18n.Errors.Posts.NotFound(ctx, id.String())
        return nil, errs.NewNotFound().WithMessage(msg)
    }
    return post, nil
}
```

---

## CLI Commands Reference

| Command                        | Description                                | Flags                                                                                         |
| :----------------------------- | :----------------------------------------- | :-------------------------------------------------------------------------------------------- |
| `glib init [dir]`              | Initialize a new Glib project              | `--module <name>`, `--example`, `--minimal`                                                   |
| `glib make controller <name>`  | Generate a controller and models file      | `--path <dir>`, `--prefix <route>`, `--no-example`                                            |
| `glib make provider <name>`    | Generate a DI provider boilerplate         | `--path <dir>`, `--no-example`                                                                |
| `glib make middleware <name>`  | Generate a middleware boilerplate          | `--path <dir>`, `--no-example`                                                                |
| `glib generate` (alias: `gen`) | Scan annotations and generate Go code      | `--dir <path>`, `--output <dir>`, `--workers <n>`, `--verbose`, `--no-cache`, `--clear-cache` |
| `glib validate`                | Validate annotations and route definitions | `--dir <path>`, `--verbose`                                                                   |
| `glib dev`                     | Start development server with hot reload   | `--port <port>`, `--workers <n>`, `--debounce <ms>`, `--verbose`, `--no-cache`                |
| `glib version`                 | Print the Glib CLI version                 |                                                                                               |

---

## Configuration (`.config.toml`)

Project configuration is resolved in order:

1. **CLI Flags** (highest precedence)
2. **`.config.toml`** (project configuration file)
3. **Defaults** (fallback)

```toml
version = "2"
verbose = false

[generate]
output = "generated"
package = "generated"
workers = 4
cache = true

[make]
controllers = "controllers"
providers = "services"
middleware = "middleware"

[watch]
debounce = 300
exclude_dirs = ["vendor", "node_modules", ".git", ".glib", "tmp"]
include_files = ["*.go"]
exclude_files = ["*_test.go", "*.gen.go"]

[validation]
enabled = false
languages = ["en"]
default_language = "en"

[i18n]
enabled = false
locales_dir = "locales"
default_locale = "en"
supported_locales = ["en"]
detect_from = ["header", "query"]
query_param = "lang"
```

---

## Project Structure

A typical Glib application layout:

```
my-app/
├── .config.toml              # Glib CLI & code generation settings
├── .env                      # Application environment variables
├── go.mod
├── go.sum
├── main.go                   # Server initialization and graceful shutdown
├── bootstrap.go              # App bootstrap, router middleware & route registration
├── configs/
│   └── config.go             # @Config struct definitions
├── controllers/
│   ├── auth/
│   │   ├── controller.go     # @Controller with route handlers
│   │   └── models.go         # Request / response structs
│   └── posts/
│       ├── controller.go
│       └── models.go
├── services/
│   ├── database.go           # @Provider singleton for DB connection
│   ├── jwt.go                # @Provider singleton for auth tokens
│   └── logger.go             # @Provider transient for structured logging
├── middleware/
│   └── auth.go               # @Middleware definitions
├── locales/                  # Optional translation files
│   ├── en.toml
│   └── fr.toml
└── generated/                # Auto-generated code (DO NOT EDIT)
    ├── config.gen.go         # Environment loader for @Config structs
    ├── di.gen.go             # Dependency injection container
    ├── routes.gen.go         # Chi router route registrations
    ├── parsers.gen.go        # HTTP parameter parsers & handler wrappers
    ├── validator.gen.go      # Request validator initialization
    └── i18n/                 # Generated typed i18n packages
```

---

## Architecture & Code Generation

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Source Code with Annotations                             │
│    (@Controller, @Route, @Provider, @Middleware, @Config)   │
└──────────────────────────────┬──────────────────────────────┘
                               │ AST Scanning (Parallel)
┌──────────────────────────────▼──────────────────────────────┐
│ 2. Scanner & Semantic Validation                            │
│    - Extracts routes, handlers, and struct tags             │
│    - Validates types, signatures, and dependency graph      │
└──────────────────────────────┬──────────────────────────────┘
                               │ Code Generation
┌──────────────────────────────▼──────────────────────────────┐
│ 3. Generated Code (generated/)                              │
│    - di.gen.go (Topologically sorted container)             │
│    - routes.gen.go (Chi route tree setup)                   │
│    - parsers.gen.go (Type-safe request & response binding)  │
│    - config.gen.go, validator.gen.go, i18n/                 │
└──────────────────────────────┬──────────────────────────────┘
                               │ Runtime Execution
┌──────────────────────────────▼──────────────────────────────┐
│ 4. HTTP Runtime (Zero reflection in handlers)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Requirements

- **Go**: 1.22 or later

---

## Contributing

Contributions are welcome! Please open an issue or pull request on GitHub.

---

## License

MIT License. See [LICENSE](file:///home/azizndao/repos/web/glib/LICENSE) for details.
