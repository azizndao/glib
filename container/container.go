// Package container provides a service container for dependency injection.
// It supports singleton and factory bindings with type-safe resolution using generics.
package container

import (
	"fmt"
	"reflect"
	"sync"
)

// Container is a dependency injection container that manages service bindings.
type Container struct {
	bindings map[reflect.Type]*binding
	resolved map[reflect.Type]any
	mu       sync.RWMutex
}

// binding represents a service binding in the container.
type binding struct {
	resolver   any         // Factory function
	singleton  bool        // Whether this is a singleton
	resolved   bool        // Whether singleton has been resolved
	instance   any         // Cached singleton instance
	contextual map[any]any // Contextual bindings
}

// New creates a new container instance.
func New() *Container {
	return &Container{
		bindings: make(map[reflect.Type]*binding),
		resolved: make(map[reflect.Type]any),
	}
}

// Bind registers a factory binding in the container.
// The resolver function will be called each time the service is resolved.
//
// Example:
//
//	container.Bind(func(c *Container) (*Database, error) {
//	    return NewDatabase(c.MustResolve[*Config]())
//	})
func Bind[T any](c *Container, resolver func(*Container) (T, error)) error {
	var zero T
	typ := reflect.TypeOf(zero)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.bindings[typ] = &binding{
		resolver:   resolver,
		singleton:  false,
		contextual: make(map[any]any),
	}

	return nil
}

// Singleton registers a singleton binding in the container.
// The resolver function will be called only once, and the result will be cached.
//
// Example:
//
//	container.Singleton(func(c *Container) (*Config, error) {
//	    return LoadConfig()
//	})
func Singleton[T any](c *Container, resolver func(*Container) (T, error)) error {
	var zero T
	typ := reflect.TypeOf(zero)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.bindings[typ] = &binding{
		resolver:   resolver,
		singleton:  true,
		resolved:   false,
		contextual: make(map[any]any),
	}

	return nil
}

// Instance registers an existing instance as a singleton.
//
// Example:
//
//	logger := slog.Default()
//	container.Instance(logger)
func Instance[T any](c *Container, instance T) error {
	typ := reflect.TypeOf(instance)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.bindings[typ] = &binding{
		resolver:  nil,
		singleton: true,
		resolved:  true,
		instance:  instance,
	}

	return nil
}

// Resolve resolves a service from the container.
// Returns the service instance and an error if resolution fails.
//
// Example:
//
//	db, err := container.Resolve[*Database]()
//	if err != nil {
//	    log.Fatal(err)
//	}
func Resolve[T any](c *Container) (T, error) {
	var zero T
	typ := reflect.TypeOf(zero)

	c.mu.RLock()
	b, exists := c.bindings[typ]
	c.mu.RUnlock()

	if !exists {
		return zero, fmt.Errorf("no binding found for type %v", typ)
	}

	// If singleton and already resolved, return cached instance
	if b.singleton && b.resolved {
		return b.instance.(T), nil
	}

	// If singleton but not resolved, resolve and cache
	if b.singleton && !b.resolved {
		c.mu.Lock()
		defer c.mu.Unlock()

		// Double-check after acquiring write lock
		if b.resolved {
			return b.instance.(T), nil
		}

		resolver, ok := b.resolver.(func(*Container) (T, error))
		if !ok {
			return zero, fmt.Errorf("invalid resolver type for %v", typ)
		}

		instance, err := resolver(c)
		if err != nil {
			return zero, fmt.Errorf("failed to resolve %v: %w", typ, err)
		}

		b.instance = instance
		b.resolved = true

		return instance, nil
	}

	// Factory binding - resolve each time
	resolver, ok := b.resolver.(func(*Container) (T, error))
	if !ok {
		return zero, fmt.Errorf("invalid resolver type for %v", typ)
	}

	instance, err := resolver(c)
	if err != nil {
		return zero, fmt.Errorf("failed to resolve %v: %w", typ, err)
	}

	return instance, nil
}

// MustResolve resolves a service from the container and panics if resolution fails.
// Use this when you're certain the binding exists.
//
// Example:
//
//	db := container.MustResolve[*Database]()
func MustResolve[T any](c *Container) T {
	instance, err := Resolve[T](c)
	if err != nil {
		panic(err)
	}
	return instance
}

// Has checks if a binding exists for the given type.
//
// Example:
//
//	if container.Has[*Database]() {
//	    db := container.MustResolve[*Database]()
//	}
func Has[T any](c *Container) bool {
	var zero T
	typ := reflect.TypeOf(zero)

	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.bindings[typ]
	return exists
}

// Forget removes a binding from the container.
//
// Example:
//
//	container.Forget[*OldService]()
func Forget[T any](c *Container) {
	var zero T
	typ := reflect.TypeOf(zero)

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.bindings, typ)
	delete(c.resolved, typ)
}

