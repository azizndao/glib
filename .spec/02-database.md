# Phase 2: Database Layer

**Timeline**: Weeks 3-6  
**Priority**: Critical - Core functionality for most applications  
**Dependencies**: Phase 1 (Foundation)

## Overview

Build a comprehensive database layer that wraps GORM with a Laravel Eloquent-inspired API, providing:
- Multiple database connection management
- Active Record pattern for models
- Fluent query builder
- Relationships (HasOne, HasMany, BelongsTo, ManyToMany)
- Migration system for schema versioning
- Query scopes and model events

## Why GORM?

**Advantages:**
- ✅ Battle-tested and mature (v2 is stable)
- ✅ Feature-rich (relationships, hooks, transactions, migrations)
- ✅ Good performance with proper usage
- ✅ Active community and maintenance
- ✅ Supports MySQL, PostgreSQL, SQLite, SQL Server
- ✅ Plugin system for extensions

**Our Value Add:**
- Better API (Laravel-style fluent interface)
- Integrated with our container and config
- Custom logger using our slog
- Enhanced relationship syntax
- Simplified transaction handling
- Better error messages

## Components

### 2.1 Database Manager & Connections

**Location**: `database/`

#### Package Structure

```
database/
├── manager.go          # Database manager (multiple connections)
├── connection.go       # Single database connection wrapper  
├── factory.go         # Connection factory
├── gorm_logger.go     # Custom GORM logger using slog
├── transaction.go     # Transaction helpers
├── config.go          # Database configuration types
└── database_test.go   # Tests

database/drivers/
├── mysql.go           # MySQL driver initialization
├── postgres.go        # PostgreSQL driver initialization
└── sqlite.go          # SQLite driver initialization
```

#### Core Types

```go
// Manager manages multiple database connections
type Manager struct {
    connections map[string]*Connection
    config      *config.Repository
    default     string
    mu          sync.RWMutex
}

// Connection wraps a GORM database connection
type Connection struct {
    db     *gorm.DB
    name   string
    driver string
    config ConnectionConfig
}

// ConnectionConfig defines connection settings
type ConnectionConfig struct {
    Driver      string
    Host        string
    Port        int
    Database    string
    Username    string
    Password    string
    Charset     string
    Collation   string
    Prefix      string
    Timezone    string
    SSLMode     string // PostgreSQL
    Pool        PoolConfig
}

// PoolConfig defines connection pool settings
type PoolConfig struct {
    MaxOpen     int
    MaxIdle     int
    MaxLifetime time.Duration
}
```

#### API Design

```go
// Initialize database manager
manager := database.NewManager(config)

// Get default connection
db := manager.DB()

// Get named connection
analyticsDB := manager.Connection("analytics")
logsDB := manager.Connection("logs")

// Add connection at runtime
manager.AddConnection("reporting", config)

// Close all connections
defer manager.Close()

// Get underlying GORM instance for advanced usage
gormDB := manager.DB().GORM()

// Execute raw query
var users []User
manager.DB().Raw("SELECT * FROM users WHERE active = ?", true).Scan(&users)
```

#### Custom GORM Logger

Integrate GORM logging with glib's slog:

```go
// GormLogger adapts slog.Logger to GORM's logger interface
type GormLogger struct {
    logger *slog.Logger
    config LoggerConfig
}

type LoggerConfig struct {
    SlowThreshold time.Duration
    LogLevel      logger.LogLevel
    Colorful      bool
}

func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
    l.config.LogLevel = level
    return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
    l.logger.Info(fmt.Sprintf(msg, data...))
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
    l.logger.Warn(fmt.Sprintf(msg, data...))
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
    l.logger.Error(fmt.Sprintf(msg, data...))
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
    elapsed := time.Since(begin)
    sql, rows := fc()
    
    if err != nil {
        l.logger.Error("SQL Error",
            "error", err,
            "elapsed", elapsed,
            "rows", rows,
            "sql", sql,
        )
    } else if elapsed > l.config.SlowThreshold {
        l.logger.Warn("Slow SQL",
            "elapsed", elapsed,
            "rows", rows,
            "sql", sql,
        )
    } else {
        l.logger.Debug("SQL",
            "elapsed", elapsed,
            "rows", rows,
            "sql", sql,
        )
    }
}
```

#### Transaction Helpers

