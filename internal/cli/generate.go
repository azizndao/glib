package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/generator"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/spf13/cobra"
)

type generateOptions struct {
	dir        string
	output     string
	config     string
	verbose    bool
	watch      bool
	workers    int
	noCache    bool
	clearCache bool
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
	cmd.Flags().IntVar(&opts.workers, "workers", 0, "Number of parallel workers (0 = auto, -1 = disable)")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Disable file caching")
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

	return runGenerateSimple(outputDir, pkgName, cfg, opts)
}

// runGenerateSimple performs code generation with simple output
func runGenerateSimple(outputDir, pkgName string, cfg *glibConfig, opts *generateOptions) error {
	start := time.Now()

	// Clear cache if requested
	if opts.clearCache {
		cacheDir := filepath.Join(".glib", "cache")
		if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to clear cache: %v", err)))
		} else if opts.verbose {
			fmt.Println(ui.Info("Cache cleared"))
		}
	}

	// Configure scanner options
	var scanOpts []scanner.ScannerOption

	// Enable caching unless explicitly disabled
	if !opts.noCache {
		cacheDir := filepath.Join(".glib", "cache")
		scanOpts = append(scanOpts, scanner.WithCache(cacheDir))
		if opts.verbose {
			fmt.Println(ui.Info("File caching enabled"))
		}
	}

	// Enable parallel scanning if workers > 0 or auto (0)
	if opts.workers != -1 {
		workers := opts.workers
		if workers == 0 {
			workers = 4 // Default to 4 workers
		}
		scanOpts = append(scanOpts, scanner.WithParallel(workers))
		if opts.verbose {
			fmt.Printf("  %s Parallel scanning with %d workers\n", ui.IconBullet, workers)
		}
	}

	// Scan
	fmt.Println(ui.Info("Scanning project..."))
	scan, err := scanner.New(".", scanOpts...)
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to create scanner: %v", err)))
		return err
	}

	var project *scanner.Project
	if opts.verbose {
		// Use streaming for verbose mode to show progress
		project, err = scanWithProgress(scan, opts)
	} else {
		project, err = scan.Scan()
	}

	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to scan project: %v", err)))
		return err
	}

	// Validate with incremental validation if caching enabled
	var valStats *validator.ValidationStats
	if !opts.noCache {
		cacheDir := filepath.Join(".glib", "cache")
		incVal := validator.NewIncrementalValidator(cacheDir)
		if err := incVal.ValidateIncremental(project); err != nil {
			if validationErr, ok := err.(*validator.ValidationErrors); ok {
				for _, verr := range validationErr.Errors {
					fmt.Printf("  %s %s\n", ui.IconBullet, verr.Message)
				}
			}
			fmt.Println(ui.Error("Validation failed"))
			return err
		}
		stats := incVal.Stats()
		valStats = &stats
	} else {
		// Use regular validator
		val := validator.New()
		if err := val.Validate(project); err != nil {
			for _, verr := range val.Errors() {
				fmt.Printf("  %s %s\n", ui.IconBullet, verr.Message)
			}
			fmt.Println(ui.Error("Validation failed"))
			return err
		}
	}

	// Display scan summary if verbose
	if opts.verbose {
		fmt.Println()
		printScanSummary(scan.Stats(), valStats, !opts.noCache)
	}

	// Build validation config from glib.json
	// Auto-enable if languages are specified
	validationEnabled := cfg.Validation.Enabled || len(cfg.Validation.Languages) > 0
	validationCfg := generator.ValidationConfig{
		Enabled:         validationEnabled,
		Languages:       cfg.Validation.Languages,
		DefaultLanguage: cfg.Validation.DefaultLanguage,
	}

	gen := generator.NewWithValidation(project, outputDir, pkgName, validationCfg)
	if err := gen.Generate(); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to generate code: %v", err)))
		return err
	}

	duration := time.Since(start)
	fmt.Println(ui.Success(fmt.Sprintf("Generation complete (%dms)", duration.Milliseconds())))

	// Format generated code
	if err := formatGeneratedCode(outputDir, opts.verbose); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Failed to format code: %v", err)))
		// Don't return error, formatting is not critical
	}

	if opts.verbose {
		fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf("%d controllers, %d providers, %d middleware",
			len(project.Controllers), len(project.Providers), len(project.Middleware))))
	}

	return nil
}

// formatGeneratedCode runs goimports and gofmt on generated files
func formatGeneratedCode(outputDir string, verbose bool) error {
	// Find all .go files in output directory
	pattern := filepath.Join(outputDir, "*.gen.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find generated files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	if verbose {
		fmt.Println(ui.Info("Formatting generated code..."))
	}

	// Try goimports first (removes unused imports and formats)
	if err := runGoImports(files, verbose); err != nil {
		// Fall back to gofmt if goimports is not available
		if verbose {
			fmt.Println(ui.Warning("goimports not found, using gofmt only"))
		}
		return runGoFmt(files, verbose)
	}

	return nil
}

