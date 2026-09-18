# 06. Reference Examples

Complete, working reference implementations using the Glib web framework.

---

## Table of Contents

1. [Blog CRUD Controller](#1-blog-crud-controller)
2. [Authentication & Middleware Example](#2-authentication--middleware-example)
3. [Providers & Dependency Injection](#3-providers--dependency-injection)
4. [Response Metadata & Headers](#4-response-metadata--headers)
5. [Raw HTTP Handlers (CSV & SSE)](#5-raw-http-handlers-csv--sse)
6. [Internationalization (i18n)](#6-internationalization-i18n)
7. [Bootstrap & Server Lifecycle](#7-bootstrap--server-lifecycle)

---

## 1. Blog CRUD Controller

Demonstrates standard `(T, error)` and `error` handlers with path parameters, query parameters, and automatic request validation.

```go
// controllers/posts/controller.go
package posts

import (
    "context"
    "errors"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/azizndao/glib/errs"
    "my-app/models"
    "my-app/services"
)

// @Controller path=/api/v1/posts tags=api
type Controller struct {
    DB     *gorm.DB
    Logger *services.Logger
}

type ListPostsQuery struct {
    Page    int    `query:"page" validate:"required,min=1"`
    PerPage int    `query:"per_page" validate:"required,min=1,max=100"`
    Search  string `query:"search" validate:"omitempty,max=50"`
}

type CreatePostRequest struct {
    Title    string    `json:"title" validate:"required,min=3,max=200"`
    Body     string    `json:"body" validate:"required,min=10"`
    AuthorID uuid.UUID `json:"author_id" validate:"required,uuid4"`
}

type UpdatePostRequest struct {
    Title *string `json:"title" validate:"omitempty,min=3,max=200"`
    Body  *string `json:"body" validate:"omitempty,min=10"`
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context, q ListPostsQuery) ([]models.Post, error) {
    var posts []models.Post
    query := c.DB.Limit(q.PerPage).Offset((q.Page - 1) * q.PerPage)
    if q.Search != "" {
        query = query.Where("title LIKE ?", "%"+q.Search+"%")
    }
    if err := query.Find(&posts).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to fetch posts").Cause(err).Err()
    }
    return posts, nil
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Post, error) {
    var post models.Post
    if err := c.DB.First(&post, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errs.NewNotFound().WithMessage("post not found")
        }
        return nil, errs.B().Code(errs.Internal).Msg("database error").Cause(err).Err()
    }
    return &post, nil
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) (*models.Post, error) {
    post := &models.Post{
        ID:       uuid.New(),
        Title:    req.Title,
        Body:     req.Body,
        AuthorID: req.AuthorID,
    }
    if err := c.DB.Create(post).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to create post").Cause(err).Err()
    }
    return post, nil // Returns HTTP 201 Created
}

// @Route method=PUT path=/{id} tags=protected
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdatePostRequest) (*models.Post, error) {
    var post models.Post
    if err := c.DB.First(&post, "id = ?", id).Error; err != nil {
        return nil, errs.NewNotFound().WithMessage("post not found")
    }

    if req.Title != nil {
        post.Title = *req.Title
    }
    if req.Body != nil {
        post.Body = *req.Body
    }

    if err := c.DB.Save(&post).Error; err != nil {
        return nil, errs.B().Code(errs.Internal).Msg("failed to update post").Cause(err).Err()
    }
    return &post, nil
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error {
    res := c.DB.Delete(&models.Post{}, "id = ?", id)
    if res.Error != nil {
        return errs.B().Code(errs.Internal).Msg("failed to delete post").Cause(res.Error).Err()
    }
    if res.RowsAffected == 0 {
        return errs.NewNotFound().WithMessage("post not found")
    }
    return nil // Returns HTTP 204 No Content
}
```

---

## 2. Authentication & Middleware Example

Demonstrates native Glib middleware with context propagation and JWT verification.

```go
// middleware/auth.go
package middleware

import (
    "strings"

    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
    "my-app/services"
)

type ContextKey string

const UserIDKey ContextKey = "user_id"

// @Middleware name=auth target=protected order=10
func Auth(jwtService *services.JWTService) func(glib.Request, glib.Next) glib.Response {
    return func(req glib.Request, next glib.Next) glib.Response {
        authHeader := req.Header("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            return glib.Response{
                Err: errs.NewUnauthorized().WithMessage("authorization token required"),
            }
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := jwtService.Validate(token)
        if err != nil {
            return glib.Response{Err: err}
        }

        // Attach to context using immutable helper
        req = req.WithValue(UserIDKey, claims.UserID)

        // Forward to next handler
        resp := next(req)

        // Attach response header
        resp.Header().Set("X-User-ID", claims.UserID.String())
        return resp
    }
}
```

---

## 3. Providers & Dependency Injection

Demonstrates singleton and transient providers with config injection.

```go
// services/database.go
package services

import (
    "fmt"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "my-app/configs"
)

// @Provider singleton
func NewDatabase(cfg *configs.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
        cfg.Database.User, cfg.Database.Password)
    return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
```

```go
// services/logger.go
package services

import (
    "log/slog"
    "os"
)

type Logger struct {
    *slog.Logger
}

// @Provider transient
func NewLogger() *Logger {
    return &Logger{
        Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
    }
}
```

---

## 4. Response Metadata & Headers

Demonstrates custom HTTP status code and response header tags.

```go
// controllers/orders/controller.go
package orders

import (
    "context"
    "github.com/google/uuid"
)

type OrderCreatedResponse struct {
    StatusCode int          `json:"-" response:"httpstatus"`
    Location   string       `json:"-" header:"Location"`
    ETag       string       `json:"-" header:"ETag,omitempty"`
    OrderID    uuid.UUID    `json:"order_id"`
    Status     string       `json:"status"`
}

// @Controller path=/api/v1/orders
type Controller struct{}

// @Route method=POST path=/
func (c *Controller) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderCreatedResponse, error) {
    orderID := uuid.New()
    return &OrderCreatedResponse{
        StatusCode: 201,
        Location:   "/api/v1/orders/" + orderID.String(),
        ETag:       `"v1-order"`,
        OrderID:    orderID,
        Status:     "pending",
    }, nil
}
```

---

## 5. Raw HTTP Handlers (CSV & SSE)

```go
// controllers/reports/controller.go
package reports

import (
    "encoding/csv"
    "fmt"
    "net/http"
)

// @Controller path=/api/v1/reports
type Controller struct {
    EventStream chan string
}

// @Route method=GET path=/export
func (c *Controller) ExportCSV(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=report.csv")
    w.WriteHeader(http.StatusOK)

    writer := csv.NewWriter(w)
    defer writer.Flush()
    _ = writer.Write([]string{"ID", "Metric", "Value"})
    _ = writer.Write([]string{"1", "Active Users", "1200"})
}

// @Route method=GET path=/stream
func (c *Controller) StreamEvents(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
        return
    }

    for {
        select {
        case <-r.Context().Done():
            return
        case msg := <-c.EventStream:
            fmt.Fprintf(w, "data: %s\n\n", msg)
            flusher.Flush()
        }
    }
}
```

---

## 6. Internationalization (i18n)

```go
// controllers/auth/controller.go
package auth

import (
    "context"
    "glib/demo/generated/i18n"
    "github.com/azizndao/glib/errs"
    "github.com/google/uuid"
)

// @Controller path=/api/v1/users
type Controller struct {
    I18n *i18n.Translator
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*User, error) {
    user, err := findUser(id)
    if err != nil {
        msg := c.I18n.Errors.Users.NotFound(ctx, id.String())
        return nil, errs.NewNotFound().WithMessage(msg)
    }
    return user, nil
}
```

---

## 7. Bootstrap & Server Lifecycle

```go
// bootstrap.go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5/middleware"
    "my-app/generated"
)

func Bootstrap(ctx context.Context) (*http.Server, error) {
    app, err := generated.InitContainer(ctx)
    if err != nil {
        return nil, err
    }

    app.Router.Use(middleware.RequestID)
    app.Router.Use(middleware.Logger)
    app.Router.Use(middleware.Recoverer)

    if err := app.RegisterRoutes(); err != nil {
        return nil, err
    }

    return &http.Server{
        Addr:         app.Config.Addr(),
        Handler:      app.Router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
    }, nil
}
```
