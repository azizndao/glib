package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type makeOptions struct {
	path      string
	prefix    string
	noExample bool
}

func newMakeCmd() *cobra.Command {
	opts := &makeOptions{}

	cmd := &cobra.Command{
		Use:   "make <type> <name>",
		Short: "Generate boilerplate code",
		Long: `Generate boilerplate code for controllers, providers, and middleware.

Types:
  controller  - HTTP controller with CRUD methods
  provider    - Dependency injection provider
  middleware  - HTTP middleware`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			componentType := args[0]
			name := args[1]

			// Load glib.json to get configuration
			cfg, err := loadGlibrc()
			if err != nil {
				return fmt.Errorf("failed to load glib.json: %w (run 'glib init' first)", err)
			}

			switch componentType {
			case "controller":
				return makeController(name, opts, cfg)
			case "provider":
				return makeProvider(name, opts, cfg)
			case "middleware":
				return makeMiddleware(name, opts, cfg)
			default:
				return fmt.Errorf("unknown type: %s (use: controller, provider, middleware)", componentType)
			}
		},
	}

	cmd.Flags().StringVar(&opts.path, "path", "", "File path (default: inferred from name)")
	cmd.Flags().StringVar(&opts.prefix, "prefix", "", "Route prefix for controllers")
	cmd.Flags().BoolVar(&opts.noExample, "no-example", false, "Skip example code")

	return cmd
}

