package validator

import (
	"github.com/azizndao/glib/internal/scanner"
)

// ValidationStats tracks validation statistics
type ValidationStats struct {
	ComponentsValidated int // Total components validated
	CacheHits           int // Components loaded from cache
	CacheMisses         int // Components validated (not cached)
}

// IncrementalValidator validates only changed components
type IncrementalValidator struct {
	cache     *ValidationCache
	validator *Validator
	stats     ValidationStats
}

// NewIncrementalValidator creates a validator with caching support
func NewIncrementalValidator(cacheDir string) *IncrementalValidator {
	cache := NewValidationCache(cacheDir)
	if err := cache.Load(); err != nil {
		// Log error but continue - cache will be rebuilt
		// Don't fail initialization due to cache load failure
	}

	return &IncrementalValidator{
		cache:     cache,
		validator: New(),
	}
}

// ValidateIncremental validates the project, using cached results when possible
func (iv *IncrementalValidator) ValidateIncremental(project *scanner.Project) error {
	// Track which components need validation
	needsValidation := make(map[string]bool)
	componentMap := make(map[string]any)

	// Check all providers
	for _, provider := range project.Providers {
		id := componentID("provider", provider.PackagePath, provider.Name)
		hash, deps := computeComponentHash(provider, project)

		componentMap[id] = provider

		// Check cache
		iv.stats.ComponentsValidated++
		if cached, ok := iv.cache.Get(id, hash); ok {
			// Use cached validation
			iv.stats.CacheHits++
			for _, err := range cached.Errors {
				iv.validator.errors = append(iv.validator.errors, err)
			}
			for _, warn := range cached.Warnings {
				iv.validator.warnings = append(iv.validator.warnings, warn)
			}
		} else {
			// Needs validation
			iv.stats.CacheMisses++
			needsValidation[id] = true

			// If dependency changed, invalidate cache
			iv.cache.Invalidate(id)

			// Validate this provider
			iv.validator.validateProvider(provider)

			// Cache the results
			iv.cache.Set(&CachedValidation{
				ComponentID:   id,
				ComponentType: "provider",
				Hash:          hash,
				Errors:        iv.validator.errors,
				Warnings:      iv.validator.warnings,
				Dependencies:  deps,
			})
		}
	}

	// Check all controllers
	for _, controller := range project.Controllers {
		id := componentID("controller", controller.PackagePath, controller.Name)
		hash, deps := computeComponentHash(controller, project)

		componentMap[id] = controller

		iv.stats.ComponentsValidated++
		if cached, ok := iv.cache.Get(id, hash); ok {
			// Use cached validation
			iv.stats.CacheHits++
			for _, err := range cached.Errors {
				iv.validator.errors = append(iv.validator.errors, err)
			}
			for _, warn := range cached.Warnings {
				iv.validator.warnings = append(iv.validator.warnings, warn)
			}
		} else {
			iv.stats.CacheMisses++
			needsValidation[id] = true
			iv.cache.Invalidate(id)

			// Validate this controller
			iv.validator.validateController(controller)

			// Cache the results
			iv.cache.Set(&CachedValidation{
				ComponentID:   id,
				ComponentType: "controller",
				Hash:          hash,
				Errors:        iv.validator.errors,
				Warnings:      iv.validator.warnings,
				Dependencies:  deps,
			})
		}
	}

	// Check all middleware
	for _, middleware := range project.Middleware {
		id := componentID("middleware", middleware.PackagePath, middleware.Name)
		hash, deps := computeComponentHash(middleware, project)

		componentMap[id] = middleware

		iv.stats.ComponentsValidated++
		if cached, ok := iv.cache.Get(id, hash); ok {
			// Use cached validation
			iv.stats.CacheHits++
			for _, err := range cached.Errors {
				iv.validator.errors = append(iv.validator.errors, err)
			}
			for _, warn := range cached.Warnings {
				iv.validator.warnings = append(iv.validator.warnings, warn)
			}
		} else {
			iv.stats.CacheMisses++
			needsValidation[id] = true
			iv.cache.Invalidate(id)

			// Validate this middleware
			iv.validator.validateMiddleware(middleware)

			// Cache the results
			iv.cache.Set(&CachedValidation{
				ComponentID:   id,
				ComponentType: "middleware",
				Hash:          hash,
				Errors:        iv.validator.errors,
				Warnings:      iv.validator.warnings,
				Dependencies:  deps,
			})
		}
	}

	// Always validate cross-component concerns (routes, dependencies, middleware refs)
	// These can't be easily cached since they depend on entire project state
	iv.validator.validateUniqueRoutes(project.Controllers)
	iv.validator.validateDependencies(project)
	iv.validator.validateMiddlewareReferences(project)

	// Save cache
	iv.cache.Save()

	// Return errors if any
	if len(iv.validator.errors) > 0 {
		return &ValidationErrors{Errors: iv.validator.errors}
	}

	return nil
}

// GetWarnings returns all validation warnings
func (iv *IncrementalValidator) GetWarnings() []*ValidationError {
	return iv.validator.warnings
}

// GetErrors returns all validation errors
func (iv *IncrementalValidator) GetErrors() []*ValidationError {
	return iv.validator.errors
}

// ClearCache clears the validation cache
func (iv *IncrementalValidator) ClearCache() {
	iv.cache.Clear()
}

// InvalidateComponent invalidates a specific component and its dependents
func (iv *IncrementalValidator) InvalidateComponent(componentType, packagePath, name string) []string {
	id := componentID(componentType, packagePath, name)
	return iv.cache.Invalidate(id)
}

// ValidationErrors wraps multiple validation errors
type ValidationErrors struct {
	Errors []*ValidationError
}

func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}
	return ve.Errors[0].Error()
}

// Stats returns validation statistics
func (iv *IncrementalValidator) Stats() ValidationStats {
	return iv.stats
}