```go
// Transaction with automatic commit/rollback
err := manager.DB().Transaction(func(tx *Connection) error {
    // All operations in transaction
    user := &User{Name: "John"}
    if err := tx.Create(user); err != nil {
        return err // Automatic rollback
    }
    
    profile := &Profile{UserID: user.ID}
    if err := tx.Create(profile); err != nil {
        return err // Automatic rollback
    }
    
    return nil // Automatic commit
})

// Manual transaction control
tx := manager.DB().Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

tx.Create(&user)
tx.Create(&profile)

if err := tx.Commit(); err != nil {
    tx.Rollback()
    return err
}

// Savepoint support
tx.SavePoint("sp1")
// ... operations
tx.RollbackTo("sp1")
```

---

### 2.2 Model Base & Active Record Pattern

**Location**: `orm/`

#### Package Structure

```
orm/
├── model.go            # Base model struct
├── builder.go          # Query builder wrapper
├── collection.go       # Model collection type
├── scope.go           # Query scopes
├── events.go          # Model events
├── hooks.go           # GORM hooks integration
├── soft_delete.go     # Soft delete support
├── timestamps.go      # Timestamp management
└── orm_test.go        # Tests
```

#### Base Model

```go
// Model provides common fields and methods for all models
type Model struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// ModelInterface defines methods all models must implement
type ModelInterface interface {
    TableName() string
    PrimaryKey() string
    IsNew() bool
}

// Implement ModelInterface
func (m *Model) IsNew() bool {
    return m.ID == 0
}

func (m *Model) PrimaryKey() string {
    return "id"
}
```

#### User Model Example

```go
// User model with relationships
type User struct {
    orm.Model
    Name     string    `gorm:"type:varchar(100);not null" json:"name"`
    Email    string    `gorm:"uniqueIndex;not null" json:"email"`
    Password string    `gorm:"not null" json:"-"` // Hidden from JSON
    Age      int       `json:"age"`
    Active   bool      `gorm:"default:true" json:"active"`
    
    // Relationships
    Posts    []Post    `gorm:"foreignKey:UserID" json:"posts,omitempty"`
    Profile  *Profile  `gorm:"foreignKey:UserID" json:"profile,omitempty"`
    Roles    []Role    `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// TableName overrides the default table name
func (User) TableName() string {
    return "users"
}

// BeforeCreate hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
    // Hash password before saving
    if u.Password != "" {
        hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
        if err != nil {
            return err
        }
        u.Password = string(hashed)
    }
    return nil
}

// AfterFind hook
func (u *User) AfterFind(tx *gorm.DB) error {
    // Load something after finding
    return nil
}
```

#### Query Builder API

Laravel-inspired fluent interface:

```go
// Static-style queries using model type
users := []User{}
err := orm.Query(&User{}).
    Where("active = ?", true).
    Where("age >= ?", 18).
    OrderBy("name ASC").
    Limit(10).
    Find(&users)

// First result
user := &User{}
err := orm.Query(&User{}).
    Where("email = ?", "john@example.com").
    First(user)

// First or fail (returns error if not found)
user := &User{}
err := orm.Query(&User{}).
    Where("id = ?", 123).
    FirstOrFail(user)

// Count
count := orm.Query(&User{}).
    Where("active = ?", true).
    Count()

// Exists
exists := orm.Query(&User{}).
    Where("email = ?", "john@example.com").
    Exists()

// Pluck (get single column values)
emails := orm.Query(&User{}).
    Where("active = ?", true).
    Pluck("email")

// Select specific columns
users := []User{}
orm.Query(&User{}).
    Select("id", "name", "email").
    Find(&users)

// Get single value
name := orm.Query(&User{}).
    Where("id = ?", 1).
    Value("name")
```

#### Create, Update, Delete

```go
// Create
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
    Age:   30,
}
err := orm.Create(user)
// user.ID is now set

// Batch create
users := []User{
    {Name: "John", Email: "john@example.com"},
    {Name: "Jane", Email: "jane@example.com"},
}
err := orm.Create(&users)

// Update single field
err := orm.Query(&User{}).
    Where("id = ?", 1).
    Update("name", "Jane Doe")

// Update multiple fields
err := orm.Query(&User{}).
    Where("id = ?", 1).
    Updates(map[string]interface{}{
        "name": "Jane Doe",
        "age":  31,
    })

// Update from struct
user.Name = "Jane Doe"
user.Age = 31
err := orm.Save(user)

