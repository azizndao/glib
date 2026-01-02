# Glib

**Code-generation-first web framework for Go**

Glib uses annotation-based code generation to eliminate boilerplate and let you focus on business logic. Write annotations in comments, and Glib generates all the HTTP handling, routing, dependency injection, and error serialization code.

## Features

- **Annotation-Based**: Define routes, controllers, middleware, and DI providers using simple annotations
- **Code Generation**: Generates optimized, type-safe HTTP handlers and routing code
- **Result[T] Pattern**: Type-safe responses with explicit HTTP status control and fluent API
- **Automatic Validation**: Request validation using `validate:` tags with auto-generated validation code
- **Raw HTTP Support**: Full control for streaming, SSE, file uploads when needed
- **Dependency Injection**: Built-in DI container with singleton and transient lifecycles
- **Structured Errors**: Encore.dev-style error handling with HTTP status mapping and ValidationErrors support
- **Hot Reload**: Native development server with incremental code regeneration and automatic restart
- **CLI Tools**: Generate boilerplate, validate annotations, and manage development workflow

## Quick Start

### Installation

```bash
go install github.com/azizndao/glib/cmd/glib@latest
```

### Create a New Project

```bash
# Initialize new project
glib init my-app
cd my-app

# Generate a controller
glib make controller posts

# Generate code and run
glib generate
go run .
```

### Development Mode with Hot Reload

```bash
glib dev
```

This will:

1. Run initial code generation
2. Build and start your server
3. Watch for file changes (`.go` files)
4. Auto-regenerate code incrementally (fast!)
5. Rebuild and restart server automatically

Press `Ctrl+C` to stop the development server.

## Example

### Define a Controller

```go
package controllers

import (
    "context"
    "github.com/google/uuid"
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/pkg/errs"
)

// @Controller path=/api/v1/posts tags=api
type PostsController struct {
    DB *gorm.DB  // Auto-injected
}

// @Route method=GET path=/
func (c *PostsController) Index(ctx context.Context) glib.Result[[]*Post] {
    var posts []*Post
    err := c.DB.Find(&posts).Error
    if err != nil {
        return glib.Fail[[]*Post](err)
    }
    return glib.OK(posts)
}

// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    var post Post
    err := c.DB.First(&post, id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return glib.NotFound[*Post]("post not found")
    }
    if err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.OK(&post)
}

// Request model with automatic validation
type CreatePostRequest struct {
    Title   string `json:"title" validate:"required,min=3,max=200"`
    Content string `json:"content" validate:"required,min=10"`
}

// @Route method=POST path=/
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    // Validation happens automatically before this handler is called!
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    err := c.DB.Create(post).Error
    if err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.Created(post)
}

// @Route method=DELETE path=/{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    result := c.DB.Delete(&Post{}, id)
    if result.Error != nil {
        return glib.Fail[any](result.Error)
    }
    if result.RowsAffected == 0 {
        return glib.NotFound[any]("post not found")
    }
    return glib.NoContent[any]()
}
```

### Define a Provider

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

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }

    return db, nil
}

// @Provider singleton
func NewConfig() (*Config, error) {
    return LoadConfig(), nil
}
```

### Define Middleware

```go
// @Middleware auth
func AuthMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            // Validate token...
            next.ServeHTTP(w, r)
        })
    }
}
```

### Generate and Run

```bash
# Generate all code
glib generate

# Run the server
go run .
```

### Bootstrap in main.go

```go
package main

import (
    "context"
    "log"
    "net/http"

    "myapp/generated"
)

func main() {
    ctx := context.Background()

    // Bootstrap generates the entire app with DI, routes, and middleware
    handler, err := generated.Bootstrap(ctx)
    if err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }

    // Start server
    log.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", handler); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}
```

## Annotations

### @Controller

Defines a controller with a route prefix and optional tags.

```go
// @Controller path=/api/v1/posts tags=api,protected
type PostsController struct {
    // Dependencies are auto-injected by type
}
```

### @Route

Defines a route handler with optional tags and middleware.

```go
// @Route method=GET path=/{id} tags=protected with=cache
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    // ...
}
```

Supported HTTP methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`

### @Provider

Defines a dependency provider function.

