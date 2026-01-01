package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		module  string
		example bool
		minimal bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new Glib project",
		Long: `Initialize a new Glib project with scaffolding.

Creates a new project with:
  - main.go (application entry point)
  - configs/config.go (configuration struct with @Config annotation)
  - glib.json (Glib configuration)
  - .gitignore (Git ignore file)

Optional:
  --example  Include example health check controller
  --minimal  Minimal setup (no examples, no comments)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			return initProject(dir, module, example, minimal)
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module name (auto-detected if not specified)")
	cmd.Flags().BoolVar(&example, "example", false, "Include example health check controller")
	cmd.Flags().BoolVar(&minimal, "minimal", false, "Minimal setup (no examples, no comments)")

	return cmd
}

// initSpec holds the specification for initializing a project
type initSpec struct {
	absDir  string
	module  string
	example bool
	minimal bool
}

// initResult holds the result of initializing a project
type initResult struct {
	createdFiles []string
	err          error
}

// prepareInit validates and prepares the init specification
func prepareInit(dir, module string, example, minimal bool) (initSpec, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return initSpec{}, fmt.Errorf("failed to create directory: %w", err)
	}

	// Get absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return initSpec{}, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Infer module name if not provided
	if module == "" {
		module = inferModuleName(absDir)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return initSpec{}, fmt.Errorf("failed to read directory: %w", err)
	}
	if len(entries) > 0 {
		// Allow if only .git directory exists
		if len(entries) > 1 || entries[0].Name() != ".git" {
			return initSpec{}, fmt.Errorf("directory %s is not empty", dir)
		}
	}

	return initSpec{
		absDir:  absDir,
		module:  module,
		example: example,
		minimal: minimal,
	}, nil
}

// executeInit performs the actual project initialization (business logic)
func executeInit(spec initSpec) initResult {
	createdFiles := []string{}
	files := buildProjectFiles(spec.module, spec.example, spec.minimal)

	for path, content := range files {
		fullPath := filepath.Join(spec.absDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return initResult{err: fmt.Errorf("failed to create directory for %s: %w", path, err)}
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return initResult{err: fmt.Errorf("failed to write %s: %w", path, err)}
		}
		createdFiles = append(createdFiles, path)
	}

	// Initialize go.mod if it doesn't exist
	goModPath := filepath.Join(spec.absDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		if err := createGoMod(spec.absDir, spec.module); err != nil {
			return initResult{err: fmt.Errorf("failed to initialize go.mod: %w", err)}
		}
		createdFiles = append(createdFiles, "go.mod")
	}

	return initResult{createdFiles: createdFiles}
}

func initProject(dir, module string, example, minimal bool) error {
	spec, err := prepareInit(dir, module, example, minimal)
	if err != nil {
		return err
	}

	return runInit(spec)
}

// runInit executes the init operation with simple output
func runInit(spec initSpec) error {
	start := time.Now()

	result := executeInit(spec)
	if result.err != nil {
		fmt.Println(ui.Error(result.err.Error()))
		return result.err
	}

	duration := time.Since(start)
	fmt.Println(ui.Success(fmt.Sprintf("Project initialized (%dms)", duration.Milliseconds())))
	fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("Module: %s", spec.module)))
	fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("Files: %d", len(result.createdFiles))))

	return nil
}

func inferModuleName(absPath string) string {
	// Try to get from git remote
	// For now, just use the directory name
	base := filepath.Base(absPath)
	return strings.ToLower(base)
}

func createGoMod(dir, module string) error {
	content := fmt.Sprintf(`module %s

go 1.25
`, module)

	return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644)
}

func buildProjectFiles(module string, example, minimal bool) map[string]string {
	files := make(map[string]string)

	// main.go
	files["main.go"] = renderMainGo(module, minimal)

	// bootstrap.go - User's bootstrap function
	files["bootstrap.go"] = renderBootstrapGo(module)

	// configs/config.go (in separate package to avoid import cycle)
	files["configs/config.go"] = renderConfigGo(minimal)

	// glib.json
	files["glib.json"] = renderGlibRC()

	// .gitignore
	files[".gitignore"] = renderGitignore()

	// README.md (unless minimal)
	if !minimal {
		files["README.md"] = renderReadme(module)
	}

	// Example controller
	if example {
		files["health/controller.go"] = renderHealthController()
	}

	return files
}

func renderMainGo(module string, minimal bool) string {
	tmplName := "main.go.templ"
	if minimal {
		tmplName = "main_minimal.go.templ"
	}

	result, err := executeTemplate(tmplName, map[string]any{"Module": module})
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderBootstrapGo(module string) string {
	result, err := executeTemplate("bootstrap.go.templ", map[string]any{"Module": module})
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderConfigGo(minimal bool) string {
	tmplName := "config.go.templ"
	if minimal {
		tmplName = "config_minimal.go.templ"
	}

	result, err := executeTemplate(tmplName, nil)
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderGlibRC() string {
	// glib.json is JSON, keeping it inline since it's simple and doesn't need templating
	return `{
  "version": "2",
  "generate": {
    "output": "generated",
    "package": "generated"
  },
  "make": {
    "controllers": "controllers",
    "providers": "providers",
    "middleware": "middleware"
  },
  "dev": {
    "port": 8080
  }
}
`
}

func renderGitignore() string {
	result, err := executeTemplate("gitignore.templ", nil)
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderReadme(module string) string {
	result, err := executeTemplate("readme.md.templ", map[string]any{"Module": module})
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderHealthController() string {
	result, err := executeTemplate("health_controller.go.templ", nil)
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}
