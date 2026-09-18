# 02. Handler Signatures & Request/Response Processing

Comprehensive guide to handler patterns, parameter binding, validation, and response metadata in Glib.

---

## Table of Contents

1. [Overview](#overview)
2. [Pattern 1: Data & Error Handlers `(T, error)`](#pattern-1-data--error-handlers-t-error)
3. [Pattern 2: Error-Only Handlers `error`](#pattern-2-error-only-handlers-error)
4. [Pattern 3: Raw HTTP Handlers `(w, r)`](#pattern-3-raw-http-handlers-w-r)
5. [Parameter Binding Rules](#parameter-binding-rules)
6. [Request Validation System](#request-validation-system)
7. [Response Metadata (Status Codes & Headers)](#response-metadata-status-codes--headers)
8. [Best Practices](#best-practices)

---

## Overview

Glib supports **3 distinct handler patterns** tailored to different API needs:

| Pattern          | Signature                   | Typical Use Case                                  | Status Control                                     | Serialization      |
| :--------------- | :-------------------------- | :------------------------------------------------ | :------------------------------------------------- | :----------------- |
| **Data & Error** | `func(ctx, ...) (T, error)` | CRUD, JSON APIs, Queries (~90%)                   | Method default (200/201), struct tags, error codes | Automatic JSON     |
| **Error Only**   | `func(ctx, ...) error`      | Deletions, actions returning 204 No Content (~5%) | 204 No Content on nil error                        | None (empty body)  |
| **Raw HTTP**     | `func(w, r)`                | File download/upload, SSE, WebSockets (~5%)       | Direct `w.WriteHeader()`                           | Direct `w.Write()` |

---

## Pattern 1: Data & Error Handlers `(T, error)`

### Signature

```go
func (c *Controller) HandlerName(
    ctx context.Context,
    [pathParam1 Type1,]
    [paramsStruct ParamsType,]
    [reqBody BodyType],
) (ResponseType, error)
```

### Behavior

- The first parameter must always be `context.Context`.
- If the handler returns `nil, nil`, Glib writes HTTP 200 OK with `null` JSON body.
- If the handler returns `data, nil`:
    - `POST` handlers default to **HTTP 201 Created**.
    - `GET`, `PUT`, `PATCH`, `OPTIONS`, `HEAD` handlers default to **HTTP 200 OK**.
    - Response headers and custom status codes from struct tags are applied.
- If the handler returns an `error`:
    - Glib maps the error to the corresponding HTTP status using `errs.Error` code.
    - A structured JSON error payload is sent.

### Example

```go
// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Post, error) {
    post, err := c.postService.FindByID(id)
    if err != nil {
        return nil, errs.NewNotFound().WithMessage("post not found")
    }
    return post, nil
}
```

---

## Pattern 2: Error-Only Handlers `error`

### Signature

```go
func (c *Controller) HandlerName(
    ctx context.Context,
    [pathParam1 Type1,]
    [paramsStruct ParamsType,]
    [reqBody BodyType],
) error
```

### Behavior

- When returning `nil`:
    - `DELETE` handlers respond with **HTTP 204 No Content** and no body.
    - Other methods respond with **HTTP 200 OK** and no body.
- When returning `error`:
    - Serializes error with corresponding HTTP status code.

### Example

```go
// @Route method=DELETE path=/{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error {
    if err := c.postService.Delete(id); err != nil {
        return errs.B().Code(errs.Internal).Msg("failed to delete post").Cause(err).Err()
    }
    return nil // Sends HTTP 204 No Content
}
```

---

## Pattern 3: Raw HTTP Handlers `(w, r)`

### Signature

```go
func (c *Controller) HandlerName(w http.ResponseWriter, r *http.Request)
```

### Behavior

- Directly invoked by the generated routing layer.
- Bypasses automatic JSON serialization, allowing full access to headers, flushing, streaming, and custom encodings.
- Middleware chains applied to the route still execute around the raw handler.

### Example

```go
// @Route method=GET path=/export/csv
func (c *Controller) ExportCSV(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=export.csv")
    w.WriteHeader(http.StatusOK)

    writer := csv.NewWriter(w)
    defer writer.Flush()

    _ = writer.Write([]string{"ID", "Title", "Created"})
    for _, post := range c.postService.GetAll() {
        _ = writer.Write([]string{post.ID.String(), post.Title, post.CreatedAt.String()})
    }
}
```

---

## Parameter Binding Rules

Glib inspects handler parameter types and names during scanning to generate type-safe parsing:

### 1. Path Parameters

Primitive parameters (`string`, `int`, `int64`, `uint`, `bool`, etc.) or `uuid.UUID` whose parameter names match common path patterns are extracted from Chi URL path wildcards:

- Recognized names: `id`, `uuid`, `key`, `slug`, `userId`, `postId`, or any name ending in `Id` / `Key`.

```go
// Route pattern: /api/v1/users/{userId}/posts/{id}
// @Route method=GET path=/{userId}/posts/{id}
func (c *Controller) GetUserPost(ctx context.Context, userId uuid.UUID, id int) (*models.Post, error)
```

### 2. Query Parameters (`query:"..."`)

Struct fields tagged with `query:` are automatically parsed from `r.URL.Query()`:

```go
type PaginationQuery struct {
    Page    int    `query:"page" validate:"required,min=1"`
    PerPage int    `query:"per_page" validate:"required,min=1,max=100"`
    Sort    string `query:"sort" validate:"omitempty,oneof=asc desc"`
}

// @Route method=GET path=/
func (c *Controller) List(ctx context.Context, q PaginationQuery) ([]models.Post, error)
```

### 3. Header Parameters (`header:"..."`)

Struct fields tagged with `header:` are automatically parsed from incoming request headers:

```go
type AuthHeaders struct {
    ClientVersion string `header:"X-Client-Version" validate:"required"`
    RequestTrace  string `header:"X-Trace-ID"`
}
```

### 4. JSON Request Body (`json:"..."`)

Structs that do not represent query/header containers (or structs containing `json:` tags) are automatically parsed from `r.Body`:

```go
type CreatePostRequest struct {
    Title   string `json:"title" validate:"required,min=3"`
    Content string `json:"content" validate:"required,min=10"`
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) (*models.Post, error)
```

---

## Request Validation System

Glib integrates directly with `github.com/go-playground/validator/v10`.

### Automatic Validation

When a request struct contains `validate:` struct tags, Glib generates automatic validation calls prior to invoking the handler:

```go
type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
}
```

### Conditional Validation (`validator.Validable`)

If a request struct implements the `validator.Validable` interface, validation execution depends on the return value of `Validate()`:

```go
func (r RegisterRequest) Validate() bool {
    // Return false to skip validation based on custom runtime conditions
    return true
}
```

### Validation Error Output

On validation failure, Glib terminates execution and immediately writes a structured `400 Bad Request` response:

```json
{
    "error": {
        "code": "invalid_argument",
        "message": "Validation failed",
        "details": {
            "body": {
                "email": {
                    "errors": ["email must be a valid email address"]
                }
            }
        }
    }
}
```

---

## Response Metadata (Status Codes & Headers)

For handlers returning structs or struct pointers in `(T, error)`, Glib inspects struct tags to extract dynamic status codes and headers at response time:

### Custom Status Code: `response:"httpstatus"`

```go
type CustomStatusResponse struct {
    StatusCode int `json:"-" response:"httpstatus"`
    Data       any `json:"data"`
}

// Handler returns status code dynamically
func (c *Controller) CustomAction(ctx context.Context) (*CustomStatusResponse, error) {
    return &CustomStatusResponse{
        StatusCode: 202, // 202 Accepted
        Data:       "processing",
    }, nil
}
```

### Response Headers: `header:"Header-Name[,omitempty]"`

```go
type PostCreatedResponse struct {
    Location string `json:"-" header:"Location"`
    ETag     string `json:"-" header:"ETag,omitempty"`
    Post     *models.Post `json:"post"`
}

func (c *Controller) Create(ctx context.Context, req CreateRequest) (*PostCreatedResponse, error) {
    post := c.service.Create(req)
    return &PostCreatedResponse{
        Location: "/api/v1/posts/" + post.ID.String(),
        ETag:     `"xyz789"`,
        Post:     post,
    }, nil
}
```

---

## Best Practices

1. **Default to `(T, error)`**: Use `(T, error)` for almost all endpoints. It is idiomatic Go and gives full type safety.
2. **Use `error` for Deletions**: Return `error` only for `DELETE` handlers to cleanly yield `204 No Content`.
3. **Use Raw Handlers Sparingly**: Reserve `(w, r)` handlers only for streaming (SSE), WebSockets, or high-volume file transfer.
4. **Prefer Struct Tag Validation**: Use `validate:` tags on request models rather than manual `if field == ""` checks inside handlers.
5. **Leverage `errs` Package**: Return structured errors (`errs.NewNotFound()`, `errs.B().Code(...)`) so HTTP status codes are determined at the point of failure.
