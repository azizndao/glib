# 06. Complete Examples

**Status:** Specification v1.0  
**Last Updated:** 2025-12-30

---

## Table of Contents

1. [Overview](#overview)
2. [Example 1: Simple Blog API](#example-1-simple-blog-api)
3. [Example 2: E-Commerce REST API](#example-2-e-commerce-rest-api)
4. [Example 3: Microservice with Event Queue](#example-3-microservice-with-event-queue)
5. [Example 4: Real-Time Chat API](#example-4-real-time-chat-api)

---

## Overview

This document provides **complete, production-ready examples** demonstrating different application architectures and use cases with Glib 2.0.

### Examples Included

| Example          | Structure     | Features                     | Complexity   |
| ---------------- | ------------- | ---------------------------- | ------------ |
| **Simple Blog**  | Flat          | CRUD, SQLite, no auth        | Beginner     |
| **E-Commerce**   | Feature-based | Auth, payments, multi-model  | Intermediate |
| **Microservice** | Layered       | gRPC, events, observability  | Advanced     |
| **Chat API**     | Feature-based | WebSockets, real-time, Redis | Advanced     |

---

## Example 1: Simple Blog API

A minimal blog API with posts and comments. Perfect for learning Glib 2.0 basics.

### Features

- ✅ CRUD operations (posts, comments)
- ✅ SQLite database
- ✅ No authentication
- ✅ Flat project structure
- ✅ JSON responses

### Project Structure

```
blog-api/
├── main.go
├── config.go
├── posts.go
├── comments.go
├── providers.go
├── database.db
└── generated/
```

### Implementation

#### `config.go`

```go
package main

import (
    "os"
    "strconv"
)

type Config struct {
    App struct {
        Port int
        Env  string
    }
    Database struct {
        Path string
    }
}

func LoadConfig() *Config {
    cfg := &Config{}

    cfg.App.Port, _ = strconv.Atoi(getEnv("APP_PORT", "8080"))
    cfg.App.Env = getEnv("APP_ENV", "development")
    cfg.Database.Path = getEnv("DB_PATH", "database.db")

    return cfg
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

#### `providers.go`

```go
package main

import (
    "fmt"

    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error) {
    db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Auto-migrate models
    if err := db.AutoMigrate(&Post{}, &Comment{}); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }

    return db, nil
}
```

#### `posts.go`

```go
package main

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/goyave/glib/v2/errs"
    "gorm.io/gorm"
)

// Models

type Post struct {
    ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
    Title     string     `json:"title"`
    Content   string     `json:"content"`
    Published bool       `json:"published"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    Comments  []Comment  `json:"comments,omitempty" gorm:"foreignKey:PostID"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) error {
    if p.ID == uuid.Nil {
        p.ID = uuid.New()
    }
    return nil
}

// Requests

type CreatePostRequest struct {
    Title   string `json:"title" validate:"required,min=3,max=200"`
    Content string `json:"content" validate:"required"`
}

type UpdatePostRequest struct {
    Title   *string `json:"title" validate:"omitempty,min=3,max=200"`
    Content *string `json:"content" validate:"omitempty"`
}

// Controller

// @Controller path=/api/posts
type PostsController struct {
    DB *gorm.DB
}

// @Route method=GET path=/
func (c *PostsController) Index(ctx context.Context) glib.Result[[]Post] {
    var posts []Post
    if err := c.DB.Where("published = ?", true).Find(&posts).Error; err != nil {
        return glib.Fail[[]Post](errs.B().Code(errs.Internal).Msg("failed to fetch posts").Cause(err).Err())
    }
    return glib.OK(posts)
}

// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    var post Post
    err := c.DB.Preload("Comments").Where("id = ?", id).First(&post).Error
    if err == gorm.ErrRecordNotFound {
        return glib.NotFound[*Post]("post not found")
    }
    if err != nil {
        return glib.Fail[*Post](errs.B().Code(errs.Internal).Msg("failed to fetch post").Cause(err).Err())
    }
    return glib.OK(&post)
}

// @Route method=POST path=/
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    
    if err := c.DB.Create(post).Error; err != nil {
        return glib.Fail[*Post](errs.B().Code(errs.Internal).Msg("failed to create post").Cause(err).Err())
    }
    
    return glib.Created(post)
}

// @Route method=PUT path=/{id}
func (c *PostsController) Update(ctx context.Context, id uuid.UUID, req UpdatePostRequest) glib.Result[*Post] {
    var post Post
    err := c.DB.Where("id = ?", id).First(&post).Error
    if err == gorm.ErrRecordNotFound {
        return glib.NotFound[*Post]("post not found")
    }
    if err != nil {
        return glib.Fail[*Post](errs.B().Code(errs.Internal).Msg("failed to fetch post").Cause(err).Err())
    }
    
    if req.Title != nil {
        post.Title = *req.Title
    }
    if req.Content != nil {
        post.Content = *req.Content
    }
    
    if err := c.DB.Save(&post).Error; err != nil {
        return glib.Fail[*Post](errs.B().Code(errs.Internal).Msg("failed to update post").Cause(err).Err())
    }
    
    return glib.OK(&post)
}

// @Route method=DELETE path=/{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    result := c.DB.Delete(&Post{}, "id = ?", id)
    if result.Error != nil {
        return glib.Fail[any](errs.B().Code(errs.Internal).Msg("failed to delete post").Cause(result.Error).Err())
    }
    if result.RowsAffected == 0 {
        return glib.NotFound[any]("post not found")
    }
    return glib.NoContent[any]()
}