// Delete
err := orm.Delete(user)

// Delete by ID
err := orm.Query(&User{}).Where("id = ?", 1).Delete()

// Soft delete (sets deleted_at)
err := orm.Query(&User{}).Where("id = ?", 1).Delete()

// Permanent delete (force delete)
err := orm.Query(&User{}).Where("id = ?", 1).ForceDelete()

// Restore soft deleted
err := orm.Query(&User{}).Where("id = ?", 1).Restore()

// Query with soft deleted records
users := []User{}
orm.Query(&User{}).WithTrashed().Find(&users)

// Query only soft deleted records
users := []User{}
orm.Query(&User{}).OnlyTrashed().Find(&users)
```

#### Query Scopes

Reusable query logic:

```go
// Define scope
func ActiveUsers(db *gorm.DB) *gorm.DB {
    return db.Where("active = ?", true)
}

func AdultUsers(db *gorm.DB) *gorm.DB {
    return db.Where("age >= ?", 18)
}

func RecentUsers(db *gorm.DB) *gorm.DB {
    return db.Where("created_at > ?", time.Now().AddDate(0, 0, -30))
}

// Use scopes
users := []User{}
orm.Query(&User{}).
    Scopes(ActiveUsers, AdultUsers, RecentUsers).
    Find(&users)

// Model method scopes
type User struct {
    orm.Model
    // ...
}

func (u User) ScopeActive(db *gorm.DB) *gorm.DB {
    return db.Where("active = ?", true)
}

func (u User) ScopeAdult(db *gorm.DB) *gorm.DB {
    return db.Where("age >= ?", 18)
}

// Use model scopes
users := []User{}
orm.Query(&User{}).
    Scopes(User{}.ScopeActive, User{}.ScopeAdult).
    Find(&users)
```

#### Model Events

```go
// Model events interface
type ModelEvents interface {
    BeforeCreate(*gorm.DB) error
    AfterCreate(*gorm.DB) error
    BeforeUpdate(*gorm.DB) error
    AfterUpdate(*gorm.DB) error
    BeforeSave(*gorm.DB) error
    AfterSave(*gorm.DB) error
    BeforeDelete(*gorm.DB) error
    AfterDelete(*gorm.DB) error
    AfterFind(*gorm.DB) error
}

// Example implementation
type User struct {
    orm.Model
    Name     string
    Email    string
    Password string
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
    // Validate email uniqueness
    var count int64
    tx.Model(&User{}).Where("email = ?", u.Email).Count(&count)
    if count > 0 {
        return errors.New("email already exists")
    }
    
    // Hash password
    hashed, _ := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
    u.Password = string(hashed)
    
    return nil
}

func (u *User) AfterCreate(tx *gorm.DB) error {
    // Send welcome email
    go sendWelcomeEmail(u.Email)
    return nil
}

func (u *User) BeforeDelete(tx *gorm.DB) error {
    // Prevent deletion of admin users
    if u.IsAdmin() {
        return errors.New("cannot delete admin user")
    }
    return nil
}
```

---

### 2.3 Relationships

**Location**: `orm/relations/`

#### Package Structure

```
orm/relations/
├── relation.go         # Base relation interface
├── has_one.go         # One-to-one relation
├── has_many.go        # One-to-many relation
├── belongs_to.go      # Inverse of has_one/has_many
├── many_to_many.go    # Many-to-many with pivot
├── polymorphic.go     # Polymorphic relations
└── relations_test.go  # Tests
```

#### Relationship Types

**Has One (1:1)**

```go
type User struct {
    orm.Model
    Name    string
    Profile *Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    orm.Model
    UserID uint
    Bio    string
    User   *User `gorm:"foreignKey:UserID"`
}

// Query with relationship
user := &User{}
orm.Query(&User{}).
    Preload("Profile").
    First(user)

// Access relationship
fmt.Println(user.Profile.Bio)
```

**Has Many (1:N)**

```go
type User struct {
    orm.Model
    Name  string
    Posts []Post `gorm:"foreignKey:UserID"`
}

type Post struct {
    orm.Model
    UserID uint
    Title  string
    Body   string
    User   *User `gorm:"foreignKey:UserID"`
}

// Eager load posts
user := &User{}
orm.Query(&User{}).
    Preload("Posts").
    First(user)

