package validator

import (
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/azizndao/glib/internal/scanner"
)

// ValidationCache caches validation results to avoid re-validating unchanged components
type ValidationCache struct {
	mu         sync.RWMutex
	cacheDir   string
	components map[string]*CachedValidation // componentID -> validation
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
	return &ValidationCache{
		cacheDir:   cacheDir,
		components: make(map[string]*CachedValidation),
	}
}

// Load loads the cache from disk
func (vc *ValidationCache) Load() error {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	cachePath := filepath.Join(vc.cacheDir, "validation.cache")
	file, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(&vc.components)
}

// Save saves the cache to disk
func (vc *ValidationCache) Save() error {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if err := os.MkdirAll(vc.cacheDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(vc.cacheDir, "validation.cache")
	file, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(vc.components)
}

// Get retrieves cached validation for a component if it matches the current hash
func (vc *ValidationCache) Get(componentID, currentHash string) (*CachedValidation, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	cached, ok := vc.components[componentID]
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
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.components[cached.ComponentID] = cached
}

// Invalidate removes a component and all components that depend on it
func (vc *ValidationCache) Invalidate(componentID string) []string {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	invalidated := []string{componentID}
	delete(vc.components, componentID)

	// Find all components that depend on this one
	for id, cached := range vc.components {
		for _, dep := range cached.Dependencies {
			if dep == componentID {
				invalidated = append(invalidated, id)
				delete(vc.components, id)
				break
			}
		}
	}

	return invalidated
}

// Clear removes all cached validations
func (vc *ValidationCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.components = make(map[string]*CachedValidation)
}

// componentID generates a unique ID for a component
func componentID(componentType, packagePath, name string) string {
	return fmt.Sprintf("%s:%s/%s", componentType, packagePath, name)
}

// computeComponentHash computes a hash of a component and its dependencies
func computeComponentHash(component interface{}, project *scanner.Project) (string, []string) {
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