// @Route method=POST path=/{id}/publish
func (c *PostsController) Publish(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    var post Post
    err := c.DB.Where("id = ?", id).First(&post).Error
    if err == gorm.ErrRecordNotFound {
        return glib.NotFound[*Post]("post not found")
    }
    if err != nil {
        return glib.Fail[*Post](errs.B().Code(errs.Internal).Msg("failed to fetch post").Cause(err).Err())
    }
    
    post.Published = true
    if err := c.DB.Save(&post).Error; err != nil {
        return glib.Fail[*Post](errs.B().Code(errs.Internal).Msg("failed to publish post").Cause(err).Err())
    }
    
    return glib.OK(&post)
}
```

#### `comments.go`

```go
package main

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/goyave/glib/v2/errs"
    "gorm.io/gorm"
)

// Models

type Comment struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
    PostID    uuid.UUID `json:"post_id" gorm:"type:uuid"`
    Author    string    `json:"author"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
}

func (c *Comment) BeforeCreate(tx *gorm.DB) error {
    if c.ID == uuid.Nil {
        c.ID = uuid.New()
    }
    return nil
}

// Requests

type CreateCommentRequest struct {
    Author  string `json:"author" validate:"required,min=2,max=100"`
    Content string `json:"content" validate:"required,min=1,max=1000"`
}

// Controller

// @Controller path=/api/posts
type CommentsController struct {
    DB *gorm.DB
}

// @Route method=GET path=/{postID}/comments
func (c *CommentsController) Index(ctx context.Context, postID uuid.UUID) glib.Result[[]Comment] {
    var comments []Comment
    if err := c.DB.Where("post_id = ?", postID).Find(&comments).Error; err != nil {
        return glib.Fail[[]Comment](errs.B().Code(errs.Internal).Msg("failed to fetch comments").Cause(err).Err())
    }
    return glib.OK(comments)
}

// @Route method=POST path=/{postID}/comments
func (c *CommentsController) Create(ctx context.Context, postID uuid.UUID, req CreateCommentRequest) glib.Result[*Comment] {
    // Check if post exists
    var post Post
    err := c.DB.Where("id = ?", postID).First(&post).Error
    if err == gorm.ErrRecordNotFound {
        return glib.NotFound[*Comment]("post not found")
    }
    if err != nil {
        return glib.Fail[*Comment](errs.B().Code(errs.Internal).Msg("failed to fetch post").Cause(err).Err())
    }
    
    comment := &Comment{
        PostID:  postID,
        Author:  req.Author,
        Content: req.Content,
    }
    
    if err := c.DB.Create(comment).Error; err != nil {
        return glib.Fail[*Comment](errs.B().Code(errs.Internal).Msg("failed to create comment").Cause(err).Err())
    }
    
    return glib.Created(comment)
}

// @Route method=DELETE path=/{postID}/comments/{commentID}
func (c *CommentsController) Delete(ctx context.Context, postID, commentID uuid.UUID) glib.Result[any] {
    result := c.DB.Delete(&Comment{}, "id = ? AND post_id = ?", commentID, postID)
    if result.Error != nil {
        return glib.Fail[any](errs.B().Code(errs.Internal).Msg("failed to delete comment").Cause(result.Error).Err())
    }
    if result.RowsAffected == 0 {
        return glib.NotFound[any]("comment not found")
    }
    return glib.NoContent[any]()
}
```

#### `main.go`

