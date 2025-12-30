# errs - Structured Error Handling

Encore.dev-inspired structured error handling with HTTP status code mapping.

## Features

- **HTTP Code Mapping** - Errors automatically map to HTTP status codes
- **Structured Metadata** - Add context to errors with key-value pairs
- **Validation Details** - Structured validation error details
- **Error Wrapping** - Preserve error chains with `errors.Is()` and `errors.As()`
- **Fluent Builder API** - Readable error construction

## Usage

### Basic Error

```go
err := errs.B().
    Code(errs.NotFound).
    Msg("user not found").
    Err()
```

### With Metadata

```go
err := errs.B().
    Code(errs.InvalidArgument).
    Msg("invalid email").
    Meta("field", "email").
    Meta("value", "invalid@").
    Err()
```

### With Validation Details

```go
err := errs.B().
    Code(errs.InvalidArgument).
    Msg("validation failed").
    Details(map[string][]string{
        "email": {"must be valid email", "required"},
        "age": {"must be at least 18"},
    }).
    Err()
```

### Wrapping Errors

```go
dbErr := db.Query(...)
if dbErr != nil {
    return errs.B().
        Code(errs.Internal).
        Msg("failed to fetch user").
        Cause(dbErr).
        Err()
}
```

### Convenience Functions

```go
// Simple error (defaults to Internal)
err := errs.New("something went wrong")

// Formatted error
err := errs.Newf("invalid id: %d", id)

// Wrap with message
err := errs.Wrap(dbErr, "database operation failed")

// Wrap with format
err := errs.Wrapf(dbErr, "failed to fetch user %d", userID)
```

## Error Codes

| Code | HTTP Status | Use Case |
|------|-------------|----------|
| `InvalidArgument` | 400 | Invalid input, validation failed |
| `Unauthenticated` | 401 | Missing or invalid authentication |
| `PermissionDenied` | 403 | Authenticated but not authorized |
| `NotFound` | 404 | Resource doesn't exist |
| `AlreadyExists` | 409 | Resource conflict (duplicate) |
| `Internal` | 500 | Unexpected server error |
| `Unavailable` | 503 | Service temporarily unavailable |

## Checking Errors

```go
var glibErr *errs.Error
if errors.As(err, &glibErr) {
    fmt.Println("Code:", glibErr.Code().String())
    fmt.Println("HTTP Status:", glibErr.Code().HTTPStatus())
    fmt.Println("Message:", glibErr.Message())
    fmt.Println("Meta:", glibErr.Meta())
}

// Check error code
if errors.Is(err, errs.B().Code(errs.NotFound).Err()) {
    // Handle not found
}
```

## Integration with HTTP Handlers

```go
func handleError(w http.ResponseWriter, err error) {
    var glibErr *errs.Error
    if errors.As(err, &glibErr) {
        w.WriteHeader(glibErr.Code().HTTPStatus())
        json.NewEncoder(w).Encode(map[string]any{
            "error": map[string]any{
                "code":    glibErr.Code().String(),
                "message": glibErr.Message(),
                "meta":    glibErr.Meta(),
                "details": glibErr.Details(),
            },
        })
        return
    }
    
    // Fallback for non-Glib errors
    w.WriteHeader(http.StatusInternalServerError)
    json.NewEncoder(w).Encode(map[string]any{
        "error": map[string]any{
            "code":    "internal",
            "message": err.Error(),
        },
    })
}
```

## Best Practices

### ✅ DO

- Use specific error codes for client errors
- Add metadata for debugging context
- Wrap underlying errors with `.Cause()`
- Use validation details for field-level errors

### ❌ DON'T

- Don't expose internal details to clients
- Don't use generic error messages
- Don't lose the error chain
- Don't use `Internal` for validation errors

## License

MIT
