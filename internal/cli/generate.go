package cli

import (
	"fmt"
	"os"

	"github.com/goyave/glib/v2/internal/scanner"
	"github.com/goyave/glib/v2/internal/validator"
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
  - Request parsers (generated/parsers.gen.go)
  - Error handlers (generated/errors.gen.go)`,
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

	// Print results
	fmt.Printf("   Found %d controllers\n", len(project.Controllers))
	fmt.Printf("   Found %d providers\n", len(project.Providers))
	fmt.Printf("   Found %d middleware\n", len(project.Middleware))

	// Count total handlers
	totalHandlers := 0
	for _, ctrl := range project.Controllers {
		totalHandlers += len(ctrl.Handlers)
	}
	fmt.Printf("   Found %d handlers\n", totalHandlers)

	// Validate
	fmt.Println("\n🔍 Validating...")
	validator := validator.New()
	if err := validator.Validate(project); err != nil {
		// Print errors
		for _, verr := range validator.Errors() {
			fmt.Printf("   ❌ %s\n", verr.Message)
		}
		return err
	}

	// Print warnings
	if len(validator.Warnings()) > 0 {
		for _, warn := range validator.Warnings() {
			fmt.Printf("   ⚠️  %s\n", warn.Message)
		}
	}

	fmt.Println("✅ Validation passed")

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

	fmt.Println("\n✅ Scanning complete")
	fmt.Println("\n⚠️  Code generation not implemented yet")
	fmt.Println("   (Phase 5 - coming soon)")

	return nil
}
