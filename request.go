package glib

import (
	"context"
	"net/http"
)

// Request wraps http.Request to provide a clean, middleware-friendly API.
// It follows an immutable pattern: modification methods return new Request instances.
type Request struct {
	ctx context.Context
	r   *http.Request
}

// NewRequest creates a Request wrapper from an http.Request.
// This is primarily used by the generated code.
func NewRequest(r *http.Request) Request {
	return Request{
		ctx: r.Context(),
		r:   r,
	}
}

// Context returns the request's context.
func (req Request) Context() context.Context {
	return req.ctx
}

// Header returns the value of the request header with the given key.
// If the header is not present, it returns an empty string.
func (req Request) Header(key string) string {
	return req.r.Header.Get(key)
}

// Query returns the first value for the named query parameter.
// If the parameter is not present, it returns an empty string.
func (req Request) Query(key string) string {
	return req.r.URL.Query().Get(key)
}

// QuerySlice returns all values for the named query parameter.
// If the parameter is not present, it returns an empty slice.
func (req Request) QuerySlice(key string) []string {
	values := req.r.URL.Query()[key]
	if values == nil {
		return []string{}
	}
	return values
}

// PathValue returns the value of the path parameter with the given key.
// Uses Go 1.22+ http.Request.PathValue.
func (req Request) PathValue(key string) string {
	return req.r.PathValue(key)
}

// Method returns the HTTP method (GET, POST, PUT, etc.).
func (req Request) Method() string {
	return req.r.Method
}

// URL returns the full request URL as a string.
func (req Request) URL() string {
	return req.r.URL.String()
}

// Path returns the URL path (without query string).
func (req Request) Path() string {
	return req.r.URL.Path
}

// RemoteAddr returns the client's network address.
func (req Request) RemoteAddr() string {
	return req.r.RemoteAddr
}

// Value returns the value associated with the context key.
func (req Request) Value(key any) any {
	return req.ctx.Value(key)
}

// WithContext returns a new Request with the given context.
// This follows an immutable pattern.
func (req Request) WithContext(ctx context.Context) Request {
	return Request{
		ctx: ctx,
		r:   req.r,
	}
}

// WithValue is a convenience method that adds a single key-value pair to the context.
// It's equivalent to WithContext(context.WithValue(req.Context(), key, val)).
func (req Request) WithValue(key, val any) Request {
	return req.WithContext(context.WithValue(req.ctx, key, val))
}

// WithValues is a convenience method that adds multiple key-value pairs to the context.
// It's useful for adding several values at once without chaining multiple WithValue calls.
func (req Request) WithValues(values map[any]any) Request {
	ctx := req.ctx
	for k, v := range values {
		ctx = context.WithValue(ctx, k, v)
	}
	return req.WithContext(ctx)
}

// HTTPRequest returns the underlying *http.Request with the current context.
// This is used internally by the generated code to pass the request to handlers.
// The returned request has its context updated to match the Request's context.
func (req Request) HTTPRequest() *http.Request {
	if req.ctx == req.r.Context() {
		return req.r
	}
	return req.r.WithContext(req.ctx)
}

// RawHTTPRequest returns the original underlying *http.Request without context updates.
// Use this only when you need access to the original request for advanced use cases.
func (req Request) RawHTTPRequest() *http.Request {
	return req.r
}
