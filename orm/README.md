# ORM Package

The `orm` package provides a Laravel-inspired Active Record pattern implementation for Go, built on top of GORM. It offers both a traditional fluent API and a modern **type-safe generics API** (GORM v1.30+).

## Features

- **Base Model** with UUID primary keys, automatic timestamps and soft deletes
- **Type-Safe Generics API** with context support (NEW!)
- **Fluent Query Builder** with method chaining
- **Query Scopes** for reusable query logic
- **Pagination** support
- **Soft Deletes** with restore capability
- **Search** across multiple columns
- **Custom Scopes** for complex queries
- **Full GORM Compatibility** - use GORM features directly

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

### 2. Choose Your API Style

#### A. Generics API (Recommended for New Code)

Type-safe, context-aware, returns values directly:

```go
ctx := context.Background()

// Create - returns error
user := &User{Name: "Alice", Email: "alice@example.com", Age: 28}
err := orm.G[User](db).Create(ctx, user)

// Query - returns (User, error)
user, err := orm.G[User](db).Where("email = ?", "alice@example.com").First(ctx)

// Find - returns ([]User, error)
users, err := orm.G[User](db).Where("age > ?", 18).Order("name ASC").Find(ctx)

// Update - returns (rowsAffected int, error)
rowsAffected, err := orm.G[User](db).Where("id = ?", user.ID).Update(ctx, "age", 29)

// Delete - returns (rowsAffected int, error)
rowsAffected, err := orm.G[User](db).Where("id = ?", user.ID).Delete(ctx)

// Count - returns (int64, error)
count, err := orm.G[User](db).Where("active = ?", true).Count(ctx, "*")
```

**Benefits:**
- ✅ Type-safe: Returns `T` and `[]T` directly
- ✅ Context-aware: All methods require `context.Context`
- ✅ Cleaner: No pointer arguments needed
- ✅ Modern: Uses Go 1.18+ generics

#### B. Traditional API (Backward Compatible)

Classic Builder pattern with pointer arguments:

```go
// Create builder
builder := orm.NewBuilder(db, &User{})

// Simple query
var users []User
builder.Where("age > ?", 18).OrderBy("name ASC").Find(&users)

// Method chaining
builder.Where("active = ?", true).
    Where("age >= ?", 21).
    OrderBy("created_at DESC").
    Limit(10).
    Find(&users)
```

Both APIs can be used together in the same project!

### 3. Generics API - Advanced Features

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

The generics API gives you full access to GORM's features:

```go
// Using GORM's native generics API directly
users, err := gorm.G[User](db).
    Where("age > ?", 18).
    Scopes(func(stmt *gorm.Statement) {
        stmt.DB = stmt.DB.Where("active = ?", true)
    }).
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

### 4. Traditional API - Scopes

```go
// Built-in generic scopes
var activeUsers []User
orm.NewBuilder(db, &User{}).
    Scopes(orm.WhereColumn("active", true)).
    Find(&activeUsers)

// Combine multiple scopes
var results []Post
orm.NewBuilder(db, &Post{}).
    Scopes(
        orm.WhereColumn("published", true),
        orm.BelongsTo("user_id", userID), // userID can be UUID or any type
    ).
    Find(&results)

// Custom ordering
orm.NewBuilder(db, &Post{}).
    Scopes(orm.OrderByColumn("created_at", "DESC")).
    Find(&posts)

// Custom scope
seniorUsers := func(db *gorm.DB) *gorm.DB {
    return db.Where("age >= ?", 35)
}

