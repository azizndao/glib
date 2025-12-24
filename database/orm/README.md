# ORM Package

The `orm` package provides a Laravel-inspired Active Record pattern implementation for Go, built on top of GORM v1.30+. It offers a **type-safe generics API** with full context support.

## Features

- **Base Model** with UUID primary keys, automatic timestamps and soft deletes
- **Type-Safe Generics API** with context support
- **Query Scopes** for reusable query logic
- **Pagination** support with helper functions
- **Soft Deletes** with restore capability
- **Batch Processing** with Chunk helper
- **Search** across multiple columns
- **Custom Scopes** for complex queries
- **Full GORM Compatibility** - direct access to GORM v1.30+ features

## Quick Start

### 1. Define Models

Embed `orm.Model` to get automatic UUID primary key, timestamps, and soft delete support:

```go
import (
    "github.com/azizndao/glib/orm"
    "github.com/google/uuid"
)

type User struct {
    orm.Model        // Provides: ID (uuid.UUID), CreatedAt, UpdatedAt, DeletedAt
    Name   string
    Email  string `gorm:"uniqueIndex"`
    Age    int
    Active bool `gorm:"default:true"`
}
```

**Note**: UUIDs are automatically generated using UUIDv7 when creating records via the `BeforeCreate` hook.

### 2. Basic CRUD Operations

The generics API is type-safe, context-aware, and returns values directly:

```go
ctx := context.Background()

// Create - returns error
user := &User{Name: "Alice", Email: "alice@example.com", Age: 28}
err := orm.G[User](db).Create(ctx, user)

// Query single - returns (User, error)
user, err := orm.G[User](db).Where("email = ?", "alice@example.com").First(ctx)

// Find multiple - returns ([]User, error)
users, err := orm.G[User](db).Where("age > ?", 18).Order("name ASC").Find(ctx)

// Update - returns (rowsAffected int, error)
rows, err := orm.G[User](db).Where("id = ?", user.ID).Update(ctx, "age", 29)

// Delete (soft delete) - returns (rowsAffected int, error)
rows, err := orm.G[User](db).Where("id = ?", user.ID).Delete(ctx)

// Count - returns (int64, error)
count, err := orm.G[User](db).Where("active = ?", true).Count(ctx, "*")
```

**Benefits:**
- ✅ Type-safe: Returns `T` and `[]T` directly
- ✅ Context-aware: All methods require `context.Context`
- ✅ Cleaner: No pointer arguments needed
- ✅ Modern: Uses Go 1.18+ generics

### 3. Advanced Features

#### Pagination

```go
ctx := context.Background()
builder := orm.G[User](db).Where("active = ?", true).Order("name ASC")
paginator, err := orm.Paginate(ctx, builder, 1, 15)  // page 1, 15 per page

fmt.Printf("Page %d of %d\n", paginator.CurrentPage, paginator.LastPage)
fmt.Printf("Total: %d\n", paginator.Total)
for _, user := range paginator.Data {
    fmt.Printf("- %s\n", user.Name)
}
```

#### Batch Processing

```go
builder := orm.G[User](db).Where("active = ?", true)
err := orm.Chunk(ctx, builder, 100, func(users []User) error {
    // Process batch of 100 users
    for _, user := range users {
        // Do something with user
    }
    return nil
})
```

#### Helper Functions

```go
// Check existence
exists, err := orm.Exists(ctx, orm.G[User](db).Where("email = ?", email))
doesntExist, err := orm.DoesntExist(ctx, orm.G[User](db).Where("email = ?", email))
```

#### Using GORM Features Directly

The generics API is a thin wrapper around GORM, giving you full access to GORM v1.30+ features:

```go
// Use gorm.G[T] directly for advanced GORM features
users, err := gorm.G[User](db).
    Where("age > ?", 18).
    Preload("Posts", func(db gorm.PreloadBuilder) error {
        db.Where("published = ?", true)
        db.LimitPerRecord(5)  // GORM v1.30+ feature
        return nil
    }).
    Joins(clause.LeftJoin.Association("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
        db.Where(map[string]any{"name": "Acme"})
        return nil
    }).
    Find(ctx)
```

