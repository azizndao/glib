# 02. Handler Signatures

**Status:** Specification (Under Development)  
**Last Updated:** 2025-12-31

---

## Table of Contents

1. [Overview](#overview)
2. [Pattern: (T, error) - Standard Go Handlers](#pattern-t-error---standard-go-handlers)
3. [Pattern: Raw HTTP Handlers](#pattern-raw-http-handlers)
4. [Code Generation Behavior](#code-generation-behavior)
5. [Best Practices](#best-practices)

---

## Overview

Glib supports **2 handler patterns**, providing flexibility while keeping the API simple and type-safe.

### Design Philosophy

- **(T, error) for idiomatic Go** - Standard Go error handling with struct tag metadata
- **Raw HTTP when you need control** - Streaming, SSE, file uploads, websockets
- **Minimal runtime reflection** - Metadata cached at first use per type

### Pattern Matrix

| Pattern    | Signature                         | Use Case              | Status Control            | Type Safety | Performance |
| ---------- | --------------------------------- | --------------------- | ------------------------- | ----------- | ----------- |
| (T, error) | `func(ctx, params...) (T, error)` | Modern idiomatic APIs | Struct tags + error codes | ✅ Full     | ⚡ Fastest  |
| Raw HTTP   | `func(w, r)`                      | Streaming, SSE, files | Manual via w.WriteHeader()| ⚠️ Manual   | ⚡ Fastest  |

---

## Pattern: (T, error) - Standard Go Handlers

### Signature

```go
func (c *Controller) HandlerName(
    ctx context.Context,
    [pathParam1 Type1,]
    [pathParam2 Type2,]
    [req RequestStruct]
) (ResponseType, error)
```

### When to Use

**✅ Use (T, error) pattern for:**

- Modern idiomatic Go APIs
- When you prefer standard Go error handling
- Response metadata via struct tags
- ~95% of your endpoints (recommended default)

**❌ Don't use (T, error) pattern for:**

- Streaming responses
- File downloads with custom logic
- Server-Sent Events (SSE)
- WebSockets

### Features

1. **Idiomatic Go** - Standard `(T, error)` signature
2. **Struct Tag Metadata** - Control HTTP status and headers via tags
3. **Error Mapping** - `errs.Error` automatically mapped to HTTP status
4. **Type-Safe** - Generic type inference for responses
5. **Cached Reflection** - Metadata analyzed once per type

### Response Metadata Tags

#### `response:"httpstatus"` Tag

Control HTTP status code via struct field:

```go
type CreateResponse struct {
    ID     string `json:"id"`
    Status int    `response:"httpstatus"`  // Must be int type
}

func (c *Controller) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
    item := c.Service.Create(req)
    return &CreateResponse{
        ID:     item.ID,
        Status: 201,  // Returns 201 Created
    }, nil
}
```

**Rules:**
- Field **must** be `int` type (not `int64`, `uint`, etc.)
- If field is `0`, defaults to 200
- If field is invalid (<100 or >599), defaults to 200
- Only one `response:"httpstatus"` field per struct
- Framework logs warning if field type is wrong

#### `header:"Name"` Tag

Set HTTP response headers via struct fields:

```go
type CreateResponse struct {
    ID       string `json:"id"`
    Location string `header:"Location"`              // Always set
    ETag     string `header:"ETag,omitempty"`        // Only if non-empty
    Status   int    `response:"httpstatus"`
}

func (c *Controller) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
    item := c.Service.Create(req)
    return &CreateResponse{
        ID:       item.ID,
        Location: fmt.Sprintf("/api/items/%s", item.ID),
        ETag:     item.Version,
        Status:   201,
    }, nil
}
```

**Response:**
```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /api/items/123
ETag: v1.0.0

{"id":"123"}
```

**Tag Format:**
- `header:"Name"` - Always set header (even if empty string)
- `header:"Name,omitempty"` - Only set if non-zero value
- Header name must be valid HTTP header (e.g., `Location`, `ETag`, `X-Custom`)

**Supported Types:**
- `string` → Direct value
- `int`, `int8`, `int16`, `int32`, `int64` → Converted via `strconv.FormatInt`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` → Converted via `strconv.FormatUint`
- `float32`, `float64` → Converted via `strconv.FormatFloat`
- `bool` → `"true"` or `"false"`
- Pointers → Dereferenced (nil = omitted if `omitempty`)

### Basic Examples

#### Simple GET - Return Data

```go
// @Route method=GET path=/
func (c *PostsController) Index(ctx context.Context) ([]Post, error) {
    posts := c.Service.GetAll()
    return posts, nil
}
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

[{"id":"1","title":"Post 1"},{"id":"2","title":"Post 2"}]
```

#### GET with Path Parameter

```go
// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    post, err := c.Service.GetByID(id)
    if err != nil {
        return nil, errs.NewNotFound().Msgf("post %s not found", id).Err()
    }
    return post, nil
}
```

**Success (200):**
```json
{"id":"123","title":"My Post","content":"..."}
```

**Error (404):**
```json
{
    "error": {
        "code": "not_found",
        "message": "post 123 not found"
    }
}
```

#### POST with 201 Status

```go
type CreatePostResponse struct {
    ID       string `json:"id"`
    Location string `header:"Location"`
    Status   int    `response:"httpstatus"`
}

// @Route method=POST path=/
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) (*CreatePostResponse, error) {
    post, err := c.Service.Create(req.Title, req.Content)
    if err != nil {
        return nil, err  // Auto-mapped to HTTP status
    }
    return &CreatePostResponse{
        ID:       post.ID,
        Location: fmt.Sprintf("/api/posts/%s", post.ID),
        Status:   201,
    }, nil
}
```

**Response:**
```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /api/posts/123

{"id":"123"}
```

#### PUT with ETag

```go
type UpdatePostResponse struct {
    Post   *Post  `json:"post"`
    ETag   string `header:"ETag"`
}

// @Route method=PUT path=/{id}
func (c *PostsController) Update(
    ctx context.Context,
    id uuid.UUID,
    req UpdatePostRequest,
) (*UpdatePostResponse, error) {
    post, err := c.Service.Update(id, req)
    if err != nil {
        return nil, err
    }
    return &UpdatePostResponse{
        Post: post,
        ETag: fmt.Sprintf(`"%s"`, post.Version),
    }, nil
}
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json
ETag: "v2.0.1"

{"post":{"id":"123","title":"Updated","version":"v2.0.1"}}
```

#### DELETE - No Content

```go
// @Route method=DELETE path=/{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) (any, error) {
    if err := c.Service.Delete(id); err != nil {
        return nil, err
    }
    return nil, nil  // Returns 204 No Content when response is nil
}
```

**Response:**
```http
HTTP/1.1 204 No Content
```

### Advanced Examples

#### Conditional Headers with `omitempty`

```go
type PostResponse struct {
    Post         *Post  `json:"post"`
    ETag         string `header:"ETag,omitempty"`          // Only if non-empty
    CacheControl string `header:"Cache-Control,omitempty"` // Only if non-empty
}

// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*PostResponse, error) {
    post, err := c.Service.GetByID(id)
    if err != nil {
        return nil, errs.NewNotFound().Msgf("post not found").Err()
    }

    resp := &PostResponse{Post: post}
    
    // Only set ETag for published posts
    if post.Published {
        resp.ETag = fmt.Sprintf(`"%s"`, post.Version)
        resp.CacheControl = "max-age=3600"
    }
    
    return resp, nil
}
```

#### Multiple Headers

```go
type DownloadResponse struct {
    Data        []byte `json:"data"`
    Filename    string `header:"Content-Disposition"`
    ContentType string `header:"Content-Type"`
    Size        int64  `header:"Content-Length"`
}

// @Route method=GET path=/{id}/download
func (c *PostsController) Download(ctx context.Context, id uuid.UUID) (*DownloadResponse, error) {
    data, err := c.Service.Export(id)
    if err != nil {
        return nil, err
    }
    
    return &DownloadResponse{
        Data:        data,
        Filename:    fmt.Sprintf("attachment; filename=post-%s.json", id),
        ContentType: "application/json",
        Size:        int64(len(data)),
    }, nil
}
```

#### Custom Status Codes

```go
type AsyncResponse struct {
    JobID  string `json:"job_id"`
    Status int    `response:"httpstatus"`
}

// @Route method=POST path=/bulk
func (c *PostsController) BulkCreate(ctx context.Context, req BulkRequest) (*AsyncResponse, error) {
    jobID, err := c.Queue.Enqueue(req)
    if err != nil {
        return nil, err
    }
    return &AsyncResponse{
        JobID:  jobID,
        Status: 202,  // Accepted
    }, nil
}
```

#### Error Handling with Status Tags

```go
type ErrorResponse struct {
    Message string `json:"message"`
    Status  int    `response:"httpstatus"`
}

// @Route method=GET path=/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    post, err := c.Service.GetByID(id)
    if err != nil {
        // Option 1: Use errs.Error (recommended)
        return nil, errs.NewNotFound().Msgf("post not found").Err()
        
        // Option 2: Not possible - can't return ErrorResponse here
        // This pattern returns the success type or error, not a custom error struct
    }
    return post, nil
}
```

**Note:** The `(T, error)` pattern returns either:
- Success: `(T, nil)` where T can have `response:"httpstatus"` and `header:` tags
- Error: `(nil, error)` where error is mapped via `errs.Error` codes

You **cannot** return a custom error response struct with status tags. Use `errs.Error` for errors.

### Performance Characteristics

**Metadata Caching:**
```go
// First request for type *CreateResponse
// - Analyzes struct tags via reflection
// - Caches result in map[reflect.Type]*responseMetadataCache
// - Thread-safe with sync.RWMutex

