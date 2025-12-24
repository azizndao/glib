# Glib CLI Tool Example

This example demonstrates the code generation capabilities of the `glib` CLI tool.

**Note:** This is a **structure demonstration** example showing the output of CLI generators. It contains generated code files but no `main.go` to run. Use the generated files as reference for your own projects.

## Project Structure

The glib project uses **Go workspaces** for clean separation between framework and CLI:

- **Framework** (`github.com/azizndao/glib`) - Core framework, no CLI dependencies
- **CLI Tool** (`github.com/azizndao/glib/tools/cli`) - Separate module with its own dependencies
- **Examples** - Each example is standalone with its own `go.mod`

This ensures framework users don't pull in unnecessary CLI dependencies like Cobra.

## Installation

```bash
# Install the glib CLI globally
go install github.com/azizndao/glib/tools/cli@latest

# Verify installation
glib --version
```

## Available Commands

### 1. Generate Model

Create a new model with optional migration and controller:

```bash
# Simple model
glib make model User

# Model with migration
glib make model User --migration

# Model with migration and controller
glib make model Product --migration --controller
```

**Generated files:**
- `app/models/product.go` - Model definition with `orm.Model` base
- `database/migrations/20241224_create_products_table.sql` - Goose migration
- `app/controllers/product_controller.go` - Resource controller with CRUD methods

### 2. Generate Controller

Create a new controller:

```bash
# Simple controller
glib make controller UserController

# Resource controller with CRUD methods
glib make controller ProductController --resource
```

### 3. Generate Migration

Create a database migration:

```bash
# SQL migration
glib make migration create_users_table

# Go migration
glib make migration add_email_to_users --type=go
```

### 4. Generate Middleware

Create a new middleware:

```bash
glib make middleware Auth
glib make middleware AdminOnly
```

## Example Usage

This directory was created using the following commands:

```bash
# Create the example directory
mkdir -p example/cli-demo
cd example/cli-demo

# Generate a Product model with migration and controller
glib make model Product --migration --controller
```

## Generated Code

### Model (`app/models/product.go`)

```go
package models

import (
	"github.com/azizndao/glib/orm"
)

// Product represents a product record
type Product struct {
	orm.Model
}

// TableName overrides the default table name
func (Product) TableName() string {
	return "products"
}
```

### Controller (`app/controllers/product_controller.go`)

```go
package controllers

import (
	"github.com/azizndao/glib"
)

// ProductController handles HTTP requests for product resources
type ProductController struct{}

// Index lists all products
// GET /products
func (ctrl *ProductController) Index(c *glib.Ctx) error {
	// TODO: Implement
	return c.JSON(map[string]string{"message": "Index"})
}

// Show displays a specific product
// GET /products/{id}
func (ctrl *ProductController) Show(c *glib.Ctx) error {
	id := c.PathValue("id")
	// TODO: Implement
	return c.JSON(map[string]string{"id": id})
}

// Store creates a new product
// POST /products
func (ctrl *ProductController) Store(c *glib.Ctx) error {
	// TODO: Implement
	return c.Status(201).JSON(map[string]string{"message": "Created"})
}

// Update modifies an existing product
// PUT /products/{id}
func (ctrl *ProductController) Update(c *glib.Ctx) error {
	id := c.PathValue("id")
	// TODO: Implement
	return c.JSON(map[string]string{"id": id, "message": "Updated"})
}

// Destroy deletes a product
// DELETE /products/{id}
func (ctrl *ProductController) Destroy(c *glib.Ctx) error {
	id := c.PathValue("id")
	// TODO: Implement
	return c.NoContent()
}
```

### Migration (`database/migrations/20241224_create_products_table.sql`)

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id CHAR(36) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE products;
-- +goose StatementEnd
```

## Features

- ✅ Model generation with orm.Model base
- ✅ Resource controllers with CRUD methods
- ✅ Goose migrations (SQL and Go)
- ✅ Middleware generation
- ✅ Automatic file organization
- ✅ Snake_case conversion for file/table names
- ✅ Proper imports and package names

## What's Next?

The CLI tool is being actively developed. Upcoming features:

- `glib new` - Create new project with full structure
- `glib serve` - Development server with hot reload
- `glib migrate` - Run migrations
- `glib migrate:rollback` - Rollback migrations
- `glib migrate:fresh` - Fresh migrations
- `glib migrate:status` - Migration status

## Template Engine

The CLI uses Go's `text/template` with embedded templates located in `internal/generators/templates/`. Custom template functions include:

- `toSnake` - Convert to snake_case
- `toCamel` - Convert to camelCase
- `toPlural` - Convert to plural form
- `toSingular` - Convert to singular form
- `timestamp` - Generate migration timestamp

## Contributing

To add a new generator:

1. Create a template in `internal/generators/templates/`
2. Add data struct in `generator.go`
3. Create command in `cmd/glib/commands/`
4. Register command in `main.go`
