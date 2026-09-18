# 03. Code Generation System

Technical architecture and specification of Glib's compile-time code generator.

---

## Table of Contents

1. [Overview](#overview)
2. [Generated Files Overview](#generated-files-overview)
3. [Generation Pipeline](#generation-pipeline)
4. [Scanner Phase (AST Analysis)](#scanner-phase-ast-analysis)
5. [Dependency Graph & Topological Sort](#dependency-graph--topological-sort)
6. [Middleware Chain Resolution](#middleware-chain-resolution)
7. [Parser & Wrapper Generation](#parser--wrapper-generation)
8. [i18n Code Generation](#i18n-code-generation)
9. [Incremental Caching](#incremental-caching)

---

## Overview

Glib scans your Go source code using Go's standard `go/ast` and `go/parser` packages to discover controllers, routes, dependency injection providers, middleware, configs, and translation files. It compiles these declarative annotations into type-safe Go code with zero runtime reflection.

### Triggering Code Generation

```bash
# Explicit generation
glib generate

# Continuous incremental generation during development
glib dev
```

---

## Generated Files Overview

Glib outputs generated code into the `generated/` package (configurable via `.config.toml`):

```
generated/
├── config.gen.go       # Loaders for @Config structs from environment variables
├── di.gen.go           # Dependency Injection container with InitContainer(ctx)
├── routes.gen.go       # Route registrations for chi.Router
├── parsers.gen.go      # Request/response parsers & handler wrappers
├── validator.gen.go    # Validator initialization (go-playground/validator/v10)
└── i18n/               # Type-safe internationalization (when enabled)
    ├── translator.go   # Central Translator struct & Chi middleware
    ├── errors.go       # Type-safe localized error messages
    ├── success.go      # Type-safe localized success messages
    ├── validation.go   # Type-safe localized validation messages
    └── messages.go     # General message catalogs
```

---

## Generation Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│ 1. SCAN                                                     │
│    - Read .go files (respecting watch.exclude_dirs/files)   │
│    - Parse AST with go/parser & extract comment annotations │
│    - Resolve imports and struct type definitions            │
└──────────────────────────────┬──────────────────────────────┘
                               │ AST Metadata
┌──────────────────────────────▼──────────────────────────────┐
│ 2. VALIDATE                                                 │
│    - Construct Dependency Graph and detect circular cycles  │
│    - Verify handler parameters and return signatures        │
│    - Validate middleware names and route paths              │
└──────────────────────────────┬──────────────────────────────┘
                               │ Validated Project Model
┌──────────────────────────────▼──────────────────────────────┐
│ 3. GENERATE                                                 │
│    - Render di.gen.go with topologically sorted providers   │
│    - Render routes.gen.go with Chi router mounting          │
│    - Render parsers.gen.go with request binding & validation│
│    - Render config.gen.go, validator.gen.go, and i18n/      │
│    - Format with goimports / go/format                      │
└─────────────────────────────────────────────────────────────┘
```

---

## Scanner Phase (AST Analysis)

The scanner parses comment groups to extract annotations:

- `// @Controller path=/api/v1/posts tags=api,protected`
- `// @Route method=GET path=/{id} tags=protected with=auth`
- `// @Provider singleton` or `// @Provider transient`
- `// @Middleware name=auth target=protected order=10`
- `// @Config`

### Type Resolution & Struct Tag Parsing

The scanner identifies:

1. **Receiver structs**: Associates methods with their parent `@Controller`.
2. **Handler signatures**: Determines whether the method is `(T, error)`, `error`, or raw `(w, r)`.
3. **Parameter types**: Checks for path parameters (by name), `query:` tags, `header:` tags, and `json:` tags.
4. **Validation requirements**: Checks if request models contain `validate:` tags.
5. **Response metadata**: Inspects response structs for `response:"httpstatus"` and `header:"..."` tags.

---

## Dependency Graph & Topological Sort

Providers annotated with `@Provider` are assembled into a Directed Acyclic Graph (DAG):

```
Config ──────► Database ──────► PostService ──────► PostsController
   │                               ▲
   └──────────► JWTService ────────┘
```

1. **Topological Sort**: Determines the exact instantiation order required to satisfy all dependencies in `di.gen.go`.
2. **Cycle Detection**: Any cyclic dependencies (e.g. `A -> B -> A`) cause a compile-time error listing the offending cycle.
3. **Lifecycles**:
    - `singleton`: Stored directly as fields in `App.ProviderContainer`.
    - `transient`: Generated as factory functions (`AuditorFactory func() *services.Auditor`) invoked per controller instance.

---

## Middleware Chain Resolution

For each route, Glib calculates the exact middleware chain at generation time:

1. **Explicit Override (`with=...`)**:
    - If `with=none`, no middleware is applied to the handler.
    - If `with=auth,ratelimit`, only those specified middleware are executed.
2. **Tag-Based Auto-Targeting**:
    - Glib collects tags from both the parent `@Controller` and the handler `@Route`.
    - Any registered middleware whose `target` matches one of the tags (or `target=all`) is included.
3. **Chain Sorting**:
    - Middleware is sorted by `order` (ascending).
    - Ties are broken deterministically by file path and source line.

---

## Parser & Wrapper Generation (`parsers.gen.go`)

For each non-raw handler, Glib generates a dedicated HTTP handler wrapper function:

```go
// Example generated wrapper
func handlePostsControllerCreate(app *App) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // 1. Decode request body
        var req posts.CreatePostRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            glib.WriteError(w, errs.B().Code(errs.InvalidArgument).Msg("invalid request body").Cause(err).Err())
            return
        }

        // 2. Validate request
        if err := app.Validator.Validate(req); err != nil {
            glib.WriteError(w, errs.B().Code(errs.InvalidArgument).Msg("validation failed").Details(err).Err())
            return
        }

        // 3. Invoke controller method
        result, err := app.controllers.postsController.Create(ctx, req)
        if err != nil {
            glib.WriteError(w, err)
            return
        }

        // 4. Write response with metadata (status 201, headers, etc.)
        glib.WriteResponseWithMetadata(w, "POST", result)
    }))

    // 5. Apply middleware chain in reverse order
    handler = app.middleware.authMiddleware(handler)

    return handler.ServeHTTP
}
```

---

## i18n Code Generation

When i18n is enabled in `.config.toml`, Glib scans TOML files in `locales/` and generates:

- Strongly typed accessor methods with formatting arguments:
    ```go
    c.I18n.Errors.Posts.NotFound(ctx, id.String())
    ```
- Locale detection middleware matching `Accept-Language` headers and URL query parameters.
- Validation translation bridges converting `validator.ValidationErrors` into translated messages.

---

## Incremental Caching

In development mode (`glib dev`), Glib maintains an in-memory hash cache of scanned source files. When files change:

1. Only modified files are re-scanned.
2. Unchanged AST nodes are reused from cache.
3. Regeneration completes in milliseconds.