// Subsequent requests for *CreateResponse
// - Cache hit (RLock only)
// - No reflection analysis
// - ~50-80% faster than uncached
```

**Benchmark Results:**
```
BenchmarkWriteResponse_WithMetadata-8    458480   2632 ns/op   1056 B/op   10 allocs/op
BenchmarkWriteResponse_NoMetadata-8      687608   1736 ns/op   1008 B/op    9 allocs/op
```

**Cost:** ~896 ns/op overhead for metadata extraction (one-time per type, then cached).

---

## Pattern: Raw HTTP Handlers

### Signature

```go
func (c *Controller) HandlerName(w http.ResponseWriter, r *http.Request)
```

### When to Use

**✅ Use Raw HTTP pattern for:**

- File downloads with custom headers
- Streaming responses (video, audio)
- Server-Sent Events (SSE)
- WebSocket upgrades
- Multipart file uploads
- Custom binary protocols
- Fine-grained response control

**❌ Don't use Raw HTTP pattern for:**

- Standard JSON APIs (use Result[T])
- CRUD operations (use Result[T])
- Simple queries (use Result[T])

### Features

1. **Full Control** - Direct access to `http.ResponseWriter` and `*http.Request`
2. **No Auto-Generation** - Handler registered directly with router
3. **Manual Everything** - Parsing, validation, marshalling, status codes
4. **Context Access** - Via `r.Context()`

### Examples

#### File Download

```go
// @Route method=GET path=/{id}/export
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
// @Route method=GET path=/stream
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
// @Route method=POST path=/{id}/image
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
// @Route method=GET path=/{id}
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

