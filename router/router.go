// Package router provides utilities for HTTP routing using Chi
package router

import (
	"net/http"

	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/slog"
	"github.com/azizndao/glib/validation"
	"github.com/go-chi/chi/v5"
)

// router implements the Router interface using Chi router with Ctx abstraction
type router struct {
	chi       chi.Router
	config    RouterConfig
	logger    *slog.Logger
	validator *validation.Validator
}

// DefaultRouterOptions returns sensible default options
func DefaultRouterOptions() RouterConfig {
	return RouterConfig{
		AutoHEAD:              true,
		TrailingSlashRedirect: true,
	}
}

// New creates a new router with default options
func New(logger *slog.Logger, validator *validation.Validator, options ...RouterConfig) Router {
	chiRouter := chi.NewRouter()

	opts := DefaultRouterOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	r := &router{
		chi:       chiRouter,
		config:    opts,
		logger:    logger,
		validator: validator,
	}

	// Custom 404 handler using Ctx
	chiRouter.NotFound(r.wrapHandler(func(c *Ctx) error {
		return errors.NotFound("Route not found", nil)
	}))

	// Custom 405 handler using Ctx
	chiRouter.MethodNotAllowed(r.wrapHandler(func(c *Ctx) error {
		return errors.MethodNotAllowed("Method not allowed", nil)
	}))

	return r
}

// ServeHTTP implements http.Handler
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.chi.ServeHTTP(w, req)
}

// Routes implements chi.Routes interface
func (r *router) Routes() []chi.Route {
	return r.chi.Routes()
}

// Middlewares implements chi.Routes interface
func (r *router) Middlewares() chi.Middlewares {
	return r.chi.Middlewares()
}

// Match implements chi.Routes interface
func (r *router) Match(rctx *chi.Context, method, path string) bool {
	return r.chi.Match(rctx, method, path)
}

// Find implements chi.Routes interface
func (r *router) Find(rctx *chi.Context, method, path string) string {
	return r.chi.Find(rctx, method, path)
}

// Logger returns the logger instance for the router
func (r *router) Logger() *slog.Logger {
	return r.logger
}

// Use appends one or more middlewares onto the Router stack
func (r *router) Use(middlewares ...Middleware) {
	for _, mw := range middlewares {
		r.chi.Use(r.convertMiddleware(mw))
	}
}

// With adds inline middlewares for an endpoint handler
func (r *router) With(middlewares ...Middleware) Router {
	chiRouter := r.chi.With()
	for _, mw := range middlewares {
		chiRouter = chiRouter.With(r.convertMiddleware(mw))
	}

	return &router{
		chi:       chiRouter,
		config:    r.config,
		logger:    r.logger,
		validator: r.validator,
	}
}

// Group adds a new inline-Router along the current routing path
func (r *router) Group(fn func(r Router)) Router {
	im := r.With()
	if fn != nil {
		fn(im)
	}
	return im
}

// Route mounts a sub-Router along a pattern string
func (r *router) Route(pattern string, fn func(r Router)) Router {
	subRouter := New(r.logger, r.validator, r.config)
	if fn != nil {
		fn(subRouter)
	}
	r.Mount(pattern, subRouter)
	return subRouter
}

// Mount attaches another http.Handler along ./pattern/*
func (r *router) Mount(pattern string, h http.Handler) {
	r.chi.Mount(pattern, h)
}

// Handle adds routes for pattern that matches all HTTP methods
func (r *router) Handle(pattern string, h http.Handler) {
	r.chi.Handle(pattern, h)
}

// HandleFunc adds routes for pattern that matches all HTTP methods
func (r *router) HandleFunc(pattern string, h HandleFunc) {
	r.chi.HandleFunc(pattern, r.wrapHandler(h))
}

// Method adds routes for pattern that matches the method HTTP method
func (r *router) Method(method, pattern string, h http.Handler) {
	r.chi.Method(method, pattern, h)
}

// MethodFunc adds routes for pattern that matches the method HTTP method
func (r *router) MethodFunc(method, pattern string, h HandleFunc) {
	r.chi.MethodFunc(method, pattern, r.wrapHandler(h))
}

