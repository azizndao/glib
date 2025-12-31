package middleware

import (
	"context"
	"net/http"

	"github.com/azizndao/glib"
)

// Request wraps http.Request with framework-specific helpers
type Request interface {
	// Context returns the request's context
	Context() context.Context

	// WithContext returns a copy of the request with the given context
	WithContext(ctx context.Context) Request

	// PathValue returns the value for a named path parameter
	PathValue(key string) string

	// Header returns the first value for the given header key
	Header(key string) string

	// Method returns the HTTP method (GET, POST, etc.)
	Method() string

	// Path returns the URL path
	Path() string

	// HTTPRequest returns the underlying *http.Request
	HTTPRequest() *http.Request
}

// Next calls the next middleware or handler in the chain
type Next func(Request) glib.Result[any]

// Middleware is the signature for middleware functions
type Middleware func(Request, Next) glib.Result[any]

// requestImpl is the concrete implementation of Request
type requestImpl struct {
	req *http.Request
}

// NewRequest creates a new Request from an *http.Request
func NewRequest(r *http.Request) Request {
	return &requestImpl{req: r}
}

func (r *requestImpl) Context() context.Context {
	return r.req.Context()
}

func (r *requestImpl) WithContext(ctx context.Context) Request {
	return &requestImpl{req: r.req.WithContext(ctx)}
}

func (r *requestImpl) PathValue(key string) string {
	return r.req.PathValue(key)
}

func (r *requestImpl) Header(key string) string {
	return r.req.Header.Get(key)
}

func (r *requestImpl) Method() string {
	return r.req.Method
}

func (r *requestImpl) Path() string {
	return r.req.URL.Path
}

func (r *requestImpl) HTTPRequest() *http.Request {
	return r.req
}