// Flush removes all bindings from the container.
func (c *Container) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bindings = make(map[reflect.Type]*binding)
	c.resolved = make(map[reflect.Type]any)
}

// Contextual creates a contextual binding.
// Useful when you need different implementations based on context.
//
// Example:
//
//	container.Contextual[*PaymentController, PaymentGateway](
//	    func(c *Container) (PaymentGateway, error) {
//	        return NewStripeGateway(), nil
//	    },
//	)
func Contextual[Concrete any, Abstract any](c *Container, resolver func(*Container) (Abstract, error)) error {
	var concrete Concrete
	var abstract Abstract

	concreteType := reflect.TypeOf(concrete)
	abstractType := reflect.TypeOf(abstract)

	c.mu.Lock()
	defer c.mu.Unlock()

	b, exists := c.bindings[abstractType]
	if !exists {
		b = &binding{
			contextual: make(map[any]any),
		}
		c.bindings[abstractType] = b
	}

	b.contextual[concreteType] = resolver

	return nil
}

// Call invokes a function with dependency injection for its parameters.
// This is useful for calling functions/methods with automatic dependency resolution.
//
// Example:
//
//	err := container.Call(func(db *Database, cache *Cache) error {
//	    // Use db and cache
//	    return nil
//	})
func (c *Container) Call(fn any) error {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("expected function, got %v", fnType.Kind())
	}

	// Build arguments
	args := make([]reflect.Value, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argType := fnType.In(i)

		// Check if this is the container itself
		if argType == reflect.TypeFor[*Container]() {
			args[i] = reflect.ValueOf(c)
			continue
		}

		// Try to resolve from container
		c.mu.RLock()
		b, exists := c.bindings[argType]
		c.mu.RUnlock()

		if !exists {
			return fmt.Errorf("cannot resolve argument %d of type %v", i, argType)
		}

		// Resolve the argument
		var instance any
		var err error

		if b.singleton && b.resolved {
			instance = b.instance
		} else if b.resolver != nil {
			// Call resolver - we need to use reflection here
			resolverValue := reflect.ValueOf(b.resolver)
			results := resolverValue.Call([]reflect.Value{reflect.ValueOf(c)})

			if len(results) != 2 {
				return fmt.Errorf("resolver must return (T, error)")
			}

			if !results[1].IsNil() {
				err = results[1].Interface().(error)
				return fmt.Errorf("failed to resolve argument %d: %w", i, err)
			}

			instance = results[0].Interface()

			if b.singleton {
				c.mu.Lock()
				b.instance = instance
				b.resolved = true
				c.mu.Unlock()
			}
		} else {
			return fmt.Errorf("no resolver for type %v", argType)
		}

		args[i] = reflect.ValueOf(instance)
	}

	// Call the function
	results := fnValue.Call(args)

	// Check if function returns error
	if fnType.NumOut() > 0 && fnType.Out(fnType.NumOut()-1).Implements(reflect.TypeFor[error]()) {
		if !results[len(results)-1].IsNil() {
			return results[len(results)-1].Interface().(error)
		}
	}

	return nil
}

// Tag associates a tag with multiple service types.
// Useful for grouping related services.
//
// Example:
//
//	container.Tag[*EmailNotifier]("notifiers")
//	container.Tag[*SMSNotifier]("notifiers")
//	notifiers := container.Tagged("notifiers")
func Tag[T any](c *Container, tag string) {
	var zero T
	typ := reflect.TypeOf(zero)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.resolved == nil {
		c.resolved = make(map[reflect.Type]any)
	}

	// Store tag association (we'll use a special type for tags)
	tagType := reflect.TypeFor[string]()
	tags, _ := c.resolved[tagType].(map[string][]reflect.Type)
	if tags == nil {
		tags = make(map[string][]reflect.Type)
		c.resolved[tagType] = tags
	}

	tags[tag] = append(tags[tag], typ)
}

// Tagged resolves all services with the given tag.
//
// Example:
//
//	notifiers := container.Tagged("notifiers")
//	for _, notifier := range notifiers {
//	    notifier.Send(message)
//	}
func (c *Container) Tagged(tag string) []any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tagType := reflect.TypeFor[string]()
	tags, ok := c.resolved[tagType].(map[string][]reflect.Type)
	if !ok {
		return nil
	}

	types, exists := tags[tag]
	if !exists {
		return nil
	}

	var instances []any
	for _, typ := range types {
		b, exists := c.bindings[typ]
		if !exists {
			continue
		}

		if b.singleton && b.resolved {
			instances = append(instances, b.instance)
		}
	}

	return instances
}

// TypeName returns the type name of a value.
// Useful for getting unique identifiers for types.
func TypeName(v any) string {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Pointer {
		return typ.Elem().String()
	}
	return typ.String()
}
