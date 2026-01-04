# 04. Error Handling

**Status:** Specification (Under Development)  
**Last Updated:** 2025-12-31

---

## Table of Contents

1. [Overview](#overview)
2. [Result[T] Error Handling](#resultt-error-handling)
3. [Error Types](#error-types)
4. [HTTP Status Codes](#http-status-codes)
5. [Error Builder API](#error-builder-api)
6. [ValidationErrors](#validationerrors)
7. [Generated Error Handling](#generated-error-handling)
8. [Custom Error Types](#custom-error-types)
9. [Best Practices](#best-practices)

---

## Overview

Glib uses **Encore.dev-style error handling** with structured errors that automatically map to HTTP status codes.

### Design Principles

1. **Type-safe errors** - Compile-time checking
2. **HTTP code mapping** - Errors know their status codes
3. **Structured metadata** - Errors can carry context
4. **User-friendly messages** - Separate internal/external messages
5. **Stack traces** - Debug information in development

### Error Flow (Result[T] Pattern)

```
Handler Returns glib.Result[T]
    ↓
Result.Write(w) Method Called
    ↓
Check Result.Error()
    ↓
If Error: Extract HTTP Status & Write JSON
    ↓
If Success: Write Data as JSON
    ↓
Send to Client
```

---

## Result[T] Error Handling

Glib uses the `Result[T]` type for handlers, which provides explicit error control:

### Using Result[T] with Errors

```go
import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
)

// @Route GET /posts/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    if id == uuid.Nil {
        return glib.BadRequest[*Post]("invalid id")
    }
    
    post, err := c.Service.GetByID(id)
    if err != nil {
        return glib.Fail[*Post](err)  // Auto-extracts status from errs.Error
    }
    
    return glib.OK(post)
}
```

### Result[T] Error Helpers

Quick error responses without building `errs.Error`:

```go
// Common errors
glib.BadRequest[T](msg)      // 400
glib.Unauthorized[T](msg)    // 401
glib.Forbidden[T](msg)       // 403
glib.NotFound[T](msg)        // 404
glib.Conflict[T](msg)        // 409
glib.InternalError[T](msg)   // 500

// Auto-mapping from errs.Error
glib.Fail[T](err)            // Extracts status from errs.Error
```

---

## Error Types

### Using Result[T] Helpers (Recommended)

```go
// @Route GET /posts/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return glib.NotFound[*Post]("post not found")
    }
    if err != nil {
        return glib.InternalError[*Post]("database error")
    }
    return glib.OK(post)
}
```

### Using Structured Errors with glib.Fail()

```go
// @Route GET /posts/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if errors.Is(err, gorm.ErrRecordNotFound) {
        structuredErr := errs.B().
            Code(errs.NotFound).
            Msg("post not found").
            Meta("id", id.String()).
            Err()
        return glib.Fail[*Post](structuredErr)  // Auto-maps to 404
    }
    if err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.OK(post)
}
```

**Benefits:**
- Explicit HTTP status codes via `glib.Fail()`
- Structured metadata and details
- Better error messages
- Automatic JSON formatting

---

## HTTP Status Codes

### Error Code Mapping

Glib provides predefined error codes that map to HTTP status codes:

```go
package errs

type Code int

const (
    // 4xx Client Errors
    InvalidArgument  Code = 400  // Bad Request
    Unauthenticated  Code = 401  // Unauthorized
    PermissionDenied Code = 403  // Forbidden
    NotFound         Code = 404  // Not Found
    AlreadyExists    Code = 409  // Conflict
    
    // 5xx Server Errors
    Internal         Code = 500  // Internal Server Error
    Unavailable      Code = 503  // Service Unavailable
)
```

### Complete Mapping Table

| Error Code | HTTP Status | Use Case |
|------------|-------------|----------|
| `InvalidArgument` | 400 | Invalid input, validation failed |
| `Unauthenticated` | 401 | Missing or invalid authentication |
| `PermissionDenied` | 403 | Authenticated but not authorized |
| `NotFound` | 404 | Resource doesn't exist |
| `AlreadyExists` | 409 | Resource conflict (duplicate) |
| `Internal` | 500 | Unexpected server error |
| `Unavailable` | 503 | Service temporarily unavailable |

---

## Error Builder API

### Basic Error

```go
err := errs.B().
    Code(errs.NotFound).
    Msg("post not found").
    Err()
```

### Error with Metadata

```go
err := errs.B().
    Code(errs.InvalidArgument).
    Msg("validation failed").
    Meta("field", "email").
    Meta("reason", "invalid format").
    Err()
```

**Generated JSON:**

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "validation failed",
    "meta": {
      "field": "email",
      "reason": "invalid format"
    }
  }
}
```

### Error with Details

```go
err := errs.B().
    Code(errs.InvalidArgument).
    Msg("validation failed").
    Meta("user_id", userID).
    Err()

return glib.Fail[*Post](err)
```

**Generated JSON:**

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "validation failed"
  }
}
```

Note: Meta fields are for internal logging only, not exposed in API responses.

### Wrapping Errors

```go
err := errs.B().
    Code(errs.Internal).
    Msg("failed to create post").
    Cause(dbErr).  // Wrap underlying error
    Err()
```

**Benefits:**
- Preserves error chain for `errors.Is()` / `errors.As()`
- Internal error details hidden from client
- Full stack trace in logs

---

## ValidationErrors

Glib provides built-in support for structured validation errors with field-level details.

### ValidationError Type

```go
// pkg/errs/details.go

type ValidationError struct {
    Field    string   `json:"field"`
    Messages []string `json:"messages"`
}

type ValidationErrors struct {
    Errors []ValidationError `json:"errors"`
}
```

### Creating ValidationErrors

```go
import "github.com/azizndao/glib/errs"

// @Route POST /posts
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    // Manual validation
    var validationErrs []errs.ValidationError
    
    if len(req.Title) < 3 {
        validationErrs = append(validationErrs, errs.ValidationError{
            Field:    "title",
            Messages: []string{"must be at least 3 characters"},
        })
    }
    
    if req.Content == "" {
        validationErrs = append(validationErrs, errs.ValidationError{
            Field:    "content",
            Messages: []string{"field is required"},
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

### Generated Response Format

**Request:**
```bash
POST /api/v1/posts
{
  "title": "ab",
  "content": ""
}
```

**Response (400):**
```json
{
  "error": {
    "code": "invalid_argument",
    "message": "Validation failed",
    "details": [
      {
        "field": "title",
        "messages": ["must be at least 3 characters"]
      },
      {
        "field": "content",
        "messages": ["field is required"]
      }
    ]
  }
}
```

### ValidationErrors Builder API

```go
// Create empty ValidationErrors
validationErrs := &errs.ValidationErrors{}

// Add errors one by one
validationErrs.AddError("email", "must be a valid email")
validationErrs.AddError("email", "field is required")
validationErrs.AddError("password", "must be at least 8 characters")

// Create from slice
validationErrs := errs.NewValidationErrors([]errs.ValidationError{
    {Field: "email", Messages: []string{"invalid format", "required"}},
    {Field: "password", Messages: []string{"too short"}},
})

// Use in error
err := errs.B().
    Code(errs.InvalidArgument).
    Msg("Validation failed").
    Details(validationErrs).
    Err()

return glib.Fail[*Post](err)
```

### Error Detail Conversion

The framework's `Result[T].Write()` method converts `errs.ErrDetails` to the JSON response format:

```go
// pkg/errs/details.go

// ValidationErrors contains field-level validation errors
type ValidationErrors struct {
    Errors []ValidationError
}

type ValidationError struct {
    Field    string
    Messages []string
}

// Convert to JSON format
func (v *ValidationErrors) ToJSON() map[string][]string {
    result := make(map[string][]string)
    for _, err := range v.Errors {
        result[err.Field] = err.Messages
    }
    return result
}
```

---

## Generated Error Handling

Error handling is now built into the `Result[T]` type via the `Write(w http.ResponseWriter)` method.

### Result[T].Write() Method

Located in `writer.go`:

```go
// Write writes the Result to an http.ResponseWriter
func (r Result[T]) Write(w http.ResponseWriter) {
    // Set custom headers
    for key, values := range r.Headers {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    // Handle error response
    if r.err != nil {
        writeErrorJSON(w, r.StatusCode, r.err)
        return
    }
    
    // Handle no-content response
    if r.StatusCode == http.StatusNoContent {
        w.WriteHeader(r.StatusCode)
        return
    }
    
    // Write success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(r.StatusCode)
    if err := json.NewEncoder(w).Encode(r.Data); err != nil {
        log.Printf("failed to encode response: %v", err)
    }
}
```

### Handler Wrapper Integration

**Result[T] pattern wrapper:**

```go
func handlePostsControllerCreate(container *container) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Parse request
        var req CreatePostRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            glib.BadRequest[*Post]("invalid JSON").Write(w)
            return
        }
        
        // Call handler
        result := container.controllers.postsController.Create(ctx, req)
        
        // Write result using Result.Write() method
        result.Write(w)
    }))
    
    // Apply middleware if needed
    // handler = container.middleware.authMiddleware(handler)
    
    return handler.ServeHTTP
}
```

---

## Custom Error Types

### Domain-Specific Errors

```go
// posts/errors.go
package posts

import (
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
)

// Predefined errors
func ErrPostNotFound(id string) glib.Result[*Post] {
    err := errs.B().
        Code(errs.NotFound).
        Msg("post not found").
        Meta("id", id).
        Err()
    return glib.Fail[*Post](err)
}

func ErrPostDeleted() glib.Result[*Post] {
    err := errs.B().
        Code(errs.NotFound).
        Msg("post has been deleted").
        Err()
    return glib.Fail[*Post](err)
}

func ErrUnauthorized() glib.Result[*Post] {
    return glib.Forbidden[*Post]("you don't have permission to modify this post")
}
```

**Usage:**

```go
// @Route GET /posts/{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    post, err := c.Service.GetByID(id)
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return ErrPostNotFound(id.String())  // Returns glib.Result[*Post]
    }
    
    if post.DeletedAt != nil {
        return ErrPostDeleted()
    }
    
    return glib.OK(post)
}

// @Route DELETE /posts/{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    user := auth.GetUser(ctx)
    post, _ := c.Service.GetByID(id)
    
    if post.AuthorID != user.ID {
        return ErrUnauthorized()  // Type mismatch - won't compile!
    }
    
    // Fix: Use correct type
    if post.AuthorID != user.ID {
        return glib.Forbidden[any]("you don't have permission to modify this post")
    }
    
    if err := c.Service.Delete(id); err != nil {
        return glib.Fail[any](err)
    }
    return glib.NoContent[any]()
}
```

### Error Factory Functions

```go
func ValidationFailed[T any](errors []errs.ValidationError) glib.Result[T] {
    err := errs.B().
        Code(errs.InvalidArgument).
        Msg("Validation failed").
        Details(errs.NewValidationErrors(errors)).
        Err()
    return glib.Fail[T](err)
}