```go
package main

import (
    "context"
    "log"
    "net/http"
    "fmt"

    "blog-api/generated"
)

func main() {
    ctx := context.Background()

    cfg := LoadConfig()

    handler, err := generated.Bootstrap(ctx)
    if err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }

    addr := fmt.Sprintf(":%d", cfg.App.Port)
    log.Printf("Server starting on %s", addr)

    if err := http.ListenAndServe(addr, handler); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}
```

### Usage

```bash
# Initialize project
glib init blog-api
cd blog-api

# Copy the code above into respective files

# Generate code
glib generate

# Run development server
glib dev

# Test the API
curl http://localhost:8080/api/posts
curl -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello World","content":"My first post"}'
```

### API Endpoints

```
GET    /api/posts              → List all published posts
GET    /api/posts/{id}         → Get post with comments
POST   /api/posts              → Create new post
PUT    /api/posts/{id}         → Update post
DELETE /api/posts/{id}         → Delete post
POST   /api/posts/{id}/publish → Publish post

GET    /api/posts/{postID}/comments           → List comments
POST   /api/posts/{postID}/comments           → Create comment
DELETE /api/posts/{postID}/comments/{commentID} → Delete comment
```

---

## Example 2: E-Commerce REST API

A feature-based e-commerce API with authentication, products, orders, and payments.

### Features

- ✅ JWT authentication
- ✅ User management
- ✅ Product catalog
- ✅ Shopping cart
- ✅ Order processing
- ✅ Payment integration (Stripe)
- ✅ PostgreSQL database
- ✅ Redis caching

### Project Structure

```
ecommerce-api/
├── main.go
├── config.go
├── auth/
│   ├── controller.go
│   ├── middleware.go
│   └── models.go
├── users/
│   ├── controller.go
│   ├── models.go
│   └── repository.go
├── products/
│   ├── controller.go
│   ├── models.go
│   └── repository.go
├── orders/
│   ├── controller.go
│   ├── models.go
│   └── repository.go
├── payments/
│   ├── controller.go
│   └── stripe.go
├── providers/
│   ├── database.go
│   ├── cache.go
│   └── stripe.go
├── middleware/
│   ├── auth.go
│   └── ratelimit.go
└── generated/
```

### Key Files

#### `config.go`

```go
package main

import (
    "os"
    "strconv"
)

type Config struct {
    App struct {
        Port   int
        Env    string
        Secret string
    }
    Database struct {
        Host     string
        Port     int
        Name     string
        User     string
        Password string
    }
    Redis struct {
        Host     string
        Port     int
        Password string
    }
    Stripe struct {
        SecretKey string
        WebhookSecret string
    }
}

func LoadConfig() *Config {
    cfg := &Config{}

    cfg.App.Port, _ = strconv.Atoi(getEnv("APP_PORT", "8080"))
    cfg.App.Env = getEnv("APP_ENV", "development")
    cfg.App.Secret = getEnv("APP_SECRET", "change-me-in-production")

    cfg.Database.Host = getEnv("DB_HOST", "localhost")
    cfg.Database.Port, _ = strconv.Atoi(getEnv("DB_PORT", "5432"))
    cfg.Database.Name = getEnv("DB_NAME", "ecommerce")
    cfg.Database.User = getEnv("DB_USER", "postgres")
    cfg.Database.Password = getEnv("DB_PASSWORD", "")

    cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
    cfg.Redis.Port, _ = strconv.Atoi(getEnv("REDIS_PORT", "6379"))
    cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")

    cfg.Stripe.SecretKey = getEnv("STRIPE_SECRET_KEY", "")
    cfg.Stripe.WebhookSecret = getEnv("STRIPE_WEBHOOK_SECRET", "")

    return cfg
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

#### `providers/database.go`

```go
package providers

import (
    "fmt"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.Name,
        cfg.Database.User,
        cfg.Database.Password,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Auto-migrate all models
    if err := db.AutoMigrate(
        &users.User{},
        &products.Product{},
        &orders.Order{},
        &orders.OrderItem{},
    ); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }

    return db, nil
}
```

#### `providers/cache.go`

```go
package providers

import (
    "fmt"

    "github.com/redis/go-redis/v9"
)

// @Provider singleton
func NewCache(cfg *Config) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
        Password: cfg.Redis.Password,
        DB:       0,
    })

    return client, nil
}
```

#### `auth/middleware.go`

```go
package auth

import (
    "context"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/goyave/glib/v2/errs"
)

type contextKey string

