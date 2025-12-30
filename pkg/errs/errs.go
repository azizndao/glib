// Package errs provides structured error handling with HTTP status code mapping.
// Inspired by Encore.dev's error handling approach.
package errs

import (
	"errors"
	"fmt"
)

// Code represents an error code that maps to HTTP status codes.
type Code int

// Error codes mapping to HTTP status codes.
const (
	InvalidArgument  Code = 400 // Bad Request
	Unauthenticated  Code = 401 // Unauthorized
	PermissionDenied Code = 403 // Forbidden
	NotFound         Code = 404 // Not Found
	AlreadyExists    Code = 409 // Conflict
	Internal         Code = 500 // Internal Server Error
	Unavailable      Code = 503 // Service Unavailable
)

// HTTPStatus returns the HTTP status code for this error code.
func (c Code) HTTPStatus() int {
	return int(c)
}

// String returns the string representation of the error code.
func (c Code) String() string {
	switch c {
	case InvalidArgument:
		return "invalid_argument"
	case Unauthenticated:
		return "unauthenticated"
	case PermissionDenied:
		return "permission_denied"
	case NotFound:
		return "not_found"
	case AlreadyExists:
		return "already_exists"
	case Internal:
		return "internal"
	case Unavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

// Error represents a structured error with code, message, and metadata.
type Error struct {
	code    Code
	message string
	meta    map[string]any
	details map[string][]string
	cause   error
}

// Code returns the error code.
func (e *Error) Code() Code {
	return e.code
}

// Message returns the error message.
func (e *Error) Message() string {
	return e.message
}

// Meta returns the error metadata.
func (e *Error) Meta() map[string]any {
	return e.meta
}

// Details returns the error details (e.g., validation errors).
func (e *Error) Details() map[string][]string {
	return e.details
}

// Unwrap returns the underlying cause error.
func (e *Error) Unwrap() error {
	return e.cause
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Builder provides a fluent API for building structured errors.
type Builder struct {
	err *Error
}

// B creates a new error builder.
func B() *Builder {
	return &Builder{
		err: &Error{
			code: Internal, // Default to internal error
		},
	}
}

// Code sets the error code.
func (b *Builder) Code(code Code) *Builder {
	b.err.code = code
	return b
}

// Msg sets the error message.
func (b *Builder) Msg(msg string) *Builder {
	b.err.message = msg
	return b
}

// Msgf sets the error message with formatting.
func (b *Builder) Msgf(format string, args ...any) *Builder {
	b.err.message = fmt.Sprintf(format, args...)
	return b
}

// Meta adds a metadata key-value pair.
func (b *Builder) Meta(key string, value any) *Builder {
	if b.err.meta == nil {
		b.err.meta = make(map[string]any)
	}
	b.err.meta[key] = value
	return b
}

// Details sets validation error details.
func (b *Builder) Details(details map[string][]string) *Builder {
	b.err.details = details
	return b
}

// Cause sets the underlying cause error.
func (b *Builder) Cause(err error) *Builder {
	b.err.cause = err
	return b
}

// Err returns the built error.
func (b *Builder) Err() error {
	return b.err
}

// Is checks if the target error matches this error.
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return e.code == t.code
}

// Common error constructors for convenience.

// New creates a simple error with a message (defaults to Internal).
func New(msg string) error {
	return B().Msg(msg).Err()
}

// Newf creates a simple error with formatted message (defaults to Internal).
func Newf(format string, args ...any) error {
	return B().Msgf(format, args...).Err()
}

// Wrap wraps an error with a message (defaults to Internal).
func Wrap(err error, msg string) error {
	return B().Msg(msg).Cause(err).Err()
}

// Wrapf wraps an error with a formatted message (defaults to Internal).
func Wrapf(err error, format string, args ...any) error {
	return B().Msgf(format, args...).Cause(err).Err()
}
