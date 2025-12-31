package glib

import (
	"errors"
	"net/http"

	"github.com/azizndao/glib/pkg/errs"
)

// Result represents a type-safe handler response with explicit control over
// HTTP status codes, headers, and response data.
type Result[T any] struct {
	// Data is the response payload (can be nil for no-content responses)
	Data T

	// err is an internal error (not exposed in JSON responses)
	err error

	// StatusCode is the HTTP status code (default: 200 for success, 500 for error)
	StatusCode int

	// Headers are custom HTTP headers to include in the response
	Headers http.Header
}

type Nothing Result[any]

// Success response helpers

// OK creates a 200 OK response with the given data
func OK[T any](data T) Result[T] {
	return Result[T]{
		Data:       data,
		StatusCode: http.StatusOK,
	}
}

// Created creates a 201 Created response with the given data
func Created[T any](data T) Result[T] {
	return Result[T]{
		Data:       data,
		StatusCode: http.StatusCreated,
	}
}

// Accepted creates a 202 Accepted response with the given data
func Accepted[T any](data T) Result[T] {
	return Result[T]{
		Data:       data,
		StatusCode: http.StatusAccepted,
	}
}

// NoContent creates a 204 No Content response
func NoContent[T any]() Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		StatusCode: http.StatusNoContent,
	}
}

// Redirection response helpers

// MovedPermanently creates a 301 Moved Permanently response
func MovedPermanently[T any](location string) Result[T] {
	var zero T
	result := Result[T]{
		Data:       zero,
		StatusCode: http.StatusMovedPermanently,
	}
	return result.WithHeader("Location", location)
}

// Found creates a 302 Found response
func Found[T any](location string) Result[T] {
	var zero T
	result := Result[T]{
		Data:       zero,
		StatusCode: http.StatusFound,
	}
	return result.WithHeader("Location", location)
}

// SeeOther creates a 303 See Other response
func SeeOther[T any](location string) Result[T] {
	var zero T
	result := Result[T]{
		Data:       zero,
		StatusCode: http.StatusSeeOther,
	}
	return result.WithHeader("Location", location)
}

// NotModified creates a 304 Not Modified response
func NotModified[T any]() Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		StatusCode: http.StatusNotModified,
	}
}

// TemporaryRedirect creates a 307 Temporary Redirect response
func TemporaryRedirect[T any](location string) Result[T] {
	var zero T
	result := Result[T]{
		Data:       zero,
		StatusCode: http.StatusTemporaryRedirect,
	}
	return result.WithHeader("Location", location)
}

// PermanentRedirect creates a 308 Permanent Redirect response
func PermanentRedirect[T any](location string) Result[T] {
	var zero T
	result := Result[T]{
		Data:       zero,
		StatusCode: http.StatusPermanentRedirect,
	}
	return result.WithHeader("Location", location)
}

// Error response helpers

// BadRequest creates a 400 Bad Request error response
func BadRequest[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.InvalidArgument).Msg(message).Err(),
		StatusCode: http.StatusBadRequest,
	}
}

// Unauthorized creates a 401 Unauthorized error response
func Unauthorized[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.Unauthenticated).Msg(message).Err(),
		StatusCode: http.StatusUnauthorized,
	}
}

// Forbidden creates a 403 Forbidden error response
func Forbidden[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.PermissionDenied).Msg(message).Err(),
		StatusCode: http.StatusForbidden,
	}
}

// NotFound creates a 404 Not Found error response
func NotFound[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.NotFound).Msg(message).Err(),
		StatusCode: http.StatusNotFound,
	}
}

// MethodNotAllowed creates a 405 Method Not Allowed error response
func MethodNotAllowed[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.Unimplemented).Msg(message).Err(),
		StatusCode: http.StatusMethodNotAllowed,
	}
}

// NotAcceptable creates a 406 Not Acceptable error response
func NotAcceptable[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.InvalidArgument).Msg(message).Err(),
		StatusCode: http.StatusNotAcceptable,
	}
}

// RequestTimeout creates a 408 Request Timeout error response
func RequestTimeout[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.DeadlineExceeded).Msg(message).Err(),
		StatusCode: http.StatusRequestTimeout,
	}
}

// Conflict creates a 409 Conflict error response
func Conflict[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.AlreadyExists).Msg(message).Err(),
		StatusCode: http.StatusConflict,
	}
}

