# Glib Demo Application

Complete example demonstrating Glib's annotation-based code generation for building HTTP APIs.

## Features Demonstrated

- **Result[T] Handlers**: Type-safe responses with explicit status control
- **Raw HTTP Handlers**: Full control for file exports and SSE
- **Dependency Injection**: Auto-wired services with topological sorting
- **ValidationErrors**: Structured field-level validation errors
- **Cross-Package Types**: Handler responses from `models/` package
- **Middleware**: Tag-based middleware targeting system
- **Hot Reload**: Development mode with automatic code regeneration

## Quick Start

### Prerequisites

```bash
# Build the glib CLI (from project root)
go build -o bin/glib ./cmd/glib

# Verify installation
./bin/glib version
```

### Run the Demo

From the demo directory:

```bash
# Generate code
../../bin/glib generate

# Run the server
go run .
```

Or use hot reload mode:

```bash
../../bin/glib dev
```

The server will start on port 8091 (configured in `glib.json`).

## API Endpoints

### Posts API

**List all posts:**

```bash
curl http://localhost:8091/api/v1/post
```

**Get specific post:**

```bash
curl http://localhost:8091/api/v1/post/1
```

**Create post:**

```bash
curl -X POST http://localhost:8091/api/v1/post \
  -H "Content-Type: application/json" \
  -d '{"title":"My Post","body":"Hello World"}'
```

**Update post:**

```bash
curl -X PUT http://localhost:8091/api/v1/post/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title","body":"Updated content"}'
```

**Delete post:**

```bash
curl -X DELETE http://localhost:8091/api/v1/post/550e8400-e29b-41d4-a716-446655440000
```

**Export posts as CSV (Raw HTTP):**

```bash
curl http://localhost:8091/api/v1/post/export
```

**Stream posts via SSE (Raw HTTP):**

```bash
curl http://localhost:8091/api/v1/post/stream
```

### Comments API

**Create comment:**

```bash
curl -X POST http://localhost:8091/api/v1/comment \
  -H "Content-Type: application/json" \
  -d '{"content":"Great post!","postId":1,"authorId":1}'
```

**List comments:**

```bash
curl http://localhost:8091/api/v1/comment
```

## Project Structure

```
demo/
├── controllers/
│   ├── auth/
│   │   ├── controller.go    # Auth controller with Result[T]
│   │   └── models.go         # Request/response models
│   ├── comment/
│   │   ├── controller.go    # Comment CRUD
│   │   └── models.go
│   └── post/
│       ├── controller.go    # Post CRUD + streaming examples
│       └── models.go
├── middleware/
│   └── middleware.go         # @Middleware annotations
├── models/
│   ├── post.go              # Post domain model
│   └── user.go              # User domain model
├── services/
│   ├── post.go              # @Provider singleton
│   └── user.go              # @Provider singleton
├── generated/               # ⚠️ DO NOT EDIT - Generated code
│   ├── glib.gen.go         # Bootstrap function
│   ├── di.gen.go           # DI container (topologically sorted)
│   ├── routes.gen.go       # Route registration
│   └── parsers.gen.go      # Handler wrappers
├── config.go               # Configuration loading
├── main.go                 # Application entry point
├── glib.json                 # Glib configuration
└── .air.toml               # Hot reload configuration
```

## Handler Examples

### Result[T] Pattern (Type-Safe)

Most handlers use the Result[T] pattern for type-safe responses:

```go
// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]models.Post] {
    return glib.OK(c.PostService.GetPosts())
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*models.Post] {
    post := c.PostService.GetPost(id)
    if post == nil {
        return glib.NotFound[*models.Post]("post not found")
    }
    return glib.OK(post)
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) glib.Result[*models.Post] {
    // Validation example
    if len(req.Title) < 3 {
        validationErrs := errs.NewValidationErrors([]errs.ValidationError{
            {Field: "title", Messages: []string{"must be at least 3 characters"}},
        })
        err := errs.B().
            Code(errs.InvalidArgument).
            Msg("Validation failed").
            Details(validationErrs).
            Err()
        return glib.Fail[*models.Post](err)
    }

    post := c.PostService.CreatePost(req)
    return glib.Created(post)
}

// @Route method=DELETE path=/{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    if err := c.PostService.Delete(id); err != nil {
        return glib.Fail[any](err)
    }
    return glib.NoContent[any]()
}
```

**Available Result[T] Helpers:**

- `glib.OK[T](data)` - 200 OK
- `glib.Created[T](data)` - 201 Created
- `glib.NoContent[T]()` - 204 No Content
- `glib.NotFound[T](msg)` - 404 Not Found
- `glib.BadRequest[T](msg)` - 400 Bad Request
- `glib.Fail[T](err)` - Auto-extract status from errs.Error

### Raw HTTP Pattern (Advanced)

Use the Raw HTTP pattern when you need full control:

