package validator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

func createLargeProject() *scanner.Project {
	project := &scanner.Project{
		Module: "testproject",
	}

	// Create 50 providers
	for i := 0; i < 50; i++ {
		provider := &scanner.Provider{
			Name:        "NewService" + string(rune('A'+i%26)),
			PackagePath: "testproject",
			Lifecycle:   "singleton",
			ReturnType: &scanner.TypeInfo{
				Name:     "Service" + string(rune('A'+i%26)),
				FullName: "Service" + string(rune('A'+i%26)),
			},
		}
		project.Providers = append(project.Providers, provider)
	}

	// Create 20 controllers
	for i := 0; i < 20; i++ {
		controller := &scanner.Controller{
			Name:        "Controller" + string(rune('A'+i)),
			PackagePath: "testproject",
			RoutePrefix: "/api/v" + string(rune('1'+i)),
			Fields: []*scanner.Field{
				{
					Name: "DB",
					Type: &scanner.TypeInfo{
						Name:     "ServiceA",
						FullName: "ServiceA",
					},
				},
			},
			Handlers: []*scanner.Handler{
				{
					Name:     "List",
					Method:   "GET",
					Path:     "/",
					FullPath: "/api/v" + string(rune('1'+i)) + "/",
				},
			},
		}
		project.Controllers = append(project.Controllers, controller)
	}

	return project
}

func BenchmarkValidation(b *testing.B) {
	project := createLargeProject()

	b.Run("Regular", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validator := New()
			validator.Validate(project)
		}
	})

	b.Run("Incremental_FirstRun", func(b *testing.B) {
		cacheDir := b.TempDir()
		for i := 0; i < b.N; i++ {
			validator := NewIncrementalValidator(cacheDir)
			validator.ValidateIncremental(project)
		}
	})

	b.Run("Incremental_CachedRun", func(b *testing.B) {
		cacheDir := b.TempDir()

		// Warm up cache
		validator := NewIncrementalValidator(cacheDir)
		validator.ValidateIncremental(project)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validator := NewIncrementalValidator(cacheDir)
			validator.ValidateIncremental(project)
		}
	})
}

func BenchmarkIncrementalValidation_OneChangeOnly(b *testing.B) {
	project := createLargeProject()
	cacheDir := b.TempDir()

	// Warm up cache with full validation
	validator := NewIncrementalValidator(cacheDir)
	validator.ValidateIncremental(project)

	// Modify just one provider
	project.Providers[0].Lifecycle = "transient"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator := NewIncrementalValidator(cacheDir)
		validator.ValidateIncremental(project)

		// Flip back and forth to simulate changes
		if i%2 == 0 {
			project.Providers[0].Lifecycle = "singleton"
		} else {
			project.Providers[0].Lifecycle = "transient"
		}
	}
}
