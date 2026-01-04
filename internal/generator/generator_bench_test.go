package generator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

func createSimpleBenchmarkProject() *scanner.Project {
	return &scanner.Project{
		Module: "github.com/test/app",
		Providers: []*scanner.Provider{
			{
				Name:        "NewDatabase",
				PackageName: "services",
				PackagePath: "github.com/test/app/services",
				FilePath:    "/test/services/database.go",
				Lifecycle:   "singleton",
				ReturnType: &scanner.TypeInfo{
					Name:        "DB",
					FullName:    "gorm.DB",
					PackageName: "gorm",
				},
			},
		},
		Controllers: []*scanner.Controller{
			{
				Name:        "UserController",
				PackageName: "controllers",
				PackagePath: "github.com/test/app/controllers",
				FilePath:    "/test/controllers/user.go",
				RoutePrefix: "/api/users",
				Handlers: []*scanner.Handler{
					{
						Name:     "List",
						Method:   "GET",
						Path:     "/",
						FullPath: "/api/users/",
						Signature: &scanner.HandlerSignature{
							Pattern: "result",
						},
					},
				},
			},
		},
		Middleware: []*scanner.Middleware{
			{
				Name:        "AuthMiddleware",
				PackageName: "middleware",
				PackagePath: "github.com/test/app/middleware",
				FilePath:    "/test/middleware/auth.go",
			},
		},
		Configs: []*scanner.Config{
			{
				Name:        "Config",
				PackageName: "config",
				PackagePath: "github.com/test/app/config",
				FilePath:    "/test/config/config.go",
				Fields: []*scanner.ConfigField{
					{
						Name:         "Port",
						Type:         &scanner.TypeInfo{Name: "int"},
						EnvName:      "PORT",
						DefaultValue: "8080",
					},
				},
			},
		},
	}
}

func BenchmarkGenerator_Generate(b *testing.B) {
	project := createSimpleBenchmarkProject()

	b.ReportAllocs()

	for b.Loop() {
		outputDir := b.TempDir()
		gen := New(project, outputDir, "generated")
		err := gen.Generate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerator_LargeProject(b *testing.B) {
	project := createSimpleBenchmarkProject()

	// Scale up to 50 controllers with 5 handlers each
	project.Controllers = nil // Clear existing
	for i := range 50 {
		ctrl := &scanner.Controller{
			Name:        "Controller" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			PackageName: "controllers",
			PackagePath: "github.com/test/app/controllers",
			RoutePrefix: "/api/resource" + string(rune('A'+i%26)),
			Handlers:    make([]*scanner.Handler, 0, 5),
		}

		for j := range 5 {
			handler := &scanner.Handler{
				Name:     "Handler" + string(rune('A'+j)),
				Method:   "GET",
				Path:     "/" + string(rune('a'+j)),
				FullPath: ctrl.RoutePrefix + "/" + string(rune('a'+j)),
				Signature: &scanner.HandlerSignature{
					Pattern: "Result",
				},
			}
			ctrl.Handlers = append(ctrl.Handlers, handler)
		}

		project.Controllers = append(project.Controllers, ctrl)
	}

	// Scale up providers
	project.Providers = nil // Clear existing
	for i := range 30 {
		provider := &scanner.Provider{
			Name:        "NewService" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			PackageName: "services",
			PackagePath: "github.com/test/app/services",
			Lifecycle:   "singleton",
			ReturnType: &scanner.TypeInfo{
				Name: "Service" + string(rune('A'+i%26)),
			},
		}
		project.Providers = append(project.Providers, provider)
	}

	b.ReportAllocs()

	for b.Loop() {
		outputDir := b.TempDir()
		gen := New(project, outputDir, "generated")
		err := gen.Generate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark dependency graph generation
func BenchmarkDependencyGraph_Build(b *testing.B) {
	project := createSimpleBenchmarkProject()

	// Add complex dependency chains
	for i := range 20 {
		provider := &scanner.Provider{
			Name:        "NewService" + string(rune('A'+i)),
			PackageName: "services",
			PackagePath: "github.com/test/app/services",
			Lifecycle:   "singleton",
			ReturnType: &scanner.TypeInfo{
				Name: "Service" + string(rune('A'+i)),
			},
		}
		if i > 0 {
			// Each service depends on the previous one
			provider.Dependencies = []*scanner.Field{
				{
					Name: "dep",
					Type: &scanner.TypeInfo{
						Name: "Service" + string(rune('A'+i-1)),
					},
				},
			}
		}
		project.Providers = append(project.Providers, provider)
	}

	b.ReportAllocs()

	for b.Loop() {
		graph := NewDependencyGraph(project)
		_ = graph
		// Just benchmark graph creation since TopologicalSort is private
	}
}

// Benchmark memory usage
func BenchmarkGenerator_MemoryUsage(b *testing.B) {
	project := createSimpleBenchmarkProject()

	// Create larger project
	for i := range 100 {
		ctrl := &scanner.Controller{
			Name:        "Controller" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			PackageName: "controllers",
			PackagePath: "github.com/test/app/controllers",
			RoutePrefix: "/api/res" + string(rune('A'+i%26)),
			Handlers: []*scanner.Handler{
				{
					Name:      "Get",
					Method:    "GET",
					Path:      "/",
					FullPath:  "/api/res" + string(rune('A'+i%26)) + "/",
					Signature: &scanner.HandlerSignature{Pattern: "Result"},
				},
			},
		}
		project.Controllers = append(project.Controllers, ctrl)
	}

	b.ReportAllocs()

	for b.Loop() {
		outputDir := b.TempDir()
		gen := New(project, outputDir, "generated")
		err := gen.Generate()
		if err != nil {
			b.Fatal(err)
		}
	}
}