func DatabaseError[T any](operation string, cause error) glib.Result[T] {
    err := errs.B().
        Code(errs.Internal).
        Msg("database operation failed").
        Meta("operation", operation).
        Cause(cause).
        Err()
    return glib.Fail[T](err)
}

// Usage
if err := c.DB.Create(post).Error; err != nil {
    return DatabaseError[*Post]("create", err)
}
```

---

## Best Practices

### 1. Use Result[T] Helpers for Simple Cases

```go
// ✅ Good - Clear and concise
if post == nil {
    return glib.NotFound[*Post]("post not found")
}

// ❌ Bad - Unnecessarily verbose
if post == nil {
    err := errs.B().Code(errs.NotFound).Msg("post not found").Err()
    return glib.Fail[*Post](err)
}
```

### 2. Use glib.Fail() for Complex Errors

```go
// ✅ Good - Rich error with metadata
if err := c.DB.Create(post).Error; err != nil {
    return glib.Fail[*Post](
        errs.B().
            Code(errs.Internal).
            Msg("failed to create post").
            Meta("operation", "database.create").
            Cause(err).
            Err(),
    )
}
```

### 3. Use ValidationErrors for Field-Level Errors

```go
// ✅ Good - Structured validation errors
validationErrs := errs.NewValidationErrors([]errs.ValidationError{
    {Field: "email", Messages: []string{"invalid format", "required"}},
    {Field: "age", Messages: []string{"must be at least 18"}},
})

