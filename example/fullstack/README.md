# Fullstack Blog Application

A complete blog application demonstrating Glib framework's features for building real-world applications with database, authentication, and relationships.

## What You'll Learn

- **Foundation Module**: Application lifecycle, ServiceProviders, dependency injection
- **Controllers**: Laravel-style controllers with automatic resource routing
- **Dependency Injection**: Constructor-based DI for optimal performance
- **Database Integration**: Connection management, migrations, GORM ORM
- **Model Relationships**: One-to-many (User → Posts, Post → Comments)
- **Authentication**: JWT-based auth with middleware
- **Request Validation**: Validating complex request structures
- **Pagination**: Efficient data pagination with metadata
- **Soft Deletes**: Non-destructive data removal
- **Error Handling**: Structured error responses

## Features

### Authentication System
- User registration with password hashing
- Login with JWT token generation
- Protected routes with authentication middleware
- User context in authenticated requests

### Blog Functionality
- Create, read, update, delete posts (CRUD)
- Posts have title, content, and published status
- Only post owners can update/delete their posts
- List published posts with pagination
- Comments on posts

### Data Models
```
User
├── ID (UUID)
├── Name
├── Email (unique)
├── Password (hashed)
└── Posts (one-to-many)

Post
├── ID (UUID)
├── Title
├── Content
├── Published (boolean)
├── UserID (foreign key)
├── User (belongs to)
└── Comments (one-to-many)
    
Comment
├── ID (UUID)
├── Content
├── PostID (foreign key)
├── UserID (foreign key)
├── Post (belongs to)
└── User (belongs to)
```

## Project Structure

```
fullstack/
├── controllers/
│   ├── auth_controller.go  # Authentication controller
│   └── post_controller.go  # Post resource controller (CRUD)
├── models/
│   ├── user.go         # User model with posts relationship
│   ├── post.go         # Post model with user & comments
│   └── comment.go      # Comment model with user & post
├── middleware/
│   └── auth.go         # JWT authentication middleware
├── main.go             # Application setup & routes
├── .env.example        # Environment configuration template
├── test.http           # HTTP requests for testing
├── .air.toml           # Hot reload configuration
└── README.md           # This file
```

## Getting Started

### Prerequisites
- Go 1.23 or later
- SQLite (included) or PostgreSQL/MySQL

### Installation

1. **Clone and navigate:**
   ```bash
   cd example/fullstack
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

4. **Run the application:**
   ```bash
   go run main.go
   ```

   The server starts on http://localhost:3000 by default.

### Database Setup

The application uses **SQLite** by default (file: `blog.db`). Database tables are created automatically on startup using GORM's AutoMigrate.

**To use PostgreSQL or MySQL:**

Edit the configuration in `main.go` main function:

```go
// PostgreSQL
cfg.Set("database.default", "postgres")
cfg.Set("database.connections.postgres.driver", "postgres")
cfg.Set("database.connections.postgres.host", "localhost")
cfg.Set("database.connections.postgres.port", 5432)
cfg.Set("database.connections.postgres.database", "blog_db")
cfg.Set("database.connections.postgres.username", "postgres")
cfg.Set("database.connections.postgres.password", "secret")

// MySQL
cfg.Set("database.default", "mysql")
cfg.Set("database.connections.mysql.driver", "mysql")
cfg.Set("database.connections.mysql.host", "localhost")
cfg.Set("database.connections.mysql.port", 3306)
cfg.Set("database.connections.mysql.database", "blog_db")
cfg.Set("database.connections.mysql.username", "root")
cfg.Set("database.connections.mysql.password", "secret")
```

## API Documentation

### Public Endpoints (No Auth Required)

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "password123"
}

Response: 201 Created
{
  "token": "eyJhbGc...",
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "password123"
}

Response: 200 OK
{
  "token": "eyJhbGc...",
  "user": { ... }
}
```

#### List Posts (Paginated)
```http
GET /posts

Response: 200 OK
{
  "data": [
    {
      "id": "uuid",
      "title": "My First Post",
      "content": "Post content here...",
      "published": true,
      "user_id": "uuid",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 50,
  "per_page": 10,
  "current_page": 1,
  "last_page": 5,
  "from": 1,
  "to": 10
}
```

#### Get Single Post
```http
GET /posts/{id}

Response: 200 OK
{
  "id": "uuid",
  "title": "My First Post",
  "content": "Full post content...",
  "published": true,
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com"
  },
  "comments": [
    {
      "id": "uuid",
      "content": "Great post!",
      "user": { ... }
    }
  ]
}
```

### Protected Endpoints (Auth Required)

**Include JWT token in Authorization header:**
```
Authorization: Bearer <your-jwt-token>
```

#### Create Post
```http
POST /api/posts
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "My New Post",
  "content": "This is the post content..."
}

Response: 201 Created
{
  "id": "uuid",
  "title": "My New Post",
  "content": "This is the post content...",
  "published": false,
  "user_id": "uuid",
  "user": { ... }
}
```

