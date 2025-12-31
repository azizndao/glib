package cli

import (
	"fmt"
	"os"

	"github.com/azizndao/glib/internal/generator"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/spf13/cobra"
)

type generateOptions struct {
	dir     string
	output  string
	config  string
	verbose bool
	watch   bool
}

func newGenerateCmd() *cobra.Command {
	opts := &generateOptions{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code from annotations",
		Long: `Scan Go source code for annotations and generate code.

Generates:
  - Bootstrap code (generated/glib.gen.go)
  - DI container (generated/di.gen.go)
  - Route registration (generated/routes.gen.go)  
  - Request parsers (generated/parsers.gen.go)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "Project root directory")
	cmd.Flags().StringVar(&opts.output, "output", "", "Output directory (from .glibrc)")
	cmd.Flags().StringVar(&opts.config, "config", ".glibrc", "Config file")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Watch mode")

	return cmd
}

func runGenerate(opts *generateOptions) error {
	if opts.watch {
		return fmt.Errorf("watch mode not implemented yet")
	}

	// Change to project directory
	if opts.dir != "." {
		if err := os.Chdir(opts.dir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Load config
	cfg, err := loadGlibrc()
	if err != nil {
		return fmt.Errorf("failed to load .glibrc: %w", err)
	}

	// Determine output directory
	outputDir := opts.output
	if outputDir == "" {
		outputDir = cfg.Generate.Output
		if outputDir == "" {
			outputDir = "generated"
		}
	}

	// Determine package name
	pkgName := cfg.Generate.Package
	if pkgName == "" {
		pkgName = "generated"
	}

	fmt.Println("🔍 Scanning project...")

	// Create scanner
	scan, err := scanner.New(".")
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Scan project
	project, err := scan.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan project: %w", err)
	}

	// Count handlers
	totalHandlers := 0
	for _, ctrl := range project.Controllers {
		totalHandlers += len(ctrl.Handlers)
	}

	// Print results
	fmt.Printf("   Found %d controllers\n", len(project.Controllers))
	fmt.Printf("   Found %d providers\n", len(project.Providers))
	fmt.Printf("   Found %d middleware\n", len(project.Middleware))
	fmt.Printf("   Found %d configs\n", len(project.Configs))
	fmt.Printf("   Found %d handlers\n", totalHandlers)

	// Validate
	fmt.Println("\n🔍 Validating...")
	val := validator.New()
	if err := val.Validate(project); err != nil {
		// Print errors
		for _, verr := range val.Errors() {
			fmt.Printf("   ❌ %s\n", verr.Message)
		}
		return err
	}

	// Print warnings
	if len(val.Warnings()) > 0 {
		for _, warn := range val.Warnings() {
			fmt.Printf("   ⚠️  %s\n", warn.Message)
		}
	}

	fmt.Println("✅ Validation passed")

	// Generate code
	fmt.Println("\n🔨 Generating code...")

	gen := generator.New(project, outputDir, pkgName)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	fmt.Printf("   ✓ %s/glib.gen.go\n", outputDir)
	fmt.Printf("   ✓ %s/di.gen.go\n", outputDir)
	fmt.Printf("   ✓ %s/routes.gen.go\n", outputDir)
	fmt.Printf("   ✓ %s/parsers.gen.go\n", outputDir)

	fmt.Println("\n✅ Generation complete!")

	if opts.verbose {
		fmt.Println("\n📋 Controllers:")
		for _, ctrl := range project.Controllers {
			fmt.Printf("   - %s (%s)\n", ctrl.Name, ctrl.RoutePrefix)
			for _, handler := range ctrl.Handlers {
				fmt.Printf("     • %s %s → %s()\n", handler.Method, handler.FullPath, handler.Name)
			}
		}

		fmt.Println("\n📋 Providers:")
		for _, prov := range project.Providers {
			fmt.Printf("   - %s() [%s]\n", prov.FunctionName, prov.Lifecycle)
		}

		fmt.Println("\n📋 Middleware:")
		for _, mw := range project.Middleware {
			fmt.Printf("   - %s (from %s())\n", mw.Name, mw.FunctionName)
		}
	}

	fmt.Println("\n📝 Next steps:")
	fmt.Println("   go run .")

	return nil
}