orm.NewBuilder(db, &User{}).Scopes(seniorUsers).Find(&users)
```

## API Reference

### Generics API Methods

All generics API methods are context-aware and return typed results:

#### Query Methods
- `G[T](db, opts...)` - Create a new generic query builder
- `Where(query, args...)` - Add WHERE clause
- `Or(query, args...)` - Add OR clause
- `Not(query, args...)` - Add NOT clause
- `Scopes(fns...)` - Apply scopes
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
- `Paginate[T](ctx, chain, page, perPage)` → `(*GenericPaginator[T], error)`
- `Chunk[T](ctx, chain, batchSize, callback)` → `error`
- `Exists[T](ctx, chain)` → `(bool, error)`
- `DoesntExist[T](ctx, chain)` → `(bool, error)`

### Traditional Builder Methods

#### Query Filtering

- `Where(query, args...)` - Add WHERE clause
- `OrWhere(query, args...)` - Add OR WHERE clause
- `WhereIn(column, values)` - WHERE IN clause
- `WhereNotIn(column, values)` - WHERE NOT IN clause
- `WhereNull(column)` - WHERE column IS NULL
- `WhereNotNull(column)` - WHERE column IS NOT NULL
- `WhereBetween(column, start, end)` - WHERE BETWEEN clause

### Ordering & Limiting

- `OrderBy(order)` - Order results (e.g., "name ASC")
- `OrderByDesc(column)` - Order descending
- `Latest()` - Order by created_at DESC
- `Oldest()` - Order by created_at ASC
- `Limit(n)` - Limit results
- `Offset(n)` - Skip records
- `Take(n)` - Alias for Limit
- `Skip(n)` - Alias for Offset

### Retrieving Data

- `Find(dest)` - Get all matching records
- `First(dest)` - Get first record
- `FirstOrFail(dest)` - Get first or return error
- `Last(dest)` - Get last record
- `Get(dest)` - Alias for Find
- `Count()` - Count matching records
- `Exists()` - Check if records exist
- `DoesntExist()` - Check if no records exist
- `Pluck(column, dest)` - Get single column values

### Modifying Data

- `Create(value)` - Create new record
- `Update(column, value)` - Update single column
- `Updates(values)` - Update multiple columns
- `Delete()` - Soft delete records
- `ForceDelete()` - Permanently delete
- `Restore()` - Restore soft deleted records

### Advanced Features

- `Paginate(page, perPage, dest)` - Paginate results
- `Chunk(size, callback)` - Process records in batches
- `Scopes(scopes...)` - Apply query scopes
- `With(relations...)` - Eager load relationships
- `WithTrashed()` - Include soft deleted records
- `OnlyTrashed()` - Get only soft deleted records
- `Lock()` - Add FOR UPDATE lock
- `SharedLock()` - Add FOR SHARE lock

## Built-in Scopes

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

// Use them in queries
orm.NewBuilder(db, &User{}).Scopes(ActiveUsers).Find(&users)
orm.NewBuilder(db, &Post{}).Scopes(PublishedPosts).Find(&posts)
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

## Examples

### Complete Examples

- **Generics API Example**: `example/orm_generics/main.go`
  - Type-safe queries with context
  - Pagination and batch processing
  - Helper functions (Exists, Chunk)
  - Demonstrates all generics API features

- **Traditional API Example**: `example/orm/main.go`
  - Model definitions with relationships
  - CRUD operations
  - Query scopes (built-in and custom)
  - Pagination and soft deletes
  - Search functionality

Run examples:

```bash
# Generics API (recommended for new projects)
cd example/orm_generics
go run main.go

# Traditional API (backward compatible)
cd example/orm
go run main.go
```

## Testing

The package includes comprehensive tests:

- `model_test.go` - 15 tests for Model functionality (UUID support, hooks, methods)
- `builder_test.go` - 24 tests for Builder methods (CRUD, queries, pagination)
- `scopes_test.go` - 14 tests for query scopes (generic scopes, search, combinations)
- `generic_builder_test.go` - 16 tests for Generics API (type-safe queries, context support)

**Total: 69 tests covering all ORM functionality**

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

// Use ORM builder
builder := orm.NewBuilder(db.DB(), &User{})
builder.Where("active = ?", true).Find(&users)
```

## Notes

- **UUID Primary Keys**: All models use UUIDv7 for primary keys, automatically generated via `BeforeCreate` hook
- **SQLite Compatibility**: UUIDs stored as `char(36)` for SQLite support (can be changed to native `uuid` type for PostgreSQL)
- All queries respect soft deletes by default (use `WithTrashed()` to include deleted records)
- Timestamps (created_at, updated_at) are managed automatically by GORM
- The Builder returns `*Builder` for all query methods, enabling fluent chaining
- Error handling follows Go conventions - always check returned errors
- Access model fields directly (`.ID`, `.CreatedAt`) instead of using getters for idiomatic Go code

## Design Philosophy

The ORM package follows Laravel's Eloquent design philosophy with Go idioms:

1. **Expressive Syntax** - Readable, self-documenting queries
2. **Fluent Interface** - Method chaining for complex queries
3. **Scopes** - Reusable query logic with generic factory functions
4. **Convention Over Configuration** - Sensible defaults (UUIDs, timestamps, soft deletes)
5. **Soft Deletes** - Safe record deletion with recovery option
6. **Direct Field Access** - Go-idiomatic direct field access instead of getters
7. **Flexible Scopes** - Generic scope factories instead of opinionated assumptions about schema

## Future Enhancements

Potential additions for future versions:

- Relationship helpers (HasOne, HasMany, BelongsTo, ManyToMany)
- Model events/hooks (BeforeCreate, AfterUpdate, etc.)
- Migration system
- Query caching
- Eager loading optimization
- Global scopes
- Model observers
