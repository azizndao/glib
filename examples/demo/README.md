# Glib Demo Application

A complete reference implementation demonstrating Glib's annotation-based code generation for building HTTP APIs in Go.

---

## 🚀 Features Demonstrated

- ✅ **Idiomatic Go Handlers** - Standard `(T, error)` and `error` handler signatures with automatic JSON serialization and status code mapping.
- ✅ **Raw HTTP Handlers** - Low-level control for streaming Server-Sent Events (SSE) and CSV file downloads.
- ✅ **Compile-Time Dependency Injection** - Auto-wired singleton and transient services sorted topologically with zero reflection.
- ✅ **JWT Authentication** - Registration, login, profile management, and token validation.
- ✅ **Tag-Based Middleware System** - Auto-targeted middleware for authentication (`target=protected`) and rate limiting (`target=api`).
- ✅ **Structured Error Handling** - Encore-style structured error codes with field-level Goyave validation errors.
- ✅ **Type-Safe Internationalization (i18n)** - Generated translators for localized error and success messages (`locales/en.toml`, `locales/fr.toml`).
- ✅ **Database with Auto-Migration** - SQLite with GORM, automated migrations, and seed data.
- ✅ **Hot Reload Development** - Fast incremental code generation with `glib dev`.

---

## 📋 Prerequisites

- **Go 1.27+**
- **Glib CLI** (built from the repository root)

---

## 🏗️ Quick Start

### 1. Build the Glib CLI

From the project root:

```bash
cd /path/to/glib
go build -o glib ./cmd/glib
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

The server will start on port **8080** (or port specified in `configs/config.go` / `APP_PORT`).

---

## 🔥 Hot Reload Development

Run Glib in development mode for automatic code regeneration on file changes:

```bash
../../glib dev
```

`glib dev` will:

1. Scan project annotations and generate code.
2. Start the built-in file watcher.
3. Automatically regenerate code on `.go` and `.toml` changes.
4. Recompile and restart the server automatically.

---

## 📁 Project Structure

```
demo/
├── .config.toml             # Glib CLI and generator configuration
├── .env.example             # Example environment variables
├── main.go                  # Entry point & graceful shutdown
├── bootstrap.go             # Application bootstrap & Chi middleware setup
├── configs/
│   ├── config.go            # @Config struct for server & database
│   └── redis.go             # @Config struct for Redis
├── controllers/
│   ├── auth/
│   │   ├── controller.go    # Auth endpoints (register, login, profile)
│   │   ├── models.go        # Request/response structs with validation
│   │   └── auth.http        # REST client test requests
│   ├── post/
│   │   ├── controller.go    # Post CRUD, CSV export & SSE streaming
│   │   ├── models.go        # Post DTOs & pagination params
│   │   └── post.http        # REST client test requests
│   └── comment/
│       ├── controller.go    # Comment CRUD endpoints
│       ├── models.go        # Comment DTOs
│       └── comment.http     # REST client test requests
├── middleware/
│   └── middleware.go        # @Middleware (Auth & RateLimit)
├── models/                  # GORM database entities
│   ├── user.go
│   ├── post.go
│   └── comment.go
├── services/                # Business logic providers
│   ├── database.go          # @Provider singleton for SQLite
│   ├── user.go              # @Provider singleton for user service
│   ├── post.go              # @Provider singleton for post service
│   ├── comment.go           # @Provider singleton for comment service
│   ├── jwt.go               # @Provider singleton for JWT tokens
│   ├── logger.go            # @Provider transient for structured logging
│   └── auditor.go           # @Provider transient for audit logging
├── locales/                 # i18n translation catalogs
│   ├── en.toml
│   └── fr.toml
└── generated/               # ⚠️ Generated code (DO NOT EDIT)
    ├── config.gen.go        # Config loaders
    ├── di.gen.go            # Topologically sorted DI container
    ├── routes.gen.go        # Chi route registrations
    ├── parsers.gen.go       # Handler wrappers & request parsers
    ├── validator.gen.go     # Validator setup
    └── i18n/                # Generated type-safe translation package