```go
// @Provider singleton
func NewDatabase() (*gorm.DB, error) {
    // ...
}

// @Provider transient
func NewRequestID() (string, error) {
    return uuid.NewString(), nil
}
```

Lifecycles:

- `singleton` - Created once, shared across all requests
- `transient` - Created for each request/injection

### @Middleware

Defines a middleware function with targeting and ordering.

```go
// @Middleware name=auth target=protected order=10
func AuthMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Middleware logic
            next.ServeHTTP(w, r)
        })
    }
}
```

**Attributes:**

- `name` - Middleware identifier (required)
- `target` - Tag to target (`all`, `api`, `protected`, etc.)
- `order` - Execution order (lower = earlier, default 0)

## Handler Patterns

Glib supports **2 handler patterns**:

### Result[T] Pattern (Recommended)

```go
func (c *Controller) Handle(ctx context.Context, params...) glib.Result[T]
```

**Use for:** Most API endpoints - CRUD operations, queries, commands

**Features:**

- Explicit HTTP status control
- Type-safe generic responses
- Fluent API for common responses
- Automatic error handling

**Available Result Helpers:**

```go
// Success responses
glib.OK[T](data)           // 200 OK
glib.Created[T](data)      // 201 Created
glib.Accepted[T](data)     // 202 Accepted
glib.NoContent[T]()        // 204 No Content

// Error responses
glib.Fail[T](err)          // Auto-extract status from errs.Error
glib.BadRequest[T](msg)    // 400
glib.Unauthorized[T](msg)  // 401
glib.Forbidden[T](msg)     // 403
glib.NotFound[T](msg)      // 404
glib.Conflict[T](msg)      // 409
glib.InternalError[T](msg) // 500

// Redirects
glib.MovedPermanently[T](url)    // 301
glib.Found[T](url)                // 302
glib.TemporaryRedirect[T](url)   // 307
glib.PermanentRedirect[T](url)   // 308

// Custom
glib.WithStatus[T](data, code)   // Custom status
glib.WithError[T](err, code)     // Custom error

// Fluent headers
result.WithHeader("X-Custom", "value")
result.WithHeaders(headers)
```

**Examples:**

```go
// @Route method=GET path=/posts
func (c *Controller) Index(ctx context.Context) glib.Result[[]Post] {
    posts := c.Service.GetAll()
    return glib.OK(posts)
}

// @Route method=GET path=/posts/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if err != nil {
        return glib.NotFound[*Post]("post not found")
    }
    return glib.OK(post)
}

// @Route method=POST path=/posts
func (c *Controller) Create(ctx context.Context, req CreateRequest) glib.Result[*Post] {
    post, err := c.Service.Create(req)
    if err != nil {
        return glib.Fail[*Post](err)  // Auto-maps errs.Error to HTTP status
    }
    return glib.Created(post)
}

// @Route method=DELETE path=/posts/{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    if err := c.Service.Delete(id); err != nil {
        return glib.Fail[any](err)
    }
    return glib.NoContent[any]()
}
```

### Raw HTTP Pattern (Advanced)

```go
func (c *Controller) Handle(w http.ResponseWriter, r *http.Request)
```

**Use for:** Streaming, SSE, file uploads, websockets, custom protocols

**Features:**

- Full control over response
- No automatic parsing/marshalling
- Direct access to HTTP primitives

**Examples:**

```go
// @Route method=GET path=/export
func (c *Controller) Export(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=posts.csv")
    fmt.Fprintln(w, "id,title,created_at")
    // Write CSV data...
}

// @Route method=GET path=/stream
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher := w.(http.Flusher)
    for {
        select {
        case <-r.Context().Done():
            return
        case event := <-c.Events:
            fmt.Fprintf(w, "data: %s\n\n", event)
            flusher.Flush()
        }
    }
}
```

## Error Handling

Glib uses Encore.dev-style structured errors with automatic HTTP status mapping:

```go
import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/pkg/errs"
)

func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    var post Post
    if err := c.DB.First(&post, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return glib.NotFound[*Post]("post not found")
        }

        // Wrap database error with structured error
        return glib.Fail[*Post](
            errs.B().
                Code(errs.Internal).
                Msg("failed to fetch post").
                Cause(err).
                Err(),
        )
    }
    return glib.OK(&post)
}
```

### Error Codes and HTTP Status Mapping

