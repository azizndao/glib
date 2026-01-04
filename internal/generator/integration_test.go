package generator

import (
	"strings"
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

// TestIntegration_SingletonDependsOnTransient tests the real-world scenario
// where a singleton provider depends on a transient provider
func TestIntegration_SingletonDependsOnTransient(t *testing.T) {
	// Simulate the PostService -> Auditor scenario
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			// Database (singleton, no dependencies)
			{
				Name:         "NewDatabase",
				FunctionName: "NewDatabase",
				PackageName:  "services",
				PackagePath:  "app/services",
				Lifecycle:    scanner.LifecycleSingleton,
				ReturnType: &scanner.TypeInfo{
					FullName: "*gorm.DB",
				},
				Dependencies: nil,
			},
			// UserService (singleton, depends on Database)
			{
				Name:         "NewUserService",
				FunctionName: "NewUserService",
				PackageName:  "services",
				PackagePath:  "app/services",
				Lifecycle:    scanner.LifecycleSingleton,
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.UserService",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "db",
						Type: &scanner.TypeInfo{
							FullName: "*gorm.DB",
						},
					},
				},
			},
			// Auditor (transient, depends on UserService)
			{
				Name:         "NewAuditor",
				FunctionName: "NewAuditor",
				PackageName:  "services",
				PackagePath:  "app/services",
				Lifecycle:    scanner.LifecycleTransient,
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.Auditor",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "userService",
						Type: &scanner.TypeInfo{
							FullName: "*services.UserService",
						},
					},
				},
			},
			// PostService (singleton, depends on Database and Auditor)
			{
				Name:         "NewPostService",
				FunctionName: "NewPostService",
				PackageName:  "services",
				PackagePath:  "app/services",
				Lifecycle:    scanner.LifecycleSingleton,
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.PostService",
				},
				Dependencies: []*scanner.Field{
					{
						Name: "db",
						Type: &scanner.TypeInfo{
							FullName: "*gorm.DB",
						},
					},
					{
						Name: "auditor",
						Type: &scanner.TypeInfo{
							FullName: "*services.Auditor",
						},
					},
				},
			},
			// Logger (transient, no dependencies, only used by controllers)
			{
				Name:         "NewLogger",
				FunctionName: "NewLogger",
				PackageName:  "services",
				PackagePath:  "app/services",
				Lifecycle:    scanner.LifecycleTransient,
				ReturnType: &scanner.TypeInfo{
					FullName: "*services.Logger",
				},
				Dependencies: nil,
			},
		},
		Controllers: []*scanner.Controller{},
		Configs:     []*scanner.Config{},
	}

	// Build dependency graph
	depGraph := NewDependencyGraph(project)
	plan := depGraph.AnalyzeUsage()

	// Verify: Auditor should be in critical transients
	if len(plan.CriticalTransients) != 1 {
		t.Errorf("Expected 1 critical transient, got %d", len(plan.CriticalTransients))
	}
	if len(plan.CriticalTransients) > 0 && plan.CriticalTransients[0].Name != "NewAuditor" {
		t.Errorf("Expected NewAuditor in critical transients, got %s", plan.CriticalTransients[0].Name)
	}

	// Verify: Logger should be in non-critical transients
	if len(plan.NonCriticalTransients) != 1 {
		t.Errorf("Expected 1 non-critical transient, got %d", len(plan.NonCriticalTransients))
	}
	if len(plan.NonCriticalTransients) > 0 && plan.NonCriticalTransients[0].Name != "NewLogger" {
		t.Errorf("Expected NewLogger in non-critical transients, got %s", plan.NonCriticalTransients[0].Name)
	}

	// Verify: Database, UserService, PostService should be in singletons
	if len(plan.Singletons) != 3 {
		t.Errorf("Expected 3 singletons, got %d", len(plan.Singletons))
	}

	// Verify order: Database -> UserService -> PostService
	singletonNames := make([]string, len(plan.Singletons))
	for i, s := range plan.Singletons {
		singletonNames[i] = s.Name
	}

	// Database should come first (no dependencies)
	if singletonNames[0] != "NewDatabase" {
		t.Errorf("Expected NewDatabase first, got %s", singletonNames[0])
	}

	// UserService should come before PostService (PostService depends on UserService transitively through Auditor)
	userServiceIndex := -1
	postServiceIndex := -1
	for i, name := range singletonNames {
		if name == "NewUserService" {
			userServiceIndex = i
		}
		if name == "NewPostService" {
			postServiceIndex = i
		}
	}
	if userServiceIndex == -1 || postServiceIndex == -1 {
		t.Errorf("UserService or PostService not found in singletons")
	}
	if userServiceIndex > postServiceIndex {
		t.Errorf("Expected NewUserService (%d) before NewPostService (%d)", userServiceIndex, postServiceIndex)
	}

	// Test generation
	gen := &Generator{
		project: project,
		pkgName: "generated",
	}

	diCode, err := gen.generateDI()
	if err != nil {
		t.Fatalf("Failed to generate DI code: %v", err)
	}

	// Verify generated code structure
	// 1. auditorFactory should be defined before postService initialization
	auditorFactoryPos := strings.Index(diCode, "AuditorFactory = func()")
	postServicePos := strings.Index(diCode, "PostService = services.NewPostService")

	if auditorFactoryPos == -1 {
		t.Error("AuditorFactory not found in generated code")
	}
	if postServicePos == -1 {
		t.Error("PostService initialization not found in generated code")
	}
	if auditorFactoryPos > postServicePos {
		t.Error("AuditorFactory should be defined BEFORE PostService initialization")
	}

	// 2. Verify critical transient factory is defined before singleton that uses it
	if auditorFactoryPos > postServicePos {
		t.Error("Critical ordering: AuditorFactory must be initialized before PostService")
	}

	// 3. Verify non-critical transient factory (loggerFactory) exists
	found := strings.Contains(diCode, "LoggerFactory = func()")
	if !found {
		t.Error("LoggerFactory should be defined as a factory function")
	}
}

