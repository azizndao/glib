package validator

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/azizndao/glib/internal/cache"
	"github.com/azizndao/glib/internal/scanner"
)

// ValidationCache caches validation results to avoid re-validating unchanged components
type ValidationCache struct {
	*cache.Cache[string, *CachedValidation] // Embedded generic cache
	cacheDir                                string
}

// CachedValidation stores validation results for a component
type CachedValidation struct {
	ComponentID   string             // Unique ID for this component
	ComponentType string             // "provider", "controller", "middleware"
	Hash          string             // Hash of component + dependencies
	Errors        []*ValidationError // Validation errors found
	Warnings      []*ValidationError // Validation warnings found
	Dependencies  []string           // IDs of components this depends on
}

// NewValidationCache creates a new validation cache
func NewValidationCache(cacheDir string) *ValidationCache {
	cachePath := filepath.Join(cacheDir, "validation.cache")
	return &ValidationCache{
		Cache:    cache.New[string, *CachedValidation](cachePath),
		cacheDir: cacheDir,
	}
}

// Get retrieves cached validation for a component if it matches the current hash
func (vc *ValidationCache) Get(componentID, currentHash string) (*CachedValidation, bool) {
	cached, ok := vc.Cache.Get(componentID)
	if !ok {
		return nil, false
	}

	// Check if hash matches (component unchanged)
	if cached.Hash != currentHash {
		return nil, false
	}

	return cached, true
}

// Set stores validation results for a component
func (vc *ValidationCache) Set(cached *CachedValidation) {
	vc.Cache.Set(cached.ComponentID, cached)
}

// Invalidate removes a component and all components that depend on it
func (vc *ValidationCache) Invalidate(componentID string) []string {
	invalidated := []string{componentID}
	vc.Delete(componentID)

	// Find all components that depend on this one (collect IDs first to avoid lock issues)
	var toDelete []string
	vc.ForEach(func(id string, cached *CachedValidation) {
		if slices.Contains(cached.Dependencies, componentID) {
			toDelete = append(toDelete, id)
		}
	})

	// Delete dependent components
	for _, id := range toDelete {
		invalidated = append(invalidated, id)
		vc.Delete(id)
	}

	return invalidated
}

// componentID generates a unique ID for a component
func componentID(componentType, packagePath, name string) string {
	return fmt.Sprintf("%s:%s/%s", componentType, packagePath, name)
}

// computeComponentHash computes a hash of a component and its dependencies
func computeComponentHash(component any, project *scanner.Project) (string, []string) {
	hasher := sha256.New()
	var dependencies []string

	switch c := component.(type) {
	case *scanner.Provider:
		// Hash the provider definition
		fmt.Fprintf(hasher, "provider:%s:%s:%s", c.PackagePath, c.Name, c.Lifecycle)

		// Hash return type
		if c.ReturnType != nil {
			fmt.Fprintf(hasher, ":return:%s", c.ReturnType.FullName)
		}

		// Hash dependencies
		for _, dep := range c.Dependencies {
			if dep.Type != nil {
				fmt.Fprintf(hasher, ":dep:%s", dep.Type.FullName)

				// Track dependency IDs
				if provider := findProviderForType(project, dep.Type); provider != nil {
					dependencies = append(dependencies,
						componentID("provider", provider.PackagePath, provider.Name))
				}
			}
		}

	case *scanner.Controller:
		// Hash the controller definition
		fmt.Fprintf(hasher, "controller:%s:%s:%s", c.PackagePath, c.Name, c.RoutePrefix)

		// Hash fields (dependencies)
		for _, field := range c.Fields {
			if field.Type != nil {
				fmt.Fprintf(hasher, ":field:%s:%s", field.Name, field.Type.FullName)

				// Track dependency IDs
				if provider := findProviderForType(project, field.Type); provider != nil {
					dependencies = append(dependencies,
						componentID("provider", provider.PackagePath, provider.Name))
				}
			}
		}

		// Hash handlers
		for _, handler := range c.Handlers {
			fmt.Fprintf(hasher, ":handler:%s:%s:%s", handler.Name, handler.Method, handler.Path)
		}

	case *scanner.Middleware:
		// Hash the middleware definition
		fmt.Fprintf(hasher, "middleware:%s:%s", c.PackagePath, c.Name)

		// Hash dependencies
		for _, dep := range c.Dependencies {
			if dep.Type != nil {
				fmt.Fprintf(hasher, ":dep:%s", dep.Type.FullName)

				if provider := findProviderForType(project, dep.Type); provider != nil {
					dependencies = append(dependencies,
						componentID("provider", provider.PackagePath, provider.Name))
				}
			}
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), dependencies
}

// findProviderForType finds the provider that provides a given type
func findProviderForType(project *scanner.Project, typeInfo *scanner.TypeInfo) *scanner.Provider {
	if typeInfo == nil {
		return nil
	}

	for _, provider := range project.Providers {
		if provider.ReturnType != nil && provider.ReturnType.FullName == typeInfo.FullName {
			return provider
		}
	}

	return nil
}
