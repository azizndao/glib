package generator

import (
	"fmt"
	"strings"

	"github.com/azizndao/glib/internal/scanner"
)

// DependencyGraph represents the dependency relationships between providers
type DependencyGraph struct {
	Providers       []*scanner.Provider
	ProvidersByType map[string]*scanner.Provider
	Controllers     []*scanner.Controller

	// Graph structure
	Dependencies map[string][]string // provider type -> dependency types
	Dependents   map[string][]string // provider type -> types that depend on it
}

// InitializationPlan defines the order in which providers should be initialized
type InitializationPlan struct {
	CriticalTransients    []*scanner.Provider // Transients used by singletons (must init first)
	Singletons            []*scanner.Provider // Singletons in dependency order
	NonCriticalTransients []*scanner.Provider // Transients only used by controllers
}

// NewDependencyGraph builds a dependency graph from a scanned project
func NewDependencyGraph(project *scanner.Project) *DependencyGraph {
	g := &DependencyGraph{
		Providers:       project.Providers,
		Controllers:     project.Controllers,
		ProvidersByType: make(map[string]*scanner.Provider),
		Dependencies:    make(map[string][]string),
		Dependents:      make(map[string][]string),
	}

	// Build provider lookup map
	for _, prov := range project.Providers {
		if prov.ReturnType != nil {
			g.ProvidersByType[prov.ReturnType.FullName] = prov
		}
	}

	// Build dependency edges
	for _, prov := range project.Providers {
		if prov.ReturnType == nil {
			continue
		}

		provType := prov.ReturnType.FullName

		for _, dep := range prov.Dependencies {
			if dep.Type == nil || dep.Type.IsPrimitive {
				continue
			}

			depType := dep.Type.FullName

			// Add dependency edge: prov depends on dep
			g.Dependencies[provType] = append(g.Dependencies[provType], depType)

			// Add reverse edge: dep is depended on by prov
			g.Dependents[depType] = append(g.Dependents[depType], provType)
		}
	}

	return g
}

// AnalyzeUsage analyzes the dependency graph and produces an initialization plan
func (g *DependencyGraph) AnalyzeUsage() *InitializationPlan {
	plan := &InitializationPlan{}

	// Separate providers by lifecycle
	var singletons []*scanner.Provider
	var transients []*scanner.Provider

	for _, prov := range g.Providers {
		switch prov.Lifecycle {
		case "singleton":
			singletons = append(singletons, prov)
		case "transient":
			transients = append(transients, prov)
		}
	}

	// Identify critical transients (used directly or transitively by singletons)
	criticalTransientTypes := g.findCriticalTransients(singletons)

	// Separate transients into critical and non-critical
	for _, trans := range transients {
		if trans.ReturnType == nil {
			continue
		}

		if criticalTransientTypes[trans.ReturnType.FullName] {
			plan.CriticalTransients = append(plan.CriticalTransients, trans)
		} else {
			plan.NonCriticalTransients = append(plan.NonCriticalTransients, trans)
		}
	}

	// Sort critical transients by dependencies
	plan.CriticalTransients = g.topologicalSort(plan.CriticalTransients)

	// Sort singletons by dependencies
	plan.Singletons = g.topologicalSort(singletons)

	// Non-critical transients don't need sorting (no interdependencies matter)
	// But we'll sort them anyway for consistency
	plan.NonCriticalTransients = g.topologicalSort(plan.NonCriticalTransients)

	return plan
}

// findCriticalTransients identifies transient providers that are used (directly or transitively) by singletons
func (g *DependencyGraph) findCriticalTransients(singletons []*scanner.Provider) map[string]bool {
	critical := make(map[string]bool)
	visited := make(map[string]bool)

	// Helper function to recursively mark transitive dependencies
	var markTransitiveDeps func(provType string)
	markTransitiveDeps = func(provType string) {
		if visited[provType] {
			return
		}
		visited[provType] = true

		// Check if this provider is a transient
		if prov, exists := g.ProvidersByType[provType]; exists {
			if prov.Lifecycle == "transient" {
				critical[provType] = true
			}

			// Recursively check dependencies of this provider
			for _, depType := range g.Dependencies[provType] {
				markTransitiveDeps(depType)
			}
		}
	}

	// Start from each singleton and mark all transient dependencies
	for _, singleton := range singletons {
		if singleton.ReturnType == nil {
			continue
		}

		singletonType := singleton.ReturnType.FullName

		// Check all dependencies of this singleton
		for _, depType := range g.Dependencies[singletonType] {
			markTransitiveDeps(depType)
		}
	}

	return critical
}

