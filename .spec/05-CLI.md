# 05. CLI Commands and Configuration

**Status:** Specification v1.0  
**Last Updated:** 2025-12-30

---

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Commands](#commands)
4. [Configuration](#configuration)
5. [Project Structure](#project-structure)
6. [Development Workflow](#development-workflow)
7. [Integration with Go Tools](#integration-with-go-tools)

---

## Overview

The Glib CLI is a **development tool only** - users can build and deploy their applications using standard `go build` without any Glib CLI dependency.

### CLI Philosophy

1. **Development tool** - Not required for production builds
2. **Code generation** - Generates standard Go code
3. **Hot reload** - Built-in file watcher for development
4. **Scaffolding** - Helps create boilerplate quickly
5. **Standard Go** - Works with `go generate`, `go build`, etc.

### What the CLI Does

```
glib CLI
    ↓
Scans source code
    ↓
Generates Go code (generated/*.gen.go)
    ↓
Standard Go compiler (go build)
    ↓
Binary executable
```

---

## Installation

### Install from Source

```bash
go install github.com/goyave/glib/v2/cmd/glib@latest
```

### Verify Installation

```bash
glib version
# Output: glib v2.0.0
```

### System Requirements

- Go 1.22+ (requires `net/http` routing with `{param}` support)

---

## Commands

### `glib init`

Initialize a new Glib project or add Glib to an existing project.

```bash
glib init [directory]
```

**Options:**
- `--module <name>` - Go module name (default: current directory name)
- `--example` - Create with example code
- `--minimal` - Create minimal project (no examples)

**Examples:**

```bash
# Create new project
glib init my-api
cd my-api
go mod tidy

# Add Glib to existing project
cd existing-project
glib init
```

**What it creates:**

```
my-api/
├── go.mod
├── go.sum
├── main.go                    # Application entry point
├── config.go                  # Config struct
└── controllers/
    └── health.go              # Example health check controller
```

**Generated `main.go`:**

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "my-api/generated"
)

func main() {
    ctx := context.Background()
    
    handler, err := generated.Bootstrap(ctx)
    if err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }
    
    log.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", handler); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}
```

---

### `glib generate`

Generate code from annotations.

```bash
glib generate [flags]
```

**Options:**
- `--dir <path>` - Project root directory (default: current directory)
- `--output <path>` - Output directory (default: `generated/`)
- `--verbose` - Verbose output
- `--watch` - Watch mode (regenerate on file changes)

**Examples:**

```bash
# Generate code
glib generate

# Generate with verbose output
glib generate --verbose

# Generate and watch for changes
glib generate --watch

# Generate for specific directory
glib generate --dir ./services/api
```

**Output:**

```
🔍 Scanning project...
   Found 5 controllers
   Found 3 providers
   Found 2 middleware
   Found 17 handlers

✅ Validation passed

🔨 Generating code...
   generated/glib.gen.go
   generated/di.gen.go
   generated/routes.gen.go
   generated/parsers.gen.go

✅ Generation complete (234ms)
```

---

### `glib dev`

Start development server with hot reload.

```bash
glib dev [flags]
```

**Options:**
- `--port <port>` - Server port (default: 8080)
- `--debounce <ms>` - File watcher debounce in milliseconds (default: 300)
- `--exclude-dirs <dirs>` - Comma-separated directories to exclude (default: vendor,node_modules,.git,.glib,tmp)
- `--include-files <patterns>` - File patterns to watch (default: *.go)
- `--exclude-files <patterns>` - File patterns to ignore (default: *_test.go,*.gen.go)

**Examples:**

```bash
# Start dev server
glib dev

# Start on custom port
glib dev --port 3000

# Custom file watcher settings
glib dev --debounce 500 --exclude-dirs vendor,tmp,node_modules
```

**How it works:**

1. Runs `glib generate` to create initial code
2. Starts built-in file watcher
3. Watches for `.go` file changes (excludes test and generated files)
4. On change: runs `glib generate` → rebuilds → restarts server

**Configuration:**

All settings can be configured via CLI flags or environment variables:

```bash
# Using environment variables
export GLIB_WATCH_DEBOUNCE=500
export GLIB_WATCH_EXCLUDE_DIRS=vendor,tmp,node_modules
export GLIB_WATCH_INCLUDE_FILES=*.go
export GLIB_WATCH_EXCLUDE_FILES=*_test.go,*.gen.go

glib dev
```

---

### `glib make`

Generate boilerplate code (controllers, providers, middleware).

```bash
glib make <type> <name> [flags]
```

**Types:**
- `controller` - Create a new controller
- `provider` - Create a new provider
- `middleware` - Create a new middleware

**Options:**
- `--path <path>` - File path (default: inferred from name)
- `--prefix <prefix>` - Route prefix for controllers
- `--no-example` - Skip example code

#### Make Controller

```bash
glib make controller posts
```

**Creates `posts/controller.go`:**

```go
package posts

import (
    "context"
    
    "github.com/google/uuid"
)

// @Controller /api/v1/posts
type PostsController struct {
    // Add dependencies here (auto-injected)
}

// @Route GET /
func (c *PostsController) Index(ctx context.Context) ([]Post, error) {
    // TODO: implement
    return nil, nil
}

// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    // TODO: implement
    return nil, nil
}

// @Route POST /
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) (*Post, error) {
    // TODO: implement
    return nil, nil
}

// @Route PUT /{id}
func (c *PostsController) Update(ctx context.Context, id uuid.UUID, req UpdatePostRequest) (*Post, error) {
    // TODO: implement
    return nil, nil
}

// @Route DELETE /{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) error {
    // TODO: implement
    return nil
}
```

**Creates `posts/models.go`:**

```go
package posts

import (
    "time"
    
    "github.com/google/uuid"
)

type Post struct {
    ID        uuid.UUID  `json:"id"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}

type CreatePostRequest struct {
    // TODO: add fields
}

type UpdatePostRequest struct {
    // TODO: add fields
}
```

**With custom path prefix:**

```bash
glib make controller admin/posts --prefix /admin/api/posts
```

#### Make Provider

```bash
glib make provider database
```

**Creates `providers/database.go`:**

```go
package providers

import (
    "fmt"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.Name,
        cfg.Database.User,
        cfg.Database.Password,
    )
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    return db, nil
}
```

#### Make Middleware

```bash
glib make middleware auth
```

**Creates `middleware/auth.go`:**

```go
package middleware

import (
    "net/http"
)

// @Middleware auth
func Auth() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // TODO: implement authentication
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

### `glib validate`

Validate project structure and annotations without generating code.

```bash
glib validate [flags]
```

**Options:**
- `--dir <path>` - Project root directory
- `--verbose` - Show detailed validation results

**Examples:**

```bash
glib validate
glib validate --verbose
```

**Output:**

```
🔍 Validating project...

✅ Controllers (5)
   ✓ PostsController (/api/v1/posts)
   ✓ CommentsController (/api/v1/comments)
   ✓ UsersController (/api/v1/users)
   ✓ AuthController (/api/v1/auth)
   ✓ HealthController (/health)

✅ Providers (3)
   ✓ NewDatabase (*gorm.DB)
   ✓ NewCache (*redis.Client)
   ✓ NewLogger (*slog.Logger)

✅ Middleware (2)
   ✓ Auth
   ✓ RateLimit

✅ Dependencies
   ✓ No circular dependencies
   ✓ All dependencies satisfied

✅ Routes
   ✓ No route conflicts
   ✓ 23 routes registered

✅ All checks passed
```

**With errors:**

```
🔍 Validating project...

❌ Validation failed

Error: PostsController.Show
  → Route parameter mismatch
  → Route: GET /{id}/{slug}
  → Handler: Show(ctx context.Context, id uuid.UUID)
  → Missing parameter: slug

Error: Dependency resolution
  → Controller PostsController requires *redis.Client
  → No provider found for *redis.Client

2 errors found
```

---

### `glib version`

Show Glib version information.

```bash
glib version
```

**Output:**

```
glib v2.0.0
Go version: go1.22.0
OS/Arch: linux/amd64
```

---

### `glib help`

Show help for any command.

```bash
glib help [command]
```

**Examples:**

```bash
glib help
glib help generate
glib help make
```

---

## Configuration

### Configuration Priority

Glib uses a **CLI-first configuration approach** with the following priority order:

1. **CLI flags** (highest priority)
2. **Environment variables** (middle priority)
3. **Hardcoded defaults** (fallback)

This follows the 12-factor app methodology and eliminates the need for config files.

### Environment Variables

Override defaults via environment variables:

```bash
# Generation settings
GLIB_OUTPUT=generated
GLIB_PACKAGE=generated
GLIB_WORKERS=4
GLIB_CACHE=true

# Make command settings
GLIB_MAKE_CONTROLLERS=controllers
GLIB_MAKE_PROVIDERS=providers
GLIB_MAKE_MIDDLEWARE=middleware

# Dev/Watch settings
GLIB_WATCH_DEBOUNCE=300
GLIB_WATCH_EXCLUDE_DIRS=vendor,node_modules,.git,.glib,tmp
GLIB_WATCH_INCLUDE_FILES=*.go
GLIB_WATCH_EXCLUDE_FILES=*_test.go,*.gen.go

# Validation settings
GLIB_VALIDATION_ENABLED=false
GLIB_VALIDATION_LANGUAGES=""
GLIB_VALIDATION_DEFAULT_LANGUAGE=""

# Run with custom settings
glib generate
```

### Example Usage

```bash
# Use custom output directory and more workers
export GLIB_OUTPUT=internal/generated
export GLIB_WORKERS=8
glib generate

# Custom dev server port with different debounce
GLIB_WATCH_DEBOUNCE=500 glib dev --port 3000

# Override via CLI flags (highest priority)
glib generate --output custom_gen --workers 16
```

---

## Project Structure

Glib 2.0 **doesn't enforce** any particular structure. Here are common patterns:

### Flat Structure (Small Projects)

```
my-api/
├── main.go
├── config.go
├── posts.go                   # PostsController
├── comments.go                # CommentsController
├── providers.go               # All providers
├── middleware.go              # All middleware
└── generated/
    ├── glib.gen.go
    ├── di.gen.go
    └── routes.gen.go
```

### Feature-Based (Medium Projects)

```
my-api/
├── main.go
├── config.go
├── posts/
│   ├── controller.go          # @Controller
│   ├── models.go
│   └── repository.go
├── comments/
│   ├── controller.go
│   ├── models.go
│   └── repository.go
├── providers/
│   ├── database.go            # @Provider
│   └── cache.go               # @Provider
├── middleware/
│   ├── auth.go                # @Middleware
│   └── ratelimit.go           # @Middleware
└── generated/
```

### Layered (Large Projects)

```
my-api/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── controllers/
│   │   ├── posts.go
│   │   └── comments.go
│   ├── services/
│   │   ├── posts.go
│   │   └── comments.go
│   ├── repositories/
│   │   ├── posts.go
│   │   └── comments.go
│   ├── providers/
│   │   ├── database.go
│   │   └── cache.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── ratelimit.go
│   └── generated/
└── pkg/
    └── models/
```

### Monorepo (Multiple Services)

```
monorepo/
├── services/
│   ├── api/
│   │   ├── main.go
│   │   ├── controllers/
│   │   └── generated/
│   ├── admin/
│   │   ├── main.go
│   │   ├── controllers/
│   │   └── generated/
│   └── worker/
│       ├── main.go
│       └── generated/
└── shared/
    ├── providers/
    ├── middleware/
    └── models/
```

---

## Development Workflow

### Typical Development Flow

```bash
# 1. Create new project
glib init my-api
cd my-api

# 2. Create a controller
glib make controller posts

# 3. Add dependencies (example: database)
glib make provider database

# 4. Start dev server (with hot reload)
glib dev

# 5. Make changes to code
# → Air detects changes
# → glib generate runs
# → App rebuilds and restarts

# 6. Validate before committing
glib validate

# 7. Build for production
go build -o bin/api .
```

### Hot Reload Flow

```
Edit posts.go
    ↓
Glib watcher detects change
    ↓
Run: glib generate
    ↓
Regenerate generated/*.gen.go
    ↓
Run: go build
    ↓
Restart server
    ↓
Ready in ~500ms
```

### CI/CD Integration

**GitHub Actions Example:**

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      # Install Glib CLI
      - name: Install Glib
        run: go install github.com/goyave/glib/v2/cmd/glib@latest
      
      # Generate code
      - name: Generate code
        run: glib generate
      
      # Run tests
      - name: Test
        run: go test ./...
      
      # Build
      - name: Build
        run: go build -o bin/api .
```

**Production Build:**

```bash
# Generate code
glib generate

# Build with optimizations
go build -ldflags="-s -w" -o bin/api .

# Result: single binary, no Glib CLI dependency
./bin/api
```

---

## Integration with Go Tools

### `go generate`

Add to your `main.go`:

```go
//go:generate glib generate

package main
```

Run:

```bash
go generate ./...
```

### `make` / `Makefile`

```makefile
.PHONY: generate dev build test

generate:
	glib generate

dev:
	glib dev

build: generate
	go build -o bin/api .

test: generate
	go test ./...

clean:
	rm -rf generated/ tmp/ bin/
```

Usage:

```bash
make generate
make dev
make build
make test
```

### Docker

**Dockerfile:**

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install Glib CLI
RUN go install github.com/goyave/glib/v2/cmd/glib@latest

# Copy source
COPY . .

# Generate code and build
RUN glib generate && \
    go build -ldflags="-s -w" -o /app/bin/api .

# Runtime stage (no Glib CLI needed!)
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/api .

EXPOSE 8080

CMD ["./api"]
```

**Build:**

```bash
docker build -t my-api .
docker run -p 8080:8080 my-api
```

### VS Code Integration

**`.vscode/tasks.json`:**

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "glib generate",
      "type": "shell",
      "command": "glib generate",
      "problemMatcher": []
    },
    {
      "label": "glib dev",
      "type": "shell",
      "command": "glib dev",
      "isBackground": true,
      "problemMatcher": {
        "pattern": {
          "regexp": "^(.*):(\\d+):(\\d+):\\s+(warning|error):\\s+(.*)$",
          "file": 1,
          "line": 2,
          "column": 3,
          "severity": 4,
          "message": 5
        },
        "background": {
          "activeOnStart": true,
          "beginsPattern": "^.*restarting.*$",
          "endsPattern": "^.*started.*$"
        }
      }
    }
  ]
}
```

**Usage:**
- Press `Ctrl+Shift+B` → Select "glib generate"
- Press `Ctrl+Shift+P` → "Tasks: Run Task" → "glib dev"

---

## CLI Behavior

### Exit Codes

- `0` - Success
- `1` - General error
- `2` - Validation error
- `3` - Generation error
- `4` - Configuration error

### Error Handling

**Clear error messages:**

```
❌ Error: Controller validation failed

  File: posts/controller.go:15
  Controller: PostsController
  Handler: Show
  
  Route parameter mismatch:
    Route:   GET /{id}/{slug}
    Handler: Show(ctx context.Context, id uuid.UUID)
    
  Missing parameter: slug
  
  Fix: Add 'slug string' parameter to handler signature
```

### Verbose Output

```bash
glib generate --verbose
```

**Output:**

```
🔍 Scanning project...
   [SCAN] posts/controller.go
   [SCAN] comments/controller.go
   [SCAN] providers/database.go
   [SCAN] middleware/auth.go
   
   Found:
     • 2 controllers (5 handlers)
     • 1 provider
     • 1 middleware

✅ Validation
   [OK] Controller: PostsController
   [OK] Controller: CommentsController
   [OK] Provider: NewDatabase
   [OK] Middleware: Auth
   [OK] DI Graph: No circular dependencies
   [OK] Routes: No conflicts

🔨 Generating code...
   [GEN] generated/glib.gen.go (1.2 KB)
   [GEN] generated/di.gen.go (2.4 KB)
   [GEN] generated/routes.gen.go (3.1 KB)
   [GEN] generated/parsers.gen.go (4.5 KB)

✅ Generation complete (234ms)
```

---

## Summary

### CLI Commands

| Command | Purpose | Required for Production |
|---------|---------|-------------------------|
| `glib init` | Create new project | ❌ |
| `glib generate` | Generate code | ✅ (at build time) |
| `glib dev` | Hot reload dev server | ❌ |
| `glib make` | Create boilerplate | ❌ |
| `glib validate` | Validate project | ❌ |
| `glib version` | Show version | ❌ |

### Key Principles

1. **Development tool only** - Production uses `go build`
2. **Generate standard Go** - No runtime dependencies
3. **Hot reload built-in** - Built-in file watcher for development
4. **Flexible structure** - No enforced conventions
5. **Standard tooling** - Works with `go generate`, Docker, etc.

---

**Next:** `06-EXAMPLES.md` - Complete example applications
