# Glib 2.0 - Handler Signatures

Complete guide to the 9 flexible handler signature patterns supported by Glib 2.0.

---

## Overview

Glib 2.0 supports **9 different handler patterns**, giving you flexibility to choose the right level of abstraction for each endpoint. This design is inspired by [Encore.dev](https://encore.dev/docs/go/primitives/defining-apis).

### Philosophy

- **Raw `net/http` when you need control** (websockets, streaming, custom headers)
- **Typed requests/responses for common cases** (CRUD, APIs)
- **Mix patterns freely** (use what makes sense for each endpoint)

---

## Pattern Matrix

| Pattern | Context | Request | Response | HTTP Objects | Use Case |
|---------|---------|---------|----------|--------------|----------|
| 1 | ❌ | ❌ | ❌ | ✅ | Full control (websockets, streaming) |
| 2 | ✅ | ❌ | ❌ | ✅ | Context + full HTTP control |
| 3 | ✅ | ❌ | ❌ | ❌ | Simple action, no I/O |
| 4 | ✅ | ❌ | ✅ | ❌ | Static responses, no input |
| 5 | ✅ | ✅ | ❌ | ❌ | Commands (no response data) |
| 6 | ✅ | ✅ | ✅ | ❌ | **Most common** (CRUD APIs) |
| 7 | ✅ | ✅ | ✅ | ❌ | With path parameters |
| 8 | ✅ | ✅ | ✅ | ❌ | Multiple path parameters |
| 9 | ✅ | ✅ | ✅ | ✅ | Mix everything |

---

## Pattern 1: Raw net/http (Full Control)

**When to use:** Websockets, streaming, custom protocols, file uploads

```go
// @Route GET /ws
func (c *Controller) WebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Handle websocket...
}
```

### Features
- ✅ Full access to `http.ResponseWriter` and `*http.Request`
- ✅ Complete control over response
- ✅ No automatic parsing or marshalling
- ❌ More boilerplate

### Examples

#### File Upload
```go
// @Route POST /upload
func (c *Controller) Upload(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(10 << 20) // 10MB max
    
    file, handler, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Invalid file", http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Save file...
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "filename": handler.Filename,
    })
}
```

#### Server-Sent Events (SSE)
```go
// @Route GET /events
func (c *Controller) Events(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }
    
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-r.Context().Done():
            return
        case <-ticker.C:
            fmt.Fprintf(w, "data: %s\n\n", time.Now().Format(time.RFC3339))
            flusher.Flush()
        }
    }
}
```

---

## Pattern 2: Context + Raw HTTP

**When to use:** Need context for cancellation but want HTTP control

```go
// @Route GET /download
func (c *Controller) Download(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    // Use ctx for cancellation
    select {
    case <-ctx.Done():
        return
    default:
    }
    
    // Full HTTP control
    w.Header().Set("Content-Disposition", "attachment; filename=data.csv")
    w.WriteHeader(http.StatusOK)
    
    // Write data...
}
```

---

## Pattern 3: Context Only (No I/O)

**When to use:** Simple actions, side effects, no request/response data

```go
// @Route POST /cache/clear
func (c *Controller) ClearCache(ctx context.Context) error {
    return c.Cache.FlushAll(ctx).Err()
}

// @Route POST /notifications/send
func (c *Controller) SendNotification(ctx context.Context) error {
    // Trigger background job
    return c.Queue.Enqueue("send-notifications")
}
```

### Response

- **Success:** 200 OK with empty body
- **Error:** 500 Internal Server Error with error message

---

## Pattern 4: Context + Response (No Input)

**When to use:** Static data, health checks, status endpoints

```go
// @Route GET /health
func (c *Controller) Health(ctx context.Context) (*HealthResponse, error) {
    return &HealthResponse{
        Status: "ok",
        Time:   time.Now(),
        Version: "1.0.0",
    }, nil
}

type HealthResponse struct {
    Status  string    `json:"status"`
    Time    time.Time `json:"time"`
    Version string    `json:"version"`
}
```

### Response Types

Any Go type can be returned:

```go
// Struct
func Handle(ctx context.Context) (*Post, error)

// Slice
func Handle(ctx context.Context) ([]Post, error)

// Map
func Handle(ctx context.Context) (map[string]any, error)

// Primitive
func Handle(ctx context.Context) (string, error)
```

**Auto-marshalled to JSON** with appropriate `Content-Type: application/json` header.

---

## Pattern 5: Context + Request (No Response)

**When to use:** Commands, updates, deletions (no response data needed)

```go
// @Route POST /posts/{id}/publish
func (c *Controller) PublishPost(ctx context.Context, id uuid.UUID, req PublishRequest) error {
    post := &Post{}
    if err := c.DB.First(post, id).Error; err != nil {
        return err
    }
    
    post.PublishedAt = req.PublishAt
    post.Status = "published"
    
    return c.DB.Save(post).Error
}

type PublishRequest struct {
    PublishAt time.Time `json:"publish_at" validate:"required"`
}
```

### Response

- **Success:** 200 OK (or 204 No Content)
- **Error:** 500 Internal Server Error with error message

---

## Pattern 6: Context + Request + Response (Most Common)

**When to use:** CRUD operations, most API endpoints

```go
// @Route POST /posts
func (c *Controller) CreatePost(ctx context.Context, req CreatePostRequest) (*Post, error) {
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
        Status:  "draft",
    }
    
    if err := c.DB.Create(post).Error; err != nil {
        return nil, err
    }
    
    return post, nil
}

type CreatePostRequest struct {
    Title   string   `json:"title" validate:"required,min=3,max=200"`
    Content string   `json:"content" validate:"required,min=10"`
    Tags    []string `json:"tags"`
}
```

### Request Parsing

Request struct fields are automatically parsed from:
- **Path parameters** (`param:"name"`)
- **Query strings** (`query:"name"`)
- **Headers** (`header:"Name"`)
- **Body** (`json:"name"`) - default for fields without other tags

### Response Marshalling

Response is automatically:
- Marshalled to JSON
- Written with `Content-Type: application/json`
- Status code 200 OK (or 201 Created for POST)

---

## Pattern 7: Path Parameters + Request + Response

**When to use:** Resource operations with identifiers

```go
// @Route PUT /posts/{id}
func (c *Controller) UpdatePost(
    ctx context.Context, 
    id uuid.UUID, 
    req UpdatePostRequest,
) (*Post, error) {
    post := &Post{}
    if err := c.DB.First(post, id).Error; err != nil {
        return nil, err
    }
    
    post.Title = req.Title
    post.Content = req.Content
    
    if err := c.DB.Save(post).Error; err != nil {
        return nil, err
    }
    
    return post, nil
}

type UpdatePostRequest struct {
    Title   string `json:"title" validate:"required,min=3"`
    Content string `json:"content" validate:"required,min=10"`
}
```

### Path Parameter Types

Supported in route annotation:

```go
/{id}              // string (default)
/{id}         // int
/{id}64       // int64
/{id}        // uuid.UUID
/:slug:string    // explicit string
```

### Parameter Matching

Path parameters matched **by name and position**:

```go
// @Route GET /posts/{postID}/comments/{commentID}

// ✅ Correct
func Handle(ctx context.Context, postID uuid.UUID, commentID int) (*Response, error)

// ❌ Wrong order (but still works - matched by name)
func Handle(ctx context.Context, commentID int, postID uuid.UUID) (*Response, error)

// ✅ Also correct (via request struct)
type Request struct {
    PostID    uuid.UUID `param:"postID"`
    CommentID int       `param:"commentID"`
}
func Handle(ctx context.Context, req Request) (*Response, error)
```

---

## Pattern 8: Multiple Path Parameters

**When to use:** Nested resources, complex routes

```go
// @Route GET /posts/{postID}/comments/{commentID}
func (c *Controller) GetComment(
    ctx context.Context, 
    postID uuid.UUID, 
    commentID uuid.UUID,
) (*Comment, error) {
    comment := &Comment{}
    err := c.DB.
        Where("post_id = ? AND id = ?", postID, commentID).
        First(comment).
        Error
    
    return comment, err
}
```

### Wildcard Parameters

Use `*name` for catch-all:

```go
// @Route GET /static/{path...}
func (c *Controller) ServeStatic(ctx context.Context, path string) {
    // path contains everything after /static/
    // /static/css/main.css → path = "css/main.css"
}
```

---

## Pattern 9: Mix Everything

**When to use:** Need partial control (e.g., custom headers but typed request)

```go
// @Route POST /posts/{id}/upload
func (c *Controller) UploadImage(
    ctx context.Context,
    id uuid.UUID,
    w http.ResponseWriter,
    r *http.Request,
    req UploadImageRequest,
) error {
    // Path parameter: id
    // Request struct: req (parsed from query/headers)
    // Raw HTTP: w, r (for multipart form)
    
    // Parse multipart form
    file, header, err := r.FormFile("image")
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Use request struct data
    filename := fmt.Sprintf("%s-%s", id, req.Name)
    
    // Custom response
    w.Header().Set("X-Upload-ID", uuid.NewString())
    w.WriteHeader(http.StatusCreated)
    
    return nil
}

type UploadImageRequest struct {
    Name string `query:"name" validate:"required"`
    Alt  string `query:"alt"`
}
```

---

## Code Generation Behavior

### Detection Rules

Generator analyzes handler signature:

1. **Has `(http.ResponseWriter, *http.Request)`?**
   - → Raw handler, no code generation
   - → Direct registration with `http.ServeMux`

2. **Has `context.Context` as first param?**
   - → Generates wrapper with context

3. **Has struct parameter?**
   - → Generates request parser
   - → Validates before calling handler

4. **Returns `(T, error)` where T != void?**
   - → Generates response marshaller
   - → Handles errors automatically

5. **Has path parameters in route?**
   - → Generates path parameter extraction
   - → Type conversion and validation

### Generated Wrapper Example

**User writes:**
```go
// @Route POST /posts
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) (*Post, error) {
    // ...
}
```

**Generator creates:**
```go
func registerPostsControllerCreate(mux *http.ServeMux, ctrl *PostsController) {
    mux.HandleFunc("POST /posts", func(w http.ResponseWriter, r *http.Request) {
        // Parse request
        req, err := parseCreatePostRequest(r)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Call handler
        result, err := ctrl.Create(r.Context(), req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        // Marshal response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(result)
    })
}

func parseCreatePostRequest(r *http.Request) (CreatePostRequest, error) {
    var req CreatePostRequest
    
    // Parse body
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        return req, fmt.Errorf("invalid JSON: %w", err)
    }
    
    // Validate
    validate := validator.New()
    if err := validate.Struct(req); err != nil {
        return req, fmt.Errorf("validation failed: %w", err)
    }
    
    return req, nil
}
```

---

## Error Handling

All handler patterns can return `error`. Generator handles errors automatically:

### Error Detection

```go
func Handle(...) error {
    return errors.New("something failed")
}
```

**Default behavior:** 500 Internal Server Error

### Custom Error Status

Use custom error type for specific status codes:

```go
type HTTPError struct {
    Status  int
    Message string
}

func (e *HTTPError) Error() string {
    return e.Message
}

// Usage
func Handle(...) error {
    return &HTTPError{Status: 404, Message: "Post not found"}
}
```

### Error Type Detection

Generator can detect common error types:

```go
// 404 Not Found
if errors.Is(err, gorm.ErrRecordNotFound) {
    return &HTTPError{Status: 404, Message: "Not found"}
}

// 401 Unauthorized
if errors.Is(err, ErrUnauthorized) {
    return &HTTPError{Status: 401, Message: "Unauthorized"}
}
```

---

## HTTP Status Codes

### Automatic Status Codes

| Pattern | Success Status |
|---------|---------------|
| Raw `net/http` | User controls |
| Context only | 200 OK |
| Context + Response | 200 OK |
| Context + Request | 200 OK (204 No Content optional) |
| Context + Request + Response | 200 OK (POST → 201 Created) |

### Custom Status Codes

For typed responses, add `HttpStatus` field:

```go
type Response struct {
    Message string `json:"message"`
    Status  int    `json:"-"` // Excluded from JSON
}

// Set custom status
func Handle(...) (*Response, error) {
    return &Response{
        Message: "Created",
        Status:  201,
    }, nil
}
```

Or use response struct tag:

```go
type Response struct {
    Message    string `json:"message"`
    HTTPStatus int    `json:"-" httpstatus:""`
}
```

---

## Best Practices

### 1. Choose the Right Pattern

- **Pattern 1 (Raw):** Only when absolutely necessary
- **Pattern 6 (Request + Response):** Default choice for most APIs
- **Pattern 3 (Context only):** Simple actions, health checks
- **Pattern 9 (Mix):** Rare cases needing partial control

### 2. Request Structs

- Use descriptive names: `CreatePostRequest`, `UpdateUserRequest`
- Group related fields
- Add validation tags
- Document complex rules

```go
type CreatePostRequest struct {
    // Title of the blog post (3-200 characters)
    Title string `json:"title" validate:"required,min=3,max=200"`
    
    // Post content in Markdown format
    Content string `json:"content" validate:"required,min=10"`
    
    // Tags for categorization (2-10 chars each)
    Tags []string `json:"tags" validate:"dive,min=2,max=10"`
    
    // Draft status (defaults to true)
    Draft bool `json:"draft" default:"true"`
}
```

### 3. Response Structs

- Use pointers for optional fields
- Consistent naming
- JSON tags for field names

```go
type PostResponse struct {
    ID          uuid.UUID  `json:"id"`
    Title       string     `json:"title"`
    Content     string     `json:"content"`
    Author      *User      `json:"author,omitempty"`
    PublishedAt *time.Time `json:"published_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

### 4. Error Handling

- Return specific errors
- Use custom error types for status codes
- Log errors before returning

```go
func (c *Controller) GetPost(ctx context.Context, id uuid.UUID) (*Post, error) {
    post := &Post{}
    err := c.DB.First(post, id).Error
    
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, &HTTPError{Status: 404, Message: "Post not found"}
    }
    
    if err != nil {
        c.Logger.Error("database error", "error", err)
        return nil, err
    }
    
    return post, nil
}
```

---

## Summary

| Pattern | Signature | Use Case |
|---------|-----------|----------|
| 1 | `(w, r)` | Full control |
| 2 | `(ctx, w, r)` | Context + control |
| 3 | `(ctx) error` | Simple action |
| 4 | `(ctx) (T, error)` | Static response |
| 5 | `(ctx, req) error` | Command |
| 6 | `(ctx, req) (T, error)` | **Most common** |
| 7 | `(ctx, id, req) (T, error)` | With path param |
| 8 | `(ctx, id1, id2) (T, error)` | Multiple params |
| 9 | `(ctx, id, w, r, req) error` | Mix everything |

**Choose based on needs:** Start with Pattern 6, use others when necessary.
