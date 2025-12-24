# Glib Framework Examples

This directory contains real-world application examples to help you learn and get started with the Glib framework.

## 📚 Examples

### 1. Quickstart - Simple REST API

**Location:** [`quickstart/`](./quickstart/)

**What you'll learn:**
- Setting up a Glib HTTP server
- Creating REST API endpoints (GET, POST, PUT, DELETE)
- Request validation with struct tags
- Error handling with structured responses
- Route grouping and organization
- Basic middleware usage

**Use case:** Perfect for getting started quickly or building simple APIs without database complexity.

**Run it:**
```bash
cd quickstart
go run main.go
```

---

### 2. Fullstack - Complete Blog Application

**Location:** [`fullstack/`](./fullstack/)

**What you'll learn:**
- Application architecture with Foundation module
- Database integration with GORM
- Model relationships (User → Posts → Comments)
- JWT authentication and authorization
- ServiceProvider pattern for dependency injection
- Database migrations
- Middleware chaining
- Pagination and filtering
- Soft deletes

**Use case:** Real-world application showing how all Glib modules work together.

**Run it:**
```bash
cd fullstack
cp .env.example .env
# Edit .env with your database credentials
go run main.go
```

---

## 🎯 Which Example Should I Use?

| Your Goal | Example | Why |
|-----------|---------|-----|
| "I want to learn Glib basics" | **quickstart** | Simple, focused, no dependencies |
| "I'm building a simple API" | **quickstart** | Fast to set up, easy to understand |
| "I need database integration" | **fullstack** | Complete database + ORM example |
| "I want authentication" | **fullstack** | JWT auth with middleware |
| "I need a project template" | **fullstack** | Production-ready structure |
| "Show me best practices" | **fullstack** | Clean architecture, DI patterns |

---

## 🚀 Running the Examples

All examples include:
- `README.md` - Detailed documentation and learning guide
- `go.mod` - Go module definition
- `.env.example` - Environment variable template
- `test.http` - HTTP requests for testing (use with VS Code REST Client)
- `.air.toml` - Hot reload configuration (optional)

### Prerequisites

```bash
# Install Go 1.21 or later
go version

# Optional: Install Air for hot reload
go install github.com/air-verse/air@latest
```

### Development Mode

Each example supports hot reload with [Air](https://github.com/air-verse/air):

```bash
cd quickstart  # or fullstack
air
```

---

## 📖 Additional Resources

- **Framework Documentation:** [Main README](../README.md)
- **Migration Guide:** [MIGRATION.md](../MIGRATION.md) - Upgrading from v1.x
- **Module Documentation:**
  - HTTP Server: [http/README.md](../http/README.md)
  - Database & ORM: [database/README.md](../database/README.md)
  - Foundation (DI): [foundation/README.md](../foundation/README.md)
  - Common Utilities: [common/README.md](../common/README.md)

---

## 🤝 Contributing

Found an issue or have an improvement? Examples should be:
- **Educational** - Clear, well-commented code
- **Realistic** - Real-world use cases
- **Minimal** - Only necessary complexity
- **Runnable** - Works out of the box

Open an issue or pull request on [GitHub](https://github.com/azizndao/glib).

---

## 📝 License

These examples are part of the Glib framework and share the same license.
