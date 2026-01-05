# Glib Demo Application

A complete reference implementation demonstrating Glib's annotation-based code generation for building HTTP APIs in Go.

## 🚀 Features Demonstrated

- ✅ **Result[T] Handlers** - Type-safe responses with explicit status control
- ✅ **Raw HTTP Handlers** - Full control for CSV exports and Server-Sent Events
- ✅ **Dependency Injection** - Auto-wired services with topological sorting
- ✅ **JWT Authentication** - Registration, login, and protected routes
- ✅ **Middleware System** - Tag-based middleware targeting (auth, rate limiting)
- ✅ **Error Handling** - Structured field-level validation errors
- ✅ **Database Integration** - SQLite with GORM, auto-migration, and seed data
- ✅ **Hot Reload** - Development mode with automatic code regeneration

## 📋 Prerequisites

- **Go 1.21+** ([install](https://golang.org/doc/install))
- **glib CLI** (built from project root)
- **Optional:** [VS Code REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) for API testing

## 🏗️ Quick Start

### 1. Build the glib CLI

From the project root:

```bash
cd /path/to/glib
go build -o glib ./cmd/glib

# Verify installation
./glib version
# Output: glib 0.2.1
```

### 2. Run the Demo

From the demo directory:

```bash
cd examples/demo

# Generate code
../../glib generate

# Run the server
go run .
```

The server will start on **<http://localhost:8091>**

### 3. Test the API

Use the HTTP test files in `api-tests/`:

```bash
# Install VS Code REST Client extension, then open:
- api-tests/auth.http
- api-tests/posts.http
- api-tests/comments.http

# Or use curl:
curl http://localhost:8091/api/v1/post
```

## 🔥 Hot Reload Development

For automatic code regeneration and server restart on file changes:

```bash
../../glib dev
```

This will:

1. Run initial code generation
2. Start built-in file watcher
3. Auto-regenerate code on changes
4. Rebuild and restart server automatically

**Configuration:**

You can customize the file watcher behavior using `.config.toml`:

```toml
# .config.toml
[watch]
debounce = 500
exclude_dirs = ["vendor", "tmp", "node_modules"]
include_files = ["*.go"]
exclude_files = ["*_test.go", "*.gen.go"]
```

Or override with CLI flags:

```bash
# Custom debounce delay
../../glib dev --debounce 500

# Custom excluded directories
../../glib dev --exclude-dirs vendor,tmp,node_modules
```

## 📁 Project Structure

```
demo/
├── .config.toml            # Glib configuration
├── controllers/          # HTTP controllers
│   ├── auth/
│   │   ├── controller.go # Auth endpoints (register, login, profile)
│   │   └── models.go     # Request/response DTOs
│   ├── post/
│   │   ├── controller.go # Post CRUD + streaming examples
│   │   └── models.go     # Post DTOs
│   └── comment/
│       ├── controller.go # Comment CRUD
│       └── models.go     # Comment DTOs
├── middleware/
│   └── middleware.go     # @Middleware annotations (auth, rate limit)
├── models/               # Domain models
│   ├── user.go           # User entity
│   ├── post.go           # Post entity
│   └── comment.go        # Comment entity
├── services/             # Business logic
│   ├── database.go       # @Provider for database
│   ├── user.go           # @Provider for user service
│   ├── post.go           # @Provider for post service
│   ├── comment.go        # @Provider for comment service
│   ├── jwt.go            # @Provider for JWT service
│   └── logger.go         # @Provider for logger (transient)
├── generated/            # ⚠️ DO NOT EDIT - Generated code
│   ├── config.gen.go     # Config loaders
│   ├── di.gen.go         # DI container (topologically sorted)
│   ├── routes.gen.go     # Route registration
│   └── parsers.gen.go    # Handler wrappers + middleware chains
├── api-tests/            # HTTP test files
│   ├── auth.http
│   ├── posts.http
│   ├── comments.http
│   └── README.md
├── configs/
│   ├── config.go         # App configuration
│   └── redis.go          # Redis configuration
├── bootstrap.go          # Application bootstrap
├── main.go               # Entry point
├── .env.example          # Environment variables template
└── demo.db               # SQLite database (auto-created)
```

## 🔐 Authentication

The demo includes a complete JWT-based authentication system.

### Demo Users

The database is pre-seeded with 3 users (password: `password123`):

| Username   | Email              | UUID                                 |
| ---------- | ------------------ | ------------------------------------ |
| john_doe   | <john@example.com> | a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d |
| jane_smith | <jane@example.com> | b2c3d4e5-f6a7-4b5c-9d0e-1f2a3b4c5d6e |
| bob_wilson | <bob@example.com>  | c3d4e5f6-a7b8-4c5d-0e1f-2a3b4c5d6e7f |

### Authentication Flow

```bash
# 1. Login to get JWT token
curl -X POST http://localhost:8091/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"john_doe","password":"password123"}'

# Response:
{
  "user": {...},
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}

# 2. Use token for protected endpoints
curl http://localhost:8091/api/v1/auth/me \
  -H "Authorization: Bearer <your-token>"
```

## 📡 API Endpoints

### Authentication (`/api/v1/auth`)

| Method | Path        | Auth      | Description       |
| ------ | ----------- | --------- | ----------------- |
| POST   | /register   | Public    | Register new user |
| POST   | /login      | Public    | Login and get JWT |
| GET    | /me         | Protected | Get current user  |
| PUT    | /me         | Protected | Update profile    |
| GET    | /users/{id} | Public    | Get user by ID    |
| DELETE | /logout     | Protected | Logout            |

### Posts (`/api/v1/post`)

| Method | Path    | Auth      | Description                  |
| ------ | ------- | --------- | ---------------------------- |
| GET    | /       | Public    | List all posts               |
| GET    | /{id}   | Public    | Get single post              |
| POST   | /       | Protected | Create post                  |
| PUT    | /{id}   | Protected | Update post                  |
| DELETE | /{id}   | Protected | Delete post                  |
| GET    | /export | Public    | Export posts as CSV          |
| GET    | /stream | Public    | Stream posts (SSE)           |
| GET    | /health | Public    | Health check (no middleware) |

### Comments (`/api/v1/comment`)

| Method | Path  | Auth      | Description        |
| ------ | ----- | --------- | ------------------ |
| GET    | /     | Public    | List all comments  |
| GET    | /{id} | Public    | Get single comment |
| POST   | /     | Protected | Create comment     |
| PUT    | /{id} | Protected | Update comment     |
| DELETE | /{id} | Protected | Delete comment     |

## 💡 Handler Examples

### Result[T] Pattern (Type-Safe)

Most handlers use the Result[T] pattern for type-safe responses:

```go
// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*models.Post] {
    post, err := c.PostService.GetPost(id)
    if err != nil {
        return glib.NotFound[*models.Post]("post not found")
    }
    return glib.OK(post)
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) glib.Result[*models.Post] {
    post := c.Service.Create(req)
    return glib.Created(post)  // Returns 201 Created
}
```

**Available Result[T] Helpers:**

- `glib.OK[T](data)` - 200 OK
- `glib.Created[T](data)` - 201 Created
- `glib.NoContent[T]()` - 204 No Content
- `glib.NotFound[T](msg)` - 404 Not Found
- `glib.BadRequest[T](msg)` - 400 Bad Request
- `glib.Unauthorized[T](msg)` - 401 Unauthorized
- `glib.Fail[T](err)` - Auto-extract status from errs.Error

### Raw HTTP Pattern (Advanced)

Use Raw HTTP when you need full control:

```go
// @Route method=GET path=/export
func (c *Controller) Export(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=posts.csv")
    w.WriteHeader(http.StatusOK)

    fmt.Fprintln(w, "id,title,created_at")
    // Write CSV data...
}

// @Route method=GET path=/stream
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher := w.(http.Flusher)

    for event := range c.Events {
        fmt.Fprintf(w, "data: %s\n\n", event)
        flusher.Flush()
    }
}
```

## 🔧 Middleware System

Middleware is applied automatically based on tags:

```go
// middleware/middleware.go

// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) middleware.Middleware {
    return func(req middleware.Request, next middleware.Next) glib.Result[any] {
        token := req.Header("Authorization")
        if token == "" {
            return glib.Unauthorized[any]("authorization required")
        }
        // Validate token...
        return next(req)
    }
}

// @Middleware name=ratelimit target=api order=5
func RateLimit() middleware.Middleware {
    // Rate limiting logic...
}
```

**Middleware Targeting:**

- `target=all` - Applies to all routes
- `target=api` - Applies to routes/controllers tagged with `tags=api`
- `target=protected` - Applies to routes tagged with `tags=protected`

**Execution Order:**
Routes with `tags=api,protected` will have middleware applied in order:

1. Rate Limiter (order=5)
2. Auth (order=10)
3. Handler

## 🗄️ Database

### Schema

The demo uses SQLite with GORM for database operations. Tables are auto-migrated on startup:

- **users** - User accounts
- **posts** - Blog posts
- **comments** - Post comments

### Seed Data

On first run, the database is automatically seeded with:

- 3 demo users
- 5 sample posts
- 5 comments

### Reset Database

To start fresh:

```bash
rm demo.db
# Database will be recreated on next startup
```

## 🧪 Testing the API

### Option 1: VS Code REST Client

1. Install the [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) extension
2. Open any `.http` file in `api-tests/`
3. Click "Send Request" or press `Ctrl+Alt+R` / `Cmd+Alt+R`

### Option 2: curl

```bash
# List all posts
curl http://localhost:8091/api/v1/post

# Login
curl -X POST http://localhost:8091/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"john_doe","password":"password123"}'

# Create post (requires auth)
curl -X POST http://localhost:8091/api/v1/post \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Post",
    "body": "Content here",
    "slug": "my-post",
    "published": true,
    "author_id": "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d",
    "tags": "golang,web"
  }'
```

## 🛠️ Development Workflow

### Creating New Components

```bash
# New controller
../../glib make controller products

# New provider
../../glib make provider emailService

# New middleware
../../glib make middleware cors
```

After creating components:

```bash
../../glib generate
```

### Validation

Validate annotations without generating code:

```bash
../../glib validate
```

Checks for:

- Duplicate routes
- Invalid HTTP methods
- Malformed path parameters
- Handler signature issues
- Missing dependencies

## 🏗️ Building for Production

```bash
# Generate code
../../glib generate

# Build binary
go build -o demo .

# Run
./demo
```

Or with custom configuration:

```bash
APP_PORT=3000 ./demo
```

## ⚙️ Configuration

### Glib Configuration

The demo includes a `.config.toml` file for Glib CLI configuration:

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
```

You can customize these settings:

```toml
# Custom output directory
[generate]
output = "internal/generated"

# More workers for faster generation
[generate]
workers = 8

# Faster debounce for quicker hot reload
[watch]
debounce = 200
```

### Application Configuration

### Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

**Application Configuration:**

| Variable    | Default     | Description          |
| ----------- | ----------- | -------------------- |
| APP_PORT    | 8091        | Server port          |
| APP_ENV     | development | Environment mode     |
| DB_FILENAME | demo.db     | SQLite database file |
| REDIS_HOST  | localhost   | Redis host           |
| REDIS_PORT  | 6379        | Redis port           |

**Glib CLI Configuration:**

Override Glib settings with CLI flags:

```bash
# Custom output directory
../../glib generate --output custom-gen

# More workers
../../glib generate --workers 8

# Custom watch settings
../../glib dev --debounce 200
```

## 🐛 Troubleshooting

### Port Already in Use

```bash
# Change port in .env
echo "APP_PORT=3000" > .env

# Or use environment variable
APP_PORT=3000 go run .
```

### Generated Code Not Updating

```bash
# Clean and regenerate
rm -rf generated/
../../glib generate
```

### Build Errors

```bash
# Ensure dependencies are installed
go mod tidy

# Verify glib binary is up to date
cd ../.. && go build -o glib ./cmd/glib
```

### Database Issues

```bash
# Reset database
rm demo.db
# Will be recreated with seed data on next startup
```

## 🎓 Learning Resources

### Handler Patterns

See `controllers/post/controller.go` for examples of:

- Result[T] handlers (Show, Create, Update, Delete)
- Raw HTTP handlers (Export, Stream, Health)

### Dependency Injection

See `services/` for examples of:

- Singleton providers (database, services)
- Transient providers (logger)
- Dependencies between providers

### Middleware

See `middleware/middleware.go` for examples of:

- JWT authentication
- Rate limiting
- Tag-based targeting

### Error Handling

See `controllers/auth/controller.go` for examples of:

- Structured errors with `errs.Builder`
- Validation errors
- HTTP status mapping

## 📚 Key Takeaways

1. **Use Result[T] for APIs** - Type-safe responses for most endpoints (~95%)
2. **Use Raw HTTP for streaming** - Full control when needed (~5%)
3. **Providers are auto-sorted** - Dependencies initialized in correct order
4. **Middleware is tag-based** - Automatic application based on route tags
5. **Generated code is read-only** - Never edit files in `generated/`
6. **Database auto-seeds** - Fresh data on first run

## 🔗 Next Steps

- Read the main [README](../../README.md) for framework overview
- Explore specifications in `../../.spec/` for detailed documentation
- Learn about [handler patterns](../../.spec/02-HANDLERS.md)
- Understand [error handling](../../.spec/04-ERROR-HANDLING.md)
- Study [code generation](../../.spec/03-CODE-GENERATION.md)

## 💬 Need Help?

- Check the main README at the project root
- Review the specifications in `../../.spec/`
- Open an issue on GitHub

---

**Status:** Production-Ready Demo  
**Version:** Glib 0.2.1  
**License:** MIT
