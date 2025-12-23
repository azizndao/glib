# Phase 3: CLI Tool (Artisan-Style)

**Timeline**: Weeks 7-8  
**Priority**: High - Dramatically improves developer productivity  
**Dependencies**: Phase 1 (Foundation), Phase 2 (Database)

## Overview

Build a comprehensive command-line tool (`glib`) inspired by Laravel's Artisan that provides:
- Project scaffolding (`glib new`)
- Code generators (`glib make:*`)
- Database operations (`glib migrate`, `glib db:seed`)
- Development server (`glib serve`)
- Application maintenance commands

## Package Structure

```
cmd/glib/
├── main.go                 # CLI entry point
├── app.go                  # CLI application setup
├── command.go              # Base command interface
└── commands/               # All commands
    ├── new.go              # Create new project
    ├── serve.go            # Development server
    ├── make_model.go       # Generate model
    ├── make_controller.go  # Generate controller
    ├── make_middleware.go  # Generate middleware
    ├── make_migration.go   # Generate migration
    ├── make_seeder.go      # Generate seeder
    ├── make_factory.go     # Generate factory
    ├── make_policy.go      # Generate policy
    ├── migrate.go          # Run migrations
    ├── migrate_rollback.go # Rollback migrations
    ├── migrate_fresh.go    # Fresh migrations
    ├── db_seed.go          # Run seeders
    ├── route_list.go       # List routes
    ├── queue_work.go       # Queue worker
    └── cache_clear.go      # Clear caches

internal/generators/        # Code generation
├── generator.go            # Base generator
├── model.go                # Model generator
├── controller.go           # Controller generator
├── migration.go            # Migration generator
└── templates/              # Code templates
    ├── model.tmpl
    ├── controller.tmpl
    ├── middleware.tmpl
    ├── migration.tmpl
    ├── policy.tmpl
    └── ...
```

## CLI Framework

**Library Choice**: Use `spf13/cobra` for command structure

### Base Command Structure

```go
package commands

import "github.com/spf13/cobra"

// Command interface for all glib commands
type Command interface {
    Configure() *cobra.Command
}

// BaseCommand provides common functionality
type BaseCommand struct {
    Name        string
    Description string
    Args        cobra.PositionalArgs
    Flags       map[string]interface{}
}
```

## Core Commands

### 1. New Project (`glib new`)

Creates a complete project structure with all boilerplate:

```bash
glib new blog
glib new blog --template=api  # API-only template
glib new blog --no-git        # Skip git initialization
glib new blog --database=postgres
```

**Generated Structure:**

```
blog/
├── cmd/
│   └── main.go
├── app/
│   ├── controllers/
│   │   └── .gitkeep
│   ├── models/
│   │   └── .gitkeep
│   ├── middleware/
│   │   └── .gitkeep
│   └── policies/
│       └── .gitkeep
├── config/
│   ├── app.go
│   ├── database.go
│   ├── cache.go
│   ├── queue.go
│   └── auth.go
├── database/
│   ├── migrations/
│   │   └── .gitkeep
│   ├── seeders/
│   │   └── database_seeder.go
│   └── factories/
│       └── .gitkeep
├── routes/
│   ├── api.go
│   └── web.go
├── storage/
│   ├── logs/
│   ├── cache/
│   └── app/
├── tests/
│   ├── unit/
│   └── integration/
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

### 2. Development Server (`glib serve`)

Runs development server with hot reload:

```bash
glib serve                    # Default :8080
glib serve --port=3000       # Custom port
glib serve --host=0.0.0.0    # Bind to all interfaces
```

Uses `cosmtrek/air` for hot reload or implements file watcher.

### 3. Code Generators (`glib make:*`)

#### Make Model

```bash
glib make:model User
glib make:model User --migration  # Also create migration
glib make:model Post --migration --controller
```

**Generated `app/models/user.go`:**

```go
package models

import (
    "github.com/azizndao/glib/orm"
    "time"
)

type User struct {
    orm.Model
    // Add your fields here
}

