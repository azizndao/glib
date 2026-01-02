# Glib 2.0 - Code Generation Framework

## Executive Summary

**Glib 2.0** is a complete redesign of the Glib framework, transforming it into a **code generation-first** web framework for Go. The framework uses static analysis and code generation to eliminate boilerplate, provide compile-time safety, and deliver an exceptional developer experience.

## Core Philosophy

1. **Code Generation Over Runtime Reflection** - All wiring happens at compile-time
2. **Type Safety Everywhere** - No `interface{}`, no string keys, no runtime type assertions
3. **Flexible Structure** - Users organize code however they want
4. **Standard Go** - Uses `net/http`, standard library patterns, integrates with `go generate`
5. **Convention as Option** - Conventions help but never constrain

## Key Features

### 1. Dependency Injection (Compile-Time)

- Annotated provider functions with `@Provider`
- Auto-wiring by type (no tags needed)
- Dependency graph validation at compile-time
- Generated initialization code (Wire-style)

### 2. HTTP Routing (Flexible Handlers)

- `@Controller` and `@Route` annotations
- 2 flexible handler signature patterns (Result[T] and Raw HTTP)
- Support for raw `net/http` or type-safe Result[T]
- Auto-parsing from path/query/headers/body for Result[T] pattern

### 3. Middleware System

- `@Middleware` annotation for discovery
- Tag-based targeting (`target=all`, `target=api`, `target=protected`)
- Handler-level override with `with` attribute
- Execution order control via `order` attribute
- Standard `net/http` middleware signature

### 4. Type-Safe Configuration

- Auto-discover `type Config struct`
- Generate loader from environment variables
- Validation at startup
- Type-safe access throughout app

### 5. Hot Reload Development

- `glib dev` command
- Watches for code changes
- Auto-regenerates code
- Integrates with Air for hot reload

## Architecture Overview

```
User Code (Declarative)
├── Any Project Structure
│   ├── Flat, layered, feature-based, or custom
│   ├── Controllers with @Controller and @Route
│   ├── Providers with @Provider
│   ├── Middleware with @Middleware
│   └── Config struct anywhere
│
└── Annotations → Code Generator

Generated Code (Type-Safe)
├── generated/
│   ├── glib.gen.go      # Application bootstrap
│   ├── di.gen.go        # DI container (topologically sorted)
│   ├── routes.gen.go    # Route registration
│   └── parsers.gen.go   # Handler wrappers (Result[T] + Raw HTTP)
│
└── Standard Go Build

Final Binary
└── Zero runtime overhead, fully type-safe
```

## Design Decisions (Locked)

| Decision               | Choice              | Rationale                            |
| ---------------------- | ------------------- | ------------------------------------ |
| **Router**             | Standard `net/http` | No abstractions, full control        |
| **DI Tags**            | None needed         | Auto-wire by type from `@Controller` |
| **Structure**          | User decides        | No enforced conventions              |
| **Config Location**    | Auto-discover       | Find `type Config struct` anywhere   |
| **Generated Code**     | `generated/` dir    | 4 files (no errors.gen.go)           |
| **Handler Signatures** | 2 patterns          | Result[T] and Raw HTTP               |
| **Middleware**         | Standard `net/http` | `func(http.Handler) http.Handler`    |
| **Build System**       | Standard `go build` | No custom tooling                    |
| **Hot Reload**         | Built-in watcher    | Native file watching with debounce   |

## Comparison: Glib 1.0 vs 2.0

| Feature                | Glib 1.0                  | Glib 2.0                |
| ---------------------- | ------------------------- | ----------------------- |
| **DI Registration**    | Manual, runtime           | Auto, compile-time      |
| **Type Safety**        | `interface{}` casts       | Fully typed             |
| **Route Registration** | Manual per route          | Auto from annotations   |
| **Request Parsing**    | Manual `ctx.BodyParser()` | Auto-generated parsers  |
| **Config Access**      | String keys               | Type-safe fields        |
| **Boilerplate**        | High                      | Minimal                 |
| **Error Detection**    | Runtime panics            | Compile-time errors     |
| **Development**        | Manual restart            | Hot reload with codegen |

## Handler Signature Flexibility

Glib supports **2 handler patterns** for maximum flexibility:

```go
// Pattern 1: Result[T] - Type-safe JSON handlers (~95% of endpoints)
func (c *Controller) Handle(ctx context.Context, req Request) glib.Result[Response]

// Pattern 2: Raw HTTP - Full control for streaming, files, etc. (~5% of endpoints)
func (c *Controller) Handle(w http.ResponseWriter, r *http.Request)
```

## Development Workflow