// Lazy load posts
user := &User{ID: 1}
orm.Query(&Post{}).
    Where("user_id = ?", user.ID).
    Find(&user.Posts)

// Query relationship
posts := []Post{}
orm.Query(&Post{}).
    Where("user_id = ?", user.ID).
    Where("published = ?", true).
    OrderBy("created_at DESC").
    Find(&posts)
```

**Belongs To (Inverse)**

```go
type Post struct {
    orm.Model
    UserID uint
    Title  string
    User   *User `gorm:"foreignKey:UserID"`
}

// Eager load user
post := &Post{}
orm.Query(&Post{}).
    Preload("User").
    First(post)

fmt.Println(post.User.Name)
```

**Many to Many (N:M)**

```go
type User struct {
    orm.Model
    Name  string
    Roles []Role `gorm:"many2many:user_roles;"`
}

type Role struct {
    orm.Model
    Name  string
    Users []User `gorm:"many2many:user_roles;"`
}

// Pivot table: user_roles
// Columns: user_id, role_id

// Eager load roles
user := &User{}
orm.Query(&User{}).
    Preload("Roles").
    First(user)

// Attach role to user
admin := &Role{Name: "admin"}
orm.Create(admin)

user.Roles = append(user.Roles, *admin)
orm.Save(user)

// Or using association
orm.Model(user).Association("Roles").Append(admin)

// Detach role
orm.Model(user).Association("Roles").Delete(admin)

// Replace all roles
orm.Model(user).Association("Roles").Replace([]Role{role1, role2})

// Clear all roles
orm.Model(user).Association("Roles").Clear()

// Count roles
count := orm.Model(user).Association("Roles").Count()
```

**Polymorphic Relations**

```go
// Comment can belong to Post or Video
type Comment struct {
    orm.Model
    CommentableID   uint
    CommentableType string
    Body            string
}

type Post struct {
    orm.Model
    Title    string
    Comments []Comment `gorm:"polymorphic:Commentable;"`
}

type Video struct {
    orm.Model
    Title    string
    Comments []Comment `gorm:"polymorphic:Commentable;"`
}

// Eager load comments
post := &Post{}
orm.Query(&Post{}).
    Preload("Comments").
    First(post)

// Create comment
comment := &Comment{
    CommentableID:   post.ID,
    CommentableType: "posts",
    Body:            "Great post!",
}
orm.Create(comment)
```

#### Eager Loading (Prevent N+1)

```go
// N+1 Problem (BAD)
users := []User{}
orm.Query(&User{}).Find(&users)

for _, user := range users {
    // This executes a query for EACH user!
    orm.Query(&Post{}).Where("user_id = ?", user.ID).Find(&user.Posts)
}
// Total queries: 1 + N

// Solution: Eager Load (GOOD)
users := []User{}
orm.Query(&User{}).
    Preload("Posts").          // Load all posts in single query
    Find(&users)
// Total queries: 2

// Nested eager loading
users := []User{}
orm.Query(&User{}).
    Preload("Posts.Comments").  // Load posts and their comments
    Preload("Profile").
    Find(&users)

// Conditional eager loading
users := []User{}
orm.Query(&User{}).
    Preload("Posts", "published = ?", true).  // Only published posts
    Find(&users)

// Custom preload query
users := []User{}
orm.Query(&User{}).
    Preload("Posts", func(db *gorm.DB) *gorm.DB {
        return db.Where("published = ?", true).
            OrderBy("created_at DESC").
            Limit(5)
    }).
    Find(&users)
```

---

### 2.4 Migrations System

**Location**: `database/migrations/`

#### Package Structure

```
database/migrations/
├── migrator.go         # Migration runner
├── migration.go        # Migration interface
├── repository.go      # Track migration state in DB
├── schema.go          # Schema builder helpers
├── blueprint.go       # Table blueprint for DDL
└── migrations_test.go # Tests

# User's migrations
database/migrations/
├── 2024_12_23_000001_create_users_table.go
├── 2024_12_23_000002_create_posts_table.go
└── 2024_12_23_000003_add_published_to_posts.go
```

#### Migration Interface

```go
// Migration defines the interface for database migrations
type Migration interface {
    Up(*gorm.DB) error      // Apply migration
    Down(*gorm.DB) error    // Rollback migration
    Version() string        // Unique version identifier
}