err := errs.B().
    Code(errs.InvalidArgument).
    Msg("Validation failed").
    Details(validationErrs).
    Err()

return glib.Fail[*User](err)

// ❌ Bad - Generic error for validation
if req.Email == "" {
    return glib.BadRequest[*User]("email is required")  // No field-level detail
}
```

### 4. Don't Expose Internal Details

```go
// ❌ Bad - Exposes database schema
return glib.InternalError[*Post]("column 'email' violates unique constraint")

// ✅ Good - User-friendly message
if isDuplicateKeyError(err) {
    return glib.Conflict[*Post]("an account with this email already exists")
}
```

### 5. Use Meta for Logging, Not Client Response

```go
// ✅ Good - Meta for internal debugging
err := errs.B().
    Code(errs.Internal).
    Msg("database connection failed").
    Meta("host", dbHost).
    Meta("attempt", retryCount).
    Cause(dbErr).
    Err()

// Meta fields are NOT sent to client, only logged
return glib.Fail[*Post](err)
```

### 6. Wrap Database Errors

```go
// ❌ Bad - Leaks internal details
if err := c.DB.Create(post).Error; err != nil {
    return glib.Fail[*Post](err)  // Exposes GORM error message
}

// ✅ Good - Wraps with user-friendly message
if err := c.DB.Create(post).Error; err != nil {
    return glib.Fail[*Post](
        errs.B().
            Code(errs.Internal).
            Msg("failed to create post").
            Cause(err).  // Original error preserved for logging
            Err(),
    )
}
```

### 7. Define Domain Errors as Functions (Not Variables)

```go
// ✅ Good - Functions prevent mutation and allow parameters
func ErrPostNotFound(id string) glib.Result[*Post] {
    err := errs.B().
        Code(errs.NotFound).
        Msg("post not found").
        Meta("id", id).
        Err()
    return glib.Fail[*Post](err)
}