#### Update Post
```http
PUT /api/posts/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "Updated Title",
  "published": true
}

Response: 200 OK
```

#### Delete Post
```http
DELETE /api/posts/{id}
Authorization: Bearer <token>

Response: 204 No Content
```

#### Add Comment
```http
POST /api/posts/{id}/comments
Authorization: Bearer <token>
Content-Type: application/json

{
  "content": "Great post! Thanks for sharing."
}

Response: 201 Created
```

## Testing with HTTP Client

Use the included `test.http` file with VSCode REST Client extension or similar tools.

**Testing workflow:**

1. **Register a user** → Save the token
2. **Login** → Verify token
3. **Create a post** (use saved token)
4. **List posts** → See your post
5. **Get single post** → View details
6. **Add a comment** (use saved token)
7. **Update post** → Change title or publish
8. **Delete post** (soft delete)

## Key Concepts Explained

### 1. Controllers with Dependency Injection

This application uses **Laravel-style controllers** with constructor-based dependency injection for clean, testable code:

```go
// Controller structure
type PostController struct {
	db *database.Manager
}

// Constructor with dependency injection
func NewPostController(app *foundation.Application) *PostController {
	dbManager := container.Resolve[*database.Manager](app.Container())
	return &PostController{db: dbManager}
}

// Controller methods
func (ctrl *PostController) Index(c *glib.Ctx) error {
	db := ctrl.db.Connection()
	// Use db to query posts...
}
```

**Why this pattern?**
- Dependencies resolved **once at startup**, not per-request
- Zero per-request dependency resolution overhead (~1-5µs saved per request)
- Clean, testable controllers
- Familiar pattern for Laravel/NestJS developers

**Controller registration:**
```go
// Enable DI for controllers
server.SetApplicationDirect(app)

// APIResource automatically creates routes:
// POST   /api/posts       → Store
// PUT    /api/posts/{id}  → Update
// PATCH  /api/posts/{id}  → Update
// DELETE /api/posts/{id}  → Destroy
api.APIResource("posts", func(app *foundation.Application) glib.Controller {
	return controllers.NewPostController(app)
}, glib.ResourceOptions{
	Except: []string{"Index", "Show"}, // Public routes registered separately
})
```

**Mixed patterns supported:**
You can use both controllers and inline handlers in the same application:

```go
// Inline handler (simple routes)
router.Get("/posts", func(c *glib.Ctx) error {
	postCtrl := controllers.NewPostController(app)
	return postCtrl.Index(c)
})

// Resource controller (CRUD operations)
api.APIResource("posts", controllers.NewPostController)
```

**Controller types:**

1. **ResourceController** - Full CRUD with forms (7 routes):
   - Index, Create, Store, Show, Edit, Update, Destroy

2. **APIResourceController** - API-only CRUD (5 routes):
   - Index, Store, Show, Update, Destroy

3. **InvokableController** - Single action:
   - Invoke

See [`http/CONTROLLERS.md`](../../http/CONTROLLERS.md) for complete documentation.

### 2. Foundation Application

The Foundation module provides application lifecycle management:

```go
app := foundation.New(".")           // Create application
app.SetConfig(cfg)                   // Configure
app.Register(&database.ServiceProvider{})  // Register providers
app.Bootstrap()                      // Initialize all
```

**ServiceProviders** encapsulate service registration logic. The database provider registers the database manager in the container for dependency injection.

### 3. Database Connection

Database manager handles multiple connections:

```go
// Get manager from container
dbManager, _ := container.Resolve[*database.Manager](app.Container())

// Get default connection
conn, _ := dbManager.DB()

// Get underlying *gorm.DB
db := conn.DB()
```

### 4. ORM Relationships

**User has many Posts:**
```go
type User struct {
    ID    uuid.UUID `gorm:"type:uuid;primary_key"`
    Posts []Post    `gorm:"foreignKey:UserID"`
}
```

**Post belongs to User, has many Comments:**
```go
type Post struct {
    UserID   uuid.UUID `gorm:"type:uuid;not null"`
    User     User      `gorm:"foreignKey:UserID"`
    Comments []Comment `gorm:"foreignKey:PostID"`
}
```

**Eager loading relationships:**
```go
db.Preload("User").Preload("Comments.User").First(&post)
```

### 5. JWT Authentication

**Token creation** (simplified for demo):
```go
token := base64.URLEncoding.EncodeToString([]byte(
    fmt.Sprintf("%s:%s:%d", userID, email, time.Now().Unix())
))
```

**Middleware extracts user from token:**
```go
func Auth(c *glib.Ctx) error {
    header := c.Get("Authorization")
    // Verify and extract user ID
    c.SetLocal("user_id", userID)
    return c.Next()
}
```

