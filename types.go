package grouter

import (
	"net/http"
)

type Router interface {
	RouteGroup

	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Handler() http.Handler
}

type RouteGroup interface {
	// HTTP method routing within the group
	Get(pattern string, handler Handler, middleware ...Middleware)
	Post(pattern string, handler Handler, middleware ...Middleware)
	Put(pattern string, handler Handler, middleware ...Middleware)
	Patch(pattern string, handler Handler, middleware ...Middleware)
	Delete(pattern string, handler Handler, middleware ...Middleware)
	Option(pattern string, handler Handler, middleware ...Middleware)
	Head(pattern string, handler Handler, middleware ...Middleware)

	// Advanced routing within the group
	Handle(method, pattern string, handler Handler, middleware ...Middleware)

	Route(prefix string, handler http.HandlerFunc)

	// Nested groups
	Group(prefix string, middleware ...Middleware) RouteGroup

	// Group middleware
	Use(middleware ...Middleware)
}

type Middleware func(http.Handler) http.Handler

// Handler is the function signature for route handlers that can return errors
type Handler func(*Ctx) error

// RouteInfo contains information about a registered route
type RouteInfo struct {
	Method      string
	Pattern     string
	Handler     http.HandlerFunc
	Middleware  []Middleware
	Group       string
	Description string
}

type RouterOptions struct {
	AutoOPTIONS bool

	AutoHEAD bool

	TrailingSlashRedirect bool

	EnableLogging bool
}