// ⚠️ OK but limited - No parameters
var ErrUnauthorized = glib.Forbidden[*Post]("unauthorized")

// ❌ Bad - Shared error instance (can cause issues)
var ErrPostNotFound = errs.B().Code(errs.NotFound).Msg("post not found").Err()
```

### 8. Match Result[T] Type Parameter

```go
// ✅ Good - Type matches function signature
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*Post] {
    return glib.NotFound[*Post]("post not found")  // Correct type
}

// ❌ Bad - Type mismatch won't compile
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*Post] {
    return glib.NotFound[*User]("post not found")  // Wrong type!
}
```

---

## Complete Example

```go
// posts/controller.go
package posts

import (
    "context"
    "errors"
    
    "github.com/google/uuid"
    "github.com/azizndao/glib"
    "github.com/azizndao/glib/errs"
    "gorm.io/gorm"
)

// @Controller /api/v1/posts
type PostsController struct {
    DB *gorm.DB
}

// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    if id == uuid.Nil {
        return glib.BadRequest[*Post]("id cannot be empty")
    }
    
    var post Post
    err := c.DB.Where("id = ?", id).First(&post).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return glib.NotFound[*Post]("post not found")
    }
    if err != nil {
        return glib.Fail[*Post](
            errs.B().
                Code(errs.Internal).
                Msg("failed to fetch post").
                Cause(err).
                Err(),
        )
    }
    
    return glib.OK(&post)
}