// TableName overrides the default table name
func (User) TableName() string {
    return "users"
}
```

#### Make Controller

```bash
glib make:controller UserController
glib make:controller UserController --resource  # CRUD methods
glib make:controller Api/UserController        # Namespaced
```

**Generated `app/controllers/user_controller.go`:**

```go
package controllers

import "github.com/azizndao/glib"

type UserController struct{}

// Index lists all users
func (ctrl *UserController) Index(c *glib.Ctx) error {
    // TODO: Implement
    return c.JSON(map[string]string{"message": "Index"})
}

// Show displays a specific user
func (ctrl *UserController) Show(c *glib.Ctx) error {
    id := c.PathValue("id")
    // TODO: Implement
    return c.JSON(map[string]string{"id": id})
}

// Store creates a new user
func (ctrl *UserController) Store(c *glib.Ctx) error {
    // TODO: Implement
    return c.Status(201).JSON(map[string]string{"message": "Created"})
}

// Update modifies an existing user
func (ctrl *UserController) Update(c *glib.Ctx) error {
    id := c.PathValue("id")
    // TODO: Implement
    return c.JSON(map[string]string{"id": id, "message": "Updated"})
}

// Destroy deletes a user
func (ctrl *UserController) Destroy(c *glib.Ctx) error {
    id := c.PathValue("id")
    // TODO: Implement
    return c.NoContent()
}
```

#### Make Migration

```bash
glib make:migration create_users_table
glib make:migration add_published_to_posts
```

**Generated `database/migrations/2024_12_23_100000_create_users_table.go`:**

```go
package migrations

import "gorm.io/gorm"

type CreateUsersTable struct{}

func (m *CreateUsersTable) Up(db *gorm.DB) error {
    // TODO: Implement migration
    return db.AutoMigrate(&models.User{})
}

func (m *CreateUsersTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable("users")
}

func (m *CreateUsersTable) Version() string {
    return "2024_12_23_100000_create_users_table"
}
```

#### Make Middleware

```bash
glib make:middleware AdminOnly
glib make:middleware Auth/RequireRole
```

#### Make Policy

```bash
glib make:policy PostPolicy
```

#### Make Seeder

```bash
glib make:seeder UserSeeder
```

#### Make Factory

```bash
glib make:factory UserFactory
```

### 4. Database Commands

#### Migrations

```bash
# Run all pending migrations
glib migrate

# Rollback last batch
glib migrate:rollback

# Rollback specific steps
glib migrate:rollback --step=2

# Reset all migrations
glib migrate:reset

# Drop all tables and re-run migrations
glib migrate:fresh

# Rollback and re-run
glib migrate:refresh

# Show migration status
glib migrate:status
```

#### Seeders

```bash
# Run all seeders
glib db:seed

# Run specific seeder
glib db:seed --class=UserSeeder

# Fresh database with seeds
glib migrate:fresh --seed
```

### 5. Route Commands

```bash
# List all routes
glib route:list

# Filter by method
glib route:list --method=GET

# Filter by path
glib route:list --path=/api
```

**Output:**

```
+--------+-------------------------+------------------+
| Method | Path                    | Handler          |
+--------+-------------------------+------------------+
| GET    | /api/users              | UserController@Index |
| POST   | /api/users              | UserController@Store |
| GET    | /api/users/{id}         | UserController@Show  |
| PUT    | /api/users/{id}         | UserController@Update|
| DELETE | /api/users/{id}         | UserController@Destroy |
+--------+-------------------------+------------------+
```

### 6. Queue Commands

```bash
# Start queue worker
glib queue:work

# Specify queue
glib queue:work --queue=emails

# Specify connection
glib queue:work --connection=redis

# Number of jobs to process
glib queue:work --tries=3

# Worker timeout
glib queue:work --timeout=60

# Start scheduler
glib schedule:work

# Run scheduled tasks once (for cron)
glib schedule:run
```

### 7. Cache Commands

```bash
# Clear all caches
glib cache:clear

# Clear specific cache store
glib cache:clear --store=redis

