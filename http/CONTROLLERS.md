# Controllers in Glib

Laravel-style controllers with automatic dependency injection and resource routing.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Controller Types](#controller-types)
- [Dependency Injection](#dependency-injection)
- [Resource Controllers](#resource-controllers)
- [API Resource Controllers](#api-resource-controllers)
- [Invokable Controllers](#invokable-controllers)
- [Route Registration](#route-registration)
- [Resource Options](#resource-options)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Glib's controller system provides a Laravel-inspired pattern for organizing your HTTP handlers with:

- **Dependency Injection**: Dependencies resolved once at startup, not per-request
- **Resource Routing**: Automatic RESTful route generation
- **Type Safety**: Full Go type checking
- **Performance**: Zero per-request overhead for dependency resolution
- **Clean Code**: Handlers focus on business logic, not container access

## Quick Start

### 1. Create a Controller

```go
package controllers

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/foundation"
    "github.com/azizndao/glib/common/container"
    "github.com/azizndao/glib/database"
    "gorm.io/gorm"
)

type PostController struct {
    db *gorm.DB
}

// Constructor with dependency injection - returns interface type
func NewPostController(app *foundation.Application) glib.APIResourceController {
    dbManager, _ := container.Resolve[*database.Manager](app.Container())
    conn, _ := dbManager.DB()
    
    return &PostController{
        db: conn.DB(),
    }
}

// Handler methods use injected dependencies
func (ctrl *PostController) Index(c *glib.Ctx) error {
    var posts []models.Post
    ctrl.db.Find(&posts) // Use db directly - already injected!
    return c.JSON(posts)
}
```

### 2. Register the Controller

```go
// In main.go
server := glib.New(glib.Config{})
server.SetApplicationDirect(app)

router := server.Router()

// Register as API resource - pass constructor function directly
router.APIResource("posts", controllers.NewPostController)
```

That's it! This automatically creates 5 RESTful routes.

## Controller Types

### 1. Resource Controller

Full CRUD with 7 methods (including form views):

```go
type ResourceController interface {
    Index(c *Ctx) error    // GET /resource
    Create(c *Ctx) error   // GET /resource/create (form)
    Store(c *Ctx) error    // POST /resource
    Show(c *Ctx) error     // GET /resource/{id}
    Edit(c *Ctx) error     // GET /resource/{id}/edit (form)
    Update(c *Ctx) error   // PUT/PATCH /resource/{id}
    Destroy(c *Ctx) error  // DELETE /resource/{id}
}
```

**Use when**: Building web apps with server-rendered forms.

### 2. API Resource Controller

API-only CRUD with 5 methods (no form views):

```go
type APIResourceController interface {
    Index(c *Ctx) error    // GET /resource
    Store(c *Ctx) error    // POST /resource
    Show(c *Ctx) error     // GET /resource/{id}
    Update(c *Ctx) error   // PUT/PATCH /resource/{id}
    Destroy(c *Ctx) error  // DELETE /resource/{id}
}
```

**Use when**: Building REST APIs, SPAs, mobile backends.

### 3. Invokable Controller

Single-action controller:

```go
type InvokableController interface {
    Invoke(c *Ctx) error
}
```

**Use when**: One specific action (e.g., publish post, send email, generate report).

## Dependency Injection

Controllers use **constructor injection** for dependencies.

### How It Works

1. **Define Constructor**: Function that accepts `*foundation.Application`
2. **Resolve Dependencies**: Use container to get services
3. **Store in Struct**: Save dependencies as struct fields
4. **Use in Handlers**: Access dependencies directly

### Example: Multi-Dependency Controller

```go
type UserController struct {
    db     *gorm.DB
    logger *slog.Logger
    cache  *redis.Client
    mailer *mail.Service
}

func NewUserController(app *foundation.Application) glib.APIResourceController {
    // Resolve all dependencies once at startup
    dbManager, _ := container.Resolve[*database.Manager](app.Container())
    logger, _ := container.Resolve[*slog.Logger](app.Container())
    cache, _ := container.Resolve[*redis.Client](app.Container())
    mailer, _ := container.Resolve[*mail.Service](app.Container())
    
    conn, _ := dbManager.DB()
    
    return &UserController{
        db:     conn.DB(),
        logger: logger,
        cache:  cache,
        mailer: mailer,
    }
}

func (ctrl *UserController) Store(c *glib.Ctx) error {
    // All dependencies available without container access!
    user := models.User{...}
    ctrl.db.Create(&user)
    ctrl.logger.Info("User created", "id", user.ID)
    ctrl.cache.Del("users:all")
    ctrl.mailer.SendWelcome(user.Email)
    return c.JSON(user)
}
```

### Performance Benefits

**Without Controllers** (per-request resolution):
```go
func handler(c *glib.Ctx) error {
    dbManager, _ := container.Resolve[*database.Manager](app.Container()) // Every request!
    conn, _ := dbManager.DB()
    db := conn.DB()
    // ...
}
```

**With Controllers** (one-time resolution):
```go
func (ctrl *Controller) Handler(c *glib.Ctx) error {
    ctrl.db.Find(&items) // Dependencies already resolved!
    // ...
}
```

**Result**: ~1-5µs saved per request. For 10,000 req/s, that's 10-50ms/second saved.

## Resource Controllers

### Full Resource Registration

```go
router.Resource("posts", controllers.NewPostController)
```

**Generated Routes:**

| Method | Path | Handler | Name |
|--------|------|---------|------|
| GET | /posts | Index | posts.index |
| GET | /posts/create | Create | posts.create |
| POST | /posts | Store | posts.store |
| GET | /posts/{id} | Show | posts.show |
| GET | /posts/{id}/edit | Edit | posts.edit |
| PUT | /posts/{id} | Update | posts.update |
| PATCH | /posts/{id} | Update | posts.update |
| DELETE | /posts/{id} | Destroy | posts.destroy |

### Implementing Resource Controller

```go
type PostController struct {
    db *gorm.DB
}

func NewPostController(app *foundation.Application) glib.ResourceController {
    dbManager, _ := container.Resolve[*database.Manager](app.Container())
    conn, _ := dbManager.DB()
    return &PostController{db: conn.DB()}
}

// Implement all 7 methods
func (ctrl *PostController) Index(c *glib.Ctx) error {
    var posts []models.Post
    ctrl.db.Find(&posts)
    return c.JSON(posts)
}

func (ctrl *PostController) Create(c *glib.Ctx) error {
    // Return form HTML
    return c.HTML(`<form>...</form>`)
}

func (ctrl *PostController) Store(c *glib.Ctx) error {
    var post models.Post
    c.ParseBody(&post)
    ctrl.db.Create(&post)
    return c.JSON(post)
}

func (ctrl *PostController) Show(c *glib.Ctx) error {
    id := c.PathValue("id")
    var post models.Post
    ctrl.db.First(&post, id)
    return c.JSON(post)
}

func (ctrl *PostController) Edit(c *glib.Ctx) error {
    id := c.PathValue("id")
    var post models.Post
    ctrl.db.First(&post, id)
    return c.HTML(fmt.Sprintf(`<form>...</form>`, post))
}

func (ctrl *PostController) Update(c *glib.Ctx) error {
    id := c.PathValue("id")
    var post models.Post
    ctrl.db.First(&post, id)
    c.ParseBody(&post)
    ctrl.db.Save(&post)
    return c.JSON(post)
}

func (ctrl *PostController) Destroy(c *glib.Ctx) error {
    id := c.PathValue("id")
    ctrl.db.Delete(&models.Post{}, id)
    return c.NoContent()
}
```

## API Resource Controllers

### API Resource Registration

```go
router.APIResource("posts", controllers.NewPostController)
```

**Generated Routes:**

| Method | Path | Handler |
|--------|------|---------|
| GET | /posts | Index |
| POST | /posts | Store |
| GET | /posts/{id} | Show |
| PUT | /posts/{id} | Update |
| PATCH | /posts/{id} | Update |
| DELETE | /posts/{id} | Destroy |

**Note:** No `Create` or `Edit` (form routes) - perfect for APIs!

### Implementing API Resource Controller

```go
type PostController struct {
    db *gorm.DB
}

func NewPostController(app *foundation.Application) glib.APIResourceController {
    dbManager, _ := container.Resolve[*database.Manager](app.Container())
    conn, _ := dbManager.DB()
    return &PostController{db: conn.DB()}
}

// Implement 5 methods
func (ctrl *PostController) Index(c *glib.Ctx) error {
    var posts []models.Post
    ctrl.db.Find(&posts)
    return c.JSON(posts)
}

func (ctrl *PostController) Store(c *glib.Ctx) error {
    type CreateRequest struct {
        Title   string `json:"title" validate:"required"`
        Content string `json:"content" validate:"required"`
    }
    
    var req CreateRequest
    if err := c.ValidateBody(&req); err != nil {
        return err
    }
    
    post := models.Post{Title: req.Title, Content: req.Content}
    ctrl.db.Create(&post)
    return c.Status(201).JSON(post)
}

func (ctrl *PostController) Show(c *glib.Ctx) error {
    id := c.PathValue("id")
    var post models.Post
    if err := ctrl.db.First(&post, id).Error; err != nil {
        return errors.NotFound("Post not found", err)
    }
    return c.JSON(post)
}

func (ctrl *PostController) Update(c *glib.Ctx) error {
    id := c.PathValue("id")
    var post models.Post
    if err := ctrl.db.First(&post, id).Error; err != nil {
        return errors.NotFound("Post not found", err)
    }
    
    var updates models.Post
    c.ParseBody(&updates)
    ctrl.db.Model(&post).Updates(updates)
    return c.JSON(post)
}

func (ctrl *PostController) Destroy(c *glib.Ctx) error {
    id := c.PathValue("id")
    result := ctrl.db.Delete(&models.Post{}, id)
    if result.RowsAffected == 0 {
        return errors.NotFound("Post not found", nil)
    }
    return c.NoContent()
}
```

## Invokable Controllers

Single-action controllers for specific tasks.

### Example: Publish Post

```go
type PublishPostController struct {
    db *gorm.DB
}

func NewPublishPostController(app *foundation.Application) glib.InvokableController {
    dbManager, _ := container.Resolve[*database.Manager](app.Container())
    conn, _ := dbManager.DB()
    return &PublishPostController{db: conn.DB()}
}

func (ctrl *PublishPostController) Invoke(c *glib.Ctx) error {
    id := c.PathValue("id")
    
    var post models.Post
    if err := ctrl.db.First(&post, id).Error; err != nil {
        return errors.NotFound("Post not found", err)
    }
    
    post.Published = true
    post.PublishedAt = time.Now()
    ctrl.db.Save(&post)
    
    return c.JSON(map[string]interface{}{
        "message": "Post published successfully",
        "post": post,
    })
}
```

### Registration

```go
router.InvokableController("POST", "/posts/{id}/publish", controllers.NewPublishPostController)
```

## Route Registration

### Basic Registration

```go
// Resource (all 7 routes)
router.Resource("posts", NewPostController)

// API Resource (5 routes)
router.APIResource("posts", NewPostController)

// Invokable (1 route)
router.InvokableController("POST", "/posts/{id}/publish", NewPublishController)
```

### Auto-Detection (Convenience Method)

The `Controller()` method automatically detects the controller type and registers appropriate routes:

```go
// Automatically detects APIResourceController and registers 5 routes
// Pass constructor function directly
router.Controller("/posts", controllers.NewPostController)

// Or if your constructor signature matches ControllerConstructor:
router.Controller("/posts", controllers.NewPostController)

// Automatically detects ResourceController and registers 7 routes
router.Controller("/articles", controllers.NewArticleController)
```

**How it works:**
1. Inspects the controller instance to determine its type
2. If it implements `ResourceController` → calls `Resource()`
3. If it implements `APIResourceController` → calls `APIResource()`
4. If it implements `InvokableController` → returns error (requires explicit HTTP method)
5. Otherwise → returns error (unknown controller type)

**Example with error handling:**

```go
// Recommended: Explicit registration (clear and self-documenting)
router.APIResource("/posts", controllers.NewPostController)

// Alternative: Auto-detection (convenient but less explicit)
if _, err := router.Controller("/posts", controllers.NewPostController); err != nil {
    log.Fatalf("Failed to register controller: %v", err)
}
```

**When to use:**
- ✅ Use `Controller()` for quick prototyping or when the controller type is obvious
- ✅ Use explicit methods (`Resource`, `APIResource`) for production code (more readable)
- ❌ Don't use `Controller()` for `InvokableController` (requires HTTP method specification)

**Detection logs:**
Auto-detection logs the detected controller type at INFO level:
```
INFO: Auto-detected APIResourceController for pattern: /posts
```

### With Middleware

```go
// Apply middleware to resource
router.Route("/api", func(api glib.Router) {
    api.Use(middleware.Auth)
    api.APIResource("posts", NewPostController)
})
```

### Mixed Patterns

```go
// Some routes as resource, some custom
router.APIResource("posts", NewPostController, glib.ResourceOptions{
    Only: []string{"Index", "Show"}, // Only public routes
})

// Protected routes
router.Route("/api", func(api glib.Router) {
    api.Use(middleware.Auth)
    
    api.APIResource("posts", NewPostController, glib.ResourceOptions{
        Except: []string{"Index", "Show"}, // All except public
    })
})
```

## Resource Options

Customize resource route generation.

### Only Specific Methods

```go
router.Resource("posts", NewPostController, glib.ResourceOptions{
    Only: []string{"Index", "Show"},
})
// Only generates Index and Show routes
```

### Exclude Methods

```go
router.APIResource("posts", NewPostController, glib.ResourceOptions{
    Except: []string{"Destroy"},
})
// Generates all except Destroy
```

### Custom Route Names

```go
router.Resource("posts", NewPostController, glib.ResourceOptions{
    Names: map[string]string{
        "Index": "posts.list",
        "Show":  "posts.detail",
    },
})
```

### Custom Parameter Names

```go
router.Resource("posts", NewPostController, glib.ResourceOptions{
    Params: map[string]string{
        "id": "postId",
    },
})
// Changes /posts/{id} to /posts/{postId}
```

## Best Practices

### 1. One Controller Per Resource

```go
// ✅ Good
type PostController struct {}
type CommentController struct {}
type UserController struct {}

// ❌ Bad
type BlogController struct {} // Too broad
```

### 2. Resolve Dependencies in Constructor

```go
// ✅ Good
func NewPostController(app *foundation.Application) *PostController {
    db, _ := container.Resolve[*gorm.DB](app.Container())
    return &PostController{db: db}
}

// ❌ Bad
func (ctrl *PostController) Index(c *glib.Ctx) error {
    db, _ := container.Resolve[*gorm.DB](c.App().Container()) // Don't resolve per-request!
}
```

### 3. Use Request Validation

```go
func (ctrl *PostController) Store(c *glib.Ctx) error {
    type CreateRequest struct {
        Title string `json:"title" validate:"required,min=5"`
    }
    
    var req CreateRequest
    if err := c.ValidateBody(&req); err != nil {
        return err // Returns 400 with validation errors
    }
    
    // Use validated req
}
```

### 4. Return Structured Errors

```go
func (ctrl *PostController) Show(c *glib.Ctx) error {
    var post models.Post
    if err := ctrl.db.First(&post, id).Error; err != nil {
        return errors.NotFound("Post not found", err) // Returns 404
    }
    return c.JSON(post)
}
```

### 5. Use Custom Actions for Non-CRUD

```go
// Add custom methods to controller
func (ctrl *PostController) Publish(c *glib.Ctx) error {
    // Custom action logic
}

// Register separately
router.Post("/posts/{id}/publish", postCtrl.Publish)
```

## Examples

### Complete Blog API

```go
// Controller
type PostController struct {
    db *gorm.DB
}

func NewPostController(app *foundation.Application) *PostController {
    dbManager, _ := container.Resolve[*database.Manager](app.Container())
    conn, _ := dbManager.DB()
    return &PostController{db: conn.DB()}
}

func (ctrl *PostController) Index(c *glib.Ctx) error {
    var posts []models.Post
    ctrl.db.Preload("Author").Find(&posts)
    return c.JSON(posts)
}

func (ctrl *PostController) Store(c *glib.Ctx) error {
    var post models.Post
    if err := c.ValidateBody(&post); err != nil {
        return err
    }
    ctrl.db.Create(&post)
    return c.Status(201).JSON(post)
}

// ... other methods

// Registration
func main() {
    app := foundation.New(".")
    server := glib.New(glib.Config{})
    server.SetApplicationDirect(app)
    
    router := server.Router()
    router.APIResource("posts", controllers.NewPostController)
    
    server.ListenWithGracefulShutdown()
}
```

### Nested Resources

```go
// Posts have comments
router.APIResource("posts", NewPostController)

// Nested comment routes
router.Route("/posts/{postId}/comments", func(r glib.Router) {
    r.APIResource("", controllers.NewCommentController)
})
```

### Auth-Protected Resources

```go
// Public routes
router.Get("/posts", publicPostCtrl.Index)
router.Get("/posts/{id}", publicPostCtrl.Show)

// Protected routes
router.Route("/api", func(api glib.Router) {
    api.Use(middleware.Auth)
    
    api.APIResource("posts", NewPostController, glib.ResourceOptions{
        Except: []string{"Index", "Show"},
    })
})
```

## Summary

**Controllers in Glib provide:**

✅ Laravel-style patterns familiar to many developers  
✅ Constructor dependency injection (resolved once at startup)  
✅ Automatic RESTful route generation  
✅ Clean separation of concerns  
✅ Type-safe handlers with full Go checking  
✅ Zero per-request overhead for DI  
✅ Flexible resource options (Only, Except, custom names)  

**Choose your controller type:**

- **Resource**: Full CRUD + forms (7 methods)
- **API Resource**: API-only CRUD (5 methods)
- **Invokable**: Single action (1 method)

**Performance:** Dependencies resolved once at startup, not per-request.

**See also:**
- [Example: Fullstack Blog](/example/fullstack)
- [Foundation Module](/foundation)
- [Database Module](/database)
