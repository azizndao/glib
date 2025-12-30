package validator

import (
	"fmt"
	"strings"

	"github.com/goyave/glib/v2/internal/scanner"
)

// ValidationError represents a validation error
type ValidationError struct {
	Type     string // "error", "warning"
	Location string // File:Line
	Message  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Location, e.Message)
}

// Validator validates a scanned project
type Validator struct {
	errors   []*ValidationError
	warnings []*ValidationError
}

// New creates a new validator
func New() *Validator {
	return &Validator{}
}

// Validate validates the project and returns errors
func (v *Validator) Validate(project *scanner.Project) error {
	// Validate controllers
	for _, ctrl := range project.Controllers {
		v.validateController(ctrl)
	}

	// Validate providers
	for _, prov := range project.Providers {
		v.validateProvider(prov)
	}

	// Validate middleware
	for _, mw := range project.Middleware {
		v.validateMiddleware(mw)
	}

	// Check for duplicate routes
	v.validateUniqueRoutes(project.Controllers)

	// Check for dependency cycles (simple check)
	v.validateDependencies(project)

	if len(v.errors) > 0 {
		return fmt.Errorf("validation failed with %d errors", len(v.errors))
	}

	return nil
}

// Errors returns all validation errors
func (v *Validator) Errors() []*ValidationError {
	return v.errors
}

// Warnings returns all validation warnings
func (v *Validator) Warnings() []*ValidationError {
	return v.warnings
}

// validateController validates a controller
func (v *Validator) validateController(ctrl *scanner.Controller) {
	location := fmt.Sprintf("%s:%d", ctrl.FilePath, ctrl.Position.Line)

	// Check route prefix
	if !strings.HasPrefix(ctrl.RoutePrefix, "/") {
		v.addError(location, fmt.Sprintf("controller route prefix must start with '/': %s", ctrl.RoutePrefix))
	}

	// Check handlers
	if len(ctrl.Handlers) == 0 {
		v.addWarning(location, fmt.Sprintf("controller %s has no handlers", ctrl.Name))
	}

	for _, handler := range ctrl.Handlers {
		v.validateHandler(handler, ctrl)
	}

	// Check middleware references
	for _, mwName := range ctrl.Middlewares {
		if mwName == "" {
			v.addError(location, "empty middleware name in controller")
		}
	}
}

// validateHandler validates a handler
func (v *Validator) validateHandler(handler *scanner.Handler, ctrl *scanner.Controller) {
	location := fmt.Sprintf("%s:%d", ctrl.FilePath, handler.Position.Line)

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true,
		"DELETE": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[handler.Method] {
		v.addError(location, fmt.Sprintf("invalid HTTP method: %s", handler.Method))
	}

	// Validate path
	if !strings.HasPrefix(handler.Path, "/") {
		v.addError(location, fmt.Sprintf("handler path must start with '/': %s", handler.Path))
	}

	// Validate signature pattern
	if handler.Signature == nil {
		v.addError(location, "handler signature could not be parsed")
		return
	}

	if handler.Signature.Pattern < 1 || handler.Signature.Pattern > 9 {
		v.addError(location, fmt.Sprintf("invalid handler signature pattern: %d", handler.Signature.Pattern))
	}

	// Check middleware references
	for _, mwName := range handler.Middlewares {
		if mwName == "" {
			v.addError(location, "empty middleware name in handler")
		}
	}
}

// validateProvider validates a provider
func (v *Validator) validateProvider(prov *scanner.Provider) {
	location := fmt.Sprintf("%s:%d", prov.FilePath, prov.Position.Line)

	// Validate lifecycle
	if prov.Lifecycle != "singleton" && prov.Lifecycle != "transient" {
		v.addError(location, fmt.Sprintf("invalid provider lifecycle: %s (must be 'singleton' or 'transient')", prov.Lifecycle))
	}

	// Validate return type
	if prov.ReturnType == nil {
		v.addError(location, "provider must have a return type")
	}
}

// validateMiddleware validates middleware
func (v *Validator) validateMiddleware(mw *scanner.Middleware) {
	location := fmt.Sprintf("%s:%d", mw.FilePath, mw.Position.Line)

	// Validate name
	if mw.Name == "" {
		v.addError(location, "middleware name cannot be empty")
	}
}

// validateUniqueRoutes checks for duplicate routes
func (v *Validator) validateUniqueRoutes(controllers []*scanner.Controller) {
	seen := make(map[string]*scanner.Handler)

	for _, ctrl := range controllers {
		for _, handler := range ctrl.Handlers {
			key := handler.Method + " " + handler.FullPath
			if existing, ok := seen[key]; ok {
				location := fmt.Sprintf("%s:%d", ctrl.FilePath, handler.Position.Line)
				existingLoc := fmt.Sprintf("%s:%d", existing.Position.Filename, existing.Position.Line)
				v.addError(location, fmt.Sprintf("duplicate route %s (also defined at %s)", key, existingLoc))
			} else {
				seen[key] = handler
			}
		}
	}
}

// validateDependencies performs basic dependency validation
func (v *Validator) validateDependencies(project *scanner.Project) {
	// Build a map of available types from providers
	providedTypes := make(map[string]bool)
	for _, prov := range project.Providers {
		if prov.ReturnType != nil {
			providedTypes[prov.ReturnType.FullName] = true
		}
	}

	// Check controller dependencies
	for _, ctrl := range project.Controllers {
		location := fmt.Sprintf("%s:%d", ctrl.FilePath, ctrl.Position.Line)
		for _, field := range ctrl.Fields {
			if field.Type != nil && !field.Type.IsPrimitive {
				// Check if this type is provided
				if !providedTypes[field.Type.FullName] {
					v.addWarning(location, fmt.Sprintf("controller %s depends on %s but no provider found", ctrl.Name, field.Type.FullName))
				}
			}
		}
	}
}

// addError adds a validation error
func (v *Validator) addError(location, message string) {
	v.errors = append(v.errors, &ValidationError{
		Type:     "error",
		Location: location,
		Message:  message,
	})
}

// addWarning adds a validation warning
func (v *Validator) addWarning(location, message string) {
	v.warnings = append(v.warnings, &ValidationError{
		Type:     "warning",
		Location: location,
		Message:  message,
	})
}