```bash
# 1. Create new project
glib init myapp
cd myapp

# 2. Write code with annotations
# - Define controllers with @Controller
# - Define providers with @Provider
# - Define middleware with @Middleware

# 3. Generate code
glib generate

# 4. Development with hot reload
glib dev  # Watches files, regenerates, restarts

# 5. Build for production
go build -o myapp
```

## Project Structure (User's Choice)

### Option 1: Flat

```
myapp/
├── main.go
├── config.go
├── auth.go        # @Controller
├── posts.go       # @Controller
└── database.go    # @Provider
```

### Option 2: Feature-Based

```
myapp/
├── main.go
├── features/
│   ├── auth/
│   │   ├── controller.go
│   │   └── service.go
│   └── posts/
│       ├── controller.go
│       └── service.go
```

### Option 3: Layered

```
myapp/
├── main.go
├── controllers/
├── services/
├── repositories/
└── providers/
```

**All work!** Scanner finds annotations anywhere.

## Generated Code Example

**User writes:**

```go
// @Controller path=/api/posts tags=api
type PostsController struct {
    DB *gorm.DB  // Auto-injected!
}

// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.DB.First(&Post{}, id)
    if err != nil {
        return glib.NotFound[*Post]("post not found")
    }
    return glib.OK(post)
}
```

**Generator creates:**

```go
// generated/parsers.gen.go
func handlePostsControllerShow(container *container) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Extract path param
        idStr := r.PathValue("id")
        id, err := uuid.Parse(idStr)
        if err != nil {
            glib.BadRequest[*Post](
                fmt.Sprintf("invalid path parameter 'id': %v", err)).Write(w)
            return
        }

        // Call handler
        result := container.controllers.postsController.Show(ctx, id)

        // Write result
        result.Write(w)
    }))

    return handler.ServeHTTP
}
```

## Success Metrics

### Boilerplate Reduction

- **DI Setup:** 100% reduction (0 lines needed)
- **Route Registration:** 100% reduction (0 lines needed)
- **Request Parsing:** 90% reduction (auto-generated)
- **Config Access:** Type-safe (no string keys)

### Developer Experience

- **Time to "Hello World":** < 2 minutes
- **Time to CRUD API:** < 15 minutes
- **Error Detection:** Compile-time vs runtime
- **Type Safety:** 100% (no `interface{}`)

### Performance

- **No runtime reflection:** All wiring at compile-time
- **Zero overhead:** Generated code = hand-written code
- **Build time:** +1-2s for code generation (acceptable)

## Implementation Status

- [x] Architecture Design
- [x] Annotation Syntax Defined
- [x] Handler Patterns Defined
- [ ] CLI Implementation (Phase 1)
- [ ] Scanner Implementation (Phase 2)
- [ ] Code Generator (Phase 3)
- [ ] Hot Reload Integration (Phase 4)
- [ ] Testing & Documentation (Phase 5)

## Documentation Structure

- **00-OVERVIEW.md** - This file (architecture, philosophy)
- **01-ANNOTATIONS.md** - Complete annotation reference
- **02-HANDLERS.md** - Handler signature patterns
- **03-CODE-GENERATION.md** - How code generation works
- **04-CLI.md** - CLI commands and usage
- **05-EXAMPLES.md** - Full example applications
- **06-MIGRATION.md** - Migrating from Glib 1.0
- **07-IMPLEMENTATION.md** - Phase-by-phase implementation plan

## Questions & Answers

### Why code generation instead of runtime reflection?

- Compile-time safety catches errors before deployment
- Zero runtime overhead (performance = hand-written code)
- Better IDE support (generated code is readable)
- Easier debugging (can step through generated code)

### Why not use existing solutions like Wire?

- Wire was archived (no longer maintained)
- Glib 2.0 is more than just DI (routing, config, middleware)
- Integrated experience (one tool for everything)
- Framework-specific optimizations

### Do I have to follow conventions?

- No! Organize code however you want
- Scanner finds annotations anywhere
- Conventions are documented but optional

### Can I mix generated and manual code?

- Yes! Generated code is just Go code
- You can call generated functions manually
- You can extend generated types
- You can override generated behavior

### Is this a breaking change from Glib 1.0?

- Yes, completely breaking
- Glib 2.0 is a separate major version
- Migration guide provided
- Both versions can coexist

## Next Steps

1. Review this specification
2. Finalize any open design questions (see 07-IMPLEMENTATION.md)
3. Begin Phase 1: CLI Foundation
4. Iterate based on feedback

## Contact & Feedback

- GitHub Issues: For bugs and feature requests
- Discussions: For questions and ideas
- Discord: For real-time chat

---

**Status:** Design Complete, Implementation Pending  
**Target:** Glib 2.0.0-alpha.1  
**Timeline:** 4-6 weeks for initial release