// MigrationInfo stores metadata about migrations
type MigrationInfo struct {
    Version   string
    Batch     int
    RanAt     time.Time
}
```

#### Creating Migrations

**Using GORM AutoMigrate:**

```go
type CreateUsersTable struct{}

func (m *CreateUsersTable) Up(db *gorm.DB) error {
    return db.AutoMigrate(&User{})
}

func (m *CreateUsersTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable(&User{})
}

func (m *CreateUsersTable) Version() string {
    return "2024_12_23_000001_create_users_table"
}
```

**Using Schema Builder (More Control):**

```go
type CreatePostsTable struct{}

func (m *CreatePostsTable) Up(db *gorm.DB) error {
    return db.Exec(`
        CREATE TABLE posts (
            id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
            user_id BIGINT UNSIGNED NOT NULL,
            title VARCHAR(255) NOT NULL,
            body TEXT NOT NULL,
            published BOOLEAN DEFAULT false,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP NULL,
            INDEX idx_user_id (user_id),
            INDEX idx_published (published),
            INDEX idx_deleted_at (deleted_at),
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        )
    `).Error
}

func (m *CreatePostsTable) Down(db *gorm.DB) error {
    return db.Exec("DROP TABLE IF EXISTS posts").Error
}

func (m *CreatePostsTable) Version() string {
    return "2024_12_23_000002_create_posts_table"
}
```

**Modifying Tables:**

```go
type AddPublishedToPosts struct{}

func (m *AddPublishedToPosts) Up(db *gorm.DB) error {
    return db.Exec(`
        ALTER TABLE posts 
        ADD COLUMN published BOOLEAN DEFAULT false,
        ADD INDEX idx_published (published)
    `).Error
}

func (m *AddPublishedToPosts) Down(db *gorm.DB) error {
    return db.Exec(`
        ALTER TABLE posts 
        DROP INDEX idx_published,
        DROP COLUMN published
    `).Error
}

func (m *AddPublishedToPosts) Version() string {
    return "2024_12_23_000003_add_published_to_posts"
}
```

#### Migration Repository

Tracks which migrations have been run:

```go
// Repository manages migration state
type Repository struct {
    db *gorm.DB
}

// migrations table schema
type Migration struct {
    ID        uint      `gorm:"primaryKey"`
    Version   string    `gorm:"uniqueIndex;size:255"`
    Batch     int       `gorm:"index"`
    RanAt     time.Time
}

func (r *Repository) CreateMigrationsTable() error {
    return r.db.AutoMigrate(&Migration{})
}

func (r *Repository) GetRan() ([]string, error) {
    var migrations []Migration
    err := r.db.Order("version").Find(&migrations).Error
    if err != nil {
        return nil, err
    }
    
    versions := make([]string, len(migrations))
    for i, m := range migrations {
        versions[i] = m.Version
    }
    return versions, nil
}

func (r *Repository) Log(version string, batch int) error {
    return r.db.Create(&Migration{
        Version: version,
        Batch:   batch,
        RanAt:   time.Now(),
    }).Error
}

func (r *Repository) Delete(version string) error {
    return r.db.Where("version = ?", version).Delete(&Migration{}).Error
}

func (r *Repository) GetLastBatchNumber() (int, error) {
    var migration Migration
    err := r.db.Order("batch DESC").First(&migration).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return 0, nil
        }
        return 0, err
    }
    return migration.Batch, nil
}

func (r *Repository) GetMigrations(batch int) ([]string, error) {
    var migrations []Migration
    err := r.db.Where("batch = ?", batch).
        Order("version DESC").
        Find(&migrations).Error
    if err != nil {
        return nil, err
    }
    
    versions := make([]string, len(migrations))
    for i, m := range migrations {
        versions[i] = m.Version
    }
    return versions, nil
}
```

#### Migrator

Runs migrations:

```go
// Migrator executes migrations
type Migrator struct {
    db         *gorm.DB
    repository *Repository
    migrations []Migration
}

func NewMigrator(db *gorm.DB) *Migrator {
    return &Migrator{
        db:         db,
        repository: NewRepository(db),
        migrations: []Migration{},
    }
}

func (m *Migrator) Register(migrations ...Migration) {
    m.migrations = append(m.migrations, migrations...)
}

