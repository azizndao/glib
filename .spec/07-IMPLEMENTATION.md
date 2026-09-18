# 07. Technical Implementation Guide

Internal architecture, package breakdown, and implementation details of the Glib framework codebase.

---

## Table of Contents

1. [Codebase Layout](#codebase-layout)
2. [Package: `internal/scanner`](#package-internalscanner)
3. [Package: `internal/generator`](#package-internalgenerator)
4. [Package: `internal/cli`](#package-internalcli)
5. [Package: `validator`](#package-validator)
6. [Package: `errs`](#package-errs)
7. [Runtime Packages & Helpers](#runtime-packages--helpers)

---

## Codebase Layout

```
glib/
├── cmd/
│   └── glib/                   # CLI binary entry point (main.go)
├── internal/
│   ├── scanner/                # AST parsing, annotation extraction, type resolution
│   ├── generator/              # Code generator templates, DI sorting, Chi route mapping
│   ├── cli/                    # Cobra CLI commands, file watcher, scaffolding
│   └── cache/                  # In-memory scan cache
├── errs/                       # Structured error types, codes, builder API
├── validator/                  # Request validation and Goyave-style error formatting
├── request.go                  # glib.Request wrapper
├── middleware.go               # glib.Middleware, Next, and Response definitions
└── writer.go                   # Response writing, metadata extraction, error JSON serialization
```

---

## Package: `internal/scanner`

Responsible for AST traversal and metadata extraction without executing user code.

### Key Components

- **`scanner.go`**: Core orchestration. Manages the AST file set, package resolution, and worker pools for parallel scanning.
- **`annotations.go`**: Regex matching for `// @Controller`, `// @Route`, `// @Provider`, `// @Middleware`, and `// @Config`.
- **`controllers.go`**: Identifies controller structs, extracts base routes, and tags.
- **`handlers.go`**: Inspects method signatures to classify handlers (`PatternDataError`, `PatternErrorOnly`, `PatternRawHTTP`), analyzes parameters (path params, `query:`/`header:`/`json:` structs), and response metadata.
- **`providers.go`**: Extracts DI provider functions, parameter dependencies, return types, and lifecycles (`singleton` vs `transient`).
- **`config.go`**: Extracts `@Config` struct fields, nested structs, and `env:`/`default:` tags.
- **`cache.go`**: Calculates SHA-256 hashes of file contents to enable incremental scanning in development mode.

---

## Package: `internal/generator`

Translates scanned AST models into formatted Go code.

### Key Components

- **`generator.go`**: Main code generation driver. Orchestrates file outputs and formats code using `go/format` and `goimports`.
- **`depgraph.go`**: Constructs the dependency graph for all providers and controllers. Runs Tarjan's / Kahn's algorithm for topological sorting and circular dependency detection.
- **`di.go`**: Generates `di.gen.go` containing `App`, `ProviderContainer`, `ControllerContainer`, and `InitContainer(ctx)`.
- **`routes.go`**: Generates `routes.gen.go` which registers routes onto `chi.Router` with mounted paths and handler functions.
- **`parsers.gen.go`**: Generates `parsers.gen.go` with request JSON decoders, query/header binders, validation invocations, and error handling wrappers.
- **`config.go`**: Generates `config.gen.go` with loaders for `@Config` structs.
- **`i18n.go`**: Generates `generated/i18n` with type-safe message getters and middleware.

---

## Package: `internal/cli`

Implements CLI commands via `github.com/spf13/cobra`.

### Key Components

- **`root.go`**: Defines root CLI command and registers subcommands (`init`, `make`, `generate`, `validate`, `dev`, `version`).
- **`init.go`**: Handles project scaffolding (`main.go`, `bootstrap.go`, `.config.toml`, `.gitignore`).
- **`make.go`**: Renders templates for new controllers, providers, and middleware.
- **`dev.go` & `watcher.go`**: Implements native file watching with debouncing via `fsnotify`. Triggers incremental codegen, compiles the binary, and manages child server processes.
- **`validate.go`**: Runs scanner and dependency graph verification in dry-run mode.

---

## Package: `validator`

Provides request validation integration with `github.com/go-playground/validator/v10`.

### Key Components

- **`validator.go`**: `Validator` wrapper with language translator support.
- **`errors.go`**: `ValidationErrors` struct producing Goyave-style nested errors partitioned into `body`, `query`, and `headers`.
- **`validable.go`**: Defines `Validable` interface (`Validate() bool`).

---

## Package: `errs`

Implements Encore.dev-style structured error handling.

### Key Components

- **`codes.go`**: Defines `ErrCode` constants and their corresponding HTTP status codes.
- **`builder.go`**: Implements fluent error builder `errs.B()`.
- **`helpers.go`**: Implements helper constructors (`NewNotFound()`, `NewBadRequest()`, etc.) and `WithMessage()`.
- **`errs.go`**: Core `Error` struct with `Code`, `Message`, `Details`, `underlying` error, and wrapping utilities (`Wrap`, `WrapCode`).

---

## Runtime Packages & Helpers

- **`glib.Request`** (`request.go`): Immutable wrapper around `http.Request` providing accessor methods (`Header()`, `Query()`, `PathValue()`, `WithValue()`, `WithValues()`).
- **`glib.Response`** & **`glib.Next`** (`middleware.go`): Middleware chaining interfaces.
- **`glib.WriteResponseWithMetadata`** (`writer.go`): Analyzes response struct tags (`response:"httpstatus"`, `header:"..."`) with sync.RWMutex caching to avoid repeated reflection costs.
- **`glib.WriteError`** (`writer.go`): Serializes `*errs.Error` into JSON error responses with appropriate HTTP status codes.