# Cache configuration
glib config:cache

# Clear config cache
glib config:clear
```

## Code Generation System

### Template Engine

Use Go's `text/template` for code generation:

```go
// internal/generators/generator.go

type Generator struct {
    templates *template.Template
    data      map[string]interface{}
}

func NewGenerator() *Generator {
    return &Generator{
        templates: template.Must(template.ParseFS(templatesFS, "templates/*.tmpl")),
        data:      make(map[string]interface{}),
    }
}

func (g *Generator) Generate(templateName, outputPath string, data interface{}) error {
    file, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    return g.templates.ExecuteTemplate(file, templateName, data)
}
```

### Model Template

**`internal/generators/templates/model.tmpl`:**

```go
package {{.Package}}

import (
{{- range .Imports}}
    "{{.}}"
{{- end}}
)

{{if .Comment}}// {{.Comment}}{{end}}
type {{.Name}} struct {
    orm.Model
    {{range .Fields}}
    {{.Name}} {{.Type}} `{{.Tags}}`
    {{- end}}
    {{range .Relationships}}
    {{.Name}} {{.Type}} `{{.Tags}}`
    {{- end}}
}

{{if .TableName}}
// TableName overrides the default table name
func ({{.Name}}) TableName() string {
    return "{{.TableName}}"
}
{{end}}

{{range .Methods}}
{{.}}
{{end}}
```

### Controller Template

Similar structure for controllers, middleware, policies, etc.

## Installation

### Global Installation

```bash
# Install glib CLI globally
go install github.com/azizndao/glib/cmd/glib@latest

# Verify installation
glib --version
```

### Project-Specific

```bash
# Add to project
go get github.com/azizndao/glib/cmd/glib

# Run via go run
go run github.com/azizndao/glib/cmd/glib make:model User
```

## Interactive Prompts

Use `manifoldco/promptui` for interactive experiences:

```go
// Example: Choose database when creating project

prompt := promptui.Select{
    Label: "Select Database",
    Items: []string{"PostgreSQL", "MySQL", "SQLite"},
}

_, result, err := prompt.Run()
```

## Progress Indicators

Use `schollz/progressbar` for long-running operations:

```go
bar := progressbar.Default(100)
for i := 0; i < 100; i++ {
    bar.Add(1)
    time.Sleep(10 * time.Millisecond)
}
```

## Testing

### CLI Command Tests

```go
func TestNewProjectCommand(t *testing.T) {
    tmpDir := t.TempDir()
    
    cmd := NewProjectCommand()
    cmd.SetArgs([]string{"testproject"})
    cmd.SetOut(new(bytes.Buffer))
    
    err := cmd.Execute()
    assert.NoError(t, err)
    
    // Verify structure created
    assert.DirExists(t, filepath.Join(tmpDir, "testproject"))
    assert.FileExists(t, filepath.Join(tmpDir, "testproject/go.mod"))
}
```

### Generator Tests

```go
func TestModelGenerator(t *testing.T) {
    gen := NewModelGenerator()
    
    err := gen.Generate("User", GeneratorOptions{
        Package: "models",
        Fields: []Field{
            {Name: "Name", Type: "string"},
            {Name: "Email", Type: "string"},
        },
    })
    
    assert.NoError(t, err)
    assert.FileExists(t, "app/models/user.go")
    
    // Verify content
    content, _ := ioutil.ReadFile("app/models/user.go")
    assert.Contains(t, string(content), "type User struct")
}
```

## Success Metrics

### Phase 3 Complete When:

- ✅ `glib new` creates complete project structure
- ✅ All `make:*` commands generate valid code
- ✅ Migration commands work correctly
- ✅ `glib serve` runs development server
- ✅ `route:list` shows all registered routes
- ✅ All commands have comprehensive help text
- ✅ Interactive prompts enhance UX
- ✅ All tests pass
- ✅ Documentation complete with examples

## Next Phase

Phase 4 will build authentication system that can be generated with:

```bash
glib make:auth
# Generates: auth controllers, middleware, routes, migrations
```
