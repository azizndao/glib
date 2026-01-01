package validator

import (
	"fmt"
	"strings"

	"github.com/azizndao/glib/internal/scanner"
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
	for _, ctrl := range project.Controllers {
		v.validateController(ctrl)
	}

	for _, prov := range project.Providers {
		v.validateProvider(prov)
	}

	for _, mw := range project.Middleware {
		v.validateMiddleware(mw)
	}

	v.validateUniqueRoutes(project.Controllers)

	v.validateDependencies(project)

	v.validateMiddlewareReferences(project)

	if len(v.errors) > 0 {
		return fmt.Errorf("validation failed with %d errors", len(v.errors))
	}

	return nil
}

func (v *Validator) Errors() []*ValidationError {
	return v.errors
}

func (v *Validator) Warnings() []*ValidationError {
	return v.warnings
}

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

	// Validate handler pattern
	if handler.Signature.Pattern != scanner.PatternResult && handler.Signature.Pattern != scanner.PatternRawHTTP {
		v.addError(location, fmt.Sprintf("invalid handler signature: must return glib.Result[T] or use raw http.ResponseWriter/Request, got pattern '%s'", handler.Signature.Pattern))
	}

	// Validate path parameters match handler signature
	v.validatePathParams(handler, location)

	// Validate query parameters
	v.validateQueryParams(handler, location)

	// Validate header parameters
	v.validateHeaderParams(handler, location)
}

// validatePathParams validates that path parameters in the route match the handler signature
func (v *Validator) validatePathParams(handler *scanner.Handler, location string) {
	// Extract path parameters from the route path
	pathParams := extractPathParams(handler.Path)

	// For raw HTTP pattern, we don't validate params since handler gets raw request
	if handler.Signature.Pattern == scanner.PatternRawHTTP {
		return
	}

	// Check that signature has matching number of path parameters
	sigParams := handler.Signature.PathParams
	if len(pathParams) != len(sigParams) {
		v.addError(location, fmt.Sprintf(
			"path parameter mismatch: route has %d parameters %v but handler signature has %d parameters",
			len(pathParams), pathParams, len(sigParams)))
		return
	}

	// Check that parameter names match (order matters)
	for i, pathParam := range pathParams {
		if i >= len(sigParams) {
			break
		}
		sigParam := sigParams[i]
		if pathParam != sigParam.Name {
			v.addWarning(location, fmt.Sprintf(
				"path parameter name mismatch at position %d: route has '{%s}' but handler parameter is '%s'",
				i, pathParam, sigParam.Name))
		}
	}

	// Validate parameter types are parseable
	for _, sigParam := range sigParams {
		if sigParam.Type == nil {
			continue
		}

		// Check if the type is supported for path parameters
		if !isValidPathParamType(sigParam.Type) {
			v.addError(location, fmt.Sprintf(
				"path parameter '%s' has unsupported type '%s'. Supported types: string, int, int64, uint64, float64, bool, uuid.UUID",
				sigParam.Name, sigParam.Type.FullName))
		}
	}
}

// extractPathParams extracts parameter names from a path pattern
// Example: "/posts/{id}/comments/{commentId}" -> ["id", "commentId"]
func extractPathParams(path string) []string {
	var params []string
	inParam := false
	var currentParam strings.Builder

	for _, ch := range path {
		switch ch {
		case '{':
			inParam = true
			currentParam.Reset()
		case '}':
			if inParam {
				params = append(params, currentParam.String())
				inParam = false
			}
		default:
			if inParam {
				currentParam.WriteRune(ch)
			}
		}
	}

	return params
}

// isValidPathParamType checks if a type is valid for path parameters
func isValidPathParamType(typeInfo *scanner.TypeInfo) bool {
	// Primitives
	if typeInfo.IsPrimitive {
		validPrimitives := map[string]bool{
			"string":  true,
			"int":     true,
			"int64":   true,
			"int32":   true,
			"uint":    true,
			"uint64":  true,
			"uint32":  true,
			"float64": true,
			"float32": true,
			"bool":    true,
		}
		return validPrimitives[typeInfo.Name]
	}

	// Special types
	if typeInfo.PackageName == "uuid" && typeInfo.Name == "UUID" {
		return true
	}

	return false
}