| Function                 | Status Code | Use Case                        |
| ------------------------ | ----------- | ------------------------------- |
| `glib.OK[T](data)`       | 200         | Standard success response       |
| `glib.Created[T](data)`  | 201         | Resource created successfully   |
| `glib.Accepted[T](data)` | 202         | Request accepted for processing |
| `glib.NoContent[T]()`    | 204         | Success with no response body   |

### Redirection Responses

| Function                         | Status Code | Use Case                         |
| -------------------------------- | ----------- | -------------------------------- |
| `glib.MovedPermanently[T](url)`  | 301         | Resource moved permanently       |
| `glib.Found[T](url)`             | 302         | Temporary redirect               |
| `glib.SeeOther[T](url)`          | 303         | See other resource               |
| `glib.NotModified[T]()`          | 304         | Resource not modified (caching)  |
| `glib.TemporaryRedirect[T](url)` | 307         | Temporary redirect (keep method) |
| `glib.PermanentRedirect[T](url)` | 308         | Permanent redirect (keep method) |

### Client Error Responses

| Function                            | Status Code | Use Case                              |
| ----------------------------------- | ----------- | ------------------------------------- |
| `glib.BadRequest[T](msg)`           | 400         | Invalid request data                  |
| `glib.Unauthorized[T](msg)`         | 401         | Authentication required               |
| `glib.Forbidden[T](msg)`            | 403         | Insufficient permissions              |
| `glib.NotFound[T](msg)`             | 404         | Resource not found                    |
| `glib.MethodNotAllowed[T](msg)`     | 405         | HTTP method not allowed               |
| `glib.NotAcceptable[T](msg)`        | 406         | Cannot produce requested content-type |
| `glib.RequestTimeout[T](msg)`       | 408         | Request took too long                 |
| `glib.Conflict[T](msg)`             | 409         | Resource conflict (e.g., duplicate)   |
| `glib.Gone[T](msg)`                 | 410         | Resource permanently deleted          |
| `glib.PreconditionFailed[T](msg)`   | 412         | Precondition not met                  |
| `glib.PayloadTooLarge[T](msg)`      | 413         | Request body too large                |
| `glib.UnsupportedMediaType[T](msg)` | 415         | Content-type not supported            |
| `glib.UnprocessableEntity[T](msg)`  | 422         | Semantic validation failed            |
| `glib.Locked[T](msg)`               | 423         | Resource is locked                    |
| `glib.TooManyRequests[T](msg)`      | 429         | Rate limit exceeded                   |

