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
	GET(pattern string, handler Handler, middleware ...Middleware)
	POST(pattern string, handler Handler, middleware ...Middleware)
	PUT(pattern string, handler Handler, middleware ...Middleware)
	PATCH(pattern string, handler Handler, middleware ...Middleware)
	DELETE(pattern string, handler Handler, middleware ...Middleware)
	OPTIONS(pattern string, handler Handler, middleware ...Middleware)
	HEAD(pattern string, handler Handler, middleware ...Middleware)

	// Advanced routing within the group
	Handle(method, pattern string, handler Handler, middleware ...Middleware)

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
