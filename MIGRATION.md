# Migration Guide: v1.x → v2.0.0

This guide helps you migrate your existing glib v1.x applications to the new modular v2.0.0 architecture.

## Overview of Changes

v2.0.0 introduces a **modular architecture** where glib is split into 7 independent modules:

- `http` - HTTP server (was `core`)
- `common` - Shared utilities
- `foundation` - DI framework
- `database` - Database & ORM
- `validation` - Request validation
- `ratelimit` - Rate limiting
- `cli` - Development tools

##⏱️ Migration Time Estimate

- **Simple HTTP app**: 5-10 minutes
- **HTTP + Database**: 15-30 minutes
- **Full stack app**: 30-60 minutes

Most changes are simple import path updates.

---

## Step-by-Step Migration

### Step 1: Update Dependencies

**Before (v1.x):**
```bash
go get github.com/azizndao/glib
```

**After (v2.0.0):**
```bash
# HTTP server
go get github.com/azizndao/glib

# Add modules as needed
go get github.com/azizndao/glib/common
go get github.com/azizndao/glib/foundation
go get github.com/azizndao/glib/database
go get github.com/azizndao/glib/validation
```

### Step 2: Update Import Paths

#### HTTP Server (No change)
```go
// ✅ Same in v1.x and v2.0.0
import "github.com/azizndao/glib"
```

#### Utilities (Moved to common/)
```go
// ❌ v1.x
import "github.com/azizndao/glib/errors"
import "github.com/azizndao/glib/slog"
import "github.com/azizndao/glib/config"
import "github.com/azizndao/glib/container"

// ✅ v2.0.0
import "github.com/azizndao/glib/common/errors"
import "github.com/azizndao/glib/common/slog"
import "github.com/azizndao/glib/common/config"
import "github.com/azizndao/glib/common/container"
```

#### Foundation (Now separate module)
```go
// ❌ v1.x
import "github.com/azizndao/glib/foundation"

// ✅ v2.0.0 (path same, but separate module)
import "github.com/azizndao/glib/foundation"
```

#### Database & ORM
```go
// ❌ v1.x
import "github.com/azizndao/glib/database"
import "github.com/azizndao/glib/orm"

// ✅ v2.0.0
import "github.com/azizndao/glib/database"
import "github.com/azizndao/glib/database/orm"
```

### Step 3: Run Automated Fix

Use find-and-replace across your project:

```bash
# macOS/Linux
find . -type f -name "*.go" -exec sed -i '' \
  -e 's|github.com/azizndao/glib/errors|github.com/azizndao/glib/common/errors|g' \
  -e 's|github.com/azizndao/glib/slog|github.com/azizndao/glib/common/slog|g' \
  -e 's|github.com/azizndao/glib/config|github.com/azizndao/glib/common/config|g' \
  -e 's|github.com/azizndao/glib/container|github.com/azizndao/glib/common/container|g' \
  -e 's|github.com/azizndao/glib/util|github.com/azizndao/glib/common/util|g' \
  -e 's|github.com/azizndao/glib/orm|github.com/azizndao/glib/database/orm|g' \
  {} +

# Then run
go mod tidy
```

### Step 4: Update go.mod

```bash
cd your-project
go get github.com/azizndao/glib@v2
go get github.com/azizndao/glib/common@v2
go get github.com/azizndao/glib/foundation@v2  # if using
go get github.com/azizndao/glib/database@v2    # if using
go get github.com/azizndao/glib/validation@v2  # if using
go mod tidy
```

### Step 5: Test

```bash
go build ./...
go test ./...
```

---

## Migration Examples

### Example 1: Simple HTTP Server

**Before (v1.x):**
```go
package main

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errors"
)

func main() {
    server := glib.New(glib.Config{})
    r := server.Router()
    
    r.Get("/", func(c *glib.Ctx) error {
        return c.JSON(map[string]string{"msg": "hello"})
    })
    
    r.Get("/error", func(c *glib.Ctx) error {
        return errors.BadRequest("invalid input", nil)
    })
    
    server.ListenWithGracefulShutdown()
}
```

**After (v2.0.0):**
```go
package main

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/common/errors"  // ✅ Changed
)

func main() {
    server := glib.New(glib.Config{})
    r := server.Router()
    
    r.Get("/", func(c *glib.Ctx) error {
        return c.JSON(map[string]string{"msg": "hello"})
    })
    
    r.Get("/error", func(c *glib.Ctx) error {
        return errors.BadRequest("invalid input", nil)
    })
    
    server.ListenWithGracefulShutdown()
}
```

### Example 2: HTTP + Database

**Before (v1.x):**
```go
package main

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/orm"
    "gorm.io/driver/sqlite"
)

type User struct {
    orm.Model
    Name string
}

func main() {
    app := foundation.NewApplication()
    app.Register(&database.Provider{
        Driver: sqlite.Open("app.db"),
    })
    app.Boot()
    
    server := glib.New(glib.Config{})
    // ... routes
    server.ListenWithGracefulShutdown()
}
```

**After (v2.0.0):**
```go
package main

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/database/orm"  // ✅ Changed
    "gorm.io/driver/sqlite"
)

type User struct {
    orm.Model
    Name string
}

func main() {
    app := foundation.NewApplication()
    app.Register(&database.Provider{
        Driver: sqlite.Open("app.db"),
    })
    app.Boot()
    
    server := glib.New(glib.Config{})
    // ... routes
    server.ListenWithGracefulShutdown()
}
```

### Example 3: Custom Service Provider

**Before (v1.x):**
```go
import (
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/container"
    "github.com/azizndao/glib/slog"
)

type MyProvider struct {
    foundation.ServiceProvider
}

func (p *MyProvider) Register(app *foundation.Application) error {
    app.Bind((*MyService)(nil), func(c *container.Container) (any, error) {
        logger := c.MustResolve((*slog.Logger)(nil)).(*slog.Logger)
        return &MyService{logger: logger}, nil
    })
    return nil
}
```