⚠️ **Security Note:** This example uses simplified JWT for educational purposes. Production applications should use established libraries like `github.com/golang-jwt/jwt/v5`.

### 6. Request Validation

Validation uses struct tags:
```go
type CreatePostRequest struct {
    Title   string `json:"title" validate:"required,min=5,max=200"`
    Content string `json:"content" validate:"required,min=10"`
}

// Validate in handler
if err := c.ValidateBody(&req); err != nil {
    return err  // Returns 400 with validation errors
}
```

### 7. Pagination

Using the ORM pagination helper:
```go
chain := orm.G[models.Post](db).
    Where("published = ?", true).
    Order("created_at DESC")

paginator, err := orm.Paginate(ctx, chain, page, perPage)
```

Returns data with metadata: total, current_page, last_page, from, to.

### 8. Soft Deletes

Posts use GORM's soft delete:
```go
type Post struct {
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Delete (sets deleted_at)
db.Delete(&post)

// Query (automatically excludes deleted)
db.Find(&posts)

// Include deleted in query
db.Unscoped().Find(&posts)
```

## Generating Controllers

Use the CLI to generate controllers:

```bash
# Basic controller
glib make:controller NotificationController

# Resource controller (7 methods: Index, Create, Store, Show, Edit, Update, Destroy)
glib make:controller PostController --resource

# API Resource controller (5 methods: Index, Store, Show, Update, Destroy)
glib make:controller PostController --api

# Invokable controller (single Invoke method)
glib make:controller PublishPostController --invokable
```

The generated controllers automatically include:
- Constructor with dependency injection
- Database manager resolved from container
- Commented example code for common operations
- Type-safe handler methods

See [`http/CONTROLLERS.md`](../../http/CONTROLLERS.md) for detailed documentation.

## Development

### Hot Reload with Air

```bash
# Install Air
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

Configuration is in `.air.toml`.

### Adding Features

**Example: Add post categories using controllers**

1. **Create category model:**
   ```bash
   glib make:model Category --migration --controller
   ```

2. **Update generated model with fields:**
   ```go
   type Category struct {
       ID   uuid.UUID `gorm:"type:uuid;primary_key"`
       Name string    `gorm:"unique;not null"`
   }
   ```

3. **Add relationship to Post:**
   ```go
   type Post struct {
       // ... existing fields
       CategoryID *uuid.UUID `gorm:"type:uuid"`
       Category   *Category  `gorm:"foreignKey:CategoryID"`
   }
   ```

4. **Register the controller:**
   ```go
   // In main.go
   router.APIResource("categories", func(app *foundation.Application) glib.Controller {
       return controllers.NewCategoryController(app)
   })
   ```

5. **The controller automatically provides:**
   - GET    /categories      → List all
   - POST   /categories      → Create
   - GET    /categories/{id} → Show one
   - PUT    /categories/{id} → Update
   - DELETE /categories/{id} → Delete

## Common Issues

### Token Not Working
- Ensure `Authorization: Bearer <token>` header is included
- Token format is correct (no extra spaces)
- Token is not expired (24 hours by default)

### Database Connection Failed
- Check database credentials in configuration
- Ensure database server is running (for PostgreSQL/MySQL)
- For SQLite, ensure write permissions for `blog.db`

### Validation Errors
- Check request body matches struct fields
- Verify Content-Type is `application/json`
- Ensure required fields are included

### Relationships Not Loading
- Use `.Preload()` to eager load relationships
- Check foreign key constraints are correct
- Verify relationship tags in models

## Production Considerations

This example is educational. For production:

1. **Security:**
   - Use proper JWT library (`github.com/golang-jwt/jwt/v5`)
   - Use bcrypt for password hashing (`golang.org/x/crypto/bcrypt`)
   - Add rate limiting
   - Implement CSRF protection
   - Use HTTPS

2. **Database:**
   - Use migrations instead of AutoMigrate
   - Add database indexes
   - Implement connection pooling
   - Use read replicas for scaling

3. **Error Handling:**
   - Don't expose internal errors to clients
   - Log errors properly
   - Use error tracking (Sentry, etc.)

4. **Performance:**
   - Add caching (Redis)
   - Implement pagination for all list endpoints
   - Use database query optimization
   - Add CDN for static assets

5. **Testing:**
   - Add unit tests for handlers
   - Add integration tests for database
   - Use test database
   - Mock external dependencies

## Resources

- [Glib Documentation](https://github.com/azizndao/glib)
- [GORM Documentation](https://gorm.io)
- [Go Validator](https://github.com/go-playground/validator)
- [JWT Introduction](https://jwt.io/introduction)

## Next Steps

After completing this example:

1. Try adding new features (categories, tags, likes, etc.)
2. Implement file uploads for post images
3. Add full-text search
4. Create admin dashboard
5. Build a frontend application
6. Deploy to production (Docker, Kubernetes)

---

**Need help?** Check the main [Glib examples](../README.md) or open an issue on GitHub.
