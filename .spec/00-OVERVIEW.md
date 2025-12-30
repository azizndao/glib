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
- 9 flexible handler signature patterns (Encore.dev-inspired)
- Support for raw `net/http`, typed requests/responses, or mix
- Auto-parsing from path/query/headers/body

### 3. Middleware System
- `@Middleware` annotation for discovery
- Declarative application via annotations
- Controller-level and route-level middleware
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
│   ├── config_gen.go       # Config loader
│   ├── wire_gen.go         # DI container
│   ├── routes_gen.go       # Route registration + parsers
│   ├── middleware_gen.go   # Middleware registry
│   └── app_gen.go          # Application bootstrap
│
└── Standard Go Build

Final Binary
└── Zero runtime overhead, fully type-safe
```

## Design Decisions (Locked)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Router** | Standard `net/http` | No abstractions, full control |
| **DI Tags** | None needed | Auto-wire by type from `@Controller` |
| **Structure** | User decides | No enforced conventions |
| **Config Location** | Auto-discover | Find `type Config struct` anywhere |
| **Generated Code** | `generated/` dir | Customizable via `.glibrc` |
| **Handler Signatures** | 9 flexible patterns | From raw `net/http` to typed |
| **Middleware** | Standard `net/http` | `func(http.Handler) http.Handler` |
| **Build System** | Standard `go build` | No custom tooling |
| **Hot Reload** | Air integration | Industry standard |

## Comparison: Glib 1.0 vs 2.0

| Feature | Glib 1.0 | Glib 2.0 |
|---------|----------|----------|
| **DI Registration** | Manual, runtime | Auto, compile-time |
| **Type Safety** | `interface{}` casts | Fully typed |
| **Route Registration** | Manual per route | Auto from annotations |
| **Request Parsing** | Manual `ctx.BodyParser()` | Auto-generated parsers |
| **Config Access** | String keys | Type-safe fields |
| **Boilerplate** | High | Minimal |
| **Error Detection** | Runtime panics | Compile-time errors |
| **Development** | Manual restart | Hot reload with codegen |

## Handler Signature Flexibility

Glib 2.0 supports **9 handler patterns** (inspired by Encore.dev):

```go
// 1. Raw net/http (full control)
func (c *Controller) Handle(w http.ResponseWriter, r *http.Request)

// 2. With context
func (c *Controller) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request)

// 3. Context only
func (c *Controller) Handle(ctx context.Context) error

// 4. Context + response
func (c *Controller) Handle(ctx context.Context) (*Response, error)

// 5. Context + request
func (c *Controller) Handle(ctx context.Context, req Request) error

// 6. Context + request + response (most common)
func (c *Controller) Handle(ctx context.Context, req Request) (*Response, error)

// 7. Path params + request
func (c *Controller) Handle(ctx context.Context, id uuid.UUID, req Request) (*Response, error)

// 8. Multiple path params
func (c *Controller) Handle(ctx context.Context, postID, commentID uuid.UUID) (*Response, error)

// 9. Mix everything
func (c *Controller) Handle(ctx context.Context, id uuid.UUID, w http.ResponseWriter, r *http.Request, req Request) error
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
// @Controller /api/posts
// @Middleware auth
type PostsController struct {
    DB *gorm.DB  // Auto-injected!
}

// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    var post Post
    c.DB.First(&post, id)
    return &post, nil
}
```

**Generator creates:**
```go
// generated/routes_gen.go
func registerPostsController(mux *http.ServeMux, ctrl *PostsController) {
    mux.HandleFunc("GET /api/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
        // Extract path param
        id, err := uuid.Parse(r.PathValue("id"))
        if err != nil {
            http.Error(w, "Invalid ID", 400)
            return
        }
        
        // Call handler
        result, err := ctrl.Show(r.Context(), id)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        
        // Marshal response
        json.NewEncoder(w).Encode(result)
    })
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
