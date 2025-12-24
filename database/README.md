# Glib Database Module

[![Go Reference](https://pkg.go.dev/badge/github.com/azizndao/glib/database.svg)](https://pkg.go.dev/github.com/azizndao/glib/database)

Database management and ORM for Glib, built on [GORM v2](https://gorm.io/). Provides multi-connection management, Laravel-inspired Active Record patterns, and type-safe generic queries.

## Features

- **Database Manager** - Multi-connection support (MySQL, PostgreSQL, SQLite)
- **Connection Pooling** - Configurable connection pool settings
- **ORM Layer** - Type-safe Active Record pattern with generics
- **Base Model** - UUID primary keys, timestamps, soft deletes
- **Relationships** - HasOne, HasMany, BelongsTo, ManyToMany
- **Query Scopes** - Reusable query logic
- **Pagination** - Built-in pagination helpers
- **ServiceProvider** - Seamless foundation integration

## Installation

```bash
go get github.com/azizndao/glib/database@latest
```

### Database Drivers

Install the appropriate driver for your database:

```bash
# MySQL
go get gorm.io/driver/mysql

# PostgreSQL
go get gorm.io/driver/postgres

# SQLite
go get gorm.io/driver/sqlite
```

## Quick Start

### With Foundation (Recommended)

```go
package main

import (
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/database/orm"
    "github.com/azizndao/glib/common/container"
)

type User struct {
    orm.Model
    Name  string `json:"name"`
    Email string `json:"email" gorm:"uniqueIndex"`
}

func main() {
    app := foundation.New(".")
    
    // Register database provider
    app.Register(&database.ServiceProvider{})
    
    // Bootstrap application
    app.Bootstrap()
    
    // Get database manager
    manager := container.MustResolve[*database.Manager](app.Container())
    
    // Get default connection
    conn, _ := manager.DB()
    db := conn.DB()
    
    // Use ORM
    ctx := context.Background()
    user, err := orm.G[User](db).Where("email = ?", "user@example.com").First(ctx)
}
```

### Standalone

```go
package main

import (
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/database/orm"
    "github.com/azizndao/glib/common/config"
    "github.com/azizndao/glib/common/slog"
)

func main() {
    // Create config
    cfg := config.New()
    cfg.LoadFromEnv("DATABASE")
    
    // Create logger
    logger := slog.Create()
    
    // Create database manager
    manager := database.NewManager(cfg, logger)
    
    // Get default connection
    conn, _ := manager.DB()
    db := conn.DB()
    
    // Use GORM
    var users []User
    db.Find(&users)
}
```

## Configuration

### Environment Variables

```bash
# Default connection
DATABASE_DEFAULT=mysql

# MySQL Connection
DATABASE_MYSQL_DRIVER=mysql
DATABASE_MYSQL_HOST=localhost
DATABASE_MYSQL_PORT=3306
DATABASE_MYSQL_DATABASE=myapp
DATABASE_MYSQL_USERNAME=root
DATABASE_MYSQL_PASSWORD=secret
DATABASE_MYSQL_CHARSET=utf8mb4
DATABASE_MYSQL_COLLATION=utf8mb4_unicode_ci

# PostgreSQL Connection
DATABASE_POSTGRES_DRIVER=postgres
DATABASE_POSTGRES_HOST=localhost
DATABASE_POSTGRES_PORT=5432
DATABASE_POSTGRES_DATABASE=myapp
DATABASE_POSTGRES_USERNAME=postgres
DATABASE_POSTGRES_PASSWORD=secret
DATABASE_POSTGRES_SSLMODE=disable
DATABASE_POSTGRES_TIMEZONE=UTC

# SQLite Connection
DATABASE_SQLITE_DRIVER=sqlite
DATABASE_SQLITE_DATABASE=./database.db

# Connection Pool (all drivers)
DATABASE_MYSQL_POOL_MAX_OPEN=100
DATABASE_MYSQL_POOL_MAX_IDLE=10
DATABASE_MYSQL_POOL_MAX_LIFETIME=3600  # seconds
DATABASE_MYSQL_POOL_MAX_IDLETIME=300   # seconds
```

### Programmatic Configuration

```go
manager := database.NewManager(cfg, logger)

// Add custom connection
manager.AddConnection("analytics", database.ConnectionConfig{
    Driver:   "postgres",
    Host:     "analytics.example.com",
    Port:     5432,
    Database: "analytics",
    Username: "readonly",
    Password: "secret",
    SSLMode:  "require",
    Pool: database.PoolConfig{
        MaxOpen:     50,
        MaxIdle:     5,
        MaxLifetime: 3600,
        MaxIdleTime: 300,
    },
})

// Use custom connection
conn, _ := manager.Connection("analytics")
```

## Database Manager

### Multi-Connection Support

```go
// Get default connection
conn, err := manager.DB()

// Get named connection
mysqlConn, err := manager.Connection("mysql")
pgConn, err := manager.Connection("postgres")

// Access GORM DB
db := conn.DB()

// Connection info
name := conn.Name()
driver := conn.Driver()
```

### Connection Management

```go
// Close specific connection
conn.Close()

// Close all connections
manager.CloseAll()

// Ping connection
if err := conn.Ping(); err != nil {
    log.Fatal("Database unreachable")
}
```

## ORM

The ORM layer provides a Laravel-inspired Active Record pattern with Go generics.

### Base Model

All models should embed `orm.Model` to get:
- UUID primary key (v7)
- `created_at` timestamp
- `updated_at` timestamp
- `deleted_at` (soft deletes)

```go
type User struct {
    orm.Model
    Name     string    `json:"name"`
    Email    string    `json:"email" gorm:"uniqueIndex"`
    Password string    `json:"-"`
    Active   bool      `json:"active" gorm:"default:true"`
}

// Model provides:
// - ID (uuid.UUID)
// - CreatedAt (time.Time)
// - UpdatedAt (time.Time)
// - DeletedAt (gorm.DeletedAt)
// - IsNew() bool
// - IsDeleted() bool
```

### CRUD Operations

The ORM uses a type-safe generics API:

```go
ctx := context.Background()

// Create
user := &User{Name: "Alice", Email: "alice@example.com"}
err := orm.G[User](db).Create(ctx, user)

// Find by ID (UUID)
user, err := orm.G[User](db).Where("id = ?", userID).First(ctx)

// Find by email
user, err := orm.G[User](db).Where("email = ?", "alice@example.com").First(ctx)

// Find all
users, err := orm.G[User](db).Order("name ASC").Find(ctx)

// Find with conditions
activeUsers, err := orm.G[User](db).
    Where("active = ?", true).
    Order("created_at DESC").
    Limit(10).
    Find(ctx)

// Update
rows, err := orm.G[User](db).
    Where("id = ?", userID).
    Update(ctx, "name", "Bob")

// Update multiple fields
rows, err := orm.G[User](db).
    Where("id = ?", userID).
    Updates(ctx, map[string]any{
        "name": "Bob",
        "email": "bob@example.com",
    })

// Delete (soft delete)
rows, err := orm.G[User](db).Where("id = ?", userID).Delete(ctx)

// Force delete
rows, err := orm.G[User](db.Unscoped()).Where("id = ?", userID).Delete(ctx)

// Count
count, err := orm.G[User](db).Where("active = ?", true).Count(ctx, "*")

// Check existence
exists, err := orm.Exists(ctx, orm.G[User](db).Where("email = ?", email))
```

### Query Scopes

Reusable query logic with scopes:

```go
// Built-in scopes
ActiveUsers := orm.WhereColumn("active", true)
OrderByName := orm.OrderByColumn("name", "ASC")

// Apply scopes (to DB before G[T])
users, err := orm.G[User](ActiveUsers(db)).
    Order("name ASC").
    Find(ctx)

// Or use with GORM's Model()
db.Model(&User{}).Scopes(ActiveUsers, OrderByName).Find(&users)

// Custom scope
func AdminUsers(db *gorm.DB) *gorm.DB {
    return db.Where("role = ?", "admin")
}

admins, err := orm.G[User](AdminUsers(db)).Find(ctx)

// Combine scopes
scopedDB := ActiveUsers(db)
scopedDB = AdminUsers(scopedDB)
users, err := orm.G[User](scopedDB).Find(ctx)
```

### Pagination

```go
// Paginate results
builder := orm.G[User](db).Where("active = ?", true).Order("name ASC")
paginator, err := orm.Paginate(ctx, builder, 1, 20) // page 1, 20 per page

// Access pagination data
fmt.Printf("Page %d of %d\n", paginator.CurrentPage, paginator.LastPage)
fmt.Printf("Total: %d records\n", paginator.Total)
fmt.Printf("Per page: %d\n", paginator.PerPage)

// Iterate results
for _, user := range paginator.Data {
    fmt.Println(user.Name)
}

// Pagination scope
scopedDB := orm.PaginateScope(2, 50)(db) // page 2, 50 per page
users, err := orm.G[User](scopedDB).Find(ctx)
```

### Batch Processing

```go
// Process records in batches
builder := orm.G[User](db).Where("active = ?", true)
err := orm.Chunk(ctx, builder, 100, func(users []User) error {
    // Process batch of 100 users
    for _, user := range users {
        // Send email, update data, etc.
        if err := processUser(user); err != nil {
            return err
        }
    }
    return nil
})
```

### Search

```go
// Search across multiple columns
searchTerm := "john"
users, err := orm.G[User](orm.Search([]string{"name", "email"}, searchTerm)(db)).
    Find(ctx)

// Search with other conditions
users, err := orm.G[User](db).
    Where("active = ?", true).
    Scopes(orm.Search([]string{"name", "email", "phone"}, searchTerm)).
    Find(ctx)
```

## Relationships

### Defining Relationships

```go
// Has One (1:1)
type User struct {
    orm.Model
    Name    string
    Profile *Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    orm.Model
    UserID uuid.UUID
    Bio    string
}

// Has Many (1:N)
type User struct {
    orm.Model
    Name  string
    Posts []Post `gorm:"foreignKey:UserID"`
}

type Post struct {
    orm.Model
    UserID uuid.UUID
    Title  string
}

// Belongs To (Inverse)
type Post struct {
    orm.Model
    UserID uuid.UUID
    Title  string
    User   *User `gorm:"foreignKey:UserID"`
}

// Many to Many (N:M)
type User struct {
    orm.Model
    Name  string
    Roles []Role `gorm:"many2many:user_roles;"`
}

type Role struct {
    orm.Model
    Name string
}

// Polymorphic
type Comment struct {
    orm.Model
    CommentableID   uuid.UUID
    CommentableType string
    Body            string
}

type Post struct {
    orm.Model
    Title    string
    Comments []Comment `gorm:"polymorphic:Commentable;"`
}
```

### Eager Loading

```go
// Load single relationship
user, err := orm.G[User](db).
    Preload("Profile").
    Where("id = ?", userID).
    First(ctx)

// Load multiple relationships
user, err := orm.G[User](db).
    Preload("Profile").
    Preload("Posts").
    Where("id = ?", userID).
    First(ctx)

// Nested preloading
users, err := orm.G[User](db).
    Preload("Posts.Comments").
    Preload("Profile").
    Find(ctx)

// Conditional preloading
user, err := orm.G[User](db).
    Preload("Posts", "published = ?", true).
    Where("id = ?", userID).
    First(ctx)

// Using scope
WithPosts := orm.WithRelations("Posts")
users, err := orm.G[User](WithPosts(db)).Find(ctx)
```

### Association Helpers

```go
// Find by foreign key
posts, err := orm.G[Post](orm.BelongsTo("user_id", userID)(db)).Find(ctx)

// Association operations (GORM native)
db.Model(&user).Association("Roles").Append(&adminRole)
db.Model(&user).Association("Roles").Delete(&userRole)
db.Model(&user).Association("Roles").Clear()
count := db.Model(&user).Association("Roles").Count()
```

## Soft Deletes

```go
// Soft delete (sets deleted_at)
rows, err := orm.G[User](db).Where("id = ?", userID).Delete(ctx)

// Query only non-deleted (default)
users, err := orm.G[User](db).Find(ctx)

// Include soft deleted
users, err := orm.G[User](db.Unscoped()).Find(ctx)

// Query only deleted
users, err := orm.G[User](db).
    Where("deleted_at IS NOT NULL").
    Find(ctx)

// Restore soft deleted
db.Model(&User{}).Unscoped().
    Where("id = ?", userID).
    Update("deleted_at", nil)

// Force delete (permanent)
rows, err := orm.G[User](db.Unscoped()).
    Where("id = ?", userID).
    Delete(ctx)

// Check if deleted
if user.IsDeleted() {
    fmt.Println("User is soft deleted")
}
```

## Advanced Usage

### Transactions

```go
// Manual transaction
err := db.Transaction(func(tx *gorm.DB) error {
    // Create user
    user := &User{Name: "Alice"}
    if err := orm.G[User](tx).Create(ctx, user); err != nil {
        return err // Rollback
    }
    
    // Create profile
    profile := &Profile{UserID: user.ID, Bio: "Hello"}
    if err := orm.G[Profile](tx).Create(ctx, profile); err != nil {
        return err // Rollback
    }
    
    return nil // Commit
})

// Nested transactions
db.Transaction(func(tx1 *gorm.DB) error {
    // ... operations ...
    
    tx1.Transaction(func(tx2 *gorm.DB) error {
        // ... nested operations ...
        return nil
    })
    
    return nil
})
```

### Raw Queries

```go
// Raw SQL
var users []User
db.Raw("SELECT * FROM users WHERE active = ?", true).Scan(&users)

// Execute
db.Exec("UPDATE users SET active = ? WHERE role = ?", false, "guest")

// Named queries
db.Raw("SELECT * FROM users WHERE email = @email", 
    sql.Named("email", "user@example.com")).
    Scan(&users)
```

### Hooks

```go
type User struct {
    orm.Model
    Name  string
    Email string
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
    // Validate
    if u.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

func (u *User) AfterCreate(tx *gorm.DB) error {
    // Send welcome email
    return sendWelcomeEmail(u.Email)
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
    // Audit logging
    return logUpdate(u.ID, "user updated")
}

func (u *User) AfterDelete(tx *gorm.DB) error {
    // Cleanup related data
    return cleanupUserData(u.ID)
}
```

### Custom Types

```go
type JSON json.RawMessage

func (j *JSON) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("type assertion to []byte failed")
    }
    *j = append((*j)[0:0], bytes...)
    return nil
}

func (j JSON) Value() (driver.Value, error) {
    if len(j) == 0 {
        return nil, nil
    }
    return []byte(j), nil
}

type User struct {
    orm.Model
    Name     string
    Metadata JSON `gorm:"type:json"`
}
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/database/orm"
    "github.com/azizndao/glib/common/container"
    "github.com/google/uuid"
)

// Models
type User struct {
    orm.Model
    Name     string    `json:"name"`
    Email    string    `json:"email" gorm:"uniqueIndex"`
    Posts    []Post    `gorm:"foreignKey:UserID"`
    Profile  *Profile  `gorm:"foreignKey:UserID"`
}

type Post struct {
    orm.Model
    UserID    uuid.UUID `json:"user_id"`
    Title     string    `json:"title"`
    Body      string    `json:"body"`
    Published bool      `json:"published" gorm:"default:false"`
    User      *User     `gorm:"foreignKey:UserID"`
}

type Profile struct {
    orm.Model
    UserID uuid.UUID `json:"user_id"`
    Bio    string    `json:"bio"`
}

func main() {
    // Create application
    app := foundation.New(".")
    
    // Register database provider
    app.Register(&database.ServiceProvider{})
    
    // Bootstrap
    if err := app.Bootstrap(); err != nil {
        log.Fatal(err)
    }
    
    // Get database
    manager := container.MustResolve[*database.Manager](app.Container())
    conn, err := manager.DB()
    if err != nil {
        log.Fatal(err)
    }
    db := conn.DB()
    
    // Auto migrate
    db.AutoMigrate(&User{}, &Post{}, &Profile{})
    
    ctx := context.Background()
    
    // Create user with profile
    err = db.Transaction(func(tx *gorm.DB) error {
        user := &User{
            Name:  "Alice",
            Email: "alice@example.com",
        }
        if err := orm.G[User](tx).Create(ctx, user); err != nil {
            return err
        }
        
        profile := &Profile{
            UserID: user.ID,
            Bio:    "Software Developer",
        }
        return orm.G[Profile](tx).Create(ctx, profile)
    })
    
    // Create posts
    user, _ := orm.G[User](db).Where("email = ?", "alice@example.com").First(ctx)
    
    post := &Post{
        UserID:    user.ID,
        Title:     "My First Post",
        Body:      "Hello, World!",
        Published: true,
    }
    orm.G[Post](db).Create(ctx, post)
    
    // Query with relationships
    user, err = orm.G[User](db).
        Preload("Posts", "published = ?", true).
        Preload("Profile").
        Where("email = ?", "alice@example.com").
        First(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User: %s", user.Name)
    log.Printf("Bio: %s", user.Profile.Bio)
    log.Printf("Published posts: %d", len(user.Posts))
    
    // Pagination
    builder := orm.G[Post](db).
        Where("published = ?", true).
        Order("created_at DESC")
    
    paginator, err := orm.Paginate(ctx, builder, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Page %d of %d (Total: %d posts)",
        paginator.CurrentPage,
        paginator.LastPage,
        paginator.Total,
    )
    
    // Search
    searchTerm := "first"
    posts, err := orm.G[Post](
        orm.Search([]string{"title", "body"}, searchTerm)(db),
    ).Find(ctx)
    
    log.Printf("Found %d posts matching '%s'", len(posts), searchTerm)
}
```

## Testing

```go
package myapp_test

import (
    "context"
    "testing"
    
    "github.com/azizndao/glib/database"
    "github.com/azizndao/glib/database/orm"
    "github.com/azizndao/glib/common/config"
    "github.com/azizndao/glib/common/slog"
)

func setupTestDB(t *testing.T) *gorm.DB {
    cfg := config.NewWithMap(map[string]any{
        "database.default": "sqlite",
        "database.sqlite.driver": "sqlite",
        "database.sqlite.database": ":memory:",
    })
    
    manager := database.NewManager(cfg, slog.Create())
    conn, err := manager.DB()
    if err != nil {
        t.Fatal(err)
    }
    
    db := conn.DB()
    db.AutoMigrate(&User{})
    
    return db
}

func TestUserCRUD(t *testing.T) {
    db := setupTestDB(t)
    ctx := context.Background()
    
    // Create
    user := &User{Name: "Test", Email: "test@example.com"}
    err := orm.G[User](db).Create(ctx, user)
    if err != nil {
        t.Fatal(err)
    }
    
    // Read
    found, err := orm.G[User](db).
        Where("email = ?", "test@example.com").
        First(ctx)
    if err != nil {
        t.Fatal(err)
    }
    
    if found.Name != "Test" {
        t.Errorf("Expected name 'Test', got '%s'", found.Name)
    }
    
    // Update
    rows, err := orm.G[User](db).
        Where("id = ?", user.ID).
        Update(ctx, "name", "Updated")
    if err != nil || rows == 0 {
        t.Fatal("Update failed")
    }
    
    // Delete
    rows, err = orm.G[User](db).
        Where("id = ?", user.ID).
        Delete(ctx)
    if err != nil || rows == 0 {
        t.Fatal("Delete failed")
    }
}
```

## Module Structure

```
database/
├── manager.go          # Database connection manager
├── connection.go       # Connection wrapper
├── config.go           # Configuration structs
├── logger.go           # GORM logger adapter
├── provider.go         # Foundation service provider
└── orm/
    ├── model.go        # Base model with UUID/timestamps
    ├── builder.go      # Type-safe query builder
    ├── scopes.go       # Reusable query scopes
    ├── relations.go    # Relationship helpers
    └── soft_deletes.go # Soft delete functionality
```

## Design Philosophy

1. **Multi-Connection** - Support multiple databases in one app
2. **Type-Safe** - Leverage Go generics for compile-time safety
3. **Context-Aware** - All operations support cancellation
4. **Active Record** - Laravel-inspired model patterns
5. **GORM Foundation** - Built on battle-tested GORM
6. **UUID Primary Keys** - Modern, distributed-friendly IDs
7. **Soft Deletes** - Safe record deletion by default

## Related Modules

- **[foundation](../foundation)** - ServiceProvider integration
- **[common](../common)** - Configuration and logging
- **[http](../http)** - Web application integration

For detailed ORM documentation, see [orm/README.md](./orm/README.md).

## Examples

See [example/](../example/) directory:
- **[database](../example/database)** - Database manager example
- **[orm](../example/orm)** - Complete ORM example with relationships
- **[relationships](../example/relationships)** - Advanced relationship patterns

## Contributing

Contributions are welcome! Please ensure:

1. ✅ Tests pass
2. ✅ Documentation updated
3. ✅ GORM compatibility maintained
4. ✅ Type safety preserved

## License

This module is part of the Glib framework. See the main repository for license information.

## Roadmap

- [ ] Migration system integration
- [ ] Query caching
- [ ] Database seeding
- [ ] Model observers/events
- [ ] Global scopes
- [ ] Read/write connection splitting
