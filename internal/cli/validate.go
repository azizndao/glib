package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	dir     string
	verbose bool
}

func newValidateCmd() *cobra.Command {
	opts := &validateOptions{}

	cmd := &cobra.Command{
		Use:     "validate",
		Aliases: []string{"check"},
		Short:   "Validate project structure and annotations",
		Long: `Validate project structure and annotations without generating code.

Checks:
  - DI graph (circular dependencies, missing providers)
  - Routes (conflicts, parameter mismatches)
  - Types (signature validation)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "Project root directory")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output")

	return cmd
}

func runValidate(opts *validateOptions) error {
	// Change to project directory
	if opts.dir != "." {
		if err := os.Chdir(opts.dir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Load config from .config.toml
	cfg, err := loadConfigs()
	if err != nil {
		return err
	}

	// Priority: CLI flags > config > defaults
	if !opts.verbose {
		opts.verbose = cfg.Verbose
	}

	return runValidateSimple(cfg, opts)
}

// runValidateSimple performs validation with simple output
func runValidateSimple(cfg *glibConfig, opts *validateOptions) error {
	start := time.Now()

	fmt.Println(ui.Infof("Scanning project..."))

	// Build scanner options from config
	var scanOpts []scanner.ScannerOption

	// Enable parallel scanning if workers configured
	if cfg.Generate.Workers > 0 {
		scanOpts = append(scanOpts, scanner.WithParallel(cfg.Generate.Workers))
	}

	// Enable caching if configured
	if cfg.Generate.Cache {
		cacheDir := filepath.Join(".glib", "cache")
		scanOpts = append(scanOpts, scanner.WithCache(cacheDir))
	}

	// Apply file filtering from watch config
	if len(cfg.Watch.ExcludeDirs) > 0 {
		scanOpts = append(scanOpts, scanner.WithExcludeDirs(cfg.Watch.ExcludeDirs))
	}
	// Note: We don't pass include_files to scanner because it always scans *.go files
	// The watch config's include_files is for file watching (e.g., *.toml for locale changes)
	if len(cfg.Watch.ExcludeFiles) > 0 {
		scanOpts = append(scanOpts, scanner.WithExcludeFiles(cfg.Watch.ExcludeFiles))
	}

	// Enable i18n scanning if configured
	if cfg.I18n.Enabled && cfg.I18n.LocalesDir != "" {
		scanOpts = append(scanOpts, scanner.WithI18n(cfg.I18n.LocalesDir))
	}

	// Create scanner with options
	scan, err := scanner.New(".", scanOpts...)
	if err != nil {
		fmt.Println(ui.Errorf("Failed to create scanner: %v", err))
		return err
	}

	project, err := scan.Scan()
	if err != nil {
		fmt.Println(ui.Errorf("Failed to scan project: %v", err))
		return err
	}

	v := validator.New()
	if err := v.Validate(project); err != nil {
		fmt.Println(ui.Errorf("Validation failed"))
		for i, verr := range v.Errors() {
			fmt.Printf("  %d. %s\n", i+1, verr.Message)
		}
		return err
	}

	duration := time.Since(start)
	warnings := v.Warnings()

	if len(warnings) > 0 {
		fmt.Println(ui.Successf("Validation passed (%dms)", duration.Milliseconds()))
		fmt.Println(ui.Warningf("%d warnings", len(warnings)))
		if opts.verbose {
			for i, warn := range warnings {
				fmt.Printf("  %d. %s\n", i+1, warn.Message)
			}
		}
	} else {
		fmt.Println(ui.Successf("Validation passed (%dms)", duration.Milliseconds()))
	}

	if opts.verbose {
		fmt.Printf("  %s\n", ui.Mutedf(
			"%d controllers, %d providers, %d middleware",
			len(project.Controllers),
			len(project.Providers),
			len(project.Middleware),
		))
	}

	return nil
}
