
The server will start on port 8091 (configured in `.glibrc`).

### Testing the API

Get all posts:
```bash
curl http://localhost:8091/api/v1/posts
```

Create a post:
```bash
curl -X POST http://localhost:8091/api/v1/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"My Post","content":"Hello World"}'
```

Get a specific post:
```bash
curl http://localhost:8091/api/v1/posts/{id}
```

Create a comment:
```bash
curl -X POST http://localhost:8091/api/v1/posts/{postId}/comments \
  -H "Content-Type: application/json" \
  -d '{"content":"Great post!"}'
```

### Creating New Components

From the demo directory:

Create a controller:
```bash
../../bin/glib make controller users
```

Create a provider:
```bash
../../bin/glib make provider database
```

Create middleware:
```bash
../../bin/glib make middleware auth
```

After creating components, regenerate:
```bash
../../bin/glib generate
```

### Building for Production

Build:
```bash
../../bin/glib generate
go build -o demo .
```

Run:
```bash
./demo
```

Or with custom port:
```bash
APP_PORT=3000 ./demo
```

## Code Structure

### Controllers

Controllers use annotations to define routes:

```go
// @Controller /api/v1/posts
type PostsController struct {
    // Dependencies auto-injected by type
}

// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    // Handler implementation
}
```

### Request/Response Models

Models are plain Go structs:

```go
type CreatePostRequest struct {
    Title   string `json:"title"`
    Content string `json:"content"`
}

type Post struct {
    ID        uuid.UUID  `json:"id"`
    Title     string     `json:"title"`
    Content   string     `json:"content"`
    CreatedAt time.Time  `json:"created_at"`
}
```

### Configuration

Configuration is loaded from environment variables in `config.go`:

```go
cfg := LoadConfig()
// cfg.App.Port, cfg.Database.Host, etc.
```

Environment variables:
- `APP_PORT` - Server port (default: 8080)
- `APP_ENV` - Environment (default: development)
- Database config (if needed)

## Handler Examples

This demo showcases different handler patterns:

### Pattern 5: Context + Response
```go
// @Route GET /
func (c *PostsController) Index(ctx context.Context) ([]*Post, error)
```

### Pattern 7: Context + Path Parameter
```go
// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error)
```

### Pattern 8: Context + Path Parameter + Request
```go
// @Route POST /{postId}/comments
func (c *CommentsController) Create(
    ctx context.Context,
    postId uuid.UUID,
    req CreateCommentRequest,
) (*Comment, error)
```

### Pattern 4: Context + Error (No Response)
```go
// @Route DELETE /{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) error
```

## Validation

Validate annotations without generating:
```bash
../../bin/glib validate
```

This checks for:
- Duplicate routes
- Invalid HTTP methods
- Malformed path parameters
- Handler signature issues

## Hot Reload

The `glib dev` command provides hot reload:

1. Watches for `.go` file changes
2. Runs `glib generate` automatically
3. Rebuilds the application
4. Restarts the server

Configuration is in `.air.toml` (auto-generated).

## Troubleshooting

### Port already in use

Change port in `.glibrc`:
```json
{
  "dev": {
    "port": 3000
  }
}
```

Or use environment variable:
```bash
APP_PORT=3000 go run .
```

### Generated code not updating

Clean and regenerate:
```bash
rm -rf generated/
../../bin/glib generate
```

### Build errors

Make sure dependencies are installed:
```bash
go mod tidy
```