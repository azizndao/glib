# 04. Error Handling System

Specification and reference for structured error handling in Glib.

---

## Table of Contents

1. [Overview](#overview)
2. [Error Codes & HTTP Status Mapping](#error-codes--http-status-mapping)
3. [Error Construction & Helpers](#error-construction--helpers)
4. [Error Wrapping & Stack Preservation](#error-wrapping--stack-preservation)
5. [Validation Errors (Goyave Style)](#validation-errors-goyave-style)
6. [JSON Error Response Format](#json-error-response-format)
7. [Best Practices](#best-practices)

---

## Overview

Glib provides a robust, structured error model in package `github.com/azizndao/glib/errs` inspired by Encore.dev.

### Design Principles

1. **Explicit Error Codes**: Handlers attach domain error codes (`NotFound`, `InvalidArgument`, `PermissionDenied`, etc.) at the source.
2. **Automatic HTTP Mapping**: Glib maps error codes directly to HTTP status codes without manual status setting.
3. **Structured Details**: Errors can attach structured validation errors, metadata, and underlying causes.
4. **Safe Serialization**: Sensitive internal error details are not exposed to clients unless explicitly marked as user details.

---

## Error Codes & HTTP Status Mapping

The `errs.ErrCode` enum defines canonical error states:

| Error Code                | HTTP Status Code            | Description / Usage                            |
| :------------------------ | :-------------------------- | :--------------------------------------------- |
| `errs.InvalidArgument`    | `400 Bad Request`           | Client specified invalid argument or payload   |
| `errs.FailedPrecondition` | `400 Bad Request`           | System not in state required for execution     |
| `errs.OutOfRange`         | `400 Bad Request`           | Parameter or operation out of valid range      |
| `errs.Unauthenticated`    | `401 Unauthorized`          | Request lacks valid authentication credentials |
| `errs.PermissionDenied`   | `403 Forbidden`             | Caller lacks permission for this operation     |
| `errs.NotFound`           | `404 Not Found`             | Requested entity does not exist                |
| `errs.AlreadyExists`      | `409 Conflict`              | Attempted creation of existing entity          |
| `errs.Aborted`            | `409 Conflict`              | Operation aborted due to concurrency conflict  |
| `errs.ResourceExhausted`  | `429 Too Many Requests`     | Rate limit or quota exhausted                  |
| `errs.Canceled`           | `499 Client Closed`         | Request canceled by caller                     |
| `errs.Internal`           | `500 Internal Server Error` | Unexpected internal failure                    |
| `errs.Unknown`            | `500 Internal Server Error` | Unspecified or unclassified error              |
| `errs.DataLoss`           | `500 Internal Server Error` | Unrecoverable data corruption                  |
| `errs.Unimplemented`      | `501 Not Implemented`       | Operation not implemented                      |
| `errs.Unavailable`        | `503 Service Unavailable`   | Service temporarily unavailable                |
| `errs.DeadlineExceeded`   | `504 Gateway Timeout`       | Timeout expired before completion              |

---

## Error Construction & Helpers

### 1. Shorthand Constructor Helpers

For quick errors with a message:

```go
import "github.com/azizndao/glib/errs"

err := errs.NewNotFound().WithMessage("post not found")
err := errs.NewBadRequest().WithMessage("invalid parameters")
err := errs.NewUnauthorized().WithMessage("token expired")
err := errs.NewForbidden().WithMessage("insufficient privileges")
err := errs.NewConflict().WithMessage("email already taken")
err := errs.NewInternal().WithMessage("database connection dropped")
```

### 2. Fluent Builder Pattern (`errs.B()`)

For full control over code, message, cause, and details:

```go
err := errs.B().
    Code(errs.PermissionDenied).
    Msg("user does not have permission to delete this post").
    Cause(underlyingErr).
    Err()
```

---

## Error Wrapping & Stack Preservation

Convert standard Go errors into structured `*errs.Error` objects:

```go
// Wrap with Unknown code by default
if err := db.Save(user).Error; err != nil {
    return nil, errs.Wrap(err, "failed to save user")
}

// Wrap with specific code
if err := db.First(user, id).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errs.WrapCode(err, errs.NotFound, "user not found")
    }
    return nil, errs.WrapCode(err, errs.Internal, "database query error")
}
```

---

## Validation Errors (Goyave Style)

When request validation fails (either automatically via `validate:` tags or manually), Glib outputs Goyave-style structured field validation errors.

### Programmatic Validation Errors

```go
import "github.com/azizndao/glib/validator"

valErrs := validator.NewValidationErrors()
valErrs.AddBodyError([]string{"email"}, "email format is invalid")
valErrs.AddQueryError([]string{"per_page"}, "must be between 1 and 100")

err := errs.B().
    Code(errs.InvalidArgument).
    Msg("validation failed").
    Details(valErrs).
    Err()
```

---

## JSON Error Response Format

All errors serialized by Glib conform to a consistent top-level JSON schema:

```json
{
    "error": {
        "code": "not_found",
        "message": "post not found"
    }
}
```

With validation details:

```json
{
    "error": {
        "code": "invalid_argument",
        "message": "Validation failed",
        "details": {
            "body": {
                "title": {
                    "errors": ["title must be at least 3 characters"]
                },
                "author_id": {
                    "errors": ["author_id is required"]
                }
            },
            "query": {
                "page": {
                    "errors": ["page must be at least 1"]
                }
            }
        }
    }
}
```

---

## Best Practices

1. **Return Errors Directly**: Return `(nil, err)` or `err` from your handlers; Glib's generated wrapper handles serialization.
2. **Use Structured Codes**: Always use appropriate `errs.ErrCode` values instead of generic `errors.New()`.
3. **Wrap Causes**: Attach underlying errors using `.Cause(err)` or `errs.Wrap()` to maintain debugging context in server logs without leaking details to clients.
