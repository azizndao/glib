# glib - A Laravel-Inspired Go Backend Framework

**glib** is a comprehensive, Laravel-inspired Go backend framework that provides an elegant and expressive syntax for building modern web applications.

## 🎯 Features

- **Foundation Layer** - Application lifecycle, service providers, dependency injection
- **Routing** - Elegant HTTP routing with middleware support (powered by Chi)
- **Database** - Type-safe ORM with relationships, scopes, and soft deletes (powered by GORM)
- **Validation** - Powerful request validation
- **CLI Tool** - Code generation and project management
- **Middleware** - Built-in middleware for common tasks
- **Configuration** - Environment-based configuration management

## 📦 Project Structure

This project uses **Go workspaces** for clean separation:

```
glib/
├── go.work                 # Workspace configuration
├── go.mod                  # Framework module (no CLI deps!)
├── foundation/             # Application foundation
├── orm/                    # ORM and database
├── database/               # Database management
├── middleware/             # HTTP middleware
├── validation/             # Request validation
├── config/                 # Configuration
├── tools/
│   └── cli/               # CLI tool (separate module)
│       ├── go.mod         # CLI dependencies only
│       ├── commands/      # CLI commands
│       └── generators/    # Code generators
└── example/               # Example applications
```

## 🚀 Installation

### Framework

```bash
# Add to your project
go get github.com/azizndao/glib
```

### CLI Tool

```bash
# Install CLI globally
go install github.com/azizndao/glib/tools/cli@latest

# Verify installation
glib --version
```

## 📖 Quick Start

### 1. Create a New Project (Manual)

```go
package main

import (
    "log"
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/foundation"
)

func main() {
    // Create application
    app := foundation.NewApplication()
    
    // Register routes
    router := glib.New()
    router.Get("/", func(c *glib.Ctx) error {
        return c.JSON(map[string]string{
            "message": "Hello from glib!",
        })
    })
    
    // Start server
    log.Fatal(router.Listen(":8080"))
}
```

### 2. Use CLI for Code Generation

```bash
# Generate a model
glib make model User --migration

# Generate a controller
glib make controller UserController --resource

# Generate a migration
glib make migration create_posts_table

# Generate middleware
glib make middleware Auth
```

## 💻 CLI Commands

### Code Generation

```bash
# Models
glib make model User                    # Simple model
glib make model User --migration        # Model + migration
glib make model Post --migration --controller  # All together

# Controllers
glib make controller UserController     # Simple controller
glib make controller UserController --resource  # CRUD controller

# Migrations
glib make migration create_users_table  # SQL migration
glib make migration add_email_to_users --type=go  # Go migration

# Middleware
glib make middleware Auth
```

### Coming Soon

```bash
glib new blog                # Create new project
glib serve                   # Development server
glib migrate                 # Run migrations
glib migrate:rollback        # Rollback migrations
```

## 🗂️ Examples

The `example/` directory contains complete working examples:

- **basic** - Simple HTTP server
- **comprehensive** - Full-featured application
- **database** - Database integration
- **orm** - ORM usage with generics
- **relationships** - Model relationships
- **foundation** - Application foundation
- **cli-demo** - CLI code generation demo

Each example is a standalone project that can be copied and used as a template.

## 🗄️ Database Migrations

glib keeps the core framework lean by not including migration dependencies. Choose the approach that fits your needs:

### Option 1: Goose (Production-Grade)

For version-controlled migrations with rollback support:

```bash
# Install goose
go get github.com/pressly/goose/v3
```

```go
import "github.com/pressly/goose/v3"

sqlDB, _ := gormDB.DB()
goose.SetDialect("postgres")
goose.Up(sqlDB, "migrations")
```

**Resources:**
- [Goose Documentation](https://github.com/pressly/goose)
- [Goose Best Practices](https://github.com/pressly/goose#best-practices)

### Option 2: GORM AutoMigrate (Simple Cases)

For automatic schema generation without version control:

```go
db.AutoMigrate(&User{}, &Post{})
```

**Resources:**
- [GORM Migration Guide](https://gorm.io/docs/migration.html)

## 🏗️ Architecture

### Foundation Layer

```go
// Create and bootstrap application
app := foundation.NewApplication()

// Register service providers
app.Register(&database.DatabaseServiceProvider{})
app.Register(&YourCustomProvider{})

// Bootstrap
if err := app.Bootstrap(); err != nil {
    log.Fatal(err)
}
```

### Database & ORM

```go
// Define models
type User struct {
    orm.Model
    Name  string `json:"name"`
    Email string `json:"email" gorm:"unique"`
    Posts []Post `gorm:"foreignKey:UserID"`
}

// Query with generics
user, err := orm.First[User](ctx, db, orm.Where("email = ?", "user@example.com"))

// Use scopes
publishedPosts, err := orm.Find[Post](ctx, db, 
    Post{}.PublishedScope,
    Post{}.RecentScope,
)
```

### Relationships

```go
// Eager loading
user, err := orm.LoadWith[User](ctx, db, userID, "Posts", "Profile")

// Many-to-many
orm.Association(db, &user, "Roles").Append(&adminRole)
```

### Routing & Middleware

```go
router := glib.New()

// Global middleware
router.Use(middleware.CORS())
router.Use(middleware.RateLimit(100))

// Route groups
api := router.Group("/api")
api.Use(middleware.Auth)

// Routes
api.Get("/users", userController.Index)
api.Post("/users", userController.Store)
api.Get("/users/{id}", userController.Show)
```

## 📚 Documentation

- [Foundation Layer](foundation/README.md)
- [ORM Guide](orm/README.md)
- [CLI Tool](example/cli-demo/README.md)
- [Examples](example/)

## 🛠️ Development

### Working with the Workspace

This project uses Go workspaces for development:

```bash
# Clone the repository
git clone https://github.com/azizndao/glib.git
cd glib

# The workspace is already configured (go.work)
# Build framework and CLI together
go work sync

# Run tests
go test ./...

# Install CLI for development
go install ./tools/cli
```

### Project Structure

- **Framework code** → Root directory (`go.mod`)
- **CLI tool** → `tools/cli/` (separate `go.mod`)
- **Examples** → `example/` (each has own `go.mod`)

This structure ensures:
- ✅ Framework users don't get CLI dependencies
- ✅ CLI can evolve independently
- ✅ Examples are self-contained and copyable
- ✅ Easy development with workspace

## 🧪 Testing

```bash
# Test everything
go test ./...

# Test specific package
go test ./orm/...
go test ./foundation/...
go test ./tools/cli/generators/...

# Run with coverage
go test -cover ./...

# Run examples
cd example/relationships && go run main.go
cd example/orm && go run main.go
```

## 📋 Requirements

- Go 1.25.1 or later
- SQLite, PostgreSQL, or MySQL (for database features)

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines.

## 📄 License

MIT License - see LICENSE file for details

## 🙏 Acknowledgments

- Inspired by [Laravel](https://laravel.com/)
- Built with [Chi](https://github.com/go-chi/chi) for routing
- Powered by [GORM](https://gorm.io/) for ORM
- CLI built with [Cobra](https://github.com/spf13/cobra)
- Migrations via [Goose](https://github.com/pressly/goose) (optional)

## 🚀 Roadmap

- [x] Foundation Layer (Application, Providers, Container)
- [x] Database Layer (ORM, Relationships)
- [x] CLI Tool (Code Generators)
- [ ] Authentication & Authorization
- [ ] Queue & Job Processing
- [ ] Cache & Storage
- [ ] Testing Utilities
- [ ] API Resources & Collections

---

**Built with ❤️ and Go**