### Server Error Responses

| Function                          | Status Code | Use Case                 |
| --------------------------------- | ----------- | ------------------------ |
| `glib.InternalError[T](msg)`      | 500         | Unexpected server error  |
| `glib.NotImplemented[T](msg)`     | 501         | Feature not implemented  |
| `glib.BadGateway[T](msg)`         | 502         | Upstream server error    |
| `glib.ServiceUnavailable[T](msg)` | 503         | Service temporarily down |
| `glib.GatewayTimeout[T](msg)`     | 504         | Upstream timeout         |

### Custom Responses

| Function                         | Description                           |
| -------------------------------- | ------------------------------------- |
| `glib.WithStatus[T](data, code)` | Custom status with data               |
| `glib.WithError[T](err, code)`   | Custom status with error              |
| `glib.Fail[T](err)`              | Auto-extract status from `errs.Error` |

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

### Result[T] Pattern: Generated Wrapper

**User writes:**

```go
// @Route method=POST path=/
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
func handlePostsControllerCreate(container *container) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Parse request body
        var req CreatePostRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            glib.BadRequest[*Post](
                fmt.Sprintf("invalid request body: %v", err)).Write(w)
            return
        }

        // Call handler
        result := container.controllers.postsController.Create(ctx, req)

        // Write result using Result.Write() method
        result.Write(w)
    }))

    // Apply middleware (if any)
    // handler = container.middleware.someMiddleware(handler)

    return handler.ServeHTTP
}
```

### Raw HTTP Pattern: Direct Registration

**User writes:**

```go
// @Route method=GET path=/stream with=none
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
    // SSE streaming implementation
    w.Header().Set("Content-Type", "text/event-stream")
    // ...
}
```

**Generator creates (no middleware):**

```go
func handleControllerStream(container *container) http.HandlerFunc {
    return container.controllers.controller.Stream
}
```

**Generator creates (with middleware):**

