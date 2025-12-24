package glib

import "fmt"

// ResourceOptions configures how resource routes are generated.
//
// Example:
//
//	// Only register Index and Show
//	router.Resource("posts", NewPostController, glib.ResourceOptions{
//	    Only: []string{"Index", "Show"},
//	})
//
//	// Register all except Destroy
//	router.Resource("posts", NewPostController, glib.ResourceOptions{
//	    Except: []string{"Destroy"},
//	})
//
//	// Custom route names
//	router.Resource("posts", NewPostController, glib.ResourceOptions{
//	    Names: map[string]string{
//	        "Index": "posts.list",
//	        "Show":  "posts.detail",
//	    },
//	})
type ResourceOptions struct {
	// Only specifies which methods to register (whitelist).
	// If empty, all methods are registered (unless specified in Except).
	// Valid values: "Index", "Create", "Store", "Show", "Edit", "Update", "Destroy"
	Only []string

	// Except specifies which methods to exclude (blacklist).
	// Ignored if Only is specified.
	// Valid values: "Index", "Create", "Store", "Show", "Edit", "Update", "Destroy"
	Except []string

	// Names provides custom route names for methods.
	// Key: method name, Value: custom route name
	// Example: {"Index": "posts.list", "Show": "posts.detail"}
	Names map[string]string

	// Params provides custom parameter names in URLs.
	// Key: default name, Value: custom name
	// Example: {"id": "postId"} changes /posts/{id} to /posts/{postId}
	Params map[string]string

	// Shallow enables shallow nesting for nested resources.
	// When true, only creates nested routes for index and store.
	Shallow bool
}

// DefaultResourceOptions returns a new ResourceOptions with sensible defaults.
func DefaultResourceOptions() ResourceOptions {
	return ResourceOptions{
		Names:  make(map[string]string),
		Params: make(map[string]string),
	}
}

// ShouldRegister checks if a method should be registered based on Only/Except filters.
func (o ResourceOptions) ShouldRegister(method string) bool {
	// If Only is specified, method must be in the list
	if len(o.Only) > 0 {
		for _, m := range o.Only {
			if m == method {
				return true
			}
		}
		return false
	}

	// If Except is specified, method must NOT be in the list
	if len(o.Except) > 0 {
		for _, m := range o.Except {
			if m == method {
				return false
			}
		}
	}

	return true
}

// GetRouteName returns the route name for a method.
// Returns custom name if defined, otherwise generates default name.
func (o ResourceOptions) GetRouteName(resource, method string) string {
	if customName, ok := o.Names[method]; ok {
		return customName
	}
	return fmt.Sprintf("%s.%s", resource, toLowerFirst(method))
}

// GetParamName returns the parameter name for a resource.
// Returns custom name if defined, otherwise returns "id".
func (o ResourceOptions) GetParamName() string {
	if param, ok := o.Params["id"]; ok {
		return param
	}
	return "id"
}

// toLowerFirst converts first character to lowercase
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]+32) + s[1:] // Simple ASCII lowercase
}