const UserContextKey contextKey = "user"

// @Middleware name=auth
func Auth(cfg *Config) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                handleError(w, errs.B().Code(errs.Unauthenticated).Msg("missing authorization header").Err())
                return
            }

            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString == authHeader {
                handleError(w, errs.B().Code(errs.Unauthenticated).Msg("invalid authorization format").Err())
                return
            }

            token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, errs.B().Code(errs.Unauthenticated).Msg("invalid signing method").Err()
                }
                return []byte(cfg.App.Secret), nil
            })

            if err != nil || !token.Valid {
                handleError(w, errs.B().Code(errs.Unauthenticated).Msg("invalid token").Err())
                return
            }

            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                handleError(w, errs.B().Code(errs.Unauthenticated).Msg("invalid claims").Err())
                return
            }

            userID := claims["sub"].(string)
            ctx := context.WithValue(r.Context(), UserContextKey, userID)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func GetUserID(ctx context.Context) string {
    userID, _ := ctx.Value(UserContextKey).(string)
    return userID
}
```

#### `auth/controller.go`

```go
package auth

import (
    "context"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/goyave/glib/v2/errs"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

// Requests

type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
}

type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}

// Responses

type AuthResponse struct {
    Token string      `json:"token"`
    User  *users.User `json:"user"`
}

// Controller

// @Controller path=/api/auth
type AuthController struct {
    DB  *gorm.DB
    Cfg *Config
}

// @Route method=POST path=/register
func (c *AuthController) Register(ctx context.Context, req RegisterRequest) glib.Result[*AuthResponse] {
    // Check if user exists
    var exists users.User
    if err := c.DB.Where("email = ?", req.Email).First(&exists).Error; err == nil {
        return nil, errs.B().Code(errs.AlreadyExists).Msg("email already registered").Err()
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to hash password").Cause(err).Err()
    }

    // Create user
    user := &users.User{
        Email:    req.Email,
        Password: string(hashedPassword),
        Name:     req.Name,
    }

    if err := c.DB.Create(user).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to create user").Cause(err).Err()
    }

    // Generate token
    token, err := c.generateToken(user.ID.String())
    if err != nil {
        return nil, err
    }

    return &AuthResponse{
        Token: token,
        User:  user,
    }, nil
}

// @Route method=POST path=/login
func (c *AuthController) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
    var user users.User
    err := c.DB.Where("email = ?", req.Email).First(&user).Error
    if err == gorm.ErrRecordNotFound {
        return nil, errs.B().Code(errs.Unauthenticated).Msg("invalid credentials").Err()
    }
    if err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("database error").Cause(err).Err()
    }

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        return nil, errs.B().Code(errs.Unauthenticated).Msg("invalid credentials").Err()
    }

    // Generate token
    token, err := c.generateToken(user.ID.String())
    if err != nil {
        return nil, err
    }

    return &AuthResponse{
        Token: token,
        User:  &user,
    }, nil
}

func (c *AuthController) generateToken(userID string) (string, error) {
    claims := jwt.MapClaims{
        "sub": userID,
        "exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
        "iat": time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(c.Cfg.App.Secret))
    if err != nil {
        return "", errs.B().Code(errs.Internal).Msg("failed to generate token").Cause(err).Err()
    }

    return tokenString, nil
}
```

#### `products/controller.go`

```go
package products

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/goyave/glib/v2/errs"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

// Models

