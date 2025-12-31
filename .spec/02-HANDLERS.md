# 02. Handler Signatures

**Status:** Specification (Under Development)  
**Last Updated:** 2025-12-31

---

## Table of Contents

1. [Overview](#overview)
2. [Pattern 10: Result[T] - Type-Safe Handlers](#pattern-10-resultt---type-safe-handlers)
3. [Pattern 11: Raw HTTP Handlers](#pattern-11-raw-http-handlers)
4. [Result[T] API Reference](#resultt-api-reference)
5. [Code Generation Behavior](#code-generation-behavior)
6. [Best Practices](#best-practices)

---

## Overview

Glib supports **2 handler patterns**, providing flexibility while keeping the API simple and type-safe.

### Design Philosophy

- **Result[T] for most cases** - Type-safe, explicit status control, fluent API
- **Raw HTTP when you need control** - Streaming, SSE, file uploads, websockets
- **No runtime reflection** - Everything is generated at compile-time

### Pattern Matrix

| Pattern | Signature | Use Case | Status Control | Type Safety |
|---------|-----------|----------|----------------|-------------|
| 10 | `func(ctx, params...) glib.Result[T]` | Most API endpoints | Explicit via Result[T] | ✅ Full |
| 11 | `func(w, r)` | Streaming, SSE, files | Manual via w.WriteHeader() | ⚠️ Manual |

---

## Pattern 10: Result[T] - Type-Safe Handlers

### Signature

```go
func (c *Controller) HandlerName(
    ctx context.Context,
    [pathParam1 Type1,]
    [pathParam2 Type2,]
    [req RequestStruct]
) glib.Result[ResponseType]
```

### When to Use

**✅ Use Pattern 10 for:**
- RESTful CRUD operations
- Query endpoints
- Command endpoints
- Any JSON API
- ~95% of your endpoints

**❌ Don't use Pattern 10 for:**
- File downloads
- Streaming responses
- Server-Sent Events (SSE)
- WebSockets
- Custom binary protocols

### Features

1. **Explicit Status Control** - Choose exact HTTP status via helper functions
2. **Type-Safe** - Generic `Result[T]` ensures response type safety
3. **Fluent API** - Chain methods for headers and customization
4. **Auto Error Mapping** - `glib.Fail()` extracts status from `errs.Error`
5. **Generated Wrappers** - All parsing/marshalling code generated

### Basic Examples

#### Simple GET - Return List

```go
// @Route GET /posts
func (c *PostsController) Index(ctx context.Context) glib.Result[[]Post] {
    posts := c.Service.GetAll()
    return glib.OK(posts)
}
```

**Generated Response:**
- Status: 200 OK
- Content-Type: application/json
- Body: `[{...}, {...}]`

#### GET with Path Parameter

```go
// @Route GET /posts/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if err != nil {
        return glib.NotFound[*Post]("post not found")
    }
    return glib.OK(post)
}
```

**Success Response (200):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "title": "My Post",
  "content": "..."
}
```

**Error Response (404):**
```json
{
  "error": {
    "code": "not_found",
    "message": "post not found"
  }
}
```

#### POST with Request Body

```go
type CreatePostRequest struct {
    Title   string   `json:"title" validate:"required,min=3,max=200"`
    Content string   `json:"content" validate:"required,min=10"`
    Tags    []string `json:"tags"`
}

// @Route POST /posts
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    post, err := c.Service.Create(req.Title, req.Content, req.Tags)
    if err != nil {
        return glib.Fail[*Post](err)  // Auto-maps errs.Error to HTTP status
    }
    return glib.Created(post)
}
```

**Success Response (201):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "title": "New Post",
  "content": "...",
  "created_at": "2025-12-31T10:30:00Z"
}
```

#### PUT with Path Param + Request Body

```go
type UpdatePostRequest struct {
    Title   string `json:"title" validate:"required,min=3"`
    Content string `json:"content" validate:"required,min=10"`
}

// @Route PUT /posts/{id}
func (c *PostsController) Update(
    ctx context.Context,
    id uuid.UUID,
    req UpdatePostRequest,
) glib.Result[*Post] {
    post, err := c.Service.Update(id, req)
    if err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.OK(post)
}
```

#### DELETE - No Content Response

```go
// @Route DELETE /posts/{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    if err := c.Service.Delete(id); err != nil {
        return glib.Fail[any](err)
    }
    return glib.NoContent[any]()
}
```

**Success Response (204):**
- No body
- Status: 204 No Content

### Advanced Examples

#### Custom Headers

```go
// @Route GET /posts/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if err != nil {
        return glib.NotFound[*Post]("post not found")
    }
    
    return glib.OK(post).
        WithHeader("X-Post-Version", post.Version).
        WithHeader("Cache-Control", "max-age=3600")
}
```

#### Conditional Responses

```go
// @Route GET /posts/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if err != nil {
        return glib.NotFound[*Post]("post not found")
    }
    
    // Check If-None-Match header
    etag := fmt.Sprintf(`"%s"`, post.Version)
    if r.Header.Get("If-None-Match") == etag {
        return glib.NotModified[*Post]().WithHeader("ETag", etag)
    }
    
    return glib.OK(post).WithHeader("ETag", etag)
}
```

#### Multiple Path Parameters

```go
// @Route GET /posts/{postId}/comments/{commentId}
func (c *CommentsController) Show(
    ctx context.Context,
    postId uuid.UUID,
    commentId uuid.UUID,
) glib.Result[*Comment] {
    comment, err := c.Service.GetComment(postId, commentId)
    if err != nil {
        return glib.NotFound[*Comment]("comment not found")
    }
    return glib.OK(comment)
}
```

#### Validation Errors

```go
// @Route POST /posts
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    // Manual validation example
    var validationErrs []errs.ValidationError
    
    if len(req.Title) < 3 {
        validationErrs = append(validationErrs, errs.ValidationError{
            Field:    "title",
            Messages: []string{"must be at least 3 characters"},
        })
    }
    
    if len(req.Content) < 10 {
        validationErrs = append(validationErrs, errs.ValidationError{
            Field:    "content",
            Messages: []string{"must be at least 10 characters"},
        })
    }
    
    if len(validationErrs) > 0 {
        err := errs.B().
            Code(errs.InvalidArgument).
            Msg("Validation failed").
            Details(errs.NewValidationErrors(validationErrs)).
            Err()
        return glib.Fail[*Post](err)
    }
    
    post, err := c.Service.Create(req)
    if err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.Created(post)
}
```

**Validation Error Response (400):**
```json
{
  "error": {
    "code": "invalid_argument",
    "message": "Validation failed",
    "details": {
      "title": ["must be at least 3 characters"],
      "content": ["must be at least 10 characters"]
    }
  }
}
```

---

## Pattern 11: Raw HTTP Handlers

### Signature

```go
func (c *Controller) HandlerName(w http.ResponseWriter, r *http.Request)
```

### When to Use

**✅ Use Pattern 11 for:**
- File downloads with custom headers
- Streaming responses (video, audio)
- Server-Sent Events (SSE)
- WebSocket upgrades
- Multipart file uploads
- Custom binary protocols
- Fine-grained response control

**❌ Don't use Pattern 11 for:**
- Standard JSON APIs (use Pattern 10)
- CRUD operations (use Pattern 10)
- Simple queries (use Pattern 10)

### Features

1. **Full Control** - Direct access to `http.ResponseWriter` and `*http.Request`
2. **No Auto-Generation** - Handler registered directly with router
3. **Manual Everything** - Parsing, validation, marshalling, status codes
4. **Context Access** - Via `r.Context()`

### Examples

#### File Download

```go
// @Route GET /posts/{id}/export
func (c *PostsController) Export(w http.ResponseWriter, r *http.Request) {
    // Extract path parameter manually
    idStr := r.PathValue("id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    
    post, err := c.Service.GetByID(id)
    if err != nil {
        http.Error(w, "post not found", http.StatusNotFound)
        return
    }
    
    // Set headers for file download
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=post-%s.csv", id))
    w.WriteHeader(http.StatusOK)
    
    // Write CSV
    fmt.Fprintln(w, "id,title,content")
    fmt.Fprintf(w, "%s,%s,%s\n", post.ID, post.Title, post.Content)
}
```

#### Server-Sent Events (SSE)

```go
// @Route GET /posts/stream
func (c *PostsController) Stream(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }
    
    // Subscribe to post updates
    updates := c.Service.Subscribe()
    defer c.Service.Unsubscribe(updates)
    
    for {
        select {
        case <-r.Context().Done():
            return
        case post := <-updates:
            data, _ := json.Marshal(post)
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()
        }
    }
}
```

#### Multipart File Upload

```go
// @Route POST /posts/{id}/image
func (c *PostsController) UploadImage(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form (10MB max)
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }
    
    // Get file
    file, header, err := r.FormFile("image")
    if err != nil {
        http.Error(w, "No file provided", http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Validate file type
    if !strings.HasPrefix(header.Header.Get("Content-Type"), "image/") {
        http.Error(w, "File must be an image", http.StatusBadRequest)
        return
    }
    
    // Get post ID from path
    idStr := r.PathValue("id")
    id, _ := uuid.Parse(idStr)
    
    // Save file
    url, err := c.Storage.Save(id, file)
    if err != nil {
        http.Error(w, "Failed to save file", http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "url": url,
    })
}
```

#### Video Streaming with Range Requests

```go
// @Route GET /videos/{id}
func (c *VideosController) Stream(w http.ResponseWriter, r *http.Request) {
    idStr := r.PathValue("id")
    id, _ := uuid.Parse(idStr)
    
    video, err := c.Service.GetByID(id)
    if err != nil {
        http.Error(w, "video not found", http.StatusNotFound)
        return
    }
    
    file, err := os.Open(video.FilePath)
    if err != nil {
        http.Error(w, "Failed to open video", http.StatusInternalServerError)
        return
    }
    defer file.Close()
    
    stat, _ := file.Stat()
    
    // Handle range requests for video seeking
    http.ServeContent(w, r, video.Filename, stat.ModTime(), file)
}
```

---

## Result[T] API Reference

### Success Responses

| Function | Status Code | Use Case |
|----------|-------------|----------|
| `glib.OK[T](data)` | 200 | Standard success response |
| `glib.Created[T](data)` | 201 | Resource created successfully |
| `glib.Accepted[T](data)` | 202 | Request accepted for processing |
| `glib.NoContent[T]()` | 204 | Success with no response body |

### Redirection Responses

| Function | Status Code | Use Case |
|----------|-------------|----------|
| `glib.MovedPermanently[T](url)` | 301 | Resource moved permanently |
| `glib.Found[T](url)` | 302 | Temporary redirect |
| `glib.SeeOther[T](url)` | 303 | See other resource |
| `glib.NotModified[T]()` | 304 | Resource not modified (caching) |
| `glib.TemporaryRedirect[T](url)` | 307 | Temporary redirect (keep method) |
| `glib.PermanentRedirect[T](url)` | 308 | Permanent redirect (keep method) |

### Client Error Responses

| Function | Status Code | Use Case |
|----------|-------------|----------|
| `glib.BadRequest[T](msg)` | 400 | Invalid request data |
| `glib.Unauthorized[T](msg)` | 401 | Authentication required |
| `glib.Forbidden[T](msg)` | 403 | Insufficient permissions |
| `glib.NotFound[T](msg)` | 404 | Resource not found |
| `glib.MethodNotAllowed[T](msg)` | 405 | HTTP method not allowed |
| `glib.NotAcceptable[T](msg)` | 406 | Cannot produce requested content-type |
| `glib.RequestTimeout[T](msg)` | 408 | Request took too long |
| `glib.Conflict[T](msg)` | 409 | Resource conflict (e.g., duplicate) |
| `glib.Gone[T](msg)` | 410 | Resource permanently deleted |
| `glib.PreconditionFailed[T](msg)` | 412 | Precondition not met |
| `glib.PayloadTooLarge[T](msg)` | 413 | Request body too large |
| `glib.UnsupportedMediaType[T](msg)` | 415 | Content-type not supported |
| `glib.UnprocessableEntity[T](msg)` | 422 | Semantic validation failed |
| `glib.Locked[T](msg)` | 423 | Resource is locked |
| `glib.TooManyRequests[T](msg)` | 429 | Rate limit exceeded |

### Server Error Responses

| Function | Status Code | Use Case |
|----------|-------------|----------|
| `glib.InternalError[T](msg)` | 500 | Unexpected server error |
| `glib.NotImplemented[T](msg)` | 501 | Feature not implemented |
| `glib.BadGateway[T](msg)` | 502 | Upstream server error |
| `glib.ServiceUnavailable[T](msg)` | 503 | Service temporarily down |
| `glib.GatewayTimeout[T](msg)` | 504 | Upstream timeout |

### Custom Responses

| Function | Description |
|----------|-------------|
| `glib.WithStatus[T](data, code)` | Custom status with data |
| `glib.WithError[T](err, code)` | Custom status with error |
| `glib.Fail[T](err)` | Auto-extract status from `errs.Error` |

### Fluent API

```go
result.WithHeader(key, value string) Result[T]
result.WithHeaders(headers http.Header) Result[T]
result.Error() error  // Get internal error (for logging)
```

**Examples:**

```go
// Single header
return glib.OK(post).WithHeader("X-Version", "1.0")

// Multiple headers
headers := http.Header{}
headers.Set("Cache-Control", "max-age=3600")
headers.Set("X-Custom", "value")
return glib.OK(post).WithHeaders(headers)

// Chain headers
return glib.OK(post).
    WithHeader("X-Version", "1.0").
    WithHeader("Cache-Control", "max-age=3600")
```

---

## Code Generation Behavior

### Pattern 10: Generated Wrapper

**User writes:**
```go
// @Route POST /posts
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    post, err := c.Service.Create(req)
    if err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.Created(post)
}
```

**Generator creates:**
```go
func wrapPostsController_Create(container *container, ctrl *PostsController) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Parse request body
        var req CreatePostRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeResult(w, glib.BadRequest[*Post]("invalid JSON"))
            return
        }
        
        // Call handler
        result := ctrl.Create(r.Context(), req)
        
        // Write result
        writeResult(w, result)
    })
}

func writeResult[T any](w http.ResponseWriter, result glib.Result[T]) {
    // Set custom headers
    for key, values := range result.Headers {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    // Handle error response
    if err := result.Error(); err != nil {
        writeErrorJSON(w, result.StatusCode, err)
        return
    }
    
    // Handle no-content response
    if result.StatusCode == http.StatusNoContent {
        w.WriteHeader(result.StatusCode)
        return
    }
    
    // Write success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(result.StatusCode)
    if err := json.NewEncoder(w).Encode(result.Data); err != nil {
        log.Printf("failed to encode response: %v", err)
    }
}
```

### Pattern 11: Direct Registration

**User writes:**
```go
// @Route GET /stream
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    // Raw implementation
}
```

**Generator creates:**
```go
// Direct registration - no wrapper
mux.HandleFunc("GET /stream", ctrl.Stream)
```

---

## Best Practices

### 1. Choose the Right Pattern

```go
// ✅ Good - Pattern 10 for JSON API
func (c *Controller) Index(ctx context.Context) glib.Result[[]Post] {
    return glib.OK(c.Service.GetAll())
}

// ❌ Bad - Don't use Pattern 11 for simple JSON
func (c *Controller) Index(w http.ResponseWriter, r *http.Request) {
    posts := c.Service.GetAll()
    json.NewEncoder(w).Encode(posts)  // Unnecessary complexity
}

// ✅ Good - Pattern 11 for streaming
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    // Stream events...
}
```

### 2. Use Result[T] Helpers

```go
// ✅ Good - Use helpers for clarity
return glib.NotFound[*Post]("post not found")

// ❌ Bad - Don't build Result manually
return glib.Result[*Post]{
    StatusCode: 404,
    err: errs.B().Code(errs.NotFound).Msg("post not found").Err(),
}
```

### 3. Proper Error Handling

```go
// ✅ Good - Use glib.Fail() for errs.Error
post, err := c.Service.GetByID(id)
if err != nil {
    return glib.Fail[*Post](err)  // Auto-extracts HTTP status
}

// ✅ Good - Use specific helpers for known errors
if errors.Is(err, gorm.ErrRecordNotFound) {
    return glib.NotFound[*Post]("post not found")
}

// ❌ Bad - Don't return generic 500 for client errors
if id == uuid.Nil {
    return glib.InternalError[*Post]("invalid id")  // Should be BadRequest
}
```

### 4. Request Struct Organization

```go
// ✅ Good - Descriptive names, validation tags
type CreatePostRequest struct {
    Title   string   `json:"title" validate:"required,min=3,max=200"`
    Content string   `json:"content" validate:"required,min=10"`
    Tags    []string `json:"tags" validate:"max=10,dive,min=2,max=30"`
}

// ❌ Bad - Generic names, no validation
type Request struct {
    A string `json:"a"`
    B string `json:"b"`
}
```

### 5. Response Type Consistency

```go
// ✅ Good - Consistent response structure
type PostResponse struct {
    ID        uuid.UUID  `json:"id"`
    Title     string     `json:"title"`
    Content   string     `json:"content"`
    CreatedAt time.Time  `json:"created_at"`
}

// ✅ Good - Use pointer for optional/nullable
type UserResponse struct {
    ID      uuid.UUID       `json:"id"`
    Name    string          `json:"name"`
    Profile *ProfileDetails `json:"profile,omitempty"`
}
```

### 6. Custom Headers

```go
// ✅ Good - Add headers for caching, versioning
return glib.OK(post).
    WithHeader("Cache-Control", "max-age=3600").
    WithHeader("X-API-Version", "3.0")

// ✅ Good - ETags for conditional requests
etag := fmt.Sprintf(`"%s"`, post.Version)
return glib.OK(post).WithHeader("ETag", etag)
```

---

## Summary

### Pattern Comparison

| Aspect | Pattern 10 (Result[T]) | Pattern 11 (Raw HTTP) |
|--------|----------------------|---------------------|
| **Use Cases** | ~95% of endpoints | ~5% of endpoints |
| **Type Safety** | ✅ Full generic safety | ⚠️ Manual |
| **Status Control** | Explicit via helpers | Manual via WriteHeader |
| **Error Handling** | Auto-mapped from errs.Error | Manual |
| **Code Generation** | Wrapper generated | Direct registration |
| **Performance** | Excellent | Excellent |
| **Learning Curve** | Easy | Easy (standard Go) |

### Quick Reference

**Pattern 10 Signature:**
```go
func(ctx context.Context, [params...]) glib.Result[T]
```

**Pattern 11 Signature:**
```go
func(w http.ResponseWriter, r *http.Request)
```

**Rule of Thumb:** Start with Pattern 10. Only use Pattern 11 when you need fine-grained control over the HTTP response (streaming, file downloads, custom protocols).

---

**Next:** `03-CODE-GENERATION.md` - Code generation internals and templates
