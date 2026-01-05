package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type generateOptions struct {
	dir        string
	output     string
	verbose    bool
	watch      bool
	workers    int
	noCache    bool
	clearCache bool
}

func newGenerateCmd() *cobra.Command {
	opts := &generateOptions{}

	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "Generate code from annotations",
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
	cmd.Flags().StringVar(&opts.output, "output", "", "Output directory (default: from .config.toml or 'generated')")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output (default: from .config.toml or false)")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Watch mode")
	cmd.Flags().IntVar(&opts.workers, "workers", 0, "Number of parallel workers (default: from .config.toml or 4)")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Disable file caching (default: cache enabled from .config.toml)")
	cmd.Flags().BoolVar(&opts.clearCache, "clear-cache", false, "Clear cache before scanning")

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

	// Load config from environment variables and defaults
	cfg, err := loadConfigs()
	if err != nil {
		return err
	}

	// Priority resolution: CLI args > environment variables > defaults

	// Verbose: CLI flag OR config value
	if !opts.verbose {
		opts.verbose = cfg.Verbose
	}

	// Workers: CLI flag (if not default/0) OR config value
	if opts.workers == 0 {
		opts.workers = cfg.Generate.Workers
	}

	// Cache: CLI flag --no-cache OR config value
	if !opts.noCache {
		opts.noCache = !cfg.Generate.Cache
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

	return runGenerateSimple(outputDir, pkgName, cfg, opts)
}

// runGenerateSimple performs code generation with simple output
func runGenerateSimple(outputDir, pkgName string, cfg *glibConfig, opts *generateOptions) error {
	codegenOpts := &CodegenOptions{
		ProjectDir:   ".",
		OutputDir:    outputDir,
		PackageName:  pkgName,
		Workers:      opts.workers,
		NoCache:      opts.noCache,
		Verbose:      opts.verbose,
		ClearCache:   opts.clearCache,
		ChangedFiles: nil,  // Always full scan for generate command
		ShowProgress: true, // Generate command shows detailed output
	}

	return PerformCodeGeneration(cfg, codegenOpts)
}
