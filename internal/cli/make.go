package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

			// Load .glibrc to get configuration
			cfg, err := loadGlibrc()
			if err != nil {
				return fmt.Errorf("failed to load .glibrc: %w (run 'glib init' first)", err)
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
	data, err := os.ReadFile(".glibrc")
	if err != nil {
		return nil, err
	}

	var cfg glibConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func makeController(name string, opts *makeOptions, cfg *glibConfig) error {
	// Parse name (could be "posts" or "admin/posts")
	parts := strings.Split(name, "/")
	pkgName := parts[len(parts)-1]

	// Determine output path
	outputDir := opts.path
	if outputDir == "" {
		if cfg.Make.Controllers != "" {
			outputDir = filepath.Join(cfg.Make.Controllers, name)
		} else {
			outputDir = name
		}
	}

	// Create directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine route prefix
	routePrefix := opts.prefix
	if routePrefix == "" {
		routePrefix = "/api/v1/" + strings.ReplaceAll(name, "/", "_")
	}

	// Generate controller.go
	controllerPath := filepath.Join(outputDir, "controller.go")
	controllerCode := renderController(pkgName, routePrefix, !opts.noExample)
	if err := os.WriteFile(controllerPath, []byte(controllerCode), 0o644); err != nil {
		return fmt.Errorf("failed to write controller: %w", err)
	}

	// Generate models.go
	modelsPath := filepath.Join(outputDir, "models.go")
	modelsCode := renderModels(pkgName, !opts.noExample)
	if err := os.WriteFile(modelsPath, []byte(modelsCode), 0o644); err != nil {
		return fmt.Errorf("failed to write models: %w", err)
	}

	fmt.Printf("✅ Controller created successfully!\n\n")
	fmt.Printf("   📁 %s\n", controllerPath)
	fmt.Printf("   📁 %s\n", modelsPath)
	fmt.Printf("\n📝 Next steps:\n")
	fmt.Printf("   - Add your fields to the models\n")
	fmt.Printf("   - Implement the TODO methods\n")
	fmt.Printf("   - Run: glib generate\n")

	return nil
}

func makeProvider(name string, opts *makeOptions, cfg *glibConfig) error {
	// Determine output path
	outputDir := opts.path
	if outputDir == "" {
		if cfg.Make.Providers != "" {
			outputDir = cfg.Make.Providers
		} else {
			outputDir = "providers"
		}
	}

	// Create directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate provider file
	providerPath := filepath.Join(outputDir, name+".go")
	providerCode := renderProvider(name, !opts.noExample)
	if err := os.WriteFile(providerPath, []byte(providerCode), 0o644); err != nil {
		return fmt.Errorf("failed to write provider: %w", err)
	}

	fmt.Printf("✅ Provider created successfully!\n\n")
	fmt.Printf("   📁 %s\n", providerPath)
	fmt.Printf("\n📝 Next steps:\n")
	fmt.Printf("   - Implement the provider function\n")
	fmt.Printf("   - Add dependencies as parameters\n")
	fmt.Printf("   - Run: glib generate\n")

	return nil
}

func makeMiddleware(name string, opts *makeOptions, cfg *glibConfig) error {
	// Determine output path
	outputDir := opts.path
	if outputDir == "" {
		if cfg.Make.Middleware != "" {
			outputDir = cfg.Make.Middleware
		} else {
			outputDir = "middleware"
		}
	}

	// Create directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate middleware file
	middlewarePath := filepath.Join(outputDir, name+".go")
	middlewareCode := renderMiddleware(name, !opts.noExample)
	if err := os.WriteFile(middlewarePath, []byte(middlewareCode), 0o644); err != nil {
		return fmt.Errorf("failed to write middleware: %w", err)
	}

	fmt.Printf("✅ Middleware created successfully!\n\n")
	fmt.Printf("   📁 %s\n", middlewarePath)
	fmt.Printf("\n📝 Next steps:\n")
	fmt.Printf("   - Implement the middleware logic\n")
	fmt.Printf("   - Run: glib generate\n")

	return nil
}

func renderController(pkgName, routePrefix string, withExample bool) string {
	tmplName := "controller.go.tmpl"
	if !withExample {
		tmplName = "controller_minimal.go.tmpl"
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

	result, err := executeTemplate("models.go.tmpl", map[string]any{
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

func renderMiddleware(name string, withExample bool) string {
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