### 4. Query Scopes

Scopes are reusable query functions that can be applied to queries:

```go
// Apply scope to DB before using G[T]
scopedDB := orm.WhereColumn("active", true)(db)
activeUsers, err := orm.G[User](scopedDB).Order("name ASC").Find(ctx)

// Or use GORM's traditional Scopes with Model()
var users []User
err := db.Model(&User{}).Scopes(
    orm.WhereColumn("active", true),
    orm.OrderByColumn("created_at", "DESC"),
).Find(&users).Error

// Combine multiple scopes
scopedDB = orm.WhereColumn("published", true)(db)
scopedDB = orm.BelongsTo("user_id", userID)(scopedDB)
posts, err := orm.G[Post](scopedDB).Find(ctx)

// Custom scope
seniorUsers := func(db *gorm.DB) *gorm.DB {
    return db.Where("age >= ?", 35)
}
seniors, err := orm.G[User](seniorUsers(db)).Order("age DESC").Find(ctx)
```

## API Reference

### Generics API

All generics API methods are context-aware and return typed results:

#### Creating Queries
- `G[T](db, opts...)` - Create a new type-safe query builder (thin wrapper around `gorm.G[T]`)

#### Query Methods
- `Where(query, args...)` - Add WHERE clause
- `Or(query, args...)` - Add OR clause
- `Not(query, args...)` - Add NOT clause
- `Order(value)` - Order results
- `Limit(n)` / `Offset(n)` - Pagination
- `Distinct(args...)` / `Group(name)` / `Having(query, args...)` - Grouping
- `Joins(target, on)` - Join tables with GORM's clause API
- `Preload(association, query)` - Eager load with conditions

#### Execution Methods
- `First(ctx)` → `(T, error)` - Get first record
- `Take(ctx)` → `(T, error)` - Get one record (no ordering)
- `Last(ctx)` → `(T, error)` - Get last record
- `Find(ctx)` → `([]T, error)` - Get all matching records
- `Count(ctx, column)` → `(int64, error)` - Count records
- `Create(ctx, value)` → `error` - Insert record
- `CreateInBatches(ctx, values, batchSize)` → `error` - Batch insert
- `Update(ctx, column, value)` → `(int, error)` - Update column
- `Updates(ctx, values)` → `(int, error)` - Update multiple columns
- `Delete(ctx)` → `(int, error)` - Soft delete

#### Helper Functions
- `Paginate[T](ctx, chain, page, perPage)` → `(*Paginator[T], error)`
- `Chunk[T](ctx, chain, batchSize, callback)` → `error`
- `Exists[T](ctx, chain)` → `(bool, error)`
- `DoesntExist[T](ctx, chain)` → `(bool, error)`

## Built-in Scopes

Scopes are functions of type `func(*gorm.DB) *gorm.DB` that modify queries.

### Generic Factory Scopes
- `WhereColumn(column, value)` - Generic scope for any column/value WHERE condition
- `WhereNotColumn(column, value)` - Generic scope for any column/value WHERE NOT condition
- `OrderByColumn(column, direction)` - Generic ordering scope (direction: "ASC" or "DESC")

### Specialized Scopes
- `OrderByCreatedAt(direction)` - Order by created_at
- `OrderByUpdatedAt(direction)` - Order by updated_at
- `PaginateScope(page, perPage)` - Pagination scope
- `Search(columns, term)` - Search across multiple columns using LIKE
- `WithRelations(relations...)` - Preload relations
- `BelongsTo(foreignKey, id)` - Filter by foreign key (accepts any type including UUID)

**Design Philosophy**: The ORM provides generic factory functions (`WhereColumn`, `WhereNotColumn`, `OrderByColumn`) instead of opinionated scopes like `Active` or `Published`. This allows you to create scopes specific to your domain without assuming column names exist across all tables.