// TestIntegration_ComplexDependencyChain tests a more complex scenario
func TestIntegration_ComplexDependencyChain(t *testing.T) {
	// A more complex scenario:
	// Singleton A -> Transient B -> Singleton C -> Transient D -> Singleton E
	project := &scanner.Project{
		Providers: []*scanner.Provider{
			{
				Name:         "NewE",
				FunctionName: "NewE",
				PackageName:  "services",
				Lifecycle:    scanner.LifecycleSingleton,
				ReturnType:   &scanner.TypeInfo{FullName: "*services.E"},
				Dependencies: nil,
			},
			{
				Name:         "NewD",
				FunctionName: "NewD",
				PackageName:  "services",
				Lifecycle:    scanner.LifecycleTransient,
				ReturnType:   &scanner.TypeInfo{FullName: "*services.D"},
				Dependencies: []*scanner.Field{
					{Name: "e", Type: &scanner.TypeInfo{FullName: "*services.E"}},
				},
			},
			{
				Name:         "NewC",
				FunctionName: "NewC",
				PackageName:  "services",
				Lifecycle:    scanner.LifecycleSingleton,
				ReturnType:   &scanner.TypeInfo{FullName: "*services.C"},
				Dependencies: []*scanner.Field{
					{Name: "d", Type: &scanner.TypeInfo{FullName: "*services.D"}},
				},
			},
			{
				Name:         "NewB",
				FunctionName: "NewB",
				PackageName:  "services",
				Lifecycle:    scanner.LifecycleTransient,
				ReturnType:   &scanner.TypeInfo{FullName: "*services.B"},
				Dependencies: []*scanner.Field{
					{Name: "c", Type: &scanner.TypeInfo{FullName: "*services.C"}},
				},
			},
			{
				Name:         "NewA",
				FunctionName: "NewA",
				PackageName:  "services",
				Lifecycle:    scanner.LifecycleSingleton,
				ReturnType:   &scanner.TypeInfo{FullName: "*services.A"},
				Dependencies: []*scanner.Field{
					{Name: "b", Type: &scanner.TypeInfo{FullName: "*services.B"}},
				},
			},
		},
		Controllers: []*scanner.Controller{},
		Configs:     []*scanner.Config{},
	}

	depGraph := NewDependencyGraph(project)
	plan := depGraph.AnalyzeUsage()

	// Verify: B and D should be in critical transients
	if len(plan.CriticalTransients) != 2 {
		t.Errorf("Expected 2 critical transients, got %d", len(plan.CriticalTransients))
	}

	// Verify: A, C, E should be in singletons
	if len(plan.Singletons) != 3 {
		t.Errorf("Expected 3 singletons, got %d", len(plan.Singletons))
	}

	// Verify order: E should come before C, C before A
	singletonNames := make([]string, len(plan.Singletons))
	for i, s := range plan.Singletons {
		singletonNames[i] = s.Name
	}

	// Find indices
	eIndex := -1
	cIndex := -1
	aIndex := -1
	for i, name := range singletonNames {
		switch name {
		case "NewE":
			eIndex = i
		case "NewC":
			cIndex = i
		case "NewA":
			aIndex = i
		}
	}

	if eIndex == -1 || cIndex == -1 || aIndex == -1 {
		t.Errorf("Missing singletons in plan: E=%d, C=%d, A=%d", eIndex, cIndex, aIndex)
	}

	// E should come before C
	if eIndex > cIndex {
		t.Errorf("Expected E (%d) before C (%d)", eIndex, cIndex)
	}

	// C should come before A
	if cIndex > aIndex {
		t.Errorf("Expected C (%d) before A (%d)", cIndex, aIndex)
	}
}