// validateQueryParams validates query parameters in handler signature
func (v *Validator) validateQueryParams(handler *scanner.Handler, location string) {
	// Skip validation for raw HTTP handlers
	if handler.Signature.Pattern == scanner.PatternRawHTTP {
		return
	}

	// Validate each query parameter type
	for _, qp := range handler.Signature.QueryParams {
		if qp.Type == nil {
			continue
		}

		// Check if type is supported for query parameters
		if !isValidQueryParamType(qp.Type) {
			v.addError(location, fmt.Sprintf(
				"query parameter '%s' (field %s) has unsupported type '%s'. "+
					"Supported types: string, int, int64, uint64, float64, bool, uuid.UUID, []string, and pointer variants (*string, *int, etc.)",
				qp.ParamName, qp.FieldName, qp.Type.FullName))
		}
	}
}

// isValidQueryParamType checks if a type is valid for query parameters
func isValidQueryParamType(typeInfo *scanner.TypeInfo) bool {
	// Handle pointer types
	actualType := typeInfo
	if typeInfo.IsPointer {
		// For pointers, we need to check the underlying type
		// Just check if it's a primitive or UUID for now
		// (We don't store underlying type in TypeInfo currently)
		return typeInfo.IsPrimitive || (typeInfo.PackageName == "uuid" && typeInfo.Name == "UUID")
	}

	// Handle slice types (e.g., []string)
	if typeInfo.IsSlice {
		// For now, only support []string
		return actualType.Name == "string"
	}

	// Primitives
	if actualType.IsPrimitive {
		validPrimitives := map[string]bool{
			"string":  true,
			"int":     true,
			"int64":   true,
			"uint64":  true,
			"float64": true,
			"bool":    true,
		}
		return validPrimitives[actualType.Name]
	}

	// UUID
	if actualType.PackageName == "uuid" && actualType.Name == "UUID" {
		return true
	}

	// time.Time (future support)
	if actualType.PackageName == "time" && actualType.Name == "Time" {
		return true
	}

	return false
}

// validateHeaderParams validates header parameters in handler signature
func (v *Validator) validateHeaderParams(handler *scanner.Handler, location string) {
	// Skip validation for raw HTTP handlers
	if handler.Signature.Pattern == scanner.PatternRawHTTP {
		return
	}

	// Validate each header parameter type
	for _, hp := range handler.Signature.HeaderParams {
		if hp.Type == nil {
			continue
		}

		// Check if type is supported for headers
		if !isValidHeaderParamType(hp.Type) {
			v.addError(location, fmt.Sprintf(
				"header parameter '%s' (field %s) has unsupported type '%s'. "+
					"Supported types: string, *string",
				hp.HeaderName, hp.FieldName, hp.Type.FullName))
		}
	}
}