- `InvalidArgument` → 400 Bad Request
- `Unauthenticated` → 401 Unauthorized
- `PermissionDenied` → 403 Forbidden
- `NotFound` → 404 Not Found
- `AlreadyExists` → 409 Conflict
- `Internal` → 500 Internal Server Error
- `Unavailable` → 503 Service Unavailable

### Validation Errors

Glib provides **automatic request validation** using `go-playground/validator`. Simply add `validate:` tags to your request structs, and validation code is auto-generated.

**Define Request with Validation Tags:**

```go
type CreatePostRequest struct {
    Title    string    `json:"title" validate:"required,min=3,max=200"`
    Body     string    `json:"body" validate:"required,min=10"`
    Email    string    `json:"email" validate:"required,email"`
    Age      int       `json:"age" validate:"gte=18,lte=120"`
    Website  string    `json:"website" validate:"omitempty,url"`
    AuthorID uuid.UUID `json:"author_id" validate:"required,uuid4"`
}
```

**Validation is Automatic:**

When you run `glib generate`, the parser code automatically validates request bodies before calling your handler. No manual validation code needed!

**Common Validation Tags:**

- `required` - Field must be present
- `email` - Must be valid email
- `min=N`, `max=N` - String length or numeric range
- `gte=N`, `lte=N` - Greater/less than or equal
- `url`, `uri` - Valid URL/URI
- `uuid`, `uuid4` - Valid UUID
- `oneof=a b c` - Value must be one of the options
- `omitempty` - Skip validation if field is empty/nil

**Generated JSON Response (on validation failure):**

```json
{
    "error": {
        "code": "invalid_argument",
        "message": "Validation failed",
        "details": [
            {
                "field": "email",
                "messages": ["must be a valid email address"]
            },
            {
                "field": "title",
                "messages": ["must be at least 3 characters"]
            }
        ]
    }
}
```

**Manual Validation (when needed):**

You can also manually create validation errors:

```go
validationErrs := errs.NewValidationErrors([]errs.ValidationError{
    {Field: "custom_field", Messages: []string{"custom validation message"}},
})

err := errs.B().
    Code(errs.InvalidArgument).
    Msg("Validation failed").
    Details(validationErrs).
    Err()

return glib.Fail[*Post](err)
```

## CLI Commands

### `glib init [dir]`

Initialize a new Glib project.

```bash
glib init my-app
```

Creates:

- Project structure
- Sample `main.go` and `bootstrap.go`
- Configuration struct with `@Config` annotation
- Example controller (with `--example` flag)

**Options:**

- `--example` - Include example health check controller
- `--minimal` - Minimal setup without examples or comments
- `--module` - Specify Go module name (auto-detected if omitted)

### `glib make controller <name>`

Generate a controller boilerplate.

```bash
glib make controller posts
```

Creates `controllers/posts_controller.go` with CRUD methods.

### `glib make provider <name>`

Generate a provider boilerplate.

```bash
glib make provider database
```

### `glib make middleware <name>`

Generate middleware boilerplate.

```bash
glib make middleware auth
```

### `glib generate`

Scan and generate all code.

```bash
glib generate [--verbose] [--dir <path>]
```

Generates:

- `generated/glib.gen.go` - Bootstrap function
- `generated/di.gen.go` - Dependency injection container
- `generated/routes.gen.go` - Route registration
- `generated/parsers.gen.go` - Handler wrappers

### `glib validate`

Validate annotations without generating code.

```bash
glib validate
```

### `glib dev`

Start development server with native hot reload.

```bash
glib dev [--port 8080] [--verbose] [--workers 4] [--no-cache] [--debounce 300]
```

Features:

- Auto-regenerates code on file changes (incremental!)
- Rebuilds and restarts server automatically
- Native file watching (no external dependencies)
- Debouncing for rapid file changes (default: 300ms)
- Press Ctrl+C to stop

**Options:**

- `--port` - Server port (default: 8080, or from PORT env var)
- `--verbose` - Show detailed statistics
- `--workers` - Number of parallel workers (default: 4)
- `--no-cache` - Disable incremental caching
- `--debounce` - File watch debounce in ms (default: 300)

## Configuration

Glib uses a **CLI-first configuration approach**. Configuration is resolved in this priority order:

1. **CLI flags** (highest priority)
2. **Environment variables** (prefixed with `GLIB_`)
3. **Hardcoded defaults** (fallback)

No configuration file is needed! This makes Glib more portable and 12-factor app compliant.

### Environment Variables

You can configure Glib using these environment variables:

**Generation:**

- `GLIB_OUTPUT` - Output directory (default: `generated`)
- `GLIB_PACKAGE` - Package name (default: `generated`)
- `GLIB_WORKERS` - Number of parallel workers (default: `4`)
- `GLIB_CACHE` - Enable caching (default: `true`)

**Make Command:**

- `GLIB_MAKE_CONTROLLERS` - Controllers directory (default: `controllers`)
- `GLIB_MAKE_PROVIDERS` - Providers directory (default: `providers`)
- `GLIB_MAKE_MIDDLEWARE` - Middleware directory (default: `middleware`)

**Dev/Watch:**

- `GLIB_WATCH_DEBOUNCE` - Watch debounce in ms (default: `300`)
- `GLIB_WATCH_EXCLUDE_DIRS` - Comma-separated excluded directories (default: `vendor,node_modules,.git,.glib,tmp`)
- `GLIB_WATCH_INCLUDE_FILES` - Comma-separated file patterns (default: `*.go`)
- `GLIB_WATCH_EXCLUDE_FILES` - Comma-separated excluded patterns (default: `*_test.go,*.gen.go`)

**Validation:**

- `GLIB_VALIDATION_ENABLED` - Enable validation (default: `false`)
- `GLIB_VALIDATION_LANGUAGES` - Comma-separated supported languages (default: `""`)
- `GLIB_VALIDATION_DEFAULT_LANGUAGE` - Default language (default: `""`)

### Example Configuration

Using environment variables:

```bash
# Set custom output directory and enable verbose mode
export GLIB_OUTPUT=gen
export GLIB_VERBOSE=true
export GLIB_WORKERS=8

# Run generation
glib generate

# Or use CLI flags to override
glib generate --output custom-gen --workers 16
```

Using a `.env` file (for your application config, not Glib):

```env
# Application config
PORT=3000
DATABASE_URL=postgres://localhost/mydb

# Glib config (optional)
GLIB_OUTPUT=generated
GLIB_WORKERS=8
```

## Project Structure

```
my-app/
├── controllers/           # Your controllers
│   ├── posts_controller.go
│   └── users_controller.go
├── providers/            # Your DI providers
│   └── database.go
├── middleware/           # Your middleware
│   └── auth.go
├── configs/              # Your app configuration
│   └── config.go
├── generated/            # Generated code (don't edit)
│   ├── config.gen.go
│   ├── di.gen.go
│   ├── routes.gen.go
│   └── parsers.gen.go
├── main.go              # Your entry point
├── bootstrap.go         # Your bootstrap logic
├── .env                 # App environment variables
└── go.mod
```

## Examples

See the `examples/demo` directory for a complete working example with:

- Post controller with CRUD operations
- Comment controller with nested routes
- Request/response models
- Configuration management

```bash
cd examples/demo
../../bin/glib generate
go run .
```

## Architecture

Glib follows a code-generation-first architecture:

1. **Annotations** - You write annotations in Go comments
2. **Scanner** - AST-based scanner extracts annotations and analyzes code
3. **Validator** - Validates routes, dependencies, and handler signatures
4. **Generator** - Generates optimized, type-safe code
5. **Runtime** - Your app uses the generated code at runtime

### Generated Code

The generated code is optimized and type-safe:

- No reflection at runtime
- Direct function calls
- Type-safe request parsing
- Proper error handling
- Import resolution for cross-package types
- Topological sorting for dependency injection

### Handler Patterns

Glib uses **2 handler patterns**:

- **Result[T]**: `func(ctx, params...) glib.Result[T]` - Type-safe with explicit status control (~95% of endpoints)
- **Raw HTTP**: `func(w, r)` - Full control for streaming, SSE, file uploads (~5% of endpoints)

## Requirements

- Go 1.21 or later

## Contributing

Contributions are welcome! Please read the specifications in `.spec/` directory before contributing.

## License

MIT License - see LICENSE file for details.

---

**Status**: Experimental - Under active development

For detailed specifications and implementation details, see the `.spec/` directory.
