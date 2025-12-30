# Glib 2.0

**Code-generation-first web framework for Go**

Glib 2.0 is a complete rewrite that uses annotation-based code generation to eliminate boilerplate and let you focus on business logic. Write annotations in comments, and Glib generates all the HTTP handling, routing, dependency injection, and error serialization code.

## Features

- **Annotation-Based**: Define routes, controllers, middleware, and DI providers using simple annotations
- **Code Generation**: Generates optimized, type-safe HTTP handlers and routing code
- **9 Handler Patterns**: Support for multiple handler signatures - from raw HTTP to high-level request/response
- **Dependency Injection**: Built-in DI container with singleton and transient lifecycles
- **Structured Errors**: Encore.dev-style error handling with HTTP status mapping
- **Hot Reload**: Development server with automatic code regeneration using Air
- **CLI Tools**: Generate boilerplate, validate annotations, and manage development workflow

## Quick Start

### Installation

```bash
go install github.com/goyave/glib/v2/cmd/glib@latest
```

Optionally install Air for hot reload:

```bash
go install github.com/cosmtrek/air@latest
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
2. Start Air file watcher
3. Auto-regenerate code on file changes
4. Rebuild and restart server automatically

## Example

### Define a Controller

```go
package controllers

import (
    "context"
    "github.com/google/uuid"
)

// @Controller /api/v1/posts
// @Middleware auth,ratelimit
type PostsController struct {
    DB *gorm.DB  // Auto-injected
}

// @Route GET /
func (c *PostsController) Index(ctx context.Context) ([]*Post, error) {
    var posts []*Post
    err := c.DB.Find(&posts).Error
    return posts, err
}

// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    var post Post
    err := c.DB.First(&post, id).Error
    return &post, err
}

// @Route POST /
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) (*Post, error) {
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    err := c.DB.Create(post).Error
    return post, err
}

// @Route DELETE /{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) error {
    return c.DB.Delete(&Post{}, id).Error
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

Defines a controller with a route prefix.

```go
// @Controller /api/v1/posts
// @Middleware auth,ratelimit
type PostsController struct {
    // Dependencies are auto-injected by type
}
```

### @Route

Defines a route handler.

```go
// @Route GET /{id}
// @Middleware cache
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
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

Defines a middleware function.

```go
// @Middleware auth
func AuthMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Middleware logic
            next.ServeHTTP(w, r)
        })
    }
}
```

## Handler Patterns

Glib supports 9 different handler signatures:

### 1. Raw HTTP
```go
func (c *Controller) Handle(w http.ResponseWriter, r *http.Request)
```

### 2. Raw HTTP + Error
```go
func (c *Controller) Handle(w http.ResponseWriter, r *http.Request) error
```

### 3. Context Only
```go
func (c *Controller) Handle(ctx context.Context)
```

### 4. Context + Error
```go
func (c *Controller) Handle(ctx context.Context) error
```

### 5. Context + Response
```go
func (c *Controller) Handle(ctx context.Context) (*Response, error)
```

### 6. Context + Request
```go
func (c *Controller) Handle(ctx context.Context, req Request) (*Response, error)
```

### 7. Context + Path Parameter
```go
func (c *Controller) Handle(ctx context.Context, id uuid.UUID) (*Response, error)
```

### 8. Context + Path Parameter + Request
```go
func (c *Controller) Handle(ctx context.Context, id uuid.UUID, req Request) (*Response, error)
```

### 9. Context + Multiple Parameters
```go
func (c *Controller) Handle(ctx context.Context, userId uuid.UUID, postId uuid.UUID, req Request) (*Response, error)
```

## Error Handling

Glib uses structured errors with automatic HTTP status mapping:

```go
import "github.com/goyave/glib/v2/pkg/errs"

func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    var post Post
    if err := c.DB.First(&post, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errs.B().
                Code(errs.NotFound).
                Msg("Post not found").
                Meta("id", id).
                Err()
        }
        return nil, err
    }
    return &post, nil
}
```

Error codes and HTTP status mapping:
- `InvalidArgument` → 400
- `NotFound` → 404
- `PermissionDenied` → 403
- `Unauthenticated` → 401
- `Internal` → 500
- And more...

## CLI Commands

### `glib init [dir]`

Initialize a new Glib project.

```bash
glib init my-app
```

Creates:
- Project structure
- `.glibrc` configuration
- Sample `main.go`
- Example controller

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
- `generated/parsers.gen.go` - Request/response handlers
- `generated/errors.gen.go` - Error serialization

### `glib validate`

Validate annotations without generating code.

```bash
glib validate
```

### `glib dev`

Start development server with hot reload.

```bash
glib dev [--port 8080] [--air-config .air.toml] [--no-air]
```

Features:
- Auto-regenerates code on file changes
- Rebuilds and restarts server
- Uses Air for advanced file watching
- Falls back to basic mode if Air not installed

## Configuration

### `.glibrc`

```json
{
  "version": "2",
  "generate": {
    "output": "generated",
    "package": "generated"
  },
  "make": {
    "controllers": "controllers",
    "providers": "providers",
    "middleware": "middleware"
  },
  "dev": {
    "port": 8080
  }
}
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
├── generated/            # Generated code (don't edit)
│   ├── glib.gen.go
│   ├── di.gen.go
│   ├── routes.gen.go
│   ├── parsers.gen.go
│   └── errors.gen.go
├── main.go              # Your entry point
├── .glibrc              # Glib configuration
├── .air.toml            # Air configuration (auto-generated)
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

Glib 2.0 follows a code-generation-first architecture:

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

## Comparison with Goyave v4

| Feature | Goyave v4 | Glib 2.0 |
|---------|-----------|----------|
| Routing | Runtime registration | Generated |
| DI | Reflection-based | Generated |
| Validation | Runtime | Compile-time |
| Handlers | Interface-based | Annotation-based |
| Boilerplate | Manual | Auto-generated |
| Type Safety | Runtime checks | Compile-time |
| Performance | Good | Excellent |

## Requirements

- Go 1.21 or later
- Air (optional, for hot reload)

## Contributing

Contributions are welcome! Please read the specifications in `.spec/` directory before contributing.

## License

MIT License - see LICENSE file for details.

## Credits

Created by the Goyave team. Inspired by modern frameworks like Spring Boot (Java) and NestJS (TypeScript).

---

**Status**: Beta - API may change before v2.0.0 release

For detailed specifications and implementation details, see the `.spec/` directory.
