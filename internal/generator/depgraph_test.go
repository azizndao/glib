package generator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

// Test 1: Simple case - transient used by singleton
func TestCriticalTransient_Simple(t *testing.T) {
	// Setup: Singleton A -> Transient B
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "b",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientB",
						},
					},
				},
			},
			{
				Name:      "NewTransientB",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientB",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: B should be in critical transients
	if len(plan.CriticalTransients) != 1 {
		t.Errorf("Expected 1 critical transient, got %d", len(plan.CriticalTransients))
	}
	if len(plan.CriticalTransients) > 0 && plan.CriticalTransients[0].Name != "NewTransientB" {
		t.Errorf("Expected NewTransientB in critical transients, got %s", plan.CriticalTransients[0].Name)
	}

	// Verify: A should be in singletons
	if len(plan.Singletons) != 1 {
		t.Errorf("Expected 1 singleton, got %d", len(plan.Singletons))
	}
	if len(plan.Singletons) > 0 && plan.Singletons[0].Name != "NewSingletonA" {
		t.Errorf("Expected NewSingletonA in singletons, got %s", plan.Singletons[0].Name)
	}

	// Verify: No non-critical transients
	if len(plan.NonCriticalTransients) != 0 {
		t.Errorf("Expected 0 non-critical transients, got %d", len(plan.NonCriticalTransients))
	}
}

// Test 2: Complex chain - transient depends on singleton
func TestCriticalTransient_Chain(t *testing.T) {
	// Setup: Singleton A -> Transient B -> Singleton C
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "b",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientB",
						},
					},
				},
			},
			{
				Name:      "NewTransientB",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientB",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "c",
						Type: &scanner.TypeInfo{
							FullName: "*services.SingletonC",
						},
					},
				},
			},
			{
				Name:      "NewSingletonC",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.SingletonC",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: B should be in critical transients
	if len(plan.CriticalTransients) != 1 {
		t.Errorf("Expected 1 critical transient, got %d", len(plan.CriticalTransients))
	}

	// Verify: A and C should be in singletons
	if len(plan.Singletons) != 2 {
		t.Errorf("Expected 2 singletons, got %d", len(plan.Singletons))
	}

	// Verify order: C should come before A (A depends on B which depends on C)
	if len(plan.Singletons) == 2 {
		if plan.Singletons[0].Name != "NewSingletonC" {
			t.Errorf("Expected NewSingletonC first, got %s", plan.Singletons[0].Name)
		}
		if plan.Singletons[1].Name != "NewSingletonA" {
			t.Errorf("Expected NewSingletonA second, got %s", plan.Singletons[1].Name)
		}
	}
}

// Test 3: Transient depends on another transient
func TestCriticalTransient_TransientChain(t *testing.T) {
	// Setup: Singleton A -> Transient B -> Transient C
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "b",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientB",
						},
					},
				},
			},
			{
				Name:      "NewTransientB",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientB",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "c",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientC",
						},
					},
				},
			},
			{
				Name:      "NewTransientC",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientC",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: Both B and C should be in critical transients
	if len(plan.CriticalTransients) != 2 {
		t.Errorf("Expected 2 critical transients, got %d", len(plan.CriticalTransients))
	}

	// Verify order: C should come before B (B depends on C)
	if len(plan.CriticalTransients) == 2 {
		if plan.CriticalTransients[0].Name != "NewTransientC" {
			t.Errorf("Expected NewTransientC first, got %s", plan.CriticalTransients[0].Name)
		}
		if plan.CriticalTransients[1].Name != "NewTransientB" {
			t.Errorf("Expected NewTransientB second, got %s", plan.CriticalTransients[1].Name)
		}
	}

	// Verify: A should be in singletons
	if len(plan.Singletons) != 1 {
		t.Errorf("Expected 1 singleton, got %d", len(plan.Singletons))
	}
}

// Test 4: Non-critical transient (only used by controller)
func TestNonCriticalTransient(t *testing.T) {
	// Setup: No singleton uses TransientA
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonB",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonB",
				},
				Dependencies: nil,
			},
			{
				Name:      "NewTransientA",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientA",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: A should be in non-critical transients
	if len(plan.NonCriticalTransients) != 1 {
		t.Errorf("Expected 1 non-critical transient, got %d", len(plan.NonCriticalTransients))
	}
	if len(plan.NonCriticalTransients) > 0 && plan.NonCriticalTransients[0].Name != "NewTransientA" {
		t.Errorf("Expected NewTransientA in non-critical transients, got %s", plan.NonCriticalTransients[0].Name)
	}

	// Verify: No critical transients
	if len(plan.CriticalTransients) != 0 {
		t.Errorf("Expected 0 critical transients, got %d", len(plan.CriticalTransients))
	}
}

// Test 5: Mixed usage - transient used by both singleton and controller
func TestMixedUsage(t *testing.T) {
	// Setup: Singleton A -> Transient B (controller also uses B)
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "b",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientB",
						},
					},
				},
			},
			{
				Name:      "NewTransientB",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientB",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: B should be in critical (because singleton uses it)
	if len(plan.CriticalTransients) != 1 {
		t.Errorf("Expected 1 critical transient, got %d", len(plan.CriticalTransients))
	}
	if len(plan.CriticalTransients) > 0 && plan.CriticalTransients[0].Name != "NewTransientB" {
		t.Errorf("Expected NewTransientB in critical transients, got %s", plan.CriticalTransients[0].Name)
	}

	// Verify: Not in non-critical
	if len(plan.NonCriticalTransients) != 0 {
		t.Errorf("Expected 0 non-critical transients, got %d", len(plan.NonCriticalTransients))
	}
}