// isValidHeaderParamType checks if a type is valid for header parameters
func isValidHeaderParamType(typeInfo *scanner.TypeInfo) bool {
	// Headers are always strings or *string
	if typeInfo.IsPointer {
		return typeInfo.Name == "string"
	}
	return typeInfo.IsPrimitive && typeInfo.Name == "string"
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

	// Validate target
	if mw.Target == "" {
		v.addError(location, "middleware target cannot be empty")
	}

	// Validate order
	if mw.Order < 0 {
		v.addError(location, fmt.Sprintf("middleware order must be non-negative: %d", mw.Order))
	}

	// Validate signature
	if mw.Signature != "chi" && mw.Signature != "glib" {
		v.addError(location, fmt.Sprintf("invalid middleware signature type: %s (must be 'chi' or 'glib')", mw.Signature))
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

// validateDependencies performs dependency validation including circular dependency detection
func (v *Validator) validateDependencies(project *scanner.Project) {
	// Build a map of available types from providers
	providedTypes := make(map[string]bool)
	providersByType := make(map[string]*scanner.Provider)
	for _, prov := range project.Providers {
		if prov.ReturnType != nil {
			providedTypes[prov.ReturnType.FullName] = true
			providersByType[prov.ReturnType.FullName] = prov
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

	// Check for circular dependencies in providers using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, prov := range project.Providers {
		if prov.ReturnType == nil {
			continue
		}
		if !visited[prov.ReturnType.FullName] {
			if v.detectCircularDeps(prov, providersByType, visited, recStack, []string{}) {
				// Error already added in detectCircularDeps
				continue
			}
		}
	}
}

// detectCircularDeps performs DFS to detect circular dependencies
func (v *Validator) detectCircularDeps(prov *scanner.Provider, providersByType map[string]*scanner.Provider, visited, recStack map[string]bool, path []string) bool {
	if prov.ReturnType == nil {
		return false
	}

	typeName := prov.ReturnType.FullName
	visited[typeName] = true
	recStack[typeName] = true
	path = append(path, typeName)

	// Check all dependencies
	for _, dep := range prov.Dependencies {
		if dep.Type == nil || dep.Type.IsPrimitive {
			continue
		}

		depTypeName := dep.Type.FullName

		// If dependency is in recursion stack, we found a cycle
		if recStack[depTypeName] {
			// Find where the cycle starts
			cycleStart := -1
			for i, t := range path {
				if t == depTypeName {
					cycleStart = i
					break
				}
			}

			cyclePath := append(path[cycleStart:], depTypeName)
			location := fmt.Sprintf("%s:%d", prov.FilePath, prov.Position.Line)
			v.addError(location, fmt.Sprintf("circular dependency detected: %s", strings.Join(cyclePath, " -> ")))
			recStack[typeName] = false
			return true
		}

		// Recursively check dependency
		if depProvider, exists := providersByType[depTypeName]; exists {
			if !visited[depTypeName] {
				if v.detectCircularDeps(depProvider, providersByType, visited, recStack, path) {
					recStack[typeName] = false
					return true
				}
			}
		}
	}

	recStack[typeName] = false
	return false
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

// validateMiddlewareReferences checks that all middleware references exist
func (v *Validator) validateMiddlewareReferences(project *scanner.Project) {
	// Build map of available middleware
	availableMiddleware := make(map[string]bool)
	for _, mw := range project.Middleware {
		availableMiddleware[mw.Name] = true
	}

	// Build map of all tags used in the project
	allTags := make(map[string]bool)
	for _, ctrl := range project.Controllers {
		for _, tag := range ctrl.Tags {
			allTags[tag] = true
		}
		for _, handler := range ctrl.Handlers {
			for _, tag := range handler.Tags {
				allTags[tag] = true
			}
		}
	}

	// Validate middleware target references
	for _, mw := range project.Middleware {
		location := fmt.Sprintf("%s:%d", mw.FilePath, mw.Position.Line)

		// Skip "all" target
		if mw.Target == "all" {
			continue
		}

		// Split comma-separated targets
		targets := strings.SplitSeq(mw.Target, ",")
		for target := range targets {
			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}
			if target != "all" && !allTags[target] {
				v.addWarning(location, fmt.Sprintf("middleware '%s' targets tag '%s' which is not used by any controller or handler", mw.Name, target))
			}
		}
	}

	// Check handler @With references
	for _, ctrl := range project.Controllers {
		for _, handler := range ctrl.Handlers {
			handlerLocation := fmt.Sprintf("%s:%d", ctrl.FilePath, handler.Position.Line)

			// Skip if no explicit With annotation
			if len(handler.With) == 0 {
				continue
			}

			// Check for "none" keyword
			if len(handler.With) == 1 && handler.With[0] == "none" {
				continue
			}

			// Validate each middleware reference
			for _, mwName := range handler.With {
				if mwName == "" {
					v.addError(handlerLocation, "empty middleware name in @With")
					continue
				}
				if !availableMiddleware[mwName] {
					v.addError(handlerLocation, fmt.Sprintf("handler references undefined middleware '%s' in @With", mwName))
				}
			}
		}
	}
}