```go
// @Route method=GET path=/export
func (c *Controller) Export(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=posts.csv")
    w.WriteHeader(http.StatusOK)

    fmt.Fprintln(w, "id,title,created_at")
    for _, post := range c.PostService.GetPosts() {
        fmt.Fprintf(w, "%d,%s,%s\n", post.ID, post.Title, post.CreatedAt)
    }
}

// @Route method=GET path=/stream
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    for i := range 3 {
        select {
        case <-r.Context().Done():
            return
        case <-time.After(1 * time.Second):
            data := map[string]any{"event": i + 1, "time": time.Now()}
            jsonData, _ := json.Marshal(data)
            fmt.Fprintf(w, "data: %s\n\n", jsonData)
            flusher.Flush()
        }
    }
}
```

## Dependency Injection

Services are automatically injected by type:

```go
// services/user.go
// @Provider singleton
func NewUserService() *UserService {
    return &UserService{}
}

// services/post.go
// @Provider singleton
func NewPostService(userService *UserService) *PostService {
    return &PostService{UserService: userService}
}
```

**Generated DI Container (generated/di.gen.go):**

```go
type container struct {
    userSerivce *services.UserSerivce
    postSerivce *services.PostSerivce
}

func newContainer(ctx context.Context) (*container, error) {
    container := &container{}

    // Topologically sorted - UserService initialized first!
    container.userSerivce = services.NewUserSerivce()
    container.postSerivce = services.NewPostSerivce(container.userSerivce)

    return container, nil
}
```

## Validation Errors Example

The demo shows how to use ValidationErrors for structured field-level error responses:

```go
// In handler
validationErrs := errs.NewValidationErrors([]errs.ValidationError{
    {Field: "title", Messages: []string{"must be at least 3 characters"}},
    {Field: "body", Messages: []string{"field is required"}},
})

err := errs.B().
    Code(errs.InvalidArgument).
    Msg("Validation failed").
    Details(validationErrs).
    Err()

return glib.Fail[*Post](err)
```

**Response (400 Bad Request):**

```json
{
    "error": {
        "code": "invalid_argument",
        "message": "Validation failed",
        "details": [
            {
                "field": "title",
                "messages": ["must be at least 3 characters"]
            },
            {
                "field": "body",
                "messages": ["field is required"]
            }
        ]
    }
}
```

## Development Workflow

### Hot Reload

```bash
../../bin/glib dev
```

Watches for changes and automatically:

1. Regenerates code
2. Rebuilds application
3. Restarts server

### Creating New Components

**New controller:**

```bash
../../bin/glib make controller users
```

**New provider:**

```bash
../../bin/glib make provider database
```

**New middleware:**

```bash
../../bin/glib make middleware auth
```

After creating components:

```bash
../../bin/glib generate
```

### Validation

Validate annotations without generating code:

```bash
../../bin/glib validate
```

Checks for:

- Duplicate routes
- Invalid HTTP methods
- Malformed path parameters
- Handler signature issues
- Missing dependencies

## Configuration

Configuration is loaded from environment variables in `config.go`:

```go
type Config struct {
    App struct {
        Port int    `env:"APP_PORT" default:"8091"`
        Env  string `env:"APP_ENV" default:"development"`
    }
}

func LoadConfig() *Config {
    // Load from environment
}
```

**Environment Variables:**

- `APP_PORT` - Server port (default: 8091)
- `APP_ENV` - Environment (default: development)

**Glib Configuration (`glib.json`):**

```json
{
    "version": "2",
    "generate": {
        "output": "generated",
        "package": "generated"
    },
    "dev": {
        "port": 8091
    }
}
```

## Building for Production

```bash
# Generate code
../../bin/glib generate

# Build binary
go build -o demo .

# Run
./demo
```

Or with custom port:

```bash
APP_PORT=3000 ./demo
```

## Troubleshooting

### Port Already in Use

Change port in `glib.json` or use environment variable:

```bash
APP_PORT=3000 ../../bin/glib dev
```

### Generated Code Not Updating

Clean and regenerate:

```bash
rm -rf generated/
../../bin/glib generate
```

### Build Errors

Ensure dependencies are installed:

```bash
go mod tidy
```

### Import Errors in Generated Code

The generator automatically resolves cross-package imports. If you see import errors:

1. Make sure types are exported (capitalized)
2. Run `go mod tidy` to ensure packages exist
3. Regenerate: `../../bin/glib generate`

## Key Takeaways

1. **Use Result[T] for APIs** - Type-safe responses for most endpoints (~95%)
2. **Use Raw HTTP for streaming** - Full control when needed (~5%)
3. **Providers are auto-sorted** - Dependencies initialized in correct order via topological sort
4. **ValidationErrors for field errors** - Structured validation responses
5. **Cross-package types work** - Generator resolves imports automatically
6. **Generated code is in `generated/`** - Never edit these files (4 files: glib.gen.go, di.gen.go, routes.gen.go, parsers.gen.go)

## Next Steps

- Read the specifications in `../../.spec/`
- Explore handler patterns in `02-HANDLERS.md`
- Learn error handling in `04-ERROR-HANDLING.md`
- Understand code generation in `03-CODE-GENERATION.md`

## Need Help?

Check the main README at the project root or open an issue on GitHub.