// Run pending migrations
func (m *Migrator) Run() error {
    // Ensure migrations table exists
    if err := m.repository.CreateMigrationsTable(); err != nil {
        return err
    }
    
    // Get already run migrations
    ran, err := m.repository.GetRan()
    if err != nil {
        return err
    }
    
    // Get next batch number
    batch, err := m.repository.GetLastBatchNumber()
    if err != nil {
        return err
    }
    batch++
    
    // Run pending migrations
    for _, migration := range m.migrations {
        version := migration.Version()
        
        // Skip if already ran
        if contains(ran, version) {
            continue
        }
        
        // Run migration
        fmt.Printf("Migrating: %s\n", version)
        
        if err := migration.Up(m.db); err != nil {
            return fmt.Errorf("migration %s failed: %w", version, err)
        }
        
        // Log migration
        if err := m.repository.Log(version, batch); err != nil {
            return err
        }
        
        fmt.Printf("Migrated: %s\n", version)
    }
    
    return nil
}

// Rollback last batch
func (m *Migrator) Rollback() error {
    // Get last batch
    batch, err := m.repository.GetLastBatchNumber()
    if err != nil {
        return err
    }
    
    if batch == 0 {
        fmt.Println("Nothing to rollback")
        return nil
    }
    
    // Get migrations in last batch
    versions, err := m.repository.GetMigrations(batch)
    if err != nil {
        return err
    }
    
    // Rollback each migration
    for _, version := range versions {
        migration := m.findMigration(version)
        if migration == nil {
            return fmt.Errorf("migration not found: %s", version)
        }
        
        fmt.Printf("Rolling back: %s\n", version)
        
        if err := migration.Down(m.db); err != nil {
            return fmt.Errorf("rollback %s failed: %w", version, err)
        }
        
        // Remove from log
        if err := m.repository.Delete(version); err != nil {
            return err
        }
        
        fmt.Printf("Rolled back: %s\n", version)
    }
    
    return nil
}

// Reset (rollback all)
func (m *Migrator) Reset() error {
    for {
        batch, _ := m.repository.GetLastBatchNumber()
        if batch == 0 {
            break
        }
        if err := m.Rollback(); err != nil {
            return err
        }
    }
    return nil
}

// Refresh (reset and re-run)
func (m *Migrator) Refresh() error {
    if err := m.Reset(); err != nil {
        return err
    }
    return m.Run()
}

// Fresh (drop all tables and re-run)
func (m *Migrator) Fresh() error {
    // Drop all tables
    tables := []string{}
    m.db.Raw("SHOW TABLES").Scan(&tables)
    
    for _, table := range tables {
        if err := m.db.Migrator().DropTable(table); err != nil {
            return err
        }
    }
    
    return m.Run()
}

// Status shows migration status
func (m *Migrator) Status() error {
    ran, err := m.repository.GetRan()
    if err != nil {
        return err
    }
    
    fmt.Println("Migration Status:")
    fmt.Println("----------------------------------------")
    
    for _, migration := range m.migrations {
        version := migration.Version()
        status := "Pending"
        if contains(ran, version) {
            status = "Ran"
        }
        fmt.Printf("%s ... %s\n", version, status)
    }
    
    return nil
}
```

#### CLI Commands

```bash
# Run pending migrations
glib migrate

# Rollback last batch
glib migrate:rollback

# Rollback last N batches
glib migrate:rollback --step=3

# Reset all migrations
glib migrate:reset

# Reset and re-run
glib migrate:refresh

# Drop all tables and re-run
glib migrate:fresh

# Show migration status
glib migrate:status

# Generate migration file
glib make:migration create_users_table
glib make:migration add_published_to_posts
```

---

## Integration with Foundation

### Database Service Provider

```go
type DatabaseServiceProvider struct{}

func (p *DatabaseServiceProvider) Register(app *foundation.Application) {
    app.Container().Singleton((*database.Manager)(nil), 
        func(c *container.Container) (interface{}, error) {
            cfg := app.Config()
            return database.NewManager(cfg), nil
        })
}

func (p *DatabaseServiceProvider) Boot(app *foundation.Application) error {
    // Run migrations if configured
    if app.Config().GetBool("database.auto_migrate", false) {
        mgr := app.Container().MustResolve((*database.Manager)(nil)).(*database.Manager)
        migrator := migrations.NewMigrator(mgr.DB().GORM())
        
        // Register all migrations
        migrator.Register(migrations.All()...)
        
        // Run migrations
        return migrator.Run()
    }
    return nil
}
```

### Using in Controllers

```go
type UserController struct {
    db *database.Manager
}