// Connect adds a CONNECT route
func (r *router) Connect(pattern string, h HandleFunc) {
	r.chi.Connect(pattern, r.wrapHandler(h))
}

// Delete adds a DELETE route
func (r *router) Delete(pattern string, h HandleFunc) {
	r.chi.Delete(pattern, r.wrapHandler(h))
}

// Get adds a GET route
func (r *router) Get(pattern string, h HandleFunc) {
	r.chi.Get(pattern, r.wrapHandler(h))
}

// Head adds a HEAD route
func (r *router) Head(pattern string, h HandleFunc) {
	r.chi.Head(pattern, r.wrapHandler(h))
}

// Options adds an OPTIONS route
func (r *router) Options(pattern string, h HandleFunc) {
	r.chi.Options(pattern, r.wrapHandler(h))
}

// Patch adds a PATCH route
func (r *router) Patch(pattern string, h HandleFunc) {
	r.chi.Patch(pattern, r.wrapHandler(h))
}

// Post adds a POST route
func (r *router) Post(pattern string, h HandleFunc) {
	r.chi.Post(pattern, r.wrapHandler(h))
}

// Put adds a PUT route
func (r *router) Put(pattern string, h HandleFunc) {
	r.chi.Put(pattern, r.wrapHandler(h))
}

// Trace adds a TRACE route
func (r *router) Trace(pattern string, h HandleFunc) {
	r.chi.Trace(pattern, r.wrapHandler(h))
}

// NotFound defines a handler to respond whenever a route could not be found
func (r *router) NotFound(h HandleFunc) {
	r.chi.NotFound(r.wrapHandler(h))
}

// MethodNotAllowed defines a handler to respond whenever a method is not allowed
func (r *router) MethodNotAllowed(h HandleFunc) {
	r.chi.MethodNotAllowed(r.wrapHandler(h))
}

// wrapHandler converts a Ctx-based Handler to http.HandlerFunc with error handling
// This is the bridge between your Ctx abstraction and Chi's http.Handler
func (r *router) wrapHandler(handler HandleFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Create Ctx wrapper for this request
		ctx := newCtx(w, req, r.logger, r.validator)

		// Execute the handler with Ctx
		if err := handler(ctx); err != nil {
			var glibErr *errors.ApiError

			switch t := err.(type) {
			case *errors.ApiError:
				glibErr = t
			default:
				glibErr = errors.InternalServerError("Server Error", err)
			}

			// Set default data if nil
			data := glibErr.Data
			if data == nil {
				data = http.StatusText(glibErr.Code)
			}

			// Send error response using Ctx
			ctx.Status(glibErr.Code).JSON(glibErr)
		}
	}
}

// convertMiddleware converts a Ctx-based Middleware to Chi middleware
// This allows your existing middleware to work seamlessly with Chi
func (r *router) convertMiddleware(mw Middleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Create Ctx wrapper
			ctx := newCtx(w, req, r.logger, r.validator)

			// Wrap the next handler as a Ctx Handler
			nextHandler := func(c *Ctx) error {
				// Execute next middleware/handler in the chain
				next.ServeHTTP(c.Response, c.Request)
				return nil
			}

			// Execute middleware with Ctx
			if err := mw(nextHandler)(ctx); err != nil {
				// Handle middleware error
				var glibErr *errors.ApiError

				switch t := err.(type) {
				case *errors.ApiError:
					glibErr = t
				default:
					glibErr = errors.InternalServerError("Middleware Error", err)
				}

				data := glibErr.Data
				if data == nil {
					data = http.StatusText(glibErr.Code)
				}

				ctx.Status(glibErr.Code).JSON(glibErr)
			}
		})
	}
}

// UseHTTP is a convenience method to add Chi middleware directly to the router.
// It converts the Chi middleware to router.Middleware automatically.
//
// Example usage:
//
//	import chimiddleware "github.com/go-chi/chi/v5/middleware"
//
//	router.UseHTTP(chimiddleware.StripSlashes)
//	router.UseHTTP(chimiddleware.Heartbeat("/ping"))
func (r *router) UseHTTP(chiMiddlewares ...func(http.Handler) http.Handler) {
	for _, chiMw := range chiMiddlewares {
		r.chi.Use(chiMw)
	}
}