type Product struct {
    ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Price       float64   `json:"price"`
    Stock       int       `json:"stock"`
    ImageURL    string    `json:"image_url"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
    if p.ID == uuid.Nil {
        p.ID = uuid.New()
    }
    return nil
}

// Requests

type CreateProductRequest struct {
    Name        string  `json:"name" validate:"required,min=3"`
    Description string  `json:"description"`
    Price       float64 `json:"price" validate:"required,gt=0"`
    Stock       int     `json:"stock" validate:"required,gte=0"`
    ImageURL    string  `json:"image_url" validate:"url"`
}

// Controller

// @Controller path=/api/products
type ProductsController struct {
    DB    *gorm.DB
    Cache *redis.Client
}

// @Route method=GET path=/
func (c *ProductsController) Index(ctx context.Context) ([]Product, error) {
    // Try cache first
    cacheKey := "products:all"
    cached, err := c.Cache.Get(ctx, cacheKey).Result()
    if err == nil {
        var products []Product
        if err := json.Unmarshal([]byte(cached), &products); err == nil {
            return products, nil
        }
    }

    // Fetch from database
    var products []Product
    if err := c.DB.Find(&products).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to fetch products").Cause(err).Err()
    }

    // Cache for 5 minutes
    if data, err := json.Marshal(products); err == nil {
        c.Cache.Set(ctx, cacheKey, data, 5*time.Minute)
    }

    return products, nil
}

// @Route method=GET path=/{id}
func (c *ProductsController) Show(ctx context.Context, id uuid.UUID) (*Product, error) {
    cacheKey := fmt.Sprintf("products:%s", id.String())
    cached, err := c.Cache.Get(ctx, cacheKey).Result()
    if err == nil {
        var product Product
        if err := json.Unmarshal([]byte(cached), &product); err == nil {
            return &product, nil
        }
    }

    var product Product
    err = c.DB.Where("id = ?", id).First(&product).Error
    if err == gorm.ErrRecordNotFound {
        return nil, errs.B().Code(errs.NotFound).Msg("product not found").Err()
    }
    if err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to fetch product").Cause(err).Err()
    }

    if data, err := json.Marshal(product); err == nil {
        c.Cache.Set(ctx, cacheKey, data, 5*time.Minute)
    }

    return &product, nil
}

// @Route method=POST path=/
// @Middleware name=auth
func (c *ProductsController) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
    product := &Product{
        Name:        req.Name,
        Description: req.Description,
        Price:       req.Price,
        Stock:       req.Stock,
        ImageURL:    req.ImageURL,
    }

    if err := c.DB.Create(product).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to create product").Cause(err).Err()
    }

    // Invalidate cache
    c.Cache.Del(ctx, "products:all")

    return product, nil
}
```

### API Endpoints

```
Authentication:
POST   /api/auth/register
POST   /api/auth/login

Products:
GET    /api/products          → List products (cached)
GET    /api/products/{id}     → Get product (cached)
POST   /api/products          → Create product (auth required)

Orders:
GET    /api/orders            → List user orders (auth required)
POST   /api/orders            → Create order (auth required)
GET    /api/orders/{id}       → Get order (auth required)

Payments:
POST   /api/payments/checkout → Create payment intent (auth required)
POST   /api/payments/webhook  → Stripe webhook
```

---

## Example 3: Microservice with Event Queue

A layered microservice architecture with gRPC, event-driven communication, and observability.

### Features

- ✅ gRPC API
- ✅ HTTP REST API (gateway)
- ✅ Event-driven architecture
- ✅ Message queue (RabbitMQ)
- ✅ Distributed tracing (OpenTelemetry)
- ✅ Metrics (Prometheus)
- ✅ Structured logging

### Project Structure

```
user-service/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── controllers/
│   │   ├── grpc.go
│   │   └── http.go
│   ├── services/
│   │   └── user_service.go
│   ├── repositories/
│   │   └── user_repository.go
│   ├── events/
│   │   ├── publisher.go
│   │   └── consumer.go
│   ├── providers/
│   │   ├── database.go
│   │   ├── queue.go
│   │   ├── tracer.go
│   │   └── metrics.go
│   ├── middleware/
│   │   ├── logging.go
│   │   └── tracing.go
│   └── generated/
├── pkg/
│   └── models/
│       └── user.go
└── proto/
    └── user/
        └── v1/
            └── user.proto
```

### Key Concepts

#### Event Publishing

```go
package events

import (
    "context"
    "encoding/json"

    "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
    channel *amqp091.Channel
}

// @Provider singleton
func NewPublisher(cfg *Config) (*Publisher, error) {
    conn, err := amqp091.Dial(cfg.Queue.URL)
    if err != nil {
        return nil, err
    }

    channel, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    return &Publisher{channel: channel}, nil
}

func (p *Publisher) Publish(ctx context.Context, event string, data interface{}) error {
    body, err := json.Marshal(data)
    if err != nil {
        return err
    }

    return p.channel.PublishWithContext(
        ctx,
        "events",
        event,
        false,
        false,
        amqp091.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}
```

#### Service Layer

```go
package services

import (
    "context"

    "github.com/google/uuid"
    "user-service/internal/events"
    "user-service/internal/repositories"
    "user-service/pkg/models"
)

type UserService struct {
    repo      *repositories.UserRepository
    publisher *events.Publisher
}

// @Provider singleton
func NewUserService(repo *repositories.UserRepository, pub *events.Publisher) *UserService {
    return &UserService{
        repo:      repo,
        publisher: pub,
    }
}

func (s *UserService) Create(ctx context.Context, user *models.User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }

    // Publish event
    s.publisher.Publish(ctx, "user.created", map[string]interface{}{
        "user_id": user.ID,
        "email":   user.Email,
    })

    return nil
}
```

---

## Example 4: Real-Time Chat API

WebSocket-based chat with Redis Pub/Sub for horizontal scaling.

### Features

- ✅ WebSocket connections
- ✅ Real-time messaging
- ✅ Redis Pub/Sub
- ✅ Room management
- ✅ User presence
- ✅ Message history

### Project Structure

```
chat-api/
├── main.go
├── config.go
├── chat/
│   ├── controller.go
│   ├── hub.go
│   ├── client.go
│   └── models.go
├── messages/
│   ├── controller.go
│   └── repository.go
├── providers/
│   ├── database.go
│   └── redis.go
└── generated/
```

### Key Implementation

#### `chat/hub.go`

```go
package chat

import (
    "context"
    "encoding/json"

    "github.com/redis/go-redis/v9"
)

type Hub struct {
    clients    map[string]*Client
    broadcast  chan *Message
    register   chan *Client
    unregister chan *Client
    redis      *redis.Client
}

// @Provider singleton
func NewHub(cache *redis.Client) *Hub {
    hub := &Hub{
        clients:    make(map[string]*Client),
        broadcast:  make(chan *Message),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        redis:      cache,
    }

    go hub.run()
    go hub.subscribePubSub()

    return hub
}

func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client.id] = client

        case client := <-h.unregister:
            if _, ok := h.clients[client.id]; ok {
                delete(h.clients, client.id)
                close(client.send)
            }

        case message := <-h.broadcast:
            // Send to local clients
            for _, client := range h.clients {
                if client.roomID == message.RoomID {
                    select {
                    case client.send <- message:
                    default:
                        close(client.send)
                        delete(h.clients, client.id)
                    }
                }
            }

            // Publish to Redis for other instances
            data, _ := json.Marshal(message)
            h.redis.Publish(context.Background(), "chat:messages", data)
        }
    }
}

func (h *Hub) subscribePubSub() {
    pubsub := h.redis.Subscribe(context.Background(), "chat:messages")
    defer pubsub.Close()

    for msg := range pubsub.Channel() {
        var message Message
        if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
            continue
        }

        // Send to local clients (avoid re-publishing)
        for _, client := range h.clients {
            if client.roomID == message.RoomID {
                select {
                case client.send <- &message:
                default:
                }
            }
        }
    }
}
```

#### `chat/controller.go`

```go
package chat

import (
    "net/http"

    "github.com/gorilla/websocket"
    "github.com/goyave/glib/v2/errs"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

// @Controller path=/api/chat
// @Middleware name=auth
type ChatController struct {
    Hub *Hub
}

// @Route method=GET path=/ws
func (c *ChatController) WebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        handleError(w, errs.B().Code(errs.Internal).Msg("failed to upgrade connection").Cause(err).Err())
        return
    }

    userID := auth.GetUserID(r.Context())
    roomID := r.URL.Query().Get("room")

    client := &Client{
        hub:    c.Hub,
        conn:   conn,
        send:   make(chan *Message, 256),
        id:     userID,
        roomID: roomID,
    }

    c.Hub.register <- client

    go client.writePump()
    go client.readPump()
}
```

---

## Summary

### Examples Comparison

| Feature        | Blog   | E-Commerce    | Microservice | Chat               |
| -------------- | ------ | ------------- | ------------ | ------------------ |
| **Structure**  | Flat   | Feature-based | Layered      | Feature-based      |
| **Database**   | SQLite | PostgreSQL    | PostgreSQL   | PostgreSQL + Redis |
| **Auth**       | ❌     | JWT           | JWT + gRPC   | JWT                |
| **Caching**    | ❌     | Redis         | Redis        | Redis Pub/Sub      |
| **Real-time**  | ❌     | ❌            | Events       | WebSockets         |
| **API Style**  | REST   | REST          | REST + gRPC  | REST + WS          |
| **Complexity** | Low    | Medium        | High         | High               |

### Key Takeaways

1. **Flat structure** works for simple apps
2. **Feature-based** scales well for medium apps
3. **Layered** best for complex/microservices
4. **Mix patterns** freely (REST + gRPC + WebSockets)
5. **Standard Go** - all examples use standard libraries + Glib

---

**Next:** `07-IMPLEMENTATION.md` - Phase-by-phase implementation roadmap