// runGoImports runs goimports on the specified files
func runGoImports(files []string, verbose bool) error {
	args := append([]string{"-w"}, files...)
	cmd := exec.Command("goimports", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// runGoFmt runs gofmt on the specified files
func runGoFmt(files []string, verbose bool) error {
	args := append([]string{"-w"}, files...)
	cmd := exec.Command("gofmt", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// scanWithProgress runs scanner with streaming progress updates
func scanWithProgress(scan *scanner.Scanner, opts *generateOptions) (*scanner.Project, error) {
	events := make(chan scanner.StreamEvent, 100)

	// Start goroutine to consume events
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range events {
			switch event.Type {
			case scanner.EventProvider:
				fmt.Printf("  %s %s %s\n",
					ui.Cyan+ui.IconProvider+ui.Reset,
					ui.Muted("Provider:"),
					ui.Primary(event.Provider.PackageName+"."+event.Provider.Name))
			case scanner.EventController:
				fmt.Printf("  %s %s %s\n",
					ui.Blue+ui.IconController+ui.Reset,
					ui.Muted("Controller:"),
					ui.Primary(event.Controller.PackageName+"."+event.Controller.Name))
			case scanner.EventMiddleware:
				fmt.Printf("  %s %s %s\n",
					ui.Yellow+ui.IconMiddleware+ui.Reset,
					ui.Muted("Middleware:"),
					ui.Primary(event.Middleware.PackageName+"."+event.Middleware.Name))
			case scanner.EventConfig:
				fmt.Printf("  %s %s %s\n",
					ui.Gray+ui.IconConfig+ui.Reset,
					ui.Muted("Config:"),
					ui.Primary(event.Config.PackageName))
			case scanner.EventProgress:
				// Could add progress bar here
			case scanner.EventError:
				fmt.Println(ui.Warning(fmt.Sprintf("Warning: %v", event.Error)))
			}
		}
	}()

	// Run scan with streaming
	project, err := scan.ScanWithStream(events)
	<-done // Wait for event processing to complete

	return project, err
}

// printScanSummary prints a formatted summary table of scan statistics
func printScanSummary(scanStats scanner.ScanStats, valStats *validator.ValidationStats, cacheEnabled bool) {
	fmt.Println(ui.BoldText("  Scan Summary:"))
	fmt.Println(ui.Muted("  ┌────────────────────────┬──────────────┐"))

	// Components found
	fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" %12s "+ui.Muted("│")+"\n",
		"Providers", ui.Cyan+fmt.Sprintf("%d", scanStats.Providers)+ui.Reset)
	fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" %12s "+ui.Muted("│")+"\n",
		"Controllers", ui.Blue+fmt.Sprintf("%d", scanStats.Controllers)+ui.Reset)
	fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" %12s "+ui.Muted("│")+"\n",
		"Middleware", ui.Yellow+fmt.Sprintf("%d", scanStats.Middleware)+ui.Reset)
	fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" %12s "+ui.Muted("│")+"\n",
		"Handlers", ui.Green+fmt.Sprintf("%d", scanStats.Handlers)+ui.Reset)

	if cacheEnabled && scanStats.FilesScanned > 0 {
		fmt.Println(ui.Muted("  ├────────────────────────┼──────────────┤"))

		hitRate := float64(scanStats.CacheHits) * 100 / float64(scanStats.FilesScanned)
		fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" %12d "+ui.Muted("│")+"\n",
			"Files Scanned", scanStats.FilesScanned)

		cacheHitStr := fmt.Sprintf("%d", scanStats.CacheHits)
		fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" "+ui.Green+"%-12s"+ui.Reset+" "+ui.Muted("│")+"\n",
			"Cache Hits", fmt.Sprintf("%s (%.1f%%)", cacheHitStr, hitRate))

		fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" "+ui.Yellow+"%-12d"+ui.Reset+" "+ui.Muted("│")+"\n",
			"Cache Misses", scanStats.CacheMisses)
	}

	if valStats != nil && cacheEnabled {
		fmt.Println(ui.Muted("  ├────────────────────────┼──────────────┤"))

		valHitRate := float64(valStats.CacheHits) * 100 / float64(valStats.ComponentsValidated)
		fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" %12d "+ui.Muted("│")+"\n",
			"Components Validated", valStats.ComponentsValidated)

		valCacheStr := fmt.Sprintf("%d", valStats.CacheHits)
		fmt.Printf(ui.Muted("  │")+" %-22s "+ui.Muted("│")+" "+ui.Green+"%-12s"+ui.Reset+" "+ui.Muted("│")+"\n",
			"Validation Cached", fmt.Sprintf("%s (%.1f%%)", valCacheStr, valHitRate))
	}

	fmt.Println(ui.Muted("  └────────────────────────┴──────────────┘"))
}
