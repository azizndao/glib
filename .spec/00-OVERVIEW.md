# Glib - Overview & Architecture

## Executive Summary

**Glib** is a **code-generation-first** web framework for Go. It uses static analysis of Go AST and comment annotations to eliminate web boilerplate, ensure compile-time safety, and deliver an idiomatic Go developer experience.

With Glib, developers write standard Go functions and structs decorated with lightweight comment annotations. Glib scans the codebase and generates optimized, reflection-free HTTP routing, dependency injection wiring, parameter binding, request validation, and error serialization.

---

## Core Philosophy

1. **Code Generation Over Runtime Reflection** - All DI wiring, routing, and request parsing are generated at compile time.
2. **Idiomatic Go Everywhere** - Handlers return standard Go `(T, error)` or `error` tuples; no proprietary result wrappers required.
3. **Type Safety by Default** - Parameter extraction, query parsing, and DI injection are fully typed with no empty interface casting.
4. **Flexible Organization** - Organize your codebase however you like: layered, flat, or domain-driven.
5. **Standard Tooling Compatible** - Compatible with `go build`, `go generate`, and the standard Go ecosystem.

---

## Key Features

### 1. Compile-Time Dependency Injection

- Annotated provider functions with `// @Provider singleton` or `// @Provider transient`.
- Auto-wiring by type into controller structs—no struct tags required.
- Compile-time dependency graph validation with cycle detection and topological sorting.
- Zero reflection at runtime.

### 2. Flexible HTTP Routing & Handlers

- Annotate structs with `// @Controller` and methods with `// @Route`.
- Built on top of `github.com/go-chi/chi/v5` for high-performance, standard-compliant routing.
- Supports 3 handler signatures:
    - `(T, error)`: Standard data + error handlers (~90% of endpoints)
    - `error`: Error-only handlers (e.g. `204 No Content` for `DELETE`)
    - `(http.ResponseWriter, *http.Request)`: Full raw control for streaming, SSE, and file uploads.
- Automatic parameter binding from path wildcards, `query:` structs, `header:` structs, and `json:` bodies.

### 3. Middleware System

- `// @Middleware` annotation with automatic discovery.
- Tag-based targeting (`target=all`, `target=api`, `target=protected`).
- Per-route explicit overrides via `with=auth,ratelimit` or `with=none`.
- Priority execution order control via `order` attribute.
- Dual signature support:
    - Native Glib middleware: `func(glib.Request, glib.Next) glib.Response`
    - Standard Chi/HTTP middleware: `func(http.Handler) http.Handler`

### 4. Automatic Request Validation

- Integrated with `github.com/go-playground/validator/v10`.
- Struct validation tags (`validate:"required,min=3,email"`) on request models.
- Support for `validator.Validable` interface (`Validate() bool`) for conditional validation.
- Goyave-style structured validation error responses.

### 5. Structured Error Handling

- Encore.dev-inspired structured error model in `github.com/azizndao/glib/errs`.
- Predefined error codes (`InvalidArgument`, `NotFound`, `PermissionDenied`, `Unauthenticated`, `AlreadyExists`, etc.) mapped directly to HTTP status codes.
- Fluent builder pattern `errs.B()` and helper constructors (`errs.NewNotFound()`, `errs.NewBadRequest()`, etc.).

### 6. Type-Safe Internationalization (i18n)

- Automatic code generation from TOML translation files in `locales/`.
- Type-safe translators for errors, success messages, and validation messages in `generated/i18n`.
- Automatic locale detection from request headers and query parameters.

### 7. Type-Safe Configuration

- `@Config` annotation on struct types with `env:` and `default:` tags.
- Auto-generated configuration loaders during bootstrap.

### 8. Developer CLI & Hot Reload

- `glib dev` with native file watching, incremental scanning, and fast server restarts.
- Scaffolding commands: `glib init`, `glib make controller`, `glib make provider`, `glib make middleware`.
- Validation tool: `glib validate`.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ 1. User Application Code                                    │
│    ├── Controllers (@Controller, @Route)                    │
│    ├── Providers (@Provider singleton/transient)            │
│    ├── Middleware (@Middleware)                             │
│    ├── Configs (@Config)                                    │
│    └── Locales (locales/*.toml)                             │
└──────────────────────────────┬──────────────────────────────┘
                               │ AST Scanner & Semantic Validation
┌──────────────────────────────▼──────────────────────────────┐
│ 2. Code Generation Engine                                   │
│    ├── Dependency Graph Analysis & Cycle Detection          │
│    ├── Route Tree Construction & Middleware Resolution      │
│    └── Template Execution (Parsers, DI, Config, i18n)       │
└──────────────────────────────┬──────────────────────────────┘
                               │ Generated Output
┌──────────────────────────────▼──────────────────────────────┐
│ 3. Generated Code (generated/)                              │
│    ├── di.gen.go        # Topologically sorted container    │
│    ├── routes.gen.go    # Chi router route registrations    │
│    ├── parsers.gen.go   # Type-safe wrappers & parsers      │
│    ├── config.gen.go    # Environment variable loader       │
│    ├── validator.gen.go # Request validator initialization  │
│    └── i18n/            # Type-safe translation packages     │
└──────────────────────────────┬──────────────────────────────┘
                               │ Bootstrap & Runtime Execution
┌──────────────────────────────▼──────────────────────────────┐
│ 4. HTTP Runtime (glib & chi)                                │
│    - app.InitContainer(ctx) initializes dependencies        │
│    - app.RegisterRoutes() attaches handlers to chi.Router   │
│    - Handlers execute with zero reflection overhead         │
└─────────────────────────────────────────────────────────────┘
```

---

## Comparison: Traditional Go Frameworks vs Glib

| Feature                  | Traditional Go Frameworks                       | Glib                                                            |
| :----------------------- | :---------------------------------------------- | :-------------------------------------------------------------- |
| **Dependency Injection** | Manual wiring or runtime reflection (Dig/Fx)    | Compile-time auto-wiring & topological sort                     |
| **Route Registration**   | Manual per-endpoint route definitions           | Auto-generated from `@Controller` & `@Route`                    |
| **Request Binding**      | Manual JSON decode, query parsing, path parsing | Auto-generated type-safe parsers                                |
| **Validation**           | Manual calls to validator in every handler      | Auto-validated before handler invocation                        |
| **Error Handling**       | Inconsistent status codes and JSON formats      | Structured error codes mapped to HTTP statuses                  |
| **Performance**          | Reflection overhead during request handling     | Direct function calls with zero handler reflection              |
| **Development**          | Manual rebuilds or complex external tools       | Native hot reload with incremental code generation (`glib dev`) |

---

## Handler Signatures at a Glance

```go
// Pattern 1: Data + Error (Standard JSON API)
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Post, error)

// Pattern 2: Error Only (No Content responses)
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error

// Pattern 3: Raw HTTP (Streaming, SSE, custom formats)
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request)
```

---

## Development Workflow

```bash
# 1. Initialize project
glib init myapp --example
cd myapp

# 2. Add controllers, providers, middleware
glib make controller posts
glib make provider database
glib make middleware auth

# 3. Develop with hot reload
glib dev

# 4. Generate and build for production
glib generate
go build -o myapp
```
