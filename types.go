package glib

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router consisting of the core routing methods used by chi's Mux,
// using only the standard net/http.
type Router interface {
	http.Handler
	chi.Routes

	// Use appends one or more middlewares onto the Router stack.
	Use(middlewares ...Middleware)

	// UseHTTP appends Chi's native middleware directly onto the Router stack.
	// This allows using Chi's built-in middleware without conversion.
	UseHTTP(chiMiddlewares ...func(http.Handler) http.Handler)

	// With adds inline middlewares for an endpoint handler.
	With(middlewares ...Middleware) Router

	// Group adds a new inline-Router along the current routing
	// path, with a fresh middleware stack for the inline-Router.
	Group(fn func(r Router)) Router

	// Route mounts a sub-Router along a `pattern`` string.
	Route(pattern string, fn func(r Router)) Router

	// Mount attaches another http.Handler along ./pattern/*
	Mount(pattern string, h http.Handler)

	// Handle and HandleFunc adds routes for `pattern` that matches
	// all HTTP methods.
	Handle(pattern string, h http.Handler)
	HandleFunc(pattern string, h HandleFunc)

	// Method and MethodFunc adds routes for `pattern` that matches
	// the `method` HTTP method.
	Method(method, pattern string, h http.Handler)
	MethodFunc(method, pattern string, h HandleFunc)

	// HTTP-method routing along `pattern`
	Connect(pattern string, h HandleFunc)
	Delete(pattern string, h HandleFunc)
	Get(pattern string, h HandleFunc)
	Head(pattern string, h HandleFunc)
	Options(pattern string, h HandleFunc)
	Patch(pattern string, h HandleFunc)
	Post(pattern string, h HandleFunc)
	Put(pattern string, h HandleFunc)
	Trace(pattern string, h HandleFunc)

	// NotFound defines a handler to respond whenever a route could
	// not be found.
	NotFound(h HandleFunc)

	// MethodNotAllowed defines a handler to respond whenever a method is
	// not allowed.
	MethodNotAllowed(h HandleFunc)
}

type RouterBlock func(block func(Router))

// Deprecated: use Router instead of RouteGroup
type RouteGroup = Router

type Middleware func(HandleFunc) HandleFunc

// HandleFunc is the function signature for route handlers that can return errors
type HandleFunc func(*Ctx) error

type RouterConfig struct {
	AutoHEAD bool

	TrailingSlashRedirect bool
}