**Example**: Create your own domain-specific scopes:
```go
// Define your own scopes for your models
ActiveUsers := orm.WhereColumn("active", true)
PublishedPosts := orm.WhereColumn("published", true)
DraftPosts := orm.WhereColumn("published", false)

// Use them in queries (apply to DB first)
users, err := orm.G[User](ActiveUsers(db)).Find(ctx)
posts, err := orm.G[Post](PublishedPosts(db)).Find(ctx)

// Or use with GORM's Model()
db.Model(&User{}).Scopes(ActiveUsers).Find(&users)
```

## Model Methods

The `orm.Model` struct provides these helper methods:

- `IsNew()` - Check if model hasn't been saved (ID is uuid.Nil)
- `IsDeleted()` - Check if soft deleted

**Direct Field Access**: Access model fields directly for cleaner, more idiomatic Go code:
```go
user := User{Name: "John"}
db.Create(&user)

// Access fields directly
fmt.Println(user.ID)         // uuid.UUID
fmt.Println(user.CreatedAt)  // time.Time
fmt.Println(user.UpdatedAt)  // time.Time
fmt.Println(user.DeletedAt)  // gorm.DeletedAt

// Check status with helper methods
if user.IsNew() {
    // Not yet saved
}
if user.IsDeleted() {
    // Soft deleted
}
```

## Example

A complete example is available in `example/orm/main.go`:
- Model definitions with relationships
- Type-safe CRUD operations with generics API
- Query scopes (built-in and custom)
- Pagination and batch processing
- Soft deletes
- Search functionality

Run the example:

```bash
cd example/orm
go run main.go
```

## Testing

The package includes comprehensive tests:

- `model_test.go` - 15 tests for Model functionality (UUID support, hooks, methods)
- `builder_test.go` - 16 tests for Generics API (type-safe queries, context support)
- `scopes_test.go` - 13 tests for query scopes (generic scopes, search, combinations)

**Total: 44 tests covering all ORM functionality**

Run tests:

```bash
go test ./orm/... -v
```

## Integration with glib

The ORM package integrates seamlessly with glib's database layer:

```go
// Using with database.Manager
manager, _ := container.Resolve[*database.Manager](app.Container())
db, _ := manager.DB() // Get default connection

// Use ORM generics API
ctx := context.Background()
users, err := orm.G[User](db.DB()).Where("active = ?", true).Find(ctx)
```

## Notes

- **UUID Primary Keys**: All models use UUIDv7 for primary keys, automatically generated via `BeforeCreate` hook
- **SQLite Compatibility**: UUIDs stored as `char(36)` for SQLite support (can be changed to native `uuid` type for PostgreSQL)
- All queries respect soft deletes by default (use `db.Unscoped()` to include deleted records)
- Timestamps (created_at, updated_at) are managed automatically by GORM
- Error handling follows Go conventions - always check returned errors
- Access model fields directly (`.ID`, `.CreatedAt`) instead of using getters for idiomatic Go code
- Scopes work best when applied to `*gorm.DB` before wrapping with `G[T]()`, or used with GORM's `Model().Scopes()` pattern

## Design Philosophy

The ORM package follows Laravel's Eloquent design philosophy with Go idioms:

1. **Expressive Syntax** - Readable, self-documenting queries
2. **Type Safety** - Leverage Go generics for compile-time safety
3. **Context-Aware** - All operations support context for cancellation and tracing
4. **Scopes** - Reusable query logic with generic factory functions
5. **Convention Over Configuration** - Sensible defaults (UUIDs, timestamps, soft deletes)
6. **Soft Deletes** - Safe record deletion with recovery option
7. **Direct Field Access** - Go-idiomatic direct field access instead of getters
8. **Flexible Scopes** - Generic scope factories instead of opinionated assumptions about schema
9. **GORM Compatible** - Thin wrapper around GORM for full feature access

## Future Enhancements

Potential additions for future versions:

- Relationship helpers (HasOne, HasMany, BelongsTo, ManyToMany)
- Additional model events/hooks (AfterCreate, AfterUpdate, etc.)
- Migration system
- Query caching
- Eager loading optimization
- Global scopes
- Model observers