```

---

## 🔐 Authentication & Seed Users

On startup, the SQLite database (`demo.db`) is automatically migrated and seeded with 3 demo users (password: `password123`):

| Username     | Email              | UUID                                   |
| :----------- | :----------------- | :------------------------------------- |
| `john_doe`   | `john@example.com` | `a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d` |
| `jane_smith` | `jane@example.com` | `b2c3d4e5-f6a7-4b5c-9d0e-1f2a3b4c5d6e` |
| `bob_wilson` | `bob@example.com`  | `c3d4e5f6-a7b8-4c5d-0e1f-2a3b4c5d6e7f` |

---

## 📡 API Endpoints Reference

### Authentication (`/api/v1/auth`)

| Method   | Path          | Auth      | Description                          |
| :------- | :------------ | :-------- | :----------------------------------- |
| `POST`   | `/register`   | Public    | Register a new user                  |
| `POST`   | `/login`      | Public    | Login and receive a JWT token        |
| `GET`    | `/me`         | Protected | Get the authenticated user's profile |
| `PUT`    | `/me`         | Protected | Update profile                       |
| `GET`    | `/users/{id}` | Public    | Get user details by UUID             |
| `DELETE` | `/logout`     | Protected | Logout user session                  |

### Posts (`/api/v1/post`)

| Method   | Path      | Auth      | Description                                             |
| :------- | :-------- | :-------- | :------------------------------------------------------ |
| `GET`    | `/`       | Public    | List posts with pagination (`page`, `per_page`, `sort`) |
| `GET`    | `/{id}`   | Public    | Get a single post by UUID                               |
| `POST`   | `/`       | Protected | Create a new post                                       |
| `PUT`    | `/{id}`   | Protected | Update an existing post                                 |
| `DELETE` | `/{id}`   | Protected | Delete a post (returns HTTP 204)                        |
| `GET`    | `/export` | Public    | Export posts as CSV (Raw HTTP)                          |
| `GET`    | `/stream` | Public    | Stream posts in real time via SSE (Raw HTTP)            |
| `GET`    | `/health` | Public    | Health check (bypasses all middleware with `with=none`) |

### Comments (`/api/v1/comment`)

| Method   | Path    | Auth      | Description                |
| :------- | :------ | :-------- | :------------------------- |
| `GET`    | `/`     | Public    | List all comments          |
| `GET`    | `/{id}` | Public    | Get single comment by UUID |
| `POST`   | `/`     | Protected | Create a comment           |
| `PUT`    | `/{id}` | Protected | Update a comment           |
| `DELETE` | `/{id}` | Protected | Delete a comment           |

---

## 💡 Code Patterns in the Demo

### Standard Handler Pattern: `(T, error)`

```go
// controllers/post/controller.go
// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Post, error) {
    post, err := c.PostSerivce.GetPost(id)
    if err != nil {
        msg := c.I18n.Errors.Posts.NotFound(ctx, id.String())
        return nil, errs.NewNotFound().WithMessage(msg)
    }
    return post, nil
}
```

### Error-Only Handler: `error`

```go
// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error {
    if err := c.PostSerivce.DeletePost(id); err != nil {
        return err
    }
    return nil // Automatically returns HTTP 204 No Content
}
```

### Raw HTTP Handler: `(w, r)`

```go
// @Route method=GET path=/export
func (c *Controller) Export(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=posts.csv")
    w.WriteHeader(http.StatusOK)

    posts, _ := c.PostSerivce.GetPosts()
    fmt.Fprintln(w, "id,title,published")
    for _, post := range posts {
        fmt.Fprintf(w, "%s,%s,%t\n", post.ID, post.Title, post.Published)
    }
}
```

### Native Middleware: `func(glib.Request, glib.Next) glib.Response`

```go
// middleware/middleware.go
// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        authHeader := req.Header("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            return glib.Response{
                Err: errs.NewUnauthorized().Msg("Authorization header required").Err(),
            }
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            return glib.Response{Err: err}
        }

        req = req.WithValues(map[any]any{
            UserIDKey:   claims.UserID,
            UsernameKey: claims.Username,
            EmailKey:    claims.Email,
        })

        resp := next(req)
        resp.Header().Set("X-User-ID", claims.UserID.String())
        return resp
    }
}
```

---

## 🧪 Testing the API

### Option 1: VS Code REST Client

Open any of the HTTP test files and click **"Send Request"**:

- `controllers/auth/auth.http`
- `controllers/post/post.http`
- `controllers/comment/comment.http`

### Option 2: `curl`

```bash
# 1. Login to get JWT
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"john_doe","password":"password123"}'

# 2. List posts with pagination
curl "http://localhost:8080/api/v1/post?page=1&per_page=5"

# 3. Create a post (requires auth token)
curl -X POST http://localhost:8080/api/v1/post \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Getting Started with Glib",
    "body": "Glib makes Go web development enjoyable with compile-time code generation.",
    "author_id": "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d"
  }'
```