// @Route POST /
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    // Validate request
    var validationErrs []errs.ValidationError
    
    if len(req.Title) < 3 {
        validationErrs = append(validationErrs, errs.ValidationError{
            Field:    "title",
            Messages: []string{"must be at least 3 characters"},
        })
    }
    
    if req.Content == "" {
        validationErrs = append(validationErrs, errs.ValidationError{
            Field:    "content",
            Messages: []string{"field is required"},
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
    
    // Create post
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    
    err := c.DB.Create(post).Error
    if err != nil {
        return glib.Fail[*Post](
            errs.B().
                Code(errs.Internal).
                Msg("failed to create post").
                Cause(err).
                Err(),
        )
    }
    
    return glib.Created(post)
}

// @Route DELETE /{id}
func (c *PostsController) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
    result := c.DB.Delete(&Post{}, "id = ?", id)
    if result.Error != nil {
        return glib.Fail[any](
            errs.B().
                Code(errs.Internal).
                Msg("failed to delete post").
                Cause(result.Error).
                Err(),
        )
    }
    
    if result.RowsAffected == 0 {
        return glib.NotFound[any]("post not found")
    }
    
    return glib.NoContent[any]()
}
```

**Generated Error Responses:**

```bash
# Invalid ID
GET /api/v1/posts/00000000-0000-0000-0000-000000000000
HTTP/1.1 400 Bad Request
{
  "error": {
    "code": "invalid_argument",
    "message": "id cannot be empty"
  }
}

# Not Found
GET /api/v1/posts/123e4567-e89b-12d3-a456-426614174000
HTTP/1.1 404 Not Found
{
  "error": {
    "code": "not_found",
    "message": "post not found"
  }
}

# Validation Failed
POST /api/v1/posts
{"title": "ab", "content": ""}
HTTP/1.1 400 Bad Request
{
  "error": {
    "code": "invalid_argument",
    "message": "Validation failed",
    "details": [
      {
        "field": "title",
        "messages": ["must be at least 3 characters"]
      },
      {
        "field": "content",
        "messages": ["field is required"]
      }
    ]
  }
}
```

---

## Summary

### Key Features

1. **Result[T] Integration** - Error handling built into Result[T] type
2. **Helper Functions** - Quick error responses (BadRequest, NotFound, etc.)
3. **glib.Fail()** - Auto-extract HTTP status from errs.Error
4. **Structured Errors** - Type-safe errors with HTTP codes via errs.Builder
5. **ValidationErrors** - Field-level validation error details
6. **convertDetails()** - Generated conversion from errs.ErrDetails to JSON
7. **Type Safety** - Generic Result[T] ensures consistency

### Error Response Format

```json
{
  "error": {
    "code": "error_code",
    "message": "User-facing message",
    "details": [
      {
        "field": "field_name",
        "messages": ["error message 1", "error message 2"]
      }
    ]
  }
}
```

### Quick Reference

**Result[T] Error Helpers:**
```go
glib.BadRequest[T](msg)      // 400
glib.Unauthorized[T](msg)    // 401
glib.Forbidden[T](msg)       // 403
glib.NotFound[T](msg)        // 404
glib.Conflict[T](msg)        // 409
glib.InternalError[T](msg)   // 500
glib.Fail[T](err)            // Auto-map from errs.Error
```

**Structured Errors:**
```go
errs.B().
    Code(errs.NotFound).
    Msg("resource not found").
    Meta("id", id).              // Internal logging only
    Details(validationErrs).     // Client-facing details
    Cause(originalErr).          // Wrapped error
    Err()
```

**ValidationErrors:**
```go
validationErrs := errs.NewValidationErrors([]errs.ValidationError{
    {Field: "email", Messages: []string{"invalid format"}},
})

err := errs.B().
    Code(errs.InvalidArgument).
    Details(validationErrs).
    Err()

return glib.Fail[T](err)
```

---

**Next:** `05-CLI.md` - CLI commands and configuration
