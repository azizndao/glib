// Package grouter provides utilities for HTTP routing
package grouter

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/azizndao/grouter/errors"
)

// router implements the Router interface using Go's enhanced net/http features
type router struct {
	mux        *http.ServeMux
	options    RouterOptions
	middleware []Middleware
	routes     []RouteInfo
	prefix     string
	groupMW    []Middleware
}

// DefaultRouterOptions returns sensible default options
func DefaultRouterOptions() RouterOptions {
	return RouterOptions{
		AutoOPTIONS:           true,
		AutoHEAD:              true,
		TrailingSlashRedirect: true,
	}
}

// NewRouter creates a new router with default options
func NewRouter() Router {
	return NewRouterWithOptions(DefaultRouterOptions())
}

// NewRouterWithOptions creates a new router with custom options
func NewRouterWithOptions(options RouterOptions) Router {
	r := &router{
		mux:     http.NewServeMux(),
		options: options,
		routes:  make([]RouteInfo, 0),
	}

	return r
}

// Get registers a Get route
func (r *router) Get(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodGet, pattern, handler, middleware...)
}

// Post registers a Post route
func (r *router) Post(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodPost, pattern, handler, middleware...)
}

// Put registers a Put route
func (r *router) Put(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodPut, pattern, handler, middleware...)
}

// Patch registers a Patch route
func (r *router) Patch(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodPatch, pattern, handler, middleware...)
}

// Delete registers a Delete route
func (r *router) Delete(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodDelete, pattern, handler, middleware...)
}

// Option registers an Option route
func (r *router) Option(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodOptions, pattern, handler, middleware...)
}

// Head registers a Head route
func (r *router) Head(pattern string, handler Handler, middleware ...Middleware) {
	r.Handle(http.MethodHead, pattern, handler, middleware...)
}

// Route registers a route with a specific HTTP method
func (r *router) Route(prefix string, handler http.Handler) {
	r.mux.Handle(prefix, handler)
}

// Handle registers a route with a specific HTTP method
func (r *router) Handle(method, pattern string, handler Handler, middleware ...Middleware) {
	// Build full pattern with prefix
	fullPattern := r.buildPattern(method, pattern)

	// Combine all middleware (global + group + route-specific)
	allMiddleware := make([]Middleware, 0, len(r.middleware)+len(r.groupMW)+len(middleware))
	allMiddleware = append(allMiddleware, r.middleware...)
	allMiddleware = append(allMiddleware, r.groupMW...)
	allMiddleware = append(allMiddleware, middleware...)

	// Convert Handler to http.HandlerFunc with middleware applied
	httpHandler := r.handlerToHTTPHandler(handler, allMiddleware)

	// Register with the mux
	r.mux.Handle(fullPattern, httpHandler)

	// Store route info for introspection
	r.routes = append(r.routes, RouteInfo{
		Method:     method,
		Pattern:    pattern,
		Handler:    httpHandler,
		Middleware: allMiddleware,
		Group:      r.prefix,
	})

	// Auto-generate HEAD handler from GET if enabled
	if r.options.AutoHEAD && method == http.MethodGet {
		headPattern := r.buildPattern(http.MethodHead, pattern)
		r.mux.Handle(headPattern, httpHandler)
	}
}

// Group creates a new route group with a prefix
func (r *router) Group(prefix string, middleware ...Middleware) RouteGroup {
	// Clean and combine prefixes
	fullPrefix := path.Join(r.prefix, prefix)
	if !strings.HasSuffix(fullPrefix, "/") && strings.HasSuffix(prefix, "/") {
		fullPrefix += "/"
	}

	// Combine middleware
	groupMW := make([]Middleware, 0, len(r.groupMW)+len(middleware))
	groupMW = append(groupMW, r.groupMW...)
	groupMW = append(groupMW, middleware...)

	return &router{
		mux:        r.mux,
		options:    r.options,
		middleware: r.middleware,
		routes:     r.routes,
		prefix:     fullPrefix,
		groupMW:    groupMW,
	}
}

// Use adds middleware to the router
func (r *router) Use(middleware ...Middleware) RouteGroup {
	r.middleware = append(r.middleware, middleware...)
	return r
}

// ServeHTTP implements http.Handler
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Handler returns the underlying http.Handler
func (r *router) Handler() http.Handler {
	return r
}

// Routes returns information about all registered routes
func (r *router) Routes() []RouteInfo {
	return r.routes
}

// handlerToHTTPHandler converts a Handler to http.HandlerFunc with error handling
func (r *router) handlerToHTTPHandler(handler Handler, middleware []Middleware) http.HandlerFunc {
	// Apply middleware chain to the handler
	finalHandler := r.applyCtxMiddleware(handler, middleware)

	return func(w http.ResponseWriter, req *http.Request) {
		ctx := NewCtx(w, req)

		if err := finalHandler(ctx); err != nil {
			var grouterErr *errors.Error

			switch t := err.(type) {
			case *errors.Error:
				if t.Data == nil {
					t.Data = http.StatusText(http.StatusInternalServerError)
				}
				grouterErr = t

			default:
				grouterErr = errors.ErrorInternalServerError("Server Error", err)
			}

			ctx.Status(grouterErr.Code).JSON(grouterErr)
		}
	}
}

// buildPattern constructs the full pattern for registration
func (r *router) buildPattern(method, pattern string) string {
	// Clean the pattern
	if pattern == "" {
		pattern = "/"
	}

	// Combine prefix and pattern
	fullPath := path.Join(r.prefix, pattern)

	// Preserve trailing slash if original pattern had it
	if strings.HasSuffix(pattern, "/") && !strings.HasSuffix(fullPath, "/") && fullPath != "/" {
		fullPath += "/"
	}

	// Add method prefix for Go 1.22+ enhanced routing
	if method != "" {
		return fmt.Sprintf("%s %s", method, fullPath)
	}

	return fullPath
}

// applyCtxMiddleware applies a chain of Ctx middleware to a Handler
func (r *router) applyCtxMiddleware(handler Handler, middleware []Middleware) Handler {
	// Apply middleware in reverse order so they execute in the correct order
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
