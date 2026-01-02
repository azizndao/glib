package cli

import (
	"fmt"
	"os"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
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
	cmd.Flags().StringVar(&opts.output, "output", "", "Output directory (default: generated, override via GLIB_OUTPUT)")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output (default: false, override via GLIB_VERBOSE)")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Watch mode")
	cmd.Flags().IntVar(&opts.workers, "workers", 0, "Number of parallel workers (default: 4, override via GLIB_WORKERS)")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Disable file caching (default: cache enabled, override via GLIB_CACHE)")
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
	cfg := getDefaultConfig()

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