func NewUserController(db *database.Manager) *UserController {
    return &UserController{db: db}
}

func (ctrl *UserController) Index(c *glib.Ctx) error {
    users := []User{}
    
    err := orm.Query(&User{}).
        Where("active = ?", true).
        Preload("Posts").
        Find(&users)
    
    if err != nil {
        return err
    }
    
    return c.JSON(users)
}

func (ctrl *UserController) Show(c *glib.Ctx) error {
    id := c.PathValue("id")
    
    user := &User{}
    err := orm.Query(&User{}).
        Where("id = ?", id).
        Preload("Posts").
        Preload("Profile").
        FirstOrFail(user)
    
    if err != nil {
        return errors.NotFound("User not found", err)
    }
    
    return c.JSON(user)
}

func (ctrl *UserController) Store(c *glib.Ctx) error {
    user := &User{}
    if err := c.ValidateBody(user); err != nil {
        return err
    }
    
    if err := orm.Create(user); err != nil {
        return errors.InternalServerError("Failed to create user", err)
    }
    
    return c.Status(201).JSON(user)
}
```

---

## Testing Strategy

### Model Tests

```go
func TestUserCreate(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    user := &User{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    err := orm.Create(user)
    assert.NoError(t, err)
    assert.NotZero(t, user.ID)
}

func TestUserWithPosts(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    user := &User{Name: "John", Email: "john@example.com"}
    orm.Create(user)
    
    post := &Post{
        UserID: user.ID,
        Title:  "Test Post",
        Body:   "Content",
    }
    orm.Create(post)
    
    // Eager load
    result := &User{}
    orm.Query(&User{}).
        Preload("Posts").
        First(result)
    
    assert.Len(t, result.Posts, 1)
    assert.Equal(t, "Test Post", result.Posts[0].Title)
}
```

### Migration Tests

```go
func TestMigrationUpDown(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    migrator := migrations.NewMigrator(db)
    migration := &CreateUsersTable{}
    
    // Up
    err := migration.Up(db)
    assert.NoError(t, err)
    assert.True(t, db.Migrator().HasTable(&User{}))
    
    // Down
    err = migration.Down(db)
    assert.NoError(t, err)
    assert.False(t, db.Migrator().HasTable(&User{}))
}
```

---

## Performance Considerations

### Query Optimization

```go
// Use indexes effectively
type User struct {
    orm.Model
    Email string `gorm:"uniqueIndex"`     // Fast lookups
    Name  string `gorm:"index"`          // Fast searches
}

// Select specific columns
orm.Query(&User{}).
    Select("id", "name", "email").  // Don't load everything
    Find(&users)

// Batch loading
orm.Query(&User{}).
    Where("id IN ?", []int{1, 2, 3, 4, 5}).
    Find(&users)

// Pagination
orm.Query(&User{}).
    Limit(20).
    Offset(page * 20).
    Find(&users)
```

### Connection Pooling

```go
// Configure pool properly
config := PoolConfig{
    MaxOpen:     100,            // Maximum open connections
    MaxIdle:     10,             // Keep 10 idle connections
    MaxLifetime: 1 * time.Hour,  // Recycle connections
}
```

### Caching Queries

```go
// Cache frequent queries
func GetActiveUsers() []User {
    return cache.Remember("users:active", 10*time.Minute, func() interface{} {
        users := []User{}
        orm.Query(&User{}).Where("active = ?", true).Find(&users)
        return users
    }).([]User)
}
```

---

## Success Metrics

### Phase 2 Complete When:

- ✅ Database connections work with all supported drivers
- ✅ Models can perform CRUD operations
- ✅ Relationships load properly (eager and lazy)
- ✅ Migrations can be run and rolled back
- ✅ Query builder provides fluent API
- ✅ Scopes work for reusable queries
- ✅ Model events/hooks function correctly
- ✅ Soft deletes work properly
- ✅ All tests pass with >90% coverage
- ✅ Performance benchmarks are acceptable
- ✅ Documentation is complete with examples

---

## Next Steps

Phase 3 will build the CLI tool to:
- Generate model files: `glib make:model User`
- Generate migrations: `glib make:migration create_users_table`
- Run migrations: `glib migrate`
- Scaffold complete resources: `glib make:resource User`
