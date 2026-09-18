# 05. CLI Commands and Configuration

Specification and usage guide for the Glib CLI tool and `.config.toml` configuration.

---

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Commands Reference](#commands-reference)
    - [glib init](#glib-init)
    - [glib make](#glib-make)
    - [glib generate](#glib-generate)
    - [glib validate](#glib-validate)
    - [glib dev](#glib-dev)
    - [glib version](#glib-version)
4. [Configuration File (`.config.toml`)](#configuration-file-configtoml)
5. [Configuration Precedence](#configuration-precedence)

---

## Overview

The Glib CLI is a developer-productivity tool. It is used during development for project scaffolding, annotation validation, code generation, and hot-reload development. Production builds do not depend on the CLI—standard `go build` compiles the generated code.

---

## Installation

```bash
go install github.com/azizndao/glib/cmd/glib@latest
```

Verify installation:

```bash
glib version
```

---

## Commands Reference

### `glib init`

Initializes a new Glib project structure or configures an existing directory.

```bash
glib init [directory] [flags]
```

**Flags:**

- `--module <name>`: Specify Go module name (defaults to directory name).
- `--example`: Generate an example health-check controller.
- `--minimal`: Generate a minimal project layout without examples or comments.

**Created Files:**

- `main.go` (Server lifecycle and graceful shutdown)
- `bootstrap.go` (Application container initialization and route setup)
- `configs/config.go` (`@Config` struct definition)
- `.config.toml` (Glib CLI configuration file)
- `.gitignore`
- `health/controller.go` (When `--example` is provided)

---

### `glib make`

Generates boilerplate code for controllers, DI providers, and middleware.

```bash
glib make <type> <name> [flags]
```

**Types:**

- `controller`: Creates a controller struct with CRUD handler stubs and a `models.go` file.
- `provider`: Creates a dependency injection provider function.
- `middleware`: Creates a middleware function.

**Flags:**

- `--path <dir>`: Custom destination directory (defaults to directory configured in `.config.toml`).
- `--prefix <route>`: Base route prefix for controllers (defaults to `/api/v1/<name>`).
- `--no-example`: Skips generating sample implementation code.

**Examples:**

```bash
# Create posts controller in controllers/posts/
glib make controller posts

# Create database provider in services/database.go
glib make provider database

# Create auth middleware in middleware/auth.go
glib make middleware auth
```

---

### `glib generate` (alias: `gen`)

Scans the codebase for annotations and generates all wiring and boilerplate.

```bash
glib generate [flags]
```

**Flags:**

- `--dir <path>`: Project root directory (default `.`).
- `--output <dir>`: Output directory for generated code (default `generated`).
- `--workers <n>`: Number of parallel scanner workers (default `4`).
- `--no-cache`: Disables AST hash caching.
- `--clear-cache`: Clears cached scan data before generating.
- `--verbose`: Displays detailed timing and scanning logs.

---

### `glib validate`

Performs static analysis and validates routes, handlers, and the DI dependency graph without generating code.

```bash
glib validate [flags]
```

**Flags:**

- `--dir <path>`: Project root directory (default `.`).
- `--verbose`: Displays validation diagnostics.

---

### `glib dev`

Starts a development server with native hot reload.

```bash
glib dev [flags]
```

**Features:**

- Native file watching (no third-party dependencies required).
- Automatic incremental code regeneration when `.go` files change.
- Automatic server recompilation and restart.
- Configurable debounce delay to handle rapid editor saves.

**Flags:**

- `--port <port>`: Server port to bind (default `8080` or `PORT` env var).
- `--debounce <ms>`: File watch debounce delay in milliseconds (default `300`).
- `--workers <n>`: Scanner worker count (default `4`).
- `--no-cache`: Disables scan caching.
- `--verbose`: Enables verbose watcher logs.

---

### `glib version`

Prints the current Glib CLI version.

```bash
glib version
```

---

## Configuration File (`.config.toml`)

Project defaults can be configured in `.config.toml` at the project root:

```toml
version = "2"
verbose = false

[generate]
output = "generated"
package = "generated"
workers = 4
cache = true

[make]
controllers = "controllers"
providers = "services"
middleware = "middleware"

[watch]
debounce = 300
exclude_dirs = ["vendor", "node_modules", ".git", ".glib", "tmp"]
include_files = ["*.go"]
exclude_files = ["*_test.go", "*.gen.go"]

[validation]
enabled = false
languages = ["en"]
default_language = "en"

[i18n]
enabled = false
locales_dir = "locales"
default_locale = "en"
supported_locales = ["en"]
detect_from = ["header", "query"]
query_param = "lang"
```

---

## Configuration Precedence

Options are evaluated in the following order:

1. **CLI Flags** (highest priority)
2. **`.config.toml` Settings**
3. **Hardcoded Defaults** (fallback)