func loadGlibrc() (*glibConfig, error) {
	// Try glib.json first (new format)
	data, err := os.ReadFile("glib.json")
	if err != nil {
		// Fallback to .glibrc (legacy format for backward compatibility)
		data, err = os.ReadFile(".glibrc")
		if err != nil {
			return nil, err
		}
	}

	var cfg glibConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// makeSpec holds the specification for creating a component
type makeSpec struct {
	componentType string
	name          string
	pkgName       string
	outputDir     string
	routePrefix   string
	noExample     bool
}

// makeResult holds the result of creating a component
type makeResult struct {
	files []string
	err   error
}

// prepareMakeController prepares the spec for creating a controller
func prepareMakeController(name string, opts *makeOptions, cfg *glibConfig) makeSpec {
	parts := strings.Split(name, "/")
	pkgName := parts[len(parts)-1]

	outputDir := opts.path
	if outputDir == "" {
		if cfg.Make.Controllers != "" {
			outputDir = filepath.Join(cfg.Make.Controllers, name)
		} else {
			outputDir = name
		}
	}

	routePrefix := opts.prefix
	if routePrefix == "" {
		routePrefix = "/api/v1/" + strings.ReplaceAll(name, "/", "_")
	}

	return makeSpec{
		componentType: "controller",
		name:          name,
		pkgName:       pkgName,
		outputDir:     outputDir,
		routePrefix:   routePrefix,
		noExample:     opts.noExample,
	}
}

// prepareMakeProvider prepares the spec for creating a provider
func prepareMakeProvider(name string, opts *makeOptions, cfg *glibConfig) makeSpec {
	outputDir := opts.path
	if outputDir == "" {
		if cfg.Make.Providers != "" {
			outputDir = cfg.Make.Providers
		} else {
			outputDir = "providers"
		}
	}

	return makeSpec{
		componentType: "provider",
		name:          name,
		pkgName:       name,
		outputDir:     outputDir,
		noExample:     opts.noExample,
	}
}

// prepareMakeMiddleware prepares the spec for creating middleware
func prepareMakeMiddleware(name string, opts *makeOptions, cfg *glibConfig) makeSpec {
	outputDir := opts.path
	if outputDir == "" {
		if cfg.Make.Middleware != "" {
			outputDir = cfg.Make.Middleware
		} else {
			outputDir = "middleware"
		}
	}

	return makeSpec{
		componentType: "middleware",
		name:          name,
		pkgName:       name,
		outputDir:     outputDir,
		noExample:     opts.noExample,
	}
}

// executeMake performs the actual file creation (business logic)
func executeMake(spec makeSpec) makeResult {
	files := []string{}

	// Create directory
	if err := os.MkdirAll(spec.outputDir, 0o755); err != nil {
		return makeResult{err: fmt.Errorf("failed to create directory: %w", err)}
	}

	switch spec.componentType {
	case "controller":
		controllerPath := filepath.Join(spec.outputDir, "controller.go")
		controllerCode := renderController(spec.pkgName, spec.routePrefix, !spec.noExample)
		if err := os.WriteFile(controllerPath, []byte(controllerCode), 0o644); err != nil {
			return makeResult{err: fmt.Errorf("failed to write controller: %w", err)}
		}
		files = append(files, controllerPath)

		modelsPath := filepath.Join(spec.outputDir, "models.go")
		modelsCode := renderModels(spec.pkgName, !spec.noExample)
		if err := os.WriteFile(modelsPath, []byte(modelsCode), 0o644); err != nil {
			return makeResult{err: fmt.Errorf("failed to write models: %w", err)}
		}
		files = append(files, modelsPath)

	case "provider":
		providerPath := filepath.Join(spec.outputDir, spec.name+".go")
		providerCode := renderProvider(spec.name, !spec.noExample)
		if err := os.WriteFile(providerPath, []byte(providerCode), 0o644); err != nil {
			return makeResult{err: fmt.Errorf("failed to write provider: %w", err)}
		}
		files = append(files, providerPath)

	case "middleware":
		middlewarePath := filepath.Join(spec.outputDir, spec.name+".go")
		middlewareCode := renderMiddleware(spec.name)
		if err := os.WriteFile(middlewarePath, []byte(middlewareCode), 0o644); err != nil {
			return makeResult{err: fmt.Errorf("failed to write middleware: %w", err)}
		}
		files = append(files, middlewarePath)
	}

	return makeResult{files: files}
}

func makeController(name string, opts *makeOptions, cfg *glibConfig) error {
	spec := prepareMakeController(name, opts, cfg)
	return runMake(spec)
}

func makeProvider(name string, opts *makeOptions, cfg *glibConfig) error {
	spec := prepareMakeProvider(name, opts, cfg)
	return runMake(spec)
}

func makeMiddleware(name string, opts *makeOptions, cfg *glibConfig) error {
	spec := prepareMakeMiddleware(name, opts, cfg)
	return runMake(spec)
}

// runMake executes the make operation with simple output
func runMake(spec makeSpec) error {
	start := time.Now()

	result := executeMake(spec)
	if result.err != nil {
		fmt.Println(ui.Error(result.err.Error()))
		return result.err
	}

	duration := time.Since(start)
	fmt.Println(ui.Success(fmt.Sprintf("%s created (%dms)", cases.Title(language.English).String(spec.componentType), duration.Milliseconds())))
	for _, file := range result.files {
		fmt.Printf("  %s\n", ui.Muted(file))
	}

	return nil
}

func renderController(pkgName, routePrefix string, withExample bool) string {
	tmplName := "controller.go.templ"
	if !withExample {
		tmplName = "controller_minimal.go.templ"
	}

	// Capitalize first letter for type names
	caser := cases.Title(language.English)
	typeName := caser.String(pkgName)

	result, err := executeTemplate(tmplName, map[string]any{
		"PkgName":     pkgName,
		"RoutePrefix": routePrefix,
		"TypeName":    typeName,
	})
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderModels(pkgName string, withExample bool) string {
	if !withExample {
		return fmt.Sprintf(`package %s

// Add your models here
`, pkgName)
	}

	// Capitalize first letter for type names
	caser := cases.Title(language.English)
	typeName := caser.String(pkgName)

	result, err := executeTemplate("models.go.templ", map[string]any{
		"PkgName":  pkgName,
		"TypeName": typeName,
	})
	if err != nil {
		panic(err) // Should never happen with valid templates
	}
	return result
}

func renderProvider(name string, withExample bool) string {
	// Capitalize function name
	caser := cases.Title(language.English)
	funcName := "New" + caser.String(name)

	if !withExample {
		return fmt.Sprintf(`package providers

// @Provider singleton
func %s() (any, error) {
	// TODO: implement
	return nil, nil
}
`, funcName)
	}

	// Default to database example
	return `package providers

import (
	"fmt"
	
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @Provider singleton
func NewDatabase(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.User,
		cfg.Database.Password,
	)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	return db, nil
}
`
}

func renderMiddleware(name string) string {
	// Capitalize function name
	caser := cases.Title(language.English)
	funcName := caser.String(name)

	return fmt.Sprintf(`package middleware

import (
	"net/http"
)

// @Middleware %s
func %s() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: implement middleware logic
			
			next.ServeHTTP(w, r)
		})
	}
}
`, name, funcName)
}