```go
func handleControllerStream(container *container) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(container.controllers.controller.Stream))

    handler = container.middleware.ratelimitMiddleware(handler)
    handler = container.middleware.loggerMiddleware(handler)

    return handler.ServeHTTP
}
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

### 6. Response Metadata Tags

```go
// ✅ Good - Use struct tags for headers ((T, error) pattern)
type PostResponse struct {
    Post         *Post  `json:"post"`
    ETag         string `header:"ETag"`
    CacheControl string `header:"Cache-Control"`
}

func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*PostResponse, error) {
    post, _ := c.Service.GetByID(id)
    return &PostResponse{
        Post:         post,
        ETag:         fmt.Sprintf(`"%s"`, post.Version),
        CacheControl: "max-age=3600",
    }, nil
}

// ✅ Good - Use WithHeader for dynamic headers (Result[T] pattern)
return glib.OK(post).
    WithHeader("ETag", fmt.Sprintf(`"%s"`, post.Version)).
    WithHeader("Cache-Control", "max-age=3600")

// ✅ Good - Use omitempty for conditional headers
type Response struct {
    Data  *Post  `json:"data"`
    ETag  string `header:"ETag,omitempty"`  // Only set if non-empty
}

// ❌ Bad - Don't mix both patterns unnecessarily
type BadResponse struct {
    Data *Post `json:"data"`
    ETag string `header:"ETag"`  // Using tags...
}
return glib.OK(resp).WithHeader("Cache-Control", "max-age=3600")  // ...and WithHeader
// Choose one pattern per response
```

### 7. Custom Status Codes

```go
// ✅ Good - Use response:"httpstatus" tag ((T, error) pattern)
type CreateResponse struct {
    ID     string `json:"id"`
    Status int    `response:"httpstatus"`
}
return &CreateResponse{ID: "123", Status: 201}, nil

// ✅ Good - Use Result[T] helpers (Result[T] pattern)
return glib.Created(data)

// ❌ Bad - Wrong type for httpstatus tag
type BadResponse struct {
    Status int64 `response:"httpstatus"`  // Must be int, not int64
}

// ❌ Bad - Multiple httpstatus fields
type BadResponse struct {
    Status1 int `response:"httpstatus"`
    Status2 int `response:"httpstatus"`  // Only one allowed
}
```

---

## Summary

### Pattern Comparison

| Aspect              | (T, error) Pattern                  | Raw HTTP Pattern                    |
| ------------------- | ----------------------------------- | ----------------------------------- |
| **Use Cases**       | ~95% of endpoints (recommended)     | ~5% of endpoints                    |
| **Type Safety**     | ✅ Full                             | ⚠️ Manual                           |
| **Status Control**  | Struct tags + error codes           | Manual via WriteHeader              |
| **Header Control**  | `header:"Name"` struct tags         | Manual via `w.Header().Set()`       |
| **Error Handling**  | Standard Go `error`                 | Manual                              |
| **Code Generation** | Wrapper with parsing/serialization  | Direct method reference (optimized) |
| **Performance**     | ⚡ Excellent (cached reflection)    | ⚡ Fastest                          |
| **Learning Curve**  | Easy (idiomatic Go)                 | Easy (standard Go)                  |
| **When to Use**     | Default choice for REST APIs        | Streaming, SSE, WebSockets, files   |

### Quick Reference

**(T, error) Pattern Signature:**

```go
func(ctx context.Context, [params...]) (T, error)

// Response struct with metadata tags
type Response struct {
    Data   T      `json:"data"`
    Status int    `response:"httpstatus"`  // Optional: custom status
    Header string `header:"X-Custom"`      // Optional: custom headers
}
```

**Raw HTTP Pattern Signature:**

```go
func(w http.ResponseWriter, r *http.Request)
```

### Pattern Decision Tree

```
Start
  │
  ├─ Need streaming/SSE/WebSocket?
  │    └─ YES → Use Raw HTTP Pattern
  │
  └─ Standard REST API?
       └─ YES → Use (T, error) Pattern (RECOMMENDED DEFAULT)
```

**Rule of Thumb:** Start with `(T, error)` pattern for new projects. Only use Raw HTTP for streaming/files/custom protocols.

---

**Next:** `03-CODE-GENERATION.md` - Code generation internals and templates