// topologicalSort performs a topological sort on the given providers
// Returns providers in dependency order (dependencies first)
// This uses the global provider map to resolve dependencies, including transitive dependencies
func (g *DependencyGraph) topologicalSort(providers []*scanner.Provider) []*scanner.Provider {
	// Build a set of providers we're sorting (for filtering result)
	targetProviders := make(map[string]bool)
	for _, prov := range providers {
		if prov.ReturnType != nil {
			targetProviders[prov.ReturnType.FullName] = true
		}
	}

	// Track visited providers and build result
	visited := make(map[string]bool)
	var result []*scanner.Provider

	// Helper to find all dependencies in our target set
	// This handles both direct dependencies and transitive dependencies through other lifecycles
	var findTargetDeps func(typeName string) []*scanner.Provider
	findTargetDeps = func(typeName string) []*scanner.Provider {
		var deps []*scanner.Provider

		provider := g.ProvidersByType[typeName]
		if provider == nil {
			return deps
		}

		for _, dep := range provider.Dependencies {
			if dep.Type == nil || dep.Type.IsPrimitive {
				continue
			}

			depProvider := g.ProvidersByType[dep.Type.FullName]
			if depProvider == nil {
				continue
			}

			// If it's in our target set, add it
			if targetProviders[dep.Type.FullName] {
				deps = append(deps, depProvider)
			}

			// If it's not in our target set but exists, check its dependencies
			// This handles cases like: Singleton A -> Transient B -> Singleton C
			// When sorting singletons, we need C to come before A
			if !targetProviders[dep.Type.FullName] {
				transitiveDeps := findTargetDeps(dep.Type.FullName)
				deps = append(deps, transitiveDeps...)
			}
		}

		return deps
	}

	// Helper function for DFS
	var visit func(*scanner.Provider)
	visit = func(prov *scanner.Provider) {
		if prov.ReturnType == nil {
			return
		}

		typeName := prov.ReturnType.FullName
		if visited[typeName] {
			return
		}

		// Visit dependencies in our target set first
		deps := findTargetDeps(typeName)
		for _, dep := range deps {
			if !visited[dep.ReturnType.FullName] {
				visit(dep)
			}
		}

		// Mark as visited and add to result
		visited[typeName] = true
		result = append(result, prov)
	}

	// Visit all providers
	for _, prov := range providers {
		visit(prov)
	}

	return result
}

// IsUsedBySingleton checks if a transient provider is used by any singleton
func (g *DependencyGraph) IsUsedBySingleton(transient *scanner.Provider) bool {
	if transient.ReturnType == nil {
		return false
	}

	transientType := transient.ReturnType.FullName

	// Check all dependents of this transient
	for _, dependentType := range g.Dependents[transientType] {
		dependent := g.ProvidersByType[dependentType]
		if dependent != nil && dependent.Lifecycle == "singleton" {
			return true
		}

		// Recursively check if any dependent is used by a singleton
		if dependent != nil && dependent.Lifecycle == "transient" {
			if g.IsUsedBySingleton(dependent) {
				return true
			}
		}
	}

	return false
}

// DebugString returns a debug representation of the dependency graph
func (g *DependencyGraph) DebugString() string {
	var s strings.Builder
	s.WriteString("Dependency Graph:\n")
	fmt.Fprintf(&s, "  Providers: %d\n", len(g.Providers))
	s.WriteString("  Dependencies:\n")

	for provType, deps := range g.Dependencies {
		fmt.Fprintf(&s, "    %s ->\n", provType)
		for _, dep := range deps {
			fmt.Fprintf(&s, "      - %s\n", dep)
		}
	}

	return s.String()
}

// DebugPlan returns a debug representation of an initialization plan
func (plan *InitializationPlan) DebugString() string {
	var s strings.Builder
	s.WriteString("Initialization Plan:\n")
	fmt.Fprintf(&s, "  Phase 1 - Critical Transients: %d\n", len(plan.CriticalTransients))
	for i, prov := range plan.CriticalTransients {
		fmt.Fprintf(&s, "    %d. %s (%s)\n", i+1, prov.Name, prov.ReturnType.FullName)
	}

	fmt.Fprintf(&s, "  Phase 2 - Singletons: %d\n", len(plan.Singletons))
	for i, prov := range plan.Singletons {
		fmt.Fprintf(&s, "    %d. %s (%s)\n", i+1, prov.Name, prov.ReturnType.FullName)
	}

	fmt.Fprintf(&s, "  Phase 3 - Non-Critical Transients: %d\n", len(plan.NonCriticalTransients))
	for i, prov := range plan.NonCriticalTransients {
		fmt.Fprintf(&s, "    %d. %s (%s)\n", i+1, prov.Name, prov.ReturnType.FullName)
	}

	return s.String()
}
