package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
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
  - DI container (generated/di.gen.go)
  - Route registration (generated/routes.gen.go)  
  - Request parsers (generated/parsers.gen.go)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "Project root directory")
	cmd.Flags().StringVar(&opts.output, "output", "", "Output directory (from glib.json)")
	cmd.Flags().StringVar(&opts.config, "config", "glib.json", "Config file")
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
		return fmt.Errorf("failed to load glib.json: %w", err)
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

	return runGenerateSimple(outputDir, pkgName, opts.verbose)
}

// runGenerateSimple performs code generation with simple output
func runGenerateSimple(outputDir, pkgName string, verbose bool) error {
	start := time.Now()

	// Scan
	fmt.Println(ui.Info("Scanning project..."))
	scan, err := scanner.New(".")
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to create scanner: %v", err)))
		return err
	}

	project, err := scan.Scan()
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to scan project: %v", err)))
		return err
	}

	val := validator.New()
	if err := val.Validate(project); err != nil {
		for _, verr := range val.Errors() {
			fmt.Printf("  %s %s\n", ui.IconBullet, verr.Message)
		}
		fmt.Println(ui.Error("Validation failed"))
		return err
	}

	gen := generator.New(project, outputDir, pkgName)
	if err := gen.Generate(); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to generate code: %v", err)))
		return err
	}

	duration := time.Since(start)
	fmt.Println(ui.Success(fmt.Sprintf("Generation complete (%dms)", duration.Milliseconds())))

	if verbose {
		fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("%d controllers, %d providers, %d middleware",
			len(project.Controllers), len(project.Providers), len(project.Middleware))))
	}

	return nil
}