// Test 6: Diamond dependency
func TestDiamondDependency(t *testing.T) {
	// Setup:
	// Singleton A -> Transient B -> Singleton D
	// Singleton A -> Singleton C -> Singleton D
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "b",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientB",
						},
					},
					{
						Name: "c",
						Type: &scanner.TypeInfo{
							FullName: "*services.SingletonC",
						},
					},
				},
			},
			{
				Name:      "NewTransientB",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientB",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "d",
						Type: &scanner.TypeInfo{
							FullName: "*services.SingletonD",
						},
					},
				},
			},
			{
				Name:      "NewSingletonC",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.SingletonC",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "d",
						Type: &scanner.TypeInfo{
							FullName: "*services.SingletonD",
						},
					},
				},
			},
			{
				Name:      "NewSingletonD",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.SingletonD",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: B should be in critical transients
	if len(plan.CriticalTransients) != 1 {
		t.Errorf("Expected 1 critical transient, got %d", len(plan.CriticalTransients))
	}

	// Verify: A, C, D should be in singletons
	if len(plan.Singletons) != 3 {
		t.Errorf("Expected 3 singletons, got %d", len(plan.Singletons))
	}

	// Verify order: D should come first (no dependencies)
	// Then C (depends on D), then A (depends on B and C)
	if len(plan.Singletons) == 3 {
		if plan.Singletons[0].Name != "NewSingletonD" {
			t.Errorf("Expected NewSingletonD first, got %s", plan.Singletons[0].Name)
		}
		// C should come before A
		cIndex := -1
		aIndex := -1
		for i, prov := range plan.Singletons {
			if prov.Name == "NewSingletonC" {
				cIndex = i
			}
			if prov.Name == "NewSingletonA" {
				aIndex = i
			}
		}
		if cIndex > aIndex {
			t.Errorf("Expected NewSingletonC before NewSingletonA, got C at %d and A at %d", cIndex, aIndex)
		}
	}
}

// Test 7: IsUsedBySingleton method
func TestIsUsedBySingleton(t *testing.T) {
	// Setup: Singleton A -> Transient B
	//        Transient C (not used by any singleton)
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "b",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientB",
						},
					},
				},
			},
			{
				Name:      "NewTransientB",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientB",
				},
				Dependencies: nil,
			},
			{
				Name:      "NewTransientC",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientC",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)

	// Test: B should be used by singleton
	transientB := project.Providers[1]
	if !graph.IsUsedBySingleton(transientB) {
		t.Errorf("Expected TransientB to be used by singleton")
	}

	// Test: C should NOT be used by singleton
	transientC := project.Providers[2]
	if graph.IsUsedBySingleton(transientC) {
		t.Errorf("Expected TransientC NOT to be used by singleton")
	}
}

// Test 8: Empty project
func TestEmptyProject(t *testing.T) {
	project := &scanner.Project{
		Providers: []*scanner.Provider{},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	if len(plan.CriticalTransients) != 0 {
		t.Errorf("Expected 0 critical transients, got %d", len(plan.CriticalTransients))
	}
	if len(plan.Singletons) != 0 {
		t.Errorf("Expected 0 singletons, got %d", len(plan.Singletons))
	}
	if len(plan.NonCriticalTransients) != 0 {
		t.Errorf("Expected 0 non-critical transients, got %d", len(plan.NonCriticalTransients))
	}
}

// Test 9: Multiple singletons use same transient
func TestMultipleSingletonsUseSameTransient(t *testing.T) {
	// Setup: Singleton A -> Transient C
	//        Singleton B -> Transient C
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:      "NewSingletonA",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonA",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "c",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientC",
						},
					},
				},
			},
			{
				Name:      "NewSingletonB",
				Lifecycle: "singleton",
				ReturnType: &scanner.TypeInfo{
					FullName: "services.SingletonB",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "c",
						Type: &scanner.TypeInfo{
							FullName: "*services.TransientC",
						},
					},
				},
			},
			{
				Name:      "NewTransientC",
				Lifecycle: "transient",
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.TransientC",
				},
				Dependencies: nil,
			},
		},
	}

	graph := NewDependencyGraph(project)
	plan := graph.AnalyzeUsage()

	// Verify: C should be in critical transients (once)
	if len(plan.CriticalTransients) != 1 {
		t.Errorf("Expected 1 critical transient, got %d", len(plan.CriticalTransients))
	}
	if len(plan.CriticalTransients) > 0 && plan.CriticalTransients[0].Name != "NewTransientC" {
		t.Errorf("Expected NewTransientC in critical transients, got %s", plan.CriticalTransients[0].Name)
	}

	// Verify: A and B should be in singletons
	if len(plan.Singletons) != 2 {
		t.Errorf("Expected 2 singletons, got %d", len(plan.Singletons))
	}
}