// Gone creates a 410 Gone error response
func Gone[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.NotFound).Msg(message).Err(),
		StatusCode: http.StatusGone,
	}
}

// PreconditionFailed creates a 412 Precondition Failed error response
func PreconditionFailed[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.FailedPrecondition).Msg(message).Err(),
		StatusCode: http.StatusPreconditionFailed,
	}
}

// PayloadTooLarge creates a 413 Payload Too Large error response
func PayloadTooLarge[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.InvalidArgument).Msg(message).Err(),
		StatusCode: http.StatusRequestEntityTooLarge,
	}
}

// UnsupportedMediaType creates a 415 Unsupported Media Type error response
func UnsupportedMediaType[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.InvalidArgument).Msg(message).Err(),
		StatusCode: http.StatusUnsupportedMediaType,
	}
}

// UnprocessableEntity creates a 422 Unprocessable Entity error response
func UnprocessableEntity[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.InvalidArgument).Msg(message).Err(),
		StatusCode: http.StatusUnprocessableEntity,
	}
}

// Locked creates a 423 Locked error response
func Locked[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.FailedPrecondition).Msg(message).Err(),
		StatusCode: http.StatusLocked,
	}
}

// TooManyRequests creates a 429 Too Many Requests error response
func TooManyRequests[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.ResourceExhausted).Msg(message).Err(),
		StatusCode: http.StatusTooManyRequests,
	}
}

// InternalError creates a 500 Internal Server Error response
func InternalError[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.Internal).Msg(message).Err(),
		StatusCode: http.StatusInternalServerError,
	}
}

// NotImplemented creates a 501 Not Implemented error response
func NotImplemented[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.Unimplemented).Msg(message).Err(),
		StatusCode: http.StatusNotImplemented,
	}
}

// BadGateway creates a 502 Bad Gateway error response
func BadGateway[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.Unavailable).Msg(message).Err(),
		StatusCode: http.StatusBadGateway,
	}
}

// ServiceUnavailable creates a 503 Service Unavailable error response
func ServiceUnavailable[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.Unavailable).Msg(message).Err(),
		StatusCode: http.StatusServiceUnavailable,
	}
}

// GatewayTimeout creates a 504 Gateway Timeout error response
func GatewayTimeout[T any](message string) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        errs.B().Code(errs.DeadlineExceeded).Msg(message).Err(),
		StatusCode: http.StatusGatewayTimeout,
	}
}

// Custom response builders

// WithStatus creates a response with custom status code and data
func WithStatus[T any](data T, code int) Result[T] {
	return Result[T]{
		Data:       data,
		StatusCode: code,
	}
}

// WithError creates an error response with custom status code
func WithError[T any](err error, code int) Result[T] {
	var zero T
	return Result[T]{
		Data:       zero,
		err:        err,
		StatusCode: code,
	}
}

// Fail creates an error response from an error
// Automatically extracts status code from errs.Error if present
func Fail[T any](err error) Result[T] {
	var zero T
	statusCode := http.StatusInternalServerError

	// If it's errs.Error, extract HTTP status
	var errsErr *errs.Error
	if errors.As(err, &errsErr) {
		statusCode = errsErr.Code.HTTPStatus()
	}

	return Result[T]{
		Data:       zero,
		err:        err,
		StatusCode: statusCode,
	}
}

// Fluent API for headers

// WithHeader adds a single header to the response
func (r Result[T]) WithHeader(key, value string) Result[T] {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	r.Headers.Set(key, value)
	return r
}

// WithHeaders adds multiple headers to the response
func (r Result[T]) WithHeaders(headers http.Header) Result[T] {
	if r.Headers == nil {
		r.Headers = make(http.Header)
	}
	for key, values := range headers {
		for _, value := range values {
			r.Headers.Add(key, value)
		}
	}
	return r
}

// InternalMeta returns error metadata for logging (not for API responses)
// This includes internal debugging information that should not be exposed to users
func (r Result[T]) InternalMeta() map[string]any {
	var errsErr *errs.Error
	if errors.As(r.err, &errsErr) {
		return errsErr.Meta
	}
	return nil
}

// Error returns the internal error if present (for logging/debugging)
func (r Result[T]) Error() error {
	return r.err
}