**After (v2.0.0):**
```go
import (
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/common/container"  // ✅ Changed
    "github.com/azizndao/glib/common/slog"       // ✅ Changed
)

type MyProvider struct {
    foundation.ServiceProvider
}

func (p *MyProvider) Register(app *foundation.Application) error {
    app.Bind((*MyService)(nil), func(c *container.Container) (any, error) {
        logger := c.MustResolve((*slog.Logger)(nil)).(*slog.Logger)
        return &MyService{logger: logger}, nil
    })
    return nil
}
```

---

## Common Migration Issues

### Issue 1: Import not found

**Error:**
```
cannot find module providing package github.com/azizndao/glib/errors
```

**Solution:**
```go
// Change import
import "github.com/azizndao/glib/errors"
// To
import "github.com/azizndao/glib/common/errors"
```

```bash
# Install common module
go get github.com/azizndao/glib/common
```

### Issue 2: ORM types not found

**Error:**
```
undefined: orm.Model
```

**Solution:**
```go
// Change import
import "github.com/azizndao/glib/orm"
// To
import "github.com/azizndao/glib/database/orm"
```

### Issue 3: Multiple module versions

**Error:**
```
found multiple versions of module
```

**Solution:**
```bash
# Clean module cache
go clean -modcache

# Update all modules
go get -u github.com/azizndao/glib@v2
go get -u github.com/azizndao/glib/common@v2
go get -u github.com/azizndao/glib/foundation@v2
go get -u github.com/azizndao/glib/database@v2

go mod tidy
```

---

## Breaking Changes Summary

### Import Path Changes

| v1.x | v2.0.0 | Status |
|------|--------|--------|
| `glib` | `glib` | ✅ No change |
| `glib/errors` | `glib/common/errors` | ⚠️ Changed |
| `glib/slog` | `glib/common/slog` | ⚠️ Changed |
| `glib/config` | `glib/common/config` | ⚠️ Changed |
| `glib/container` | `glib/common/container` | ⚠️ Changed |
| `glib/util` | `glib/common/util` | ⚠️ Changed |
| `glib/foundation` | `glib/foundation` | ✅ Path same* |
| `glib/database` | `glib/database` | ✅ Path same* |
| `glib/orm` | `glib/database/orm` | ⚠️ Changed |
| `glib/validation` | `glib/validation` | ✅ Path same* |

*Path same but now separate module (requires explicit `go get`)

### API Changes

**No API-breaking changes!** All public APIs remain the same. Only import paths changed.

### Behavior Changes

**No behavior changes!** All functionality works exactly the same.

---

## Installation Patterns

### Minimal (HTTP Only)
```bash
go get github.com/azizndao/glib
```

Gets: HTTP server + common utilities

### With Database
```bash
go get github.com/azizndao/glib
go get github.com/azizndao/glib/database
```

Gets: HTTP + database + foundation + common

### Full Stack
```bash
go get github.com/azizndao/glib
go get github.com/azizndao/glib/database
go get github.com/azizndao/glib/validation
```

Gets: HTTP + database + validation + foundation + common

### CLI Tool (Separate)
```bash
go install github.com/azizndao/glib/cli@latest
```

---

## Gradual Migration Strategy

You don't have to migrate everything at once. Here's a gradual approach:

### Phase 1: Update Dependencies (5 min)
```bash
go get github.com/azizndao/glib@v2
go get github.com/azizndao/glib/common@v2
go mod tidy
```

### Phase 2: Fix Common Imports (10 min)
Update all `errors`, `slog`, `config` imports to `common/*`

### Phase 3: Fix Database Imports (5 min)
Update `orm` imports to `database/orm`

### Phase 4: Test (10 min)
```bash
go build ./...
go test ./...
```

---

## Benefits of v2.0.0

After migration, you'll enjoy:

1. **Smaller Dependencies** - Only install what you need
2. **Faster Builds** - Less code to compile
3. **Better Organization** - Clear module boundaries
4. **Independent Updates** - Modules version separately
5. **Flexible Architecture** - Use database without HTTP

---

## Need Help?

- **Issues**: https://github.com/azizndao/glib/issues
- **Discussions**: https://github.com/azizndao/glib/discussions
- **Examples**: See `example/` directory in repo

---

## Quick Reference

### All Import Changes

```bash
# One-liner to update all imports
find . -type f -name "*.go" -exec sed -i '' \
  -e 's|glib/errors|glib/common/errors|g' \
  -e 's|glib/slog|glib/common/slog|g' \
  -e 's|glib/config|glib/common/config|g' \
  -e 's|glib/container|glib/common/container|g' \
  -e 's|glib/util|glib/common/util|g' \
  -e 's|glib/typeutil|glib/common/typeutil|g' \
  -e 's|glib/orm|glib/database/orm|g' \
  {} +
```

### Verification Script

```bash
#!/bin/bash
# verify-migration.sh

echo "Checking for old imports..."
if grep -r "github.com/azizndao/glib/errors" . --include="*.go" ; then
    echo "❌ Found old error imports"
    exit 1
fi

if grep -r "github.com/azizndao/glib/slog" . --include="*.go" ; then
    echo "❌ Found old slog imports"
    exit 1
fi

if grep -r '"github.com/azizndao/glib/orm"' . --include="*.go" ; then
    echo "❌ Found old orm imports"
    exit 1
fi

echo "✅ All imports updated!"
echo "Running build..."
go build ./...

echo "Running tests..."
go test ./...

echo "✅ Migration verified!"
```

---

**Migration complete!** Welcome to glib v2.0.0 🎉
