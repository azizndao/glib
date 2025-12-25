# glib - A Laravel-Inspired Web Framework for Go

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v2.0.0--alpha-orange)](https://github.com/azizndao/glib/releases)

A modular, Laravel-inspired web framework for Go that brings the elegance and developer experience of Laravel to the Go ecosystem.

## 🎯 Philosophy

**glib** is designed to be:

- **Modular**: Use only what you need - HTTP server, database, validation, or all together
- **Laravel-Inspired**: Familiar patterns for Laravel developers, idiomatic Go for Go developers
- **Flexible**: Service Provider pattern enables powerful extensibility
- **Production-Ready**: Built on battle-tested libraries (chi, GORM, validator)

## 📦 Modular Architecture (v2.0.0)

glib is split into independent, composable modules:

### Core Modules

| Module         | Import Path                           | Description                                    |
| -------------- | ------------------------------------- | ---------------------------------------------- |
| **http**       | `github.com/azizndao/glib`            | HTTP server, routing, middleware, context      |
| **common**     | `github.com/azizndao/glib/common`     | Utilities (errors, logging, config, container) |
| **foundation** | `github.com/azizndao/glib/foundation` | DI framework, ServiceProvider pattern          |
| **database**   | `github.com/azizndao/glib/database`   | Database manager, ORM helpers, relationships   |
| **validation** | `github.com/azizndao/glib/validation` | Request validation with i18n support           |
| **ratelimit**  | `github.com/azizndao/glib/ratelimit`  | Rate limiting utilities                        |
| **cli**        | `github.com/azizndao/glib/cli`        | Code generators, project scaffolding           |

### Module Dependencies

```
common (pure utilities)
  ↑
  ├── foundation (DI framework)
  │     ↑
  │     └── database (ORM, depends on foundation for ServiceProvider)
  │
  ├── http (HTTP server, standalone)
  ├── validation (request validation)
  ├── ratelimit (rate limiting)
  └── cli (development tools, independent)
```

**Key Design**: Database depends on `foundation` (not `http`), so you can use the database without pulling in HTTP server dependencies!

## 🚀 Quick Start

### Install HTTP Server Only

```bash
go get github.com/azizndao/glib
```

```go
package main

import (
    "github.com/azizndao/glib"
)

func main() {
    server := glib.New(glib.Config{})
    r := server.Router()

    r.Get("/", func(c *glib.Ctx) error {
        return c.JSON(map[string]string{"message": "Hello World"})
    })

    server.ListenWithGracefulShutdown()
}
```

### Install with Database

```bash
go get github.com/azizndao/glib
go get github.com/azizndao/glib/database
go get gorm.io/driver/sqlite
```

```go
package main

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/foundation"
    "gorm.io/driver/sqlite"
)

func main() {
    // Create application with DI container
    app := foundation.NewApplication()

    // Register database provider
    app.Register(&database.Provider{
        Driver: sqlite.Open("app.db"),
    })

    // Bootstrap application
    app.Boot()

    // Create HTTP server
    server := glib.New(glib.Config{})

    // Use database in routes
    r := server.Router()
    r.Get("/users", func(c *glib.Ctx) error {
        db := app.MustResolve((*database.Manager)(nil)).(*database.Manager)
        var users []User
        db.Connection().Find(&users)
        return c.JSON(users)
    })

    server.ListenWithGracefulShutdown()
}
```

### Install CLI Tool

```bash
go install github.com/azizndao/glib/cli@latest
```

```bash
# Create new project
glib new myapp

# Generate code
glib make:model User
glib make:controller UserController --resource
glib make:migration create_users_table
```

## 🏗️ Project Structure

```
glib/
├── http/                 # HTTP Server Module
│   ├── glib.go          # Server implementation
│   ├── router.go        # HTTP routing (chi-based)
│   ├── ctx.go           # Request context
│   ├── middleware/      # Built-in middleware
│   └── go.mod           # module: github.com/azizndao/glib
│
├── common/              # Common Utilities Module
│   ├── errors/          # Error handling
│   ├── slog/            # Structured logging
│   ├── config/          # Configuration management
│   ├── container/       # Dependency injection container
│   ├── util/            # Utility functions
│   └── go.mod           # module: github.com/azizndao/glib/common
│
├── foundation/          # DI Framework Module
│   ├── application.go   # Application lifecycle
│   ├── provider.go      # ServiceProvider pattern
│   └── go.mod           # module: github.com/azizndao/glib/foundation
│
├── database/            # Database Module
│   ├── manager.go       # Database manager
│   ├── provider.go      # Database service provider
│   ├── orm/             # ORM helpers
│   │   ├── model.go     # Base model with UUID, timestamps
│   │   ├── scopes.go    # Query scopes
│   │   ├── relations.go # Relationship helpers
│   │   └── soft_deletes.go # Soft delete support
│   └── go.mod           # module: github.com/azizndao/glib/database
│
├── validation/          # Validation Module
│   ├── validate.go      # Validator with i18n
│   └── go.mod           # module: github.com/azizndao/glib/validation
│
├── ratelimit/           # Rate Limiting Module
│   ├── ratelimit.go     # Rate limiter
│   └── go.mod           # module: github.com/azizndao/glib/ratelimit
│
├── cli/                 # CLI Tool Module
│   ├── commands/        # CLI commands
│   ├── generators/      # Code generators
│   └── go.mod           # module: github.com/azizndao/glib/cli
│
├── example/             # Example Applications
│   ├── basic/           # Basic HTTP server
│   ├── comprehensive/   # Full-featured example
│   ├── database/        # Database usage
│   ├── orm/             # ORM examples
│   └── relationships/   # Database relationships
│
├── .spec/               # Framework Specifications
│   ├── 00-overview.md
│   ├── 01-foundation.md
│   ├── 02-database.md
│   └── ...
│
├── go.work              # Go workspace configuration
└── MIGRATION.md         # v1.x → v2.0.0 migration guide
```

## ✨ Features

### HTTP Server (`http` module)

- **Chi-based routing** with Fiber-like ergonomics
- **Middleware** - CORS, compression, rate limiting, timeout, body limit
- **Rich Context API** - Query params, path params, headers, cookies, file uploads
- **Error handling** - Structured API errors with proper HTTP status codes
- **Content negotiation** - JSON, HTML, file downloads
- **Graceful shutdown** - Clean shutdown with timeout

### Foundation (`foundation` module)

- **Service Container** - Powerful dependency injection
- **Service Providers** - Laravel-inspired provider pattern
- **Application Lifecycle** - Register, boot, shutdown phases
- **Configuration** - Environment-based configuration

### Database (`database` module)

- **GORM Integration** - Full GORM v2 support
- **Base Model** - UUID primary keys, timestamps, soft deletes
- **Relationships** - HasOne, HasMany, BelongsTo, ManyToMany helpers
- **Query Scopes** - Reusable query logic
- **Service Provider** - Easy database registration

### Validation (`validation` module)

- **go-playground/validator** integration
- **Multi-language** support (English, French, Spanish, etc.)
- **Custom validators** - Easy to extend
- **Context integration** - `c.ValidateBody(&req)`

### CLI (`cli` module)

- **Project Scaffolding** - `glib new myapp`
- **Code Generators** - Models, controllers, migrations, middleware
- **Migration Management** - Run, rollback, fresh, status
- **Development Server** - Hot reload support

## 📚 Documentation

### Getting Started

- [Installation](#-quick-start)
- [First Application](#-quick-start)
- [Configuration](./http/README.md)

### Modules

- [HTTP Server](./http/README.md) - Routing, middleware, context
- [Common Utilities](./common/README.md) - Errors, logging, config
- [Foundation](./foundation/README.md) - DI container, service providers
- [Database](./database/README.md) - ORM, relationships, migrations
- [Validation](./validation/README.md) - Request validation
- [CLI Tool](./cli/README.md) - Code generation, scaffolding

### Advanced

- [Service Providers](./foundation/README.md#service-providers)
- [Database Relationships](./database/README.md#relationships)
- [Custom Middleware](./http/README.md#custom-middleware)
- [Testing](./example/comprehensive/README.md)

### Specifications

See [.spec/](./.spec/) directory for comprehensive framework specifications and roadmap.

## 🔄 Migration from v1.x

See [MIGRATION.md](./MIGRATION.md) for detailed migration guide from v1.x to v2.0.0.

**Key Breaking Changes:**

- Modular architecture - install only what you need
- `core` renamed to `http` for clarity
- `foundation` extracted to separate module
- Database depends on `foundation` instead of `http`
- Module paths updated (see migration guide)

## 🎯 Design Principles

1. **Modularity First** - Each module is independently installable and usable
2. **Laravel Inspiration** - Familiar patterns, Go implementation
3. **Zero Magic** - Explicit configuration, no hidden behavior
4. **Standard Library** - Built on proven libraries (chi, GORM, validator)
5. **Developer Experience** - Great DX without sacrificing simplicity

## 🛠️ Development

### Prerequisites

- Go 1.25 or higher
- Git

### Setup

```bash
# Clone repository
git clone https://github.com/azizndao/glib.git
cd glib

# Install dependencies (workspace handles modules)
go work sync

# Run tests
go test ./...

# Build all modules
for dir in http common foundation database validation ratelimit cli; do
  cd $dir && go build ./... && cd ..
done
```

### Running Examples

```bash
# Basic HTTP server
cd example/basic && go run main.go

# Comprehensive example
cd example/comprehensive && go run main.go

# Database example
cd example/database && go run main.go

# ORM example
cd example/orm && go run main.go
```

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Areas to Contribute

- [ ] Additional middleware
- [ ] More database drivers
- [ ] Testing utilities
- [ ] Documentation improvements
- [ ] Example applications
- [ ] Bug fixes and optimizations

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Laravel** - For the inspiration and patterns
- **Chi** - For the excellent HTTP router
- **GORM** - For the powerful ORM
- **Fiber** - For the ergonomic API inspiration
- **Go Community** - For the amazing ecosystem

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/azizndao/glib/issues)
- **Discussions**: [GitHub Discussions](https://github.com/azizndao/glib/discussions)
- **Documentation**: [.spec/](./.spec/) directory

## 🗺️ Roadmap

See [.spec/IMPLEMENTATION-ROADMAP.md](./.spec/IMPLEMENTATION-ROADMAP.md) for detailed roadmap.

### v2.0.0 (Current - Alpha)

- ✅ Modular architecture
- ✅ HTTP server module
- ✅ Foundation module (DI, ServiceProvider)
- ✅ Database module with ORM
- ✅ Validation module
- ✅ CLI tool
- ⏳ Documentation completion
- ⏳ Beta testing

### v2.1.0 (Planned)

- Authentication system
- Session management
- OAuth2 providers
- Authorization (policies, gates)

### v2.2.0 (Planned)

- Queue system
- Task scheduling
- Job chaining
- Multiple queue drivers

### v3.0.0 (Future)

- Cache system
- File storage abstraction
- Cloud storage integration
- Enhanced testing utilities

---

**Built with ❤️ for the Go community**
